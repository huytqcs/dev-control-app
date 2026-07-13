package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"devctl/internal/applog"
)

// gitWatchDebounce absorbs the burst of filesystem events a single git
// operation produces (e.g. `checkout` writes HEAD.lock then renames it onto
// HEAD) so RefreshGitState runs once per user action, not once per event.
const gitWatchDebounce = 300 * time.Millisecond

// StartGitWatcher watches every configured service's .git/HEAD and
// refs/heads/ for changes made outside this app — e.g. `git checkout` or
// `git switch` run in an external terminal — and refreshes that service's
// in-memory git state when they change. Without this, branch/ahead/behind
// only got re-read at startup or after the app's own git actions
// (RefreshGitState's doc comment, T-058), so an external branch switch was
// invisible until the app happened to run its own git command.
//
// Safe to call with zero watchable repos (e.g. all noop in tests); it just
// runs an idle loop until ctx is done.
func (m *Manager) StartGitWatcher(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("git watcher: %w", err)
	}

	m.mu.RLock()
	ids := append([]string(nil), m.order...)
	m.mu.RUnlock()

	headDirs := make(map[string][]string)
	refsDirs := make(map[string][]string)

	for _, id := range ids {
		sr, err := m.getRuntime(id)
		if err != nil {
			continue
		}
		sr.mu.RLock()
		repoPath := sr.config.Path
		sr.mu.RUnlock()

		gitDir := resolveGitDir(repoPath)
		if gitDir == "" {
			continue
		}
		if err := watcher.Add(gitDir); err == nil {
			headDirs[gitDir] = append(headDirs[gitDir], id)
		}

		refsHeads := filepath.Join(gitDir, "refs", "heads")
		if err := watcher.Add(refsHeads); err == nil {
			refsDirs[refsHeads] = append(refsDirs[refsHeads], id)
		}
	}

	go m.runGitWatchLoop(ctx, watcher, headDirs, refsDirs)
	return nil
}

func (m *Manager) runGitWatchLoop(ctx context.Context, watcher *fsnotify.Watcher, headDirs, refsDirs map[string][]string) {
	defer watcher.Close()

	var mu sync.Mutex
	timers := make(map[string]*time.Timer)

	trigger := func(id string) {
		mu.Lock()
		defer mu.Unlock()
		if t, ok := timers[id]; ok {
			t.Stop()
		}
		timers[id] = time.AfterFunc(gitWatchDebounce, func() {
			if _, err := m.RefreshGitState(ctx, id); err != nil {
				applog.Error("runtime", "git: watch-triggered refresh for %q: %v", id, err)
			}
		})
	}

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			dir := filepath.Dir(event.Name)
			if filepath.Base(event.Name) == "HEAD" {
				for _, id := range headDirs[dir] {
					trigger(id)
				}
				continue
			}
			for _, id := range refsDirs[dir] {
				trigger(id)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			applog.Error("runtime", "git watcher error: %v", err)
		}
	}
}

// resolveGitDir returns repoPath's real .git directory, following the
// "gitdir: <path>" indirection used by worktrees and submodules where .git
// is a file rather than a directory. Returns "" if repoPath isn't a git repo.
func resolveGitDir(repoPath string) string {
	dotGit := filepath.Join(repoPath, ".git")
	info, err := os.Stat(dotGit)
	if err != nil {
		return ""
	}
	if info.IsDir() {
		return dotGit
	}

	data, err := os.ReadFile(dotGit)
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(data))
	const prefix = "gitdir:"
	if !strings.HasPrefix(line, prefix) {
		return ""
	}
	dir := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(repoPath, dir)
	}
	return dir
}

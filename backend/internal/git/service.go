// Package git executes repo-scoped git actions via the git CLI, not a Go git
// library (ARCHITECTURE.md §18) — it's a thin, testable wrapper around
// commands a human would run themselves.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	networkTimeout = 60 * time.Second
)

type Status struct {
	Branch string
	Dirty  bool
	Ahead  int
	Behind int
}

type Service struct{}

func NewService() *Service { return &Service{} }

// Status reads branch, dirty, and ahead/behind-of-upstream state. A repo
// with no upstream configured for the current branch (a common, normal case
// — e.g. a fresh local branch) reports Ahead/Behind as 0 rather than an
// error.
func (s *Service) Status(ctx context.Context, repoPath string) (Status, error) {
	branch, err := runGit(ctx, repoPath, defaultTimeout, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return Status{}, fmt.Errorf("branch: %w", err)
	}

	dirtyOut, err := runGit(ctx, repoPath, defaultTimeout, "status", "--porcelain")
	if err != nil {
		return Status{}, fmt.Errorf("dirty check: %w", err)
	}

	ahead, behind := 0, 0
	if counts, err := runGit(ctx, repoPath, defaultTimeout, "rev-list", "--left-right", "--count", "HEAD...@{upstream}"); err == nil {
		ahead, behind = parseAheadBehind(counts)
	}

	return Status{
		Branch: strings.TrimSpace(branch),
		Dirty:  strings.TrimSpace(dirtyOut) != "",
		Ahead:  ahead,
		Behind: behind,
	}, nil
}

// ListBranches returns every local branch plus every remote-tracking
// branch's short name (e.g. "origin/feature-x" contributes "feature-x"),
// deduplicated and sorted. `git checkout <name>` on a remote-only name
// auto-creates a local tracking branch (git's own DWIM behavior), so
// surfacing remote names here is what lets a search-and-checkout UI reach
// branches that haven't been checked out locally yet.
func (s *Service) ListBranches(ctx context.Context, repoPath string) ([]string, error) {
	out, err := runGit(ctx, repoPath, defaultTimeout, "for-each-ref",
		"--format=%(refname)", "refs/heads", "refs/remotes")
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}

	seen := make(map[string]bool)
	branches := []string{}
	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		branches = append(branches, name)
	}

	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "refs/heads/"):
			add(strings.TrimPrefix(line, "refs/heads/"))
		case strings.HasPrefix(line, "refs/remotes/"):
			rest := strings.TrimPrefix(line, "refs/remotes/")
			// rest is "<remote>/<branch...>" — drop the remote segment, and
			// skip the "<remote>/HEAD" pointer ref, not a real branch.
			if idx := strings.Index(rest, "/"); idx != -1 {
				if name := rest[idx+1:]; name != "HEAD" {
					add(name)
				}
			}
		}
	}

	sort.Strings(branches)
	return branches, nil
}

func (s *Service) Fetch(ctx context.Context, repoPath string) error {
	_, err := runGit(ctx, repoPath, networkTimeout, "fetch")
	return err
}

func (s *Service) Pull(ctx context.Context, repoPath string) error {
	_, err := runGit(ctx, repoPath, networkTimeout, "pull")
	return err
}

func (s *Service) Push(ctx context.Context, repoPath string) error {
	_, err := runGit(ctx, repoPath, networkTimeout, "push")
	return err
}

// validRefName rejects anything that isn't a plausible branch/ref name —
// most importantly a leading "-", which git would otherwise interpret as a
// flag instead of a ref (SPEC.md §29: don't expose raw shell/flag injection
// from a text input, even in a locally-trusted tool).
var validRefName = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)

func (s *Service) Checkout(ctx context.Context, repoPath, branch string) error {
	if err := checkRefName(branch); err != nil {
		return err
	}
	_, err := runGit(ctx, repoPath, defaultTimeout, "checkout", branch)
	return err
}

// CreateBranch creates and checks out a new branch from an existing ref
// (Plan.md §C "maybe v1.1" — checkout-only wasn't enough once someone wants
// to start new work from main). Both names go through the same validation
// as Checkout since `from` is just as attacker/typo-controlled as `name`.
func (s *Service) CreateBranch(ctx context.Context, repoPath, name, from string) error {
	if err := checkRefName(name); err != nil {
		return err
	}
	if err := checkRefName(from); err != nil {
		return err
	}
	_, err := runGit(ctx, repoPath, defaultTimeout, "checkout", "-b", name, from)
	return err
}

func checkRefName(ref string) error {
	if ref == "" || strings.HasPrefix(ref, "-") || !validRefName.MatchString(ref) {
		return fmt.Errorf("invalid branch name %q", ref)
	}
	return nil
}

func parseAheadBehind(out string) (ahead, behind int) {
	fields := strings.Fields(out)
	if len(fields) != 2 {
		return 0, 0
	}
	ahead, _ = strconv.Atoi(fields[0])
	behind, _ = strconv.Atoi(fields[1])
	return ahead, behind
}

func runGit(ctx context.Context, repoPath string, timeout time.Duration, args ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, "git", append([]string{"-C", repoPath}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}

// Package actions runs one-off config-defined commands (config.ActionConfig)
// such as "db migrate" or "npm install". Per ARCHITECTURE.md §9.8/§12.3, an
// action run is deliberately its own execution flow — its own run ID, output
// stream, and success/failure result — and must not be hacked into service
// runtime state (internal/runtime.Manager owns that instead).
package actions

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"devctl/internal/config"
	"devctl/internal/logs"
	"devctl/internal/runtime"
)

// ErrActionAlreadyRunning is returned by Run when a run is already in flight
// for the same (serviceID, actionID) pair.
var ErrActionAlreadyRunning = errors.New("action already running")

// Service runs config-defined actions as one-off OS processes, streaming
// their output into the shared logs.Manager and publishing WS events, without
// touching internal/runtime's service/worker state.
type Service struct {
	logs      *logs.Manager
	publisher runtime.EventPublisher

	mu      sync.Mutex
	nextID  uint64
	running map[string]bool // key: serviceID + ":" + actionID
}

// NewService constructs a Service. publisher may be nil, in which case
// published events are discarded (mirrors runtime.NewManager's default).
func NewService(logMgr *logs.Manager, publisher runtime.EventPublisher) *Service {
	if publisher == nil {
		publisher = runtime.NoopPublisher{}
	}
	return &Service{
		logs:      logMgr,
		publisher: publisher,
		running:   make(map[string]bool),
	}
}

// Run starts action's command in dir and returns immediately with a run ID;
// output streaming and the completion result happen asynchronously in a
// goroutine. Only one run per (serviceID, action.ID) pair may be in flight at
// a time — a second call while one is active returns ErrActionAlreadyRunning
// without starting a process.
func (s *Service) Run(ctx context.Context, serviceID string, action config.ActionConfig, dir string) (string, error) {
	key := serviceID + ":" + action.ID

	s.mu.Lock()
	if s.running[key] {
		s.mu.Unlock()
		return "", ErrActionAlreadyRunning
	}
	s.running[key] = true
	s.nextID++
	runID := "action_" + strconv.FormatUint(s.nextID, 10)
	s.mu.Unlock()

	clearRunning := func() {
		s.mu.Lock()
		delete(s.running, key)
		s.mu.Unlock()
	}

	if len(action.Command) == 0 {
		clearRunning()
		return "", fmt.Errorf("action %q: command must not be empty", action.ID)
	}

	// Plain exec.Command, not exec.CommandContext(ctx, ...): ctx here is
	// typically an HTTP request's context, which the server cancels the
	// moment the handler returns — but Run starts the process and returns
	// immediately while it keeps running in the background (that's the
	// whole point of returning a run ID instead of blocking). Tying the
	// process to that context would SIGKILL it almost instantly, as soon as
	// the response is written. Matches OSProcessRunner.Start's same
	// plain-exec.Command choice in internal/runtime/process_runner.go, for
	// the same reason.
	shell, shellArgs := loginShellCommand(action.Command)
	cmd := exec.Command(shell, shellArgs...)
	cmd.Dir = dir
	cmd.Env = mergeEnv(os.Environ(), action.Env)
	// Setsid, not Setpgid: matches OSProcessRunner.Start in
	// internal/runtime/process_runner.go — an interactive login shell (-i,
	// needed to source nvm/rbenv/Homebrew PATH setup) fights for the
	// controlling terminal under Setpgid and gets SIGTTOU'd into a frozen
	// state. Setsid gives it a fresh session with nothing to fight over.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		clearRunning()
		return "", fmt.Errorf("action %q: stdout pipe: %w", action.ID, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		clearRunning()
		return "", fmt.Errorf("action %q: stderr pipe: %w", action.ID, err)
	}

	if err := cmd.Start(); err != nil {
		clearRunning()
		return "", fmt.Errorf("action %q: start: %w", action.ID, err)
	}

	streamKey := logs.ActionStreamKey(serviceID, action.ID, runID)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		s.pipeLog(serviceID, action.ID, runID, streamKey, "stdout", stdout)
	}()
	go func() {
		defer wg.Done()
		s.pipeLog(serviceID, action.ID, runID, streamKey, "stderr", stderr)
	}()

	go func() {
		// Wait for both pipes to be fully drained before calling cmd.Wait():
		// per os/exec's docs, calling Wait before reads from StdoutPipe/
		// StderrPipe complete can truncate output, since Wait closes the
		// pipes as soon as the process exits.
		wg.Wait()
		waitErr := cmd.Wait()

		exitCode := 0
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		success := waitErr == nil && exitCode == 0
		errMsg := ""
		if !success && waitErr != nil {
			errMsg = waitErr.Error()
		}

		clearRunning()

		s.publisher.Publish(runtime.AppEvent{
			Type:      runtime.EventActionCompleted,
			ServiceID: serviceID,
			Payload: runtime.ActionCompletedPayload{
				RunID:    runID,
				ActionID: action.ID,
				ExitCode: exitCode,
				Success:  success,
				Error:    errMsg,
			},
			Time: time.Now(),
		})
	}()

	return runID, nil
}

func (s *Service) pipeLog(serviceID, actionID, runID, streamKey, source string, r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		entry := s.logs.Append(streamKey, source, scanner.Text(), time.Now())
		s.publisher.Publish(runtime.AppEvent{
			Type:      runtime.EventActionOutput,
			ServiceID: serviceID,
			Payload: runtime.ActionOutputPayload{
				RunID:    runID,
				ActionID: actionID,
				Entry:    entry,
			},
			Time: time.Now(),
		})
	}
}

// loginShellCommand, resolveShell, shellJoin, and mergeEnv are re-implemented
// here rather than imported from internal/runtime: those helpers are
// unexported there, and internal/actions is meant to stand on its own
// (mirroring how internal/git already duplicates this instead of depending on
// internal/runtime). Keep these in sync with process_runner.go by hand if the
// technique ever changes.

// loginShellCommand wraps command so it runs through the user's own login
// shell instead of being exec'd directly, so nvm/rbenv/Homebrew PATH setup
// from .zprofile/.zshrc is available (same rationale as
// internal/runtime/process_runner.go).
func loginShellCommand(command []string) (string, []string) {
	return resolveShell(), []string{"-lic", shellJoin(command)}
}

// resolveShell picks a real, executable login shell. $SHELL is trusted only
// if it actually exists on disk.
func resolveShell() string {
	candidates := []string{os.Getenv("SHELL"), "/bin/zsh", "/bin/bash", "/bin/sh"}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}
	return "/bin/sh"
}

// shellJoin quotes each argument for safe inclusion in a shell -c string.
func shellJoin(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
	}
	return strings.Join(quoted, " ")
}

// mergeEnv appends overrides on top of base, matching the merge approach in
// internal/runtime/process_runner.go's mergeEnv.
func mergeEnv(base []string, overrides map[string]string) []string {
	env := append([]string{}, base...)
	for k, v := range overrides {
		env = append(env, k+"="+v)
	}
	return env
}

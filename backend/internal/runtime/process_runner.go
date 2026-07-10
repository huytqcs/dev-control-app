// Package runtime owns process lifecycle: starting/stopping OS processes and
// tracking the in-memory state of services and their workers.
package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

type ProcessOptions struct {
	ID      string
	Command []string
	Dir     string
	Env     map[string]string
}

// RunningProcess is a handle to a started OS process. Exactly one goroutine
// (spawned internally by Start) ever calls cmd.Wait(); callers observe exit
// via Done()/ExitCode()/ExitErr() instead of touching Cmd directly.
type RunningProcess struct {
	ID     string
	Cmd    *exec.Cmd
	PID    int
	Stdout io.ReadCloser
	Stderr io.ReadCloser

	done     chan struct{}
	mu       sync.Mutex
	exitErr  error
	exitCode int
}

func (p *RunningProcess) Done() <-chan struct{} { return p.done }

func (p *RunningProcess) ExitCode() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitCode
}

func (p *RunningProcess) ExitErr() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitErr
}

type ProcessRunner interface {
	Start(ctx context.Context, opts ProcessOptions) (*RunningProcess, error)
	Stop(proc *RunningProcess, timeout time.Duration) error
}

// OSProcessRunner starts processes directly via os/exec, in their own process
// group, so Stop can signal the whole tree (e.g. a shell wrapping a real
// server) rather than just the immediate child.
type OSProcessRunner struct{}

func NewOSProcessRunner() *OSProcessRunner { return &OSProcessRunner{} }

func (r *OSProcessRunner) Start(ctx context.Context, opts ProcessOptions) (*RunningProcess, error) {
	if len(opts.Command) == 0 {
		return nil, fmt.Errorf("process %q: command must not be empty", opts.ID)
	}

	shell, shellArgs := loginShellCommand(opts.Command)
	cmd := exec.Command(shell, shellArgs...)
	cmd.Dir = opts.Dir
	cmd.Env = mergeEnv(os.Environ(), opts.Env)
	// Setsid (not just Setpgid): the interactive login shell (-i, for
	// sourcing nvm/rbenv-style .zshrc setups) tries to take control of its
	// controlling terminal on startup. Setpgid alone keeps the child in the
	// parent's session/tty, so it's a background process group trying to
	// grab a foreground-only privilege — the kernel SIGTTOUs it and it
	// freezes (state "T") before ever running the real command. Setsid
	// gives it a fresh session with no controlling terminal at all, so
	// there's nothing to fight over.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("process %q: stdout pipe: %w", opts.ID, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("process %q: stderr pipe: %w", opts.ID, err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("process %q: start: %w", opts.ID, err)
	}

	proc := &RunningProcess{
		ID:     opts.ID,
		Cmd:    cmd,
		PID:    cmd.Process.Pid,
		Stdout: stdout,
		Stderr: stderr,
		done:   make(chan struct{}),
	}

	go func() {
		waitErr := cmd.Wait()
		proc.mu.Lock()
		proc.exitErr = waitErr
		if cmd.ProcessState != nil {
			proc.exitCode = cmd.ProcessState.ExitCode()
		}
		proc.mu.Unlock()
		close(proc.done)
	}()

	return proc, nil
}

// Stop sends SIGTERM to the process group and waits up to timeout for it to
// exit, escalating to SIGKILL if it doesn't.
func (r *OSProcessRunner) Stop(proc *RunningProcess, timeout time.Duration) error {
	if proc == nil || proc.Cmd.Process == nil {
		return nil
	}

	select {
	case <-proc.Done():
		return nil
	default:
	}

	if err := syscall.Kill(-proc.PID, syscall.SIGTERM); err != nil && err != syscall.ESRCH {
		return fmt.Errorf("process %q: SIGTERM: %w", proc.ID, err)
	}

	select {
	case <-proc.Done():
		return nil
	case <-time.After(timeout):
		_ = syscall.Kill(-proc.PID, syscall.SIGKILL)
		<-proc.Done()
		return nil
	}
}

// loginShellCommand wraps command so it runs through the user's own login
// shell instead of being exec'd directly. A bare exec.Command only inherits
// this process's raw environment, which typically lacks whatever a real
// terminal session sets up via ~/.zprofile / ~/.zshrc (Homebrew's shellenv,
// nvm, rbenv, direnv, etc.) — so "npm"/"rails"/etc. can resolve to the wrong
// binary or fail to resolve at all. Running via `$SHELL -lic "<command>"`
// sources the same profile/rc files a human typing the command would get,
// matching what devctl's config author actually tested locally.
func loginShellCommand(command []string) (string, []string) {
	return resolveShell(), []string{"-lic", shellJoin(command)}
}

// resolveShell picks a real, executable login shell. $SHELL is trusted only
// if it actually exists on disk — it can be stale (e.g. inherited from a
// different environment) or point to a shell that isn't installed here.
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

func mergeEnv(base []string, overrides map[string]string) []string {
	env := append([]string{}, base...)
	for k, v := range overrides {
		env = append(env, k+"="+v)
	}
	return env
}

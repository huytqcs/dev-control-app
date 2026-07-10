package runtime

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// reconcileProbeTimeout bounds each per-service port dial at startup so a
// workspace with many services doesn't stall backend boot.
const reconcileProbeTimeout = 500 * time.Millisecond

// ReconcileOrphans probes each stopped service's configured port and, if
// something is already listening, adopts it as "running" instead of showing
// a stale "stopped". Services run Setsid'd (their own session) so they
// survive a devctl backend restart as orphans holding their port — this
// makes displayed state match reality without killing anything (BETA_PLAN
// orphan-reconciliation decision, option 1). Best-effort: a service with no
// configured port can't be probed and is left as "stopped".
func (m *Manager) ReconcileOrphans(ctx context.Context) {
	m.mu.RLock()
	ids := append([]string(nil), m.order...)
	m.mu.RUnlock()

	for _, id := range ids {
		sr, err := m.getRuntime(id)
		if err != nil {
			continue
		}
		sr.mu.RLock()
		alreadyTracked := sr.state.Status != ServiceStopped
		port := sr.config.Port
		checks := sr.config.HealthChecks
		sr.mu.RUnlock()
		if alreadyTracked || port <= 0 {
			continue
		}
		if !probeTCPPort(port, reconcileProbeTimeout) {
			continue
		}

		sr.mu.Lock()
		sr.state.Status = ServiceRunning
		sr.mu.Unlock()
		m.health.Start(id, checks)
		m.emit(EventServiceUpdated, id, sr.snapshot())
		log.Printf("reconcile: adopted orphaned service %q on port %d as running", id, port)
	}
}

func probeTCPPort(port int, timeout time.Duration) bool {
	if port <= 0 {
		return false
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// ForceKillPort kills whatever process is listening on port, regardless of
// whether this backend instance believes it owns that process. Used both as
// StopService's fallback for orphan-adopted services (no in-memory process
// handle to signal) and as the manual force-kill escape hatch (BETA_PLAN
// option 3, for when ReconcileOrphans guesses wrong).
func ForceKillPort(port int) error {
	if port <= 0 {
		return fmt.Errorf("no port configured")
	}

	out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port)).Output()
	trimmed := strings.TrimSpace(string(out))
	if err != nil {
		// lsof exits non-zero (with empty output) when nothing is listening —
		// that's success ("nothing to kill"), not a failure to report.
		if trimmed == "" {
			return nil
		}
		return fmt.Errorf("lsof :%d: %w", port, err)
	}

	var killErr error
	for _, field := range strings.Fields(trimmed) {
		pid, convErr := strconv.Atoi(field)
		if convErr != nil {
			continue
		}
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
			killErr = err
		}
	}
	return killErr
}

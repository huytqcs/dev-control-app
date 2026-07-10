// Package health runs TCP/HTTP checks against running services. Health is
// modeled separately from process-running state (ARCHITECTURE.md §17): a
// service can have a running process but still be starting up or failing its
// checks.
package health

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"devctl/internal/config"
)

type Status string

const (
	StatusUnknown   Status = "unknown"
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
)

const defaultCheckTimeout = 2 * time.Second

// Probe runs every configured check for a service and reports healthy only
// if all of them pass.
func Probe(ctx context.Context, checks []config.HealthCheck) Status {
	if len(checks) == 0 {
		return StatusUnknown
	}
	for _, c := range checks {
		if !probeOne(ctx, c) {
			return StatusUnhealthy
		}
	}
	return StatusHealthy
}

func probeOne(ctx context.Context, c config.HealthCheck) bool {
	timeout := time.Duration(c.Timeout) * time.Second
	if timeout <= 0 {
		timeout = defaultCheckTimeout
	}
	switch c.Type {
	case "tcp":
		return ProbeTCP(ctx, c.Port, timeout)
	case "http":
		return probeHTTP(ctx, c.URL, timeout)
	default:
		return false
	}
}

// ProbeTCP reports whether something accepts connections on localhost:port.
// Dials the "localhost" hostname rather than the literal 127.0.0.1 — some
// dev servers (Node 17+'s default DNS resolution order under things like
// `ng serve`/webpack-dev-server) bind only the IPv6 loopback (::1), not the
// IPv4 one. A literal 127.0.0.1 dial would get connection-refused against a
// server like that even though it's genuinely up and reachable in a
// browser (which resolves "localhost" the same dual-stack way). Dialing the
// hostname lets net.Dialer's built-in Happy-Eyeballs fallback try both.
func ProbeTCP(ctx context.Context, port int, timeout time.Duration) bool {
	if port <= 0 {
		return false
	}
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort("localhost", strconv.Itoa(port)))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func probeHTTP(ctx context.Context, url string, timeout time.Duration) bool {
	if url == "" {
		return false
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

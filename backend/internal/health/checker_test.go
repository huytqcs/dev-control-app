package health

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"
)

func listenerPort(t *testing.T, ln net.Listener) int {
	t.Helper()
	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}
	return port
}

// TestProbeTCP_IPv6OnlyListener guards against a real bug: some dev servers
// (Node 17+'s default DNS resolution order under things like
// `ng serve`/webpack-dev-server) bind only the IPv6 loopback, not the IPv4
// one — a service like that is genuinely reachable in a browser at
// http://localhost:<port> but a literal-127.0.0.1 TCP probe reports it
// unhealthy anyway.
func TestProbeTCP_IPv6OnlyListener(t *testing.T) {
	ln, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback not available in this environment: %v", err)
	}
	defer ln.Close()

	if !ProbeTCP(context.Background(), listenerPort(t, ln), time.Second) {
		t.Fatalf("expected ProbeTCP to succeed against an IPv6-only loopback listener")
	}
}

func TestProbeTCP_IPv4OnlyListener(t *testing.T) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("IPv4 loopback not available in this environment: %v", err)
	}
	defer ln.Close()

	if !ProbeTCP(context.Background(), listenerPort(t, ln), time.Second) {
		t.Fatalf("expected ProbeTCP to succeed against an IPv4-only loopback listener")
	}
}

func TestProbeTCP_NothingListening(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := listenerPort(t, ln)
	ln.Close() // free the port so nothing is listening on it

	if ProbeTCP(context.Background(), port, 200*time.Millisecond) {
		t.Fatalf("expected ProbeTCP to fail when nothing is listening")
	}
}

func TestProbeTCP_InvalidPort(t *testing.T) {
	if ProbeTCP(context.Background(), 0, time.Second) {
		t.Fatalf("expected ProbeTCP to fail for port 0")
	}
}

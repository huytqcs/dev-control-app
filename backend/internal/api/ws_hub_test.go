package api

import "testing"

// TestBroadcast_DropsMessageNotClient guards against a real bug: a burst of
// events (a SIGTERM'd process dumping dozens of log lines in a millisecond,
// which happens in practice with Ruby's `debug` gem) could fill a client's
// send buffer, and the old behavior deleted the client and closed its send
// channel over that — disconnecting the browser's WebSocket over a
// transient burst, losing every event for the whole reconnect window, not
// just the one that overflowed. Broadcast must drop the one message and
// leave the client attached.
func TestBroadcast_DropsMessageNotClient(t *testing.T) {
	hub := NewHub()
	client := &wsClient{send: make(chan []byte, 2)}
	hub.register(client)

	hub.Broadcast([]byte("1"))
	hub.Broadcast([]byte("2"))
	// Buffer (capacity 2) is now full — this one must be dropped, not the
	// client connection.
	hub.Broadcast([]byte("3"))

	hub.mu.Lock()
	_, stillRegistered := hub.clients[client]
	hub.mu.Unlock()
	if !stillRegistered {
		t.Fatalf("client was unregistered on a full buffer — should only drop the message")
	}

	// The channel must still be open (a closed channel would make this
	// send panic) and still deliver once drained.
	if got := <-client.send; string(got) != "1" {
		t.Fatalf("expected first buffered message %q, got %q", "1", got)
	}
	if got := <-client.send; string(got) != "2" {
		t.Fatalf("expected second buffered message %q, got %q", "2", got)
	}

	hub.Broadcast([]byte("4"))
	if got := <-client.send; string(got) != "4" {
		t.Fatalf("expected message after drain %q, got %q", "4", got)
	}
}

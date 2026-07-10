package logs

import (
	"fmt"
	"sync"
	"time"
)

// DefaultBufferCapacity is the per-stream line cap (SPEC.md §13.2 recommends
// 2,000-5,000; TASKS.md T-015 suggests 3,000).
const DefaultBufferCapacity = 3000

// Manager owns one ring buffer per stream key and hands out monotonically
// increasing entry IDs.
type Manager struct {
	mu      sync.Mutex
	streams map[string]*RingBuffer
	cap     int
	nextID  uint64
}

func NewManager() *Manager {
	return &Manager{streams: make(map[string]*RingBuffer), cap: DefaultBufferCapacity}
}

func (m *Manager) stream(key string) *RingBuffer {
	m.mu.Lock()
	defer m.mu.Unlock()
	rb, ok := m.streams[key]
	if !ok {
		rb = NewRingBuffer(m.cap)
		m.streams[key] = rb
	}
	return rb
}

// Append records a line and returns the created entry, so the caller can
// forward it to the WebSocket hub as a log.appended event.
func (m *Manager) Append(streamKey, source, line string, at time.Time) LogEntry {
	m.mu.Lock()
	m.nextID++
	id := m.nextID
	m.mu.Unlock()

	e := LogEntry{
		ID:        fmt.Sprintf("log_%d", id),
		StreamKey: streamKey,
		Source:    source,
		Line:      line,
		Time:      at,
	}
	m.stream(streamKey).Append(e)
	return e
}

func (m *Manager) Recent(streamKey string, limit int) []LogEntry {
	return m.stream(streamKey).Recent(limit)
}

func ServiceStreamKey(serviceID string) string {
	return "service:" + serviceID
}

func WorkerStreamKey(serviceID, workerID string) string {
	return "worker:" + serviceID + ":" + workerID
}

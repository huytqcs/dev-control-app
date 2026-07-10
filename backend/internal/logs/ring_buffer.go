// Package logs holds recent stdout/stderr lines in memory per stream, so the
// API/WS layer can serve "recent logs" without a database.
package logs

import (
	"sync"
	"time"
)

type LogEntry struct {
	ID        string    `json:"id"`
	StreamKey string    `json:"streamKey"`
	Source    string    `json:"source"` // stdout | stderr
	Line      string    `json:"line"`
	Time      time.Time `json:"time"`
	Level     string    `json:"level,omitempty"`
}

// RingBuffer is a fixed-capacity circular buffer of LogEntry, bounding memory
// use regardless of how long a service has been running.
type RingBuffer struct {
	mu      sync.RWMutex
	entries []LogEntry
	start   int
	count   int
	cap     int
}

func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = 1
	}
	return &RingBuffer{entries: make([]LogEntry, capacity), cap: capacity}
}

func (b *RingBuffer) Append(e LogEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	idx := (b.start + b.count) % b.cap
	b.entries[idx] = e
	if b.count < b.cap {
		b.count++
	} else {
		b.start = (b.start + 1) % b.cap
	}
}

// Recent returns up to limit most-recent entries, oldest first. limit <= 0
// means "all currently buffered entries".
func (b *RingBuffer) Recent(limit int) []LogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if limit <= 0 || limit > b.count {
		limit = b.count
	}
	result := make([]LogEntry, limit)
	skip := b.count - limit
	for i := 0; i < limit; i++ {
		idx := (b.start + skip + i) % b.cap
		result[i] = b.entries[idx]
	}
	return result
}

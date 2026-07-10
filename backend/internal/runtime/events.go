package runtime

import (
	"time"

	"devctl/internal/logs"
)

type EventType string

const (
	EventServiceUpdated EventType = "service.updated"
	EventLogAppended    EventType = "log.appended"
	EventHealthUpdated  EventType = "health.updated"
	EventGitUpdated     EventType = "git.updated"
	EventWorkerUpdated  EventType = "worker.updated"
)

type AppEvent struct {
	Type      EventType   `json:"type"`
	ServiceID string      `json:"serviceId,omitempty"`
	Payload   interface{} `json:"payload"`
	Time      time.Time   `json:"time"`
}

type LogAppendedPayload struct {
	Entry logs.LogEntry `json:"entry"`
}

type HealthUpdatedPayload struct {
	Health HealthState `json:"health"`
}

type GitUpdatedPayload struct {
	Git GitState `json:"git"`
}

type WorkerUpdatedPayload struct {
	Worker WorkerState `json:"worker"`
}

// EventPublisher receives runtime events for fan-out (e.g. to WebSocket
// clients). Publish must not block the runtime manager.
type EventPublisher interface {
	Publish(AppEvent)
}

// NoopPublisher discards events; useful in tests that don't care about them.
type NoopPublisher struct{}

func (NoopPublisher) Publish(AppEvent) {}

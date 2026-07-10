// Package applog provides a tiny structured-logging helper used in place of
// scattered, ad hoc log.Printf calls across devctl's backend (TASKS.md
// T-081). It intentionally stays stdlib-only (no zerolog/zap/logrus): every
// call site just wants a consistent, greppable line shape — a level tag and
// a component tag — not a new dependency.
package applog

import (
	"fmt"
	"log"
)

// Info logs a routine, expected event for component (e.g. a state
// transition or informational notice).
func Info(component, format string, args ...any) {
	logf("INFO", component, format, args...)
}

// Warn logs a recoverable problem for component — something worth a
// developer's attention but that didn't stop the operation.
func Warn(component, format string, args ...any) {
	logf("WARN", component, format, args...)
}

// Error logs a failed operation for component.
func Error(component, format string, args ...any) {
	logf("ERROR", component, format, args...)
}

// logf renders "[LEVEL] component: message" and writes it via the standard
// log package, which prepends its own date/time prefix.
func logf(level, component, format string, args ...any) {
	log.Printf("[%s] %s: %s", level, component, fmt.Sprintf(format, args...))
}

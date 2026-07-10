package applog

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

// withCapturedOutput redirects the standard logger's output for the
// duration of fn and returns what was written.
func withCapturedOutput(fn func()) string {
	var buf bytes.Buffer
	prevOut := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	}()
	fn()
	return buf.String()
}

func TestInfoProducesOutput(t *testing.T) {
	out := withCapturedOutput(func() {
		Info("test", "hello %s", "world")
	})
	if !strings.Contains(out, "[INFO]") {
		t.Errorf("expected output to contain [INFO], got %q", out)
	}
	if !strings.Contains(out, "test:") {
		t.Errorf("expected output to contain component tag %q, got %q", "test:", out)
	}
	if !strings.Contains(out, "hello world") {
		t.Errorf("expected output to contain formatted message, got %q", out)
	}
}

func TestWarnProducesOutput(t *testing.T) {
	out := withCapturedOutput(func() {
		Warn("test", "something looks off: %d", 42)
	})
	if !strings.Contains(out, "[WARN]") {
		t.Errorf("expected output to contain [WARN], got %q", out)
	}
	if !strings.Contains(out, "something looks off: 42") {
		t.Errorf("expected output to contain formatted message, got %q", out)
	}
}

func TestErrorProducesOutput(t *testing.T) {
	out := withCapturedOutput(func() {
		Error("test", "failed: %v", "boom")
	})
	if !strings.Contains(out, "[ERROR]") {
		t.Errorf("expected output to contain [ERROR], got %q", out)
	}
	if !strings.Contains(out, "failed: boom") {
		t.Errorf("expected output to contain formatted message, got %q", out)
	}
}

func TestLoggersDoNotPanicWithNoArgs(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("logging without args panicked: %v", r)
		}
	}()
	withCapturedOutput(func() {
		Info("test", "plain message")
		Warn("test", "plain message")
		Error("test", "plain message")
	})
}

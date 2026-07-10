package openhelpers

import (
	"strings"
	"testing"
)

func TestBrowserCommand(t *testing.T) {
	cmd := browserCommand("https://example.com")

	if !strings.HasSuffix(cmd.Path, "open") {
		t.Errorf("expected command path to resolve to 'open', got %q", cmd.Path)
	}
	wantArgs := []string{"open", "https://example.com"}
	if !equalArgs(cmd.Args, wantArgs) {
		t.Errorf("browserCommand args = %v, want %v", cmd.Args, wantArgs)
	}
}

func TestRepoCommand(t *testing.T) {
	cmd := repoCommand("/tmp/some-repo")

	if !strings.HasSuffix(cmd.Path, "open") {
		t.Errorf("expected command path to resolve to 'open', got %q", cmd.Path)
	}
	wantArgs := []string{"open", "/tmp/some-repo"}
	if !equalArgs(cmd.Args, wantArgs) {
		t.Errorf("repoCommand args = %v, want %v", cmd.Args, wantArgs)
	}
}

func TestTerminalCommand(t *testing.T) {
	cmd := terminalCommand("/tmp/some-repo")

	if !strings.HasSuffix(cmd.Path, "open") {
		t.Errorf("expected command path to resolve to 'open', got %q", cmd.Path)
	}
	wantArgs := []string{"open", "-a", "Terminal", "/tmp/some-repo"}
	if !equalArgs(cmd.Args, wantArgs) {
		t.Errorf("terminalCommand args = %v, want %v", cmd.Args, wantArgs)
	}
}

func TestOpenBrowser_EmptyURL(t *testing.T) {
	err := OpenBrowser("")
	if err == nil {
		t.Fatal("expected non-nil error for empty url, got nil")
	}
}

func TestOpenRepo_EmptyPath(t *testing.T) {
	err := OpenRepo("")
	if err == nil {
		t.Fatal("expected non-nil error for empty path, got nil")
	}
}

func TestOpenTerminal_EmptyPath(t *testing.T) {
	err := OpenTerminal("")
	if err == nil {
		t.Fatal("expected non-nil error for empty path, got nil")
	}
}

func equalArgs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

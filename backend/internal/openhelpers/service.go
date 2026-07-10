// Package openhelpers provides thin OS integration helpers for opening a
// browser, folder, or terminal on macOS. See docs/ARCHITECTURE.md §19
// "Open helper architecture" — these are not core domain logic, just
// wrappers around the macOS `open` command.
package openhelpers

import (
	"fmt"
	"os/exec"
)

// browserCommand constructs (without running) the command used to open a
// URL in the default browser.
func browserCommand(url string) *exec.Cmd {
	return exec.Command("open", url)
}

// repoCommand constructs (without running) the command used to open a
// folder in Finder.
func repoCommand(path string) *exec.Cmd {
	return exec.Command("open", path)
}

// terminalCommand constructs (without running) the command used to open
// Terminal.app at a given path.
func terminalCommand(path string) *exec.Cmd {
	return exec.Command("open", "-a", "Terminal", path)
}

// OpenBrowser opens url in the default browser via macOS `open <url>`.
func OpenBrowser(url string) error {
	if url == "" {
		return fmt.Errorf("no url configured")
	}
	return browserCommand(url).Run()
}

// OpenRepo opens path in Finder via macOS `open <path>`.
func OpenRepo(path string) error {
	if path == "" {
		return fmt.Errorf("no path configured")
	}
	return repoCommand(path).Run()
}

// OpenTerminal opens Terminal.app at path via macOS `open -a Terminal <path>`.
func OpenTerminal(path string) error {
	if path == "" {
		return fmt.Errorf("no path configured")
	}
	return terminalCommand(path).Run()
}

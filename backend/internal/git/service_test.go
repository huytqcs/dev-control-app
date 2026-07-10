package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "init", "-q", "-b", "main")
	run(t, dir, "config", "user.email", "test@example.com")
	run(t, dir, "config", "user.name", "Test")

	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	run(t, dir, "add", "file.txt")
	run(t, dir, "commit", "-q", "-m", "initial")
	return dir
}

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestStatus_CleanRepo(t *testing.T) {
	dir := initRepo(t)
	svc := NewService()

	status, err := svc.Status(context.Background(), dir)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Branch != "main" {
		t.Fatalf("expected branch main, got %q", status.Branch)
	}
	if status.Dirty {
		t.Fatalf("expected clean repo, got dirty")
	}
}

func TestStatus_DirtyRepo(t *testing.T) {
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	svc := NewService()
	status, err := svc.Status(context.Background(), dir)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !status.Dirty {
		t.Fatalf("expected dirty repo")
	}
}

func TestCheckout_SwitchesBranch(t *testing.T) {
	dir := initRepo(t)
	run(t, dir, "branch", "feature-x")

	svc := NewService()
	if err := svc.Checkout(context.Background(), dir, "feature-x"); err != nil {
		t.Fatalf("Checkout: %v", err)
	}

	status, err := svc.Status(context.Background(), dir)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Branch != "feature-x" {
		t.Fatalf("expected branch feature-x, got %q", status.Branch)
	}
}

func TestCheckout_RejectsInvalidRef(t *testing.T) {
	dir := initRepo(t)
	svc := NewService()

	cases := []string{"", "-force", "--hard", "; rm -rf /"}
	for _, branch := range cases {
		if err := svc.Checkout(context.Background(), dir, branch); err == nil {
			t.Fatalf("expected Checkout(%q) to be rejected, got nil error", branch)
		}
	}
}

func TestFetch_NotARepoFails(t *testing.T) {
	dir := t.TempDir()
	svc := NewService()

	if err := svc.Fetch(context.Background(), dir); err == nil {
		t.Fatalf("expected Fetch to fail outside a git repo")
	}
}

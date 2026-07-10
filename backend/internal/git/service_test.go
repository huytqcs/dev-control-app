package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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

func TestCreateBranch_CreatesAndChecksOut(t *testing.T) {
	dir := initRepo(t)
	svc := NewService()

	if err := svc.CreateBranch(context.Background(), dir, "feature-y", "main"); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}

	status, err := svc.Status(context.Background(), dir)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Branch != "feature-y" {
		t.Fatalf("expected branch feature-y, got %q", status.Branch)
	}
}

func TestCreateBranch_RejectsInvalidRef(t *testing.T) {
	dir := initRepo(t)
	svc := NewService()

	cases := []string{"", "-force", "--hard", "; rm -rf /"}
	for _, invalid := range cases {
		if err := svc.CreateBranch(context.Background(), dir, invalid, "main"); err == nil {
			t.Fatalf("expected CreateBranch(name=%q) to be rejected, got nil error", invalid)
		}
		if err := svc.CreateBranch(context.Background(), dir, "feature-z", invalid); err == nil {
			t.Fatalf("expected CreateBranch(from=%q) to be rejected, got nil error", invalid)
		}
	}
}

func TestListBranches_LocalOnly(t *testing.T) {
	dir := initRepo(t)
	run(t, dir, "branch", "feature-a")
	run(t, dir, "branch", "feature-b")

	svc := NewService()
	branches, err := svc.ListBranches(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	want := []string{"feature-a", "feature-b", "main"}
	if !reflect.DeepEqual(branches, want) {
		t.Fatalf("got %v, want %v", branches, want)
	}
}

func TestListBranches_IncludesRemoteTrackingDeduped(t *testing.T) {
	remote := t.TempDir()
	run(t, remote, "init", "-q", "--bare", "-b", "main")

	local := initRepo(t)
	run(t, local, "remote", "add", "origin", remote)
	run(t, local, "push", "-q", "origin", "main")
	run(t, local, "push", "-q", "origin", "main:remote-only")
	run(t, local, "fetch", "-q", "origin")

	svc := NewService()
	branches, err := svc.ListBranches(context.Background(), local)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	// "main" is both a local branch and a remote-tracking ref — must be
	// deduplicated to one entry. "remote-only" only exists on the remote.
	want := []string{"main", "remote-only"}
	if !reflect.DeepEqual(branches, want) {
		t.Fatalf("got %v, want %v", branches, want)
	}
}

func TestListBranches_EmptyRepoReturnsEmptySlice(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, "init", "-q", "-b", "main")

	svc := NewService()
	branches, err := svc.ListBranches(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	if branches == nil || len(branches) != 0 {
		t.Fatalf("expected empty non-nil slice, got %#v", branches)
	}
}

func TestFetch_NotARepoFails(t *testing.T) {
	dir := t.TempDir()
	svc := NewService()

	if err := svc.Fetch(context.Background(), dir); err == nil {
		t.Fatalf("expected Fetch to fail outside a git repo")
	}
}

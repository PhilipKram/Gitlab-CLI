package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestGitRepo creates a temporary git repository for testing git functions
// that require a real repo. Returns the repo path and a cleanup function.
func setupTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Initialize a git repo
	runInDir := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	runInDir("init", "-b", "main")
	runInDir("config", "user.email", "test@test.com")
	runInDir("config", "user.name", "Test")

	// Create an initial commit
	testFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runInDir("add", "README.md")
	runInDir("commit", "-m", "Initial commit")

	// Add a remote
	runInDir("remote", "add", "origin", "https://gitlab.com/test-owner/test-repo.git")
	runInDir("remote", "add", "upstream", "git@gitlab.example.com:other-owner/other-repo.git")

	return dir
}

func TestDefaultBranch_SymbolicRef(t *testing.T) {
	dir := setupTestGitRepo(t)

	// Save and restore working directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	// Set up a symbolic ref for origin/HEAD -> origin/main
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("setting up symbolic ref: %v", err)
	}

	branch, err := DefaultBranch("origin")
	if err != nil {
		t.Fatalf("DefaultBranch: %v", err)
	}
	if branch != "main" {
		t.Errorf("DefaultBranch = %q, want %q", branch, "main")
	}
}

func TestDefaultBranch_Fallback(t *testing.T) {
	dir := setupTestGitRepo(t)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	// Create a fake remote branch ref so the fallback detection works
	// Create refs/remotes/origin/main
	cmd := exec.Command("git", "update-ref", "refs/remotes/origin/main", "HEAD")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("update-ref: %v", err)
	}

	// Don't set symbolic ref, so it falls back to checking main/master
	branch, err := DefaultBranch("origin")
	if err != nil {
		t.Fatalf("DefaultBranch fallback: %v", err)
	}
	if branch != "main" {
		t.Errorf("DefaultBranch fallback = %q, want %q", branch, "main")
	}
}

func TestDefaultBranch_NoDefaultFound(t *testing.T) {
	dir := setupTestGitRepo(t)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	// Use a non-existent remote
	_, err = DefaultBranch("nonexistent-remote")
	if err == nil {
		t.Fatal("expected error for non-existent remote, got nil")
	}
}

func TestCheckoutBranch_ExistingBranch(t *testing.T) {
	dir := setupTestGitRepo(t)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	// Create a branch
	cmd := exec.Command("git", "branch", "test-branch")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("creating branch: %v", err)
	}

	// Checkout existing branch
	err = CheckoutBranch("test-branch")
	if err != nil {
		t.Fatalf("CheckoutBranch(existing): %v", err)
	}

	// Verify we're on the correct branch
	branch, err := CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if branch != "test-branch" {
		t.Errorf("CurrentBranch = %q, want %q", branch, "test-branch")
	}
}

func TestCheckoutBranch_NewBranch(t *testing.T) {
	dir := setupTestGitRepo(t)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	// Checkout a new branch (doesn't exist yet)
	err = CheckoutBranch("brand-new-branch")
	if err != nil {
		t.Fatalf("CheckoutBranch(new): %v", err)
	}

	// Verify we're on the correct branch
	branch, err := CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if branch != "brand-new-branch" {
		t.Errorf("CurrentBranch = %q, want %q", branch, "brand-new-branch")
	}
}

func TestFindRemote_ByName(t *testing.T) {
	dir := setupTestGitRepo(t)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	remote, err := FindRemote("upstream", "")
	if err != nil {
		t.Fatalf("FindRemote by name: %v", err)
	}
	if remote.Name != "upstream" {
		t.Errorf("Remote.Name = %q, want %q", remote.Name, "upstream")
	}
	if remote.Host != "gitlab.example.com" {
		t.Errorf("Remote.Host = %q, want %q", remote.Host, "gitlab.example.com")
	}
}

func TestFindRemote_ByHost(t *testing.T) {
	dir := setupTestGitRepo(t)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	remote, err := FindRemote("", "gitlab.example.com")
	if err != nil {
		t.Fatalf("FindRemote by host: %v", err)
	}
	if remote.Host != "gitlab.example.com" {
		t.Errorf("Remote.Host = %q, want %q", remote.Host, "gitlab.example.com")
	}
}

func TestFindRemote_FallbackToOrigin(t *testing.T) {
	dir := setupTestGitRepo(t)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	// Search for non-matching name and host; should fall back to origin
	remote, err := FindRemote("nonexistent", "nonexistent.host")
	if err != nil {
		t.Fatalf("FindRemote fallback: %v", err)
	}
	if remote.Name != "origin" {
		t.Errorf("Remote.Name = %q, want %q (fallback to origin)", remote.Name, "origin")
	}
}

func TestFindRemote_FallbackToFirstRemote(t *testing.T) {
	dir := t.TempDir()

	// Create a repo with only a non-origin remote
	runInDir := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	runInDir("init", "-b", "main")
	runInDir("config", "user.email", "test@test.com")
	runInDir("config", "user.name", "Test")

	testFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runInDir("add", "README.md")
	runInDir("commit", "-m", "Initial commit")
	runInDir("remote", "add", "custom", "https://gitlab.com/owner/repo.git")

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	remote, err := FindRemote("nonexistent", "nonexistent.host")
	if err != nil {
		t.Fatalf("FindRemote: %v", err)
	}
	if remote.Name != "custom" {
		t.Errorf("Remote.Name = %q, want %q (fallback to first remote)", remote.Name, "custom")
	}
}

func TestFindRemote_NoRemotes(t *testing.T) {
	dir := t.TempDir()

	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	_, err = FindRemote("", "")
	if err == nil {
		t.Fatal("expected error when no remotes configured")
	}
}

func TestRunGit_Error(t *testing.T) {
	// Test runGit with invalid command
	_, err := runGit("non-existent-command-abc123")
	if err == nil {
		t.Fatal("expected error for invalid git command")
	}
}

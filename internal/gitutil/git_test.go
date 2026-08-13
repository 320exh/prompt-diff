package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupRepo creates a throwaway git repo with one committed file and returns
// its directory. Tests run git commands with the working directory changed
// into it, since Show/Exists shell out to `git` in the current directory.
func setupRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(dir, "tracked.prompt"), []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "tracked.prompt")
	run("commit", "-m", "initial")
	return dir
}

func withDir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func TestShowExistingFile(t *testing.T) {
	dir := setupRepo(t)
	withDir(t, dir)

	data, err := Show("HEAD", "tracked.prompt")
	if err != nil {
		t.Fatalf("Show: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("data = %q", data)
	}
}

func TestShowMissingFile(t *testing.T) {
	dir := setupRepo(t)
	withDir(t, dir)

	if _, err := Show("HEAD", "does-not-exist.prompt"); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestExists(t *testing.T) {
	dir := setupRepo(t)
	withDir(t, dir)

	if !Exists("HEAD", "tracked.prompt") {
		t.Error("expected tracked.prompt to exist at HEAD")
	}
	if Exists("HEAD", "does-not-exist.prompt") {
		t.Error("expected missing file to report not existing")
	}
}

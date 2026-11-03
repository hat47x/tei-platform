package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitEnvironmentDropsRedirectionVariables(t *testing.T) {
	got := gitEnvironment([]string{"PATH=/bin", "GIT_DIR=/other", "git_work_tree=/x", "HOME=/h", "GIT_INDEX_FILE=/i"})
	if len(got) != 2 || got[0] != "PATH=/bin" || got[1] != "HOME=/h" {
		t.Fatalf("unexpected environment: %v", got)
	}
}

func TestRequireCleanRepositoryIgnoresHostGitRedirectionAndGlobalExcludes(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = gitEnvironment(os.Environ())
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	target := t.TempDir()
	other := t.TempDir()
	for _, dir := range []string{target, other} {
		run(dir, "init", "-q")
	}
	if err := os.WriteFile(filepath.Join(target, "stray.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// GIT_DIR pointing at a clean repository must not hide the target's stray file.
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	t.Setenv("GIT_WORK_TREE", other)
	if err := requireCleanRepository(target); err == nil {
		t.Fatal("untracked file in the target repository was not detected under GIT_DIR redirection")
	}
	t.Setenv("GIT_DIR", "")
	os.Unsetenv("GIT_DIR")
	os.Unsetenv("GIT_WORK_TREE")

	// A global excludes file must not hide untracked files.
	excludes := filepath.Join(t.TempDir(), "excludes")
	if err := os.WriteFile(excludes, []byte("stray.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitconfig := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(gitconfig, []byte("[core]\n\texcludesFile = "+filepath.ToSlash(excludes)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", filepath.Dir(gitconfig))
	t.Setenv("USERPROFILE", filepath.Dir(gitconfig))
	if err := os.Rename(gitconfig, filepath.Join(filepath.Dir(gitconfig), ".gitconfig")); err != nil {
		t.Fatal(err)
	}
	if err := requireCleanRepository(target); err == nil {
		t.Fatal("global excludes file hid an untracked file from the clean check")
	}
}

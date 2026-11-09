package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// #3637: `go test ./...` runs packages in parallel, and another package's
// temporary file can make the clean-check fail here. The message must name what
// was dirty, otherwise the failure reads like a mistake in this test's input.
func TestRequireCleanRepositoryNamesTheDirtyPaths(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "-q")
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "tracked.txt")
	run("-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-qm", "base")
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := requireCleanRepository(dir)
	if err == nil {
		t.Fatal("a modified file was accepted")
	}
	message := err.Error()
	if !strings.Contains(message, "tracked.txt") || !strings.Contains(message, "M ") {
		t.Fatalf("the dirty path is not named: %s", message)
	}
}

// A very dirty tree still has to produce a readable message.
func TestDirtyStatusLinesCapsTheList(t *testing.T) {
	var status strings.Builder
	for i := 0; i < 30; i++ {
		status.WriteString("?? file" + strings.Repeat("x", i%3) + ".txt\n")
	}
	rendered := dirtyStatusLines(status.String())
	if lines := strings.Count(rendered, "\n"); lines != 21 {
		t.Fatalf("rendered %d lines, want 20 entries plus the ellipsis: %s", lines, rendered)
	}
	if !strings.Contains(rendered, "…") {
		t.Fatalf("the capped list does not say it was truncated: %s", rendered)
	}
}

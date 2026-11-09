package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A root that is itself reached through a link, or spelled with a short name,
// used to make every read inside it look like an escape: the file path was
// resolved with EvalSymlinks but the root was not, so filepath.Rel started with
// "..". The Windows CI runner's temporary directory is C:\Users\RUNNER~1\...,
// which is exactly that shape (#3783).
func TestReadInsideAcceptsARootReachedThroughASymlink(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.MkdirAll(filepath.Join(real, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := "document body"
	if err := os.WriteFile(filepath.Join(real, "docs", "one.md"), []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("cannot create a symlink to a directory on this host: %v", err)
	}

	body, err := readInside(link, "docs/one.md")
	if err != nil {
		t.Fatalf("a file inside a linked root was refused: %v", err)
	}
	if string(body) != want {
		t.Fatalf("read %q, expected %q", body, want)
	}
}

// A link that really leaves the workspace is still refused.
func TestReadInsideStillRefusesALinkOutOfTheRoot(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	outside := filepath.Join(base, "outside")
	for _, dir := range []string{root, outside} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.md"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.md"), filepath.Join(root, "escape.md")); err != nil {
		t.Skipf("cannot create a symlink on this host: %v", err)
	}

	if _, err := readInside(root, "escape.md"); err == nil {
		t.Fatal("a symlink leaving the workspace was read")
	} else if !strings.Contains(err.Error(), "escapes workspace") {
		t.Fatalf("refused for the wrong reason: %v", err)
	}
}

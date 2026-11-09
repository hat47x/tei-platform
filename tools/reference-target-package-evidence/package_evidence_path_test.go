package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The repository-root guard compares directories, not spellings. Two spellings
// of the same directory must be accepted: git and os.Getwd can disagree about
// the text (separator, trailing separator) and, on the Windows CI runner, about
// a short name such as C:\Users\RUNNER~1\... (#3783).
func TestSameDirectoryAcceptsEquivalentSpellings(t *testing.T) {
	dir := t.TempDir()
	if !sameDirectory(dir, dir) {
		t.Fatal("a directory does not equal itself")
	}
	if !sameDirectory(dir, dir+string(os.PathSeparator)) {
		t.Fatal("a trailing separator changed the identity of the directory")
	}
	if !sameDirectory(dir, filepath.Join(dir, ".")) {
		t.Fatal("a '.' segment changed the identity of the directory")
	}
	if !sameDirectory(filepath.Join(dir, "."), dir) {
		t.Fatal("a '.' segment changed the identity of the directory in the other order")
	}
}

func TestSameDirectoryRejectsDifferentDirectories(t *testing.T) {
	base := t.TempDir()
	first := filepath.Join(base, "first")
	second := filepath.Join(base, "second")
	for _, dir := range []string{first, second} {
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if sameDirectory(first, second) {
		t.Fatal("two different directories were treated as the same")
	}
}

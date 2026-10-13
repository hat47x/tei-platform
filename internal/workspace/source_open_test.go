package workspace

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRootedSourceRejectsOutsideSymlinksAndNestedAliases(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("non-Linux targets fail closed")
	}
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "semantic"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "model.tei"), []byte("safe"), 0600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(outside, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "semantic", "escape.tei")); err != nil {
		t.Skip(err)
	}
	if err := os.Symlink(filepath.Dir(outside), filepath.Join(root, "alias")); err != nil {
		t.Skip(err)
	}
	for _, p := range []string{"semantic/escape.tei", "alias/secret", "../secret", "/tmp/secret"} {
		if got, err := readWorkspaceFile(root, p); err == nil {
			t.Fatalf("escaped source %q: %q", p, got)
		}
	}
	got, err := readWorkspaceFile(root, "semantic/model.tei")
	if err != nil || string(got) != "safe" {
		t.Fatalf("legitimate source: %q, %v", got, err)
	}
}

func TestRootedSourcePinsOpenedFileAcrossRename(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux handle-relative implementation")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "model.tei"), []byte("before"), 0600); err != nil {
		t.Fatal(err)
	}
	f, err := openRootedSource(root, "model.tei")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := os.Rename(filepath.Join(root, "model.tei"), filepath.Join(root, "old.tei")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "model.tei"), []byte("after"), 0600); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 6)
	n, err := f.Read(buf)
	if err != nil || n != 6 || string(buf) != "before" {
		t.Fatalf("opened handle redirected after rename: %q, %v", buf[:n], err)
	}
}

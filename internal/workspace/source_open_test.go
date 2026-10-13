package workspace

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRootedSourceRejectsOutsideSymlinksAndNestedAliases(t *testing.T) {
	if runtime.GOOS == "js" || runtime.GOOS == "wasip1" || runtime.GOOS == "plan9" {
		t.Skip("target lacks a traversal-resistant rooted reader")
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
	if err := os.Symlink("model.tei", filepath.Join(root, "semantic", "local.tei")); err != nil {
		t.Skipf("local symlinks unavailable: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../semantic", filepath.Join(root, "nested", "target")); err != nil {
		t.Skipf("nested symlinks unavailable: %v", err)
	}
	for _, p := range []string{"semantic/local.tei", "nested/target/model.tei"} {
		inside, err := readWorkspaceFile(root, p)
		if err != nil || string(inside) != "safe" {
			t.Fatalf("safe internal link %q was rejected: %q, %v", p, inside, err)
		}
	}
	got, err := readWorkspaceFile(root, "semantic/model.tei")
	if err != nil || string(got) != "safe" {
		t.Fatalf("legitimate source: %q, %v", got, err)
	}
}

func TestRootedSourcePinsOpenedFileAcrossRename(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("opened-file rename differs on this target")
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

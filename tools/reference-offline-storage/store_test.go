package offlinestorage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newStore(t *testing.T) (*Store, string) {
	t.Helper()
	root := t.TempDir()
	store, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, root
}

func TestPutGetOverwriteAndList(t *testing.T) {
	store, _ := newStore(t)
	if err := store.Put("notes/a.txt", []byte("one")); err != nil {
		t.Fatal(err)
	}
	if err := store.Put("notes/b.txt", []byte("two")); err != nil {
		t.Fatal(err)
	}
	if err := store.Put("notes/a.txt", []byte("three")); err != nil {
		t.Fatal(err)
	}
	value, err := store.Get("notes/a.txt")
	if err != nil || string(value) != "three" {
		t.Fatalf("value=%q err=%v", value, err)
	}
	keys, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(keys, ",") != "notes/a.txt,notes/b.txt" {
		t.Fatalf("keys=%v", keys)
	}
}

func TestReopenKeepsData(t *testing.T) {
	root := t.TempDir()
	first, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Put("state.json", []byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	value, err := second.Get("state.json")
	if err != nil || string(value) != `{"ok":true}` {
		t.Fatalf("value=%q err=%v", value, err)
	}
}

func TestMissingAndDelete(t *testing.T) {
	store, _ := newStore(t)
	if _, err := store.Get("missing.txt"); !os.IsNotExist(err) {
		t.Fatalf("want not-exist, got %v", err)
	}
	if err := store.Put("delete.txt", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete("delete.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("delete.txt"); !os.IsNotExist(err) {
		t.Fatalf("want not-exist after delete, got %v", err)
	}
}

func TestTraversalAndAbsoluteKeysAreRejected(t *testing.T) {
	store, _ := newStore(t)
	for _, key := range []string{"../outside.txt", "/etc/passwd"} {
		if err := store.Put(key, []byte("x")); err == nil {
			t.Fatalf("key %q was accepted", key)
		}
	}
}

func TestSymlinkOutsideRootIsRejected(t *testing.T) {
	store, root := newStore(t)
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := store.Get("link/secret.txt"); err == nil {
		t.Fatal("read through a symlink outside the root was accepted")
	}
	if err := store.Put("link/written.txt", []byte("x")); err == nil {
		t.Fatal("write through a symlink outside the root was accepted")
	}
}

func TestSecondOpenIsRejectedForSingleUser(t *testing.T) {
	root := t.TempDir()
	first, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open(root); !errors.Is(err, ErrInUse) {
		t.Fatalf("want ErrInUse, got %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestListDoesNotExposeTheLockFile(t *testing.T) {
	store, _ := newStore(t)
	if err := store.Put("a.txt", []byte("a")); err != nil {
		t.Fatal(err)
	}
	keys, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(keys, ",") != "a.txt" {
		t.Fatalf("keys=%v", keys)
	}
}

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"tei/internal/permission"
	"testing"
)

func TestWriteLineageWritesInsideTheRequestedPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lineage.json")
	if err := writeLineage(path, []byte("{}"+"\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "{}\n" {
		t.Fatalf("read: %q err=%v", content, err)
	}
	if runtime.GOOS != "windows" {
		// Windows reports a synthesised mode, so the check is meaningful only
		// where the permission bits are real.
		if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("the observation must stay private: %v %v", info, err)
		}
	}
}

func TestWriteLineageRejectsAnEmptyPath(t *testing.T) {
	if err := writeLineage("", []byte("{}")); err == nil {
		t.Fatal("an empty path must be refused before anything is written")
	}
}
func TestWriteLineageWithGrantsRefusesAPathOutsideTheGrantedScope(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	grants := []permission.Spec{{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: other}}
	if err := writeLineageWithGrants(filepath.Join(dir, "lineage.json"), []byte("{}"), grants); err == nil {
		t.Fatal("a path outside the granted scope must be refused")
	}
}

func TestLoadGrantsReadsTheGrantFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "grants.json")
	content := `{"grants":[{"kind":"filesystem","access":"write","scope":"/var/reports"}]}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	grants, err := loadGrants(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(grants) != 1 || grants[0].Scope != "/var/reports" {
		t.Fatalf("unexpected grants: %+v", grants)
	}
}

func TestWriteLineageWithGrantsWithoutGrantsWritesNothing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lineage.json")
	if err := writeLineageWithGrants(path, []byte("{}"), nil); err == nil {
		t.Fatal("no grants means nothing is written")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("a refused write must leave no file behind")
	}
}

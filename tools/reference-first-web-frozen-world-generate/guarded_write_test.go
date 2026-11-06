package main

import (
	"os"
	"path/filepath"
	"tei/internal/permission"
	"testing"
)

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

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
	const content = `{"grants":[{"issuedBy":"environment:production","kind":"filesystem","access":"write","scope":"/var/reports"}]}`
	grants, err := loadGrantsFrom(t, content)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(grants) != 1 || grants[0].Scope != "/var/reports" {
		t.Fatalf("unexpected grants: %+v", grants)
	}
}

func TestLoadGrantsRefusesAGrantTheToolIssuedItself(t *testing.T) {
	const content = `{"grants":[{"issuedBy":"tei.frozen_world.lineage","kind":"filesystem","access":"write","scope":"/var/reports"}]}`
	if _, err := loadGrantsFrom(t, content); err == nil {
		t.Fatal("the generator must not accept a grant it issued itself: the check would be its own decision")
	}
}

func TestLoadGrantsRefusesAGrantWithoutAnIssuer(t *testing.T) {
	const content = `{"grants":[{"kind":"filesystem","access":"write","scope":"/var/reports"}]}`
	if _, err := loadGrantsFrom(t, content); err == nil {
		t.Fatal("a grant that names no issuer must be refused")
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

// loadGrantsFrom writes the grants file into a temporary directory and reads it
// back through the tool's own loader.
func loadGrantsFrom(t *testing.T, content string) ([]permission.Spec, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "grants.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return loadGrants(path)
}

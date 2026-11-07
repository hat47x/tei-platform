package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/access"
	"tei/internal/permission"
)

func TestWriteLineageWithGrantsRefusesAPathOutsideTheGrantedScope(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	grants := []permission.Spec{{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: other}}
	if err := writeLineageWithGrants(filepath.Join(dir, "lineage.json"), []byte("{}"), grants, ""); err == nil {
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

func TestWriteLineageWithGrantsRecordsTheDecisionWhenAsked(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lineage.json")
	accessLog := filepath.Join(dir, "access.jsonl")
	grant := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: path}
	if err := writeLineageWithGrants(path, []byte("{}"), []permission.Spec{grant}, accessLog); err != nil {
		t.Fatalf("write: %v", err)
	}
	records, err := (access.File{Path: accessLog}).Records()
	if err != nil || len(records) != 1 {
		t.Fatalf("the decision must be recorded: %+v err=%v", records, err)
	}
	if !records[0].Allowed || records[0].Capability != string(lineageCapability) {
		t.Fatalf("unexpected record: %+v", records[0])
	}
}

// TestWriteLineageWithGrantsStillWritesWithoutAnAccessLog pins that a missing
// log does not silently turn into a refusal: the write happens and the caller is
// told the decision was not kept.
func TestWriteLineageWithGrantsStillWritesWithoutAnAccessLog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lineage.json")
	grant := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: path}
	if err := writeLineageWithGrants(path, []byte("{}"), []permission.Spec{grant}, ""); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("a write without a log must still happen: %v", err)
	}
}

// TestWriteLineageWithGrantsRefusesAChangedAccessLog checks that a journal which
// was already changed does not silently gain a valid-looking line. The write has
// already happened by then, so the caller is told the record is not trustworthy;
// the next run starts from a journal that verifies again.
func TestWriteLineageWithGrantsRefusesAChangedAccessLog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lineage.json")
	accessLog := filepath.Join(dir, "access.jsonl")
	grant := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: path}

	if err := writeLineageWithGrants(path, []byte("{}"), []permission.Spec{grant}, accessLog); err != nil {
		t.Fatalf("first write: %v", err)
	}
	raw, err := os.ReadFile(accessLog)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(accessLog, []byte(strings.Replace(string(raw), `"allowed":true`, `"allowed":false`, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeLineageWithGrants(path, []byte("{}"), []permission.Spec{grant}, accessLog); err == nil {
		t.Fatal("a changed record must not gain a valid-looking line")
	}
}

func TestWriteLineageWithGrantsWithoutGrantsWritesNothing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lineage.json")
	if err := writeLineageWithGrants(path, []byte("{}"), nil, ""); err == nil {
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
	return loadGrants(path, "")
}

// loadGrantsAgainstDeclaration writes both files and reads the grants through the
// tool's loader with the declaration attached.
func loadGrantsAgainstDeclaration(t *testing.T, content, declaration string) ([]permission.Spec, error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "grants.json")
	declarationPath := filepath.Join(dir, "declaration.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(declarationPath, []byte(declaration), 0o644); err != nil {
		t.Fatal(err)
	}
	return loadGrants(path, declarationPath)
}

func TestLoadGrantsRefusesAGrantTheDeclarationDoesNotAllow(t *testing.T) {
	// The environment declared migration-workdir for migration.atlas. It says
	// nothing about the lineage exporter, so a grant for the exporter is not
	// grounded even though it parses.
	const declaration = `{"grants":[{"issuedBy":"environment:development","capabilityId":"migration.atlas","kind":"filesystem","access":"read-write","scope":"migration-workdir"}]}`
	const grants = `{"grants":[{"issuedBy":"script:build-frozen-api","capabilityId":"tei.frozen_world.lineage","kind":"filesystem","access":"write","scope":"/tmp/observed.json"}]}`
	if _, err := loadGrantsAgainstDeclaration(t, grants, declaration); err == nil {
		t.Fatal("a grant outside the declaration must be refused")
	}
}

func TestLoadGrantsAcceptsAGrantTheDeclarationAllows(t *testing.T) {
	const declaration = `{"grants":[{"issuedBy":"environment:development","capabilityId":"tei.frozen_world.lineage","kind":"filesystem","access":"read-write","scope":"/tmp/observed.json"}]}`
	const grants = `{"grants":[{"issuedBy":"script:build-frozen-api","capabilityId":"tei.frozen_world.lineage","kind":"filesystem","access":"write","scope":"/tmp/observed.json"}]}`
	loaded, err := loadGrantsAgainstDeclaration(t, grants, declaration)
	if err != nil {
		t.Fatalf("a grant inside the declaration must be accepted: %v", err)
	}
	if len(loaded) != 1 || loaded[0].Scope != "/tmp/observed.json" {
		t.Fatalf("unexpected grants: %+v", loaded)
	}
}

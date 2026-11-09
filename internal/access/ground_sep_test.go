package access_test

import (
	"path/filepath"
	"testing"

	"tei/internal/access"
)

// TestGroundAcceptsANarrowerAccess pins the difference between "the same kind of
// access" and "the same access": a declaration of read-write must allow an
// issuance that asks only for write. Refusing that would make the declaration
// useless for the case it exists for.
func TestGroundAcceptsANarrowerAccess(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "workdir")
	declared := access.Document{Grants: []access.Entry{{
		IssuedBy:     "environment:development",
		Kind:         "filesystem",
		Access:       "read-write",
		Scope:        "migration-workdir",
		CapabilityID: "migration.atlas",
	}}}
	issued := access.Document{Grants: []access.Entry{{
		IssuedBy:     "environment:development",
		Kind:         "filesystem",
		Access:       "write",
		Scope:        filepath.Join(root, "observed.json"),
		CapabilityID: "migration.atlas",
	}}}
	resolutions := []access.Resolution{{ResolvedBy: "environment:development", Declared: "migration-workdir", Root: root}}
	if err := access.Ground(issued, declared, resolutions, "migration.atlas"); err != nil {
		t.Fatalf("a narrower access must stay inside a wider declaration: %v", err)
	}
}

// TestGroundMatchesAMixedSeparatorRoot pins that a scope built with "/" is still
// inside a root resolved with the platform separator.
func TestGroundMatchesAMixedSeparatorRoot(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "workdir")
	declared := access.Document{Grants: []access.Entry{{
		IssuedBy:     "environment:development",
		Kind:         "filesystem",
		Access:       "read-write",
		Scope:        "migration-workdir",
		CapabilityID: "migration.atlas",
	}}}
	issued := access.Document{Grants: []access.Entry{{
		IssuedBy:     "environment:development",
		Kind:         "filesystem",
		Access:       "write",
		Scope:        root + "/observed.json",
		CapabilityID: "migration.atlas",
	}}}
	resolutions := []access.Resolution{{ResolvedBy: "environment:development", Declared: "migration-workdir", Root: root}}
	if err := access.Ground(issued, declared, resolutions, "migration.atlas"); err != nil {
		t.Fatalf("root=%q scope=%q: %v", root, root+"/observed.json", err)
	}
}

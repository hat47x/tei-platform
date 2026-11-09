package access_test

import (
	"path/filepath"
	"testing"

	"tei/internal/access"
	"tei/internal/core"
)

// TestGroundFileRunsTheSameCheckForTheIssuerAndTheReader checks that the file
// form of the check is the check itself, not a second copy of it: the issuer
// (which writes a grant) and the reader (which uses one) call the same function,
// so one side cannot start accepting what the other refuses.
func TestGroundFileRunsTheSameCheckForTheReader(t *testing.T) {
	declarationPath := filepath.Join(t.TempDir(), "declaration.json")
	declared := access.Document{Grants: []access.Entry{{
		IssuedBy:     "environment:development",
		Kind:         "filesystem",
		Access:       "read-write",
		Scope:        "migration-workdir",
		CapabilityID: lineage,
	}}}
	if err := access.WriteDocument(declarationPath, declared); err != nil {
		t.Fatal(err)
	}

	inside := issuedFor("/tmp/run/observed.json")
	resolutions := []access.Resolution{{ResolvedBy: "host:build", Declared: "migration-workdir", Root: "/tmp/run"}}
	if err := access.GroundFile(inside, declarationPath, resolutions, lineage); err != nil {
		t.Fatalf("an issuance inside the declared workdir must be accepted: %v", err)
	}

	outside := issuedFor("/tmp/other/observed.json")
	if err := access.GroundFile(outside, declarationPath, resolutions, lineage); err == nil {
		t.Fatal("the reader must refuse what the issuer refuses")
	}

	// The two entry points must agree, or the pair drifts apart unnoticed.
	if err := access.Ground(outside, declared, resolutions, lineage); err == nil {
		t.Fatal("Ground accepted what GroundFile refused, so the two agree only by accident")
	}
}

func TestGroundFileRefusesAnUnreadableDeclaration(t *testing.T) {
	err := access.GroundFile(issuedFor("/tmp/run/observed.json"), filepath.Join(t.TempDir(), "missing.json"), nil, lineage)
	if err == nil {
		t.Fatal("a missing declaration must be refused, not treated as 'nothing to check'")
	}
}

func TestGroundFileRefusesWhenNoDeclarationIsNamed(t *testing.T) {
	if err := access.GroundFile(issuedFor("/tmp/run/observed.json"), "", nil, core.CanonicalID("tei.frozen_world.lineage")); err == nil {
		t.Fatal("an empty declaration path must be refused: it grounds nothing")
	}
}

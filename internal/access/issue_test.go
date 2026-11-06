package access_test

import (
	"path/filepath"
	"testing"

	"tei/internal/access"
	"tei/internal/core"
	"tei/internal/permission"
)

func TestIssueRecordsTheGrantAndItsBasis(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issued.jsonl")
	log := access.IssueFile{Path: path}
	spec := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: "/tmp/run/observed.json"}

	issued, err := access.Issue(log, "run-1", "script:build-frozen-api", core.MustCanonicalID("tei.frozen_world.lineage"), spec, "operator requested the observation path")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if issued.Scope != spec.Scope {
		t.Fatalf("the issued grant must be returned: %+v", issued)
	}
	records, err := log.Issues()
	if err != nil {
		t.Fatalf("issues: %v", err)
	}
	if len(records) != 1 || records[0].RunKey != "run-1" || records[0].Basis == "" || records[0].Issuer == "" {
		t.Fatalf("the issuance must be recorded with its issuer and basis: %+v", records)
	}
}

func TestIssueWithoutALogIssuesNothing(t *testing.T) {
	spec := permission.Spec{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"}
	if _, err := access.Issue(nil, "run-1", "environment:production", core.MustCanonicalID("tei.demo"), spec, "because"); err == nil {
		t.Fatal("an issuance that cannot be recorded must fail")
	}
}

func TestIssueRejectsAnInvalidGrant(t *testing.T) {
	log := access.IssueFile{Path: filepath.Join(t.TempDir(), "issued.jsonl")}
	spec := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: ""}
	if _, err := access.Issue(log, "run-1", "environment:production", core.MustCanonicalID("tei.demo"), spec, "because"); err == nil {
		t.Fatal("an invalid grant must be rejected")
	}
}

func TestIssueRefusesACapabilityIssuingItsOwnGrant(t *testing.T) {
	log := access.IssueFile{Path: filepath.Join(t.TempDir(), "issued.jsonl")}
	capability := core.MustCanonicalID("tei.frozen_world.lineage")
	spec := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: "/tmp/observed.json"}
	if _, err := access.Issue(log, "run-1", string(capability), capability, spec, "self-issued"); err == nil {
		t.Fatal("a capability must not issue its own grant")
	}
}

func TestIssueRefusesAnIssuanceWithoutABasis(t *testing.T) {
	log := access.IssueFile{Path: filepath.Join(t.TempDir(), "issued.jsonl")}
	spec := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: "/tmp/observed.json"}
	if _, err := access.Issue(log, "run-1", "environment:production", core.MustCanonicalID("tei.demo"), spec, ""); err == nil {
		t.Fatal("an issuance without a basis must be refused")
	}
}

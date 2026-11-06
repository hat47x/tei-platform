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

func TestIssueResolvedRecordsTheResolutions(t *testing.T) {
	log := access.IssueFile{Path: filepath.Join(t.TempDir(), "issued.jsonl")}
	spec := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: "/tmp/run/observed.json"}
	resolutions := []access.Resolution{{ResolvedBy: "environment:development", Declared: "migration-workdir", Root: "/tmp/run"}}
	if _, err := access.IssueResolved(log, access.IssueRequest{
		RunKey:      "run-1",
		Issuer:      "script:build-frozen-api",
		Capability:  core.MustCanonicalID("tei.frozen_world.lineage"),
		Permission:  spec,
		Basis:       "the run writes inside the declared workdir",
		Resolutions: resolutions,
	}); err != nil {
		t.Fatalf("issue: %v", err)
	}
	records, err := log.Issues()
	if err != nil || len(records) != 1 {
		t.Fatalf("the issuance must be recorded: %+v err=%v", records, err)
	}
	if len(records[0].Resolutions) != 1 || records[0].Resolutions[0].Declared != "migration-workdir" {
		t.Fatalf("the record must keep how the declaration was mapped: %+v", records[0].Resolutions)
	}
}

func TestIssueResolvedRefusesAResolutionTheCapabilityMade(t *testing.T) {
	log := access.IssueFile{Path: filepath.Join(t.TempDir(), "issued.jsonl")}
	capability := core.MustCanonicalID("tei.frozen_world.lineage")
	spec := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: "/tmp/run/observed.json"}
	_, err := access.IssueResolved(log, access.IssueRequest{
		RunKey:      "run-1",
		Issuer:      "script:build-frozen-api",
		Capability:  capability,
		Permission:  spec,
		Basis:       "the capability mapped its own declaration",
		Resolutions: []access.Resolution{{ResolvedBy: string(capability), Declared: "migration-workdir", Root: "/tmp/run"}},
	})
	if err == nil {
		t.Fatal("a capability must not map its own declaration")
	}
	if records, _ := log.Issues(); len(records) != 0 {
		t.Fatalf("a refused issuance must not be recorded: %+v", records)
	}
}

func TestIssueResolvedRefusesAnIncompleteResolution(t *testing.T) {
	log := access.IssueFile{Path: filepath.Join(t.TempDir(), "issued.jsonl")}
	spec := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: "/tmp/run/observed.json"}
	for name, resolution := range map[string]access.Resolution{
		"no root":     {ResolvedBy: "environment:development", Declared: "migration-workdir"},
		"no declared": {ResolvedBy: "environment:development", Root: "/tmp/run"},
		"no resolver": {Declared: "migration-workdir", Root: "/tmp/run"},
	} {
		_, err := access.IssueResolved(log, access.IssueRequest{
			RunKey:      "run-1",
			Issuer:      "script:build-frozen-api",
			Capability:  core.MustCanonicalID("tei.frozen_world.lineage"),
			Permission:  spec,
			Basis:       "incomplete",
			Resolutions: []access.Resolution{resolution},
		})
		if err == nil {
			t.Fatalf("%s must be refused", name)
		}
	}
}

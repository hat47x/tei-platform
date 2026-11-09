package main

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/access"
	"tei/internal/core"
)

// lineageCapability is the identity the migrated exporter acts as.
const lineageCapability = core.CanonicalID("tei.frozen_world.lineage")

func TestIssueGrantWritesTheGrantsFileAndTheRecord(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "grants.json")
	log := filepath.Join(dir, "issued.jsonl")

	err := runIssueGrantCommand(issueGrantOptions{
		Capability: "tei.frozen_world.lineage",
		Scope:      filepath.Join(dir, "observed.json"),
		Issuer:     "script:build-frozen-api",
		Reason:     "operator requested the observation",
		RunKey:     "run-1",
		OutPath:    out,
		LogPath:    log,
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read grants: %v", err)
	}
	document, err := access.ParseDocument(raw)
	if err != nil {
		t.Fatalf("the grants file must be readable by the adapters: %v", err)
	}
	if len(document.Grants) != 1 || document.Grants[0].Scope != filepath.Join(dir, "observed.json") {
		t.Fatalf("unexpected grants: %+v", document.Grants)
	}
	if document.Grants[0].IssuedBy != "script:build-frozen-api" {
		t.Fatalf("the grants file must name who issued it: %+v", document.Grants[0])
	}

	records, err := (access.IssueFile{Path: log}).Issues()
	if err != nil || len(records) != 1 || records[0].Basis == "" || records[0].Issuer == "" {
		t.Fatalf("the issuance must be recorded with its issuer and basis: %+v err=%v", records, err)
	}
	if records[0].Issuer != document.Grants[0].IssuedBy {
		t.Fatalf("the record and the grants file must name the same issuer: %+v", records[0])
	}
}

func TestIssueGrantRequiresItsFlags(t *testing.T) {
	if err := runIssueGrantCommand(issueGrantOptions{Capability: "tei.demo"}); err == nil {
		t.Fatal("missing flags must be reported")
	}
}

func TestIssueGrantRequiresAnIssuerAndAReason(t *testing.T) {
	dir := t.TempDir()
	for name, options := range map[string]issueGrantOptions{
		"no issuer": {
			Capability: "tei.demo",
			Scope:      filepath.Join(dir, "observed.json"),
			Reason:     "operator requested the observation",
			OutPath:    filepath.Join(dir, "grants.json"),
			LogPath:    filepath.Join(dir, "issued.jsonl"),
		},
		"no reason": {
			Capability: "tei.demo",
			Scope:      filepath.Join(dir, "observed.json"),
			Issuer:     "script:build-frozen-api",
			OutPath:    filepath.Join(dir, "grants.json"),
			LogPath:    filepath.Join(dir, "issued.jsonl"),
		},
	} {
		if err := runIssueGrantCommand(options); err == nil {
			t.Fatalf("%s must be reported", name)
		}
	}
}

func TestIssueGrantRejectsTheCapabilityIssuingItsOwnGrant(t *testing.T) {
	dir := t.TempDir()
	err := runIssueGrantCommand(issueGrantOptions{
		Capability: "tei.frozen_world.lineage",
		Scope:      filepath.Join(dir, "observed.json"),
		Issuer:     "tei.frozen_world.lineage",
		Reason:     "the capability decided for itself",
		OutPath:    filepath.Join(dir, "grants.json"),
		LogPath:    filepath.Join(dir, "issued.jsonl"),
	})
	if err == nil {
		t.Fatal("a capability must not issue its own grant")
	}
}

func TestIssueGrantRejectsANonCanonicalCapability(t *testing.T) {
	dir := t.TempDir()
	err := runIssueGrantCommand(issueGrantOptions{
		Capability: "tei.frozen-world.lineage",
		Scope:      filepath.Join(dir, "observed.json"),
		Issuer:     "script:build-frozen-api",
		Reason:     "operator requested the observation",
		OutPath:    filepath.Join(dir, "grants.json"),
		LogPath:    filepath.Join(dir, "issued.jsonl"),
	})
	if err == nil {
		t.Fatal("a non-canonical capability must be rejected")
	}
}

// declarationFor writes the file `tei grants` produces: the environment declares
// a logical scope for one capability.
func declarationFor(t *testing.T, capability core.CanonicalID, scope string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "declaration.json")
	document := access.Document{Application: "order_entry", Grants: []access.Entry{{
		IssuedBy:     "environment:development",
		Kind:         "filesystem",
		Access:       "write",
		Scope:        scope,
		CapabilityID: capability,
	}}}
	if err := access.WriteDocument(path, document); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestIssueGrantGroundedInADeclarationRecordsTheResolution(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "workdir")
	declaration := declarationFor(t, lineageCapability, "migration-workdir")

	err := runIssueGrantCommand(issueGrantOptions{
		Capability:      string(lineageCapability),
		Scope:           filepath.Join(root, "observed.json"),
		Issuer:          "script:build-frozen-api",
		Reason:          "the run writes its observation inside the declared workdir",
		RunKey:          "run-1",
		OutPath:         filepath.Join(dir, "grants.json"),
		LogPath:         filepath.Join(dir, "issued.jsonl"),
		DeclarationPath: declaration,
		DeclaredScope:   "migration-workdir",
		ResolutionRoot:  root,
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	records, err := (access.IssueFile{Path: filepath.Join(dir, "issued.jsonl")}).Issues()
	if err != nil || len(records) != 1 {
		t.Fatalf("the issuance must be recorded: %+v err=%v", records, err)
	}
	if len(records[0].Resolutions) != 1 ||
		records[0].Resolutions[0].Declared != "migration-workdir" ||
		records[0].Resolutions[0].Root != root {
		t.Fatalf("the record must say which declaration became which place: %+v", records[0].Resolutions)
	}
}

func TestIssueGrantRefusesAPlaceTheDeclarationDoesNotAllow(t *testing.T) {
	dir := t.TempDir()
	declaration := declarationFor(t, lineageCapability, "migration-workdir")
	err := runIssueGrantCommand(issueGrantOptions{
		Capability:      string(lineageCapability),
		Scope:           filepath.Join(t.TempDir(), "observed.json"),
		Issuer:          "script:build-frozen-api",
		Reason:          "outside the declared workdir",
		RunKey:          "run-1",
		OutPath:         filepath.Join(dir, "grants.json"),
		LogPath:         filepath.Join(dir, "issued.jsonl"),
		DeclarationPath: declaration,
		DeclaredScope:   "migration-workdir",
		ResolutionRoot:  filepath.Join(dir, "workdir"),
	})
	if err == nil {
		t.Fatal("a place outside the resolved declaration must be refused")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "grants.json")); !os.IsNotExist(statErr) {
		t.Fatal("a refused issuance must leave no grants file behind")
	}
	if records, _ := (access.IssueFile{Path: filepath.Join(dir, "issued.jsonl")}).Issues(); len(records) != 0 {
		t.Fatalf("a refused issuance must not be recorded: %+v", records)
	}
}

func TestIssueGrantNeedsBothHalvesOfAResolution(t *testing.T) {
	dir := t.TempDir()
	declaration := declarationFor(t, lineageCapability, "migration-workdir")
	base := issueGrantOptions{
		Capability:      string(lineageCapability),
		Scope:           filepath.Join(dir, "observed.json"),
		Issuer:          "script:build-frozen-api",
		Reason:          "the run writes its observation",
		RunKey:          "run-1",
		OutPath:         filepath.Join(dir, "grants.json"),
		LogPath:         filepath.Join(dir, "issued.jsonl"),
		DeclarationPath: declaration,
	}
	withScope := base
	withScope.DeclaredScope = "migration-workdir"
	if err := runIssueGrantCommand(withScope); err == nil {
		t.Fatal("a declared scope without a place maps nothing")
	}
	withRoot := base
	withRoot.ResolutionRoot = dir
	if err := runIssueGrantCommand(withRoot); err == nil {
		t.Fatal("a place without a declared scope cannot be traced to a declaration")
	}
}

func TestIssueGrantWithoutADeclarationIsUngrounded(t *testing.T) {
	// No -grant-declaration: the issuance is recorded, but nothing says the
	// environment allowed it. This test fixes that the two cases stay distinct.
	dir := t.TempDir()
	err := runIssueGrantCommand(issueGrantOptions{
		Capability: string(lineageCapability),
		Scope:      filepath.Join(dir, "observed.json"),
		Issuer:     "script:build-frozen-api",
		Reason:     "the run writes its observation",
		RunKey:     "run-1",
		OutPath:    filepath.Join(dir, "grants.json"),
		LogPath:    filepath.Join(dir, "issued.jsonl"),
	})
	if err != nil {
		t.Fatalf("an ungrounded issuance is still recorded, not refused: %v", err)
	}
	records, err := (access.IssueFile{Path: filepath.Join(dir, "issued.jsonl")}).Issues()
	if err != nil || len(records) != 1 || len(records[0].Resolutions) != 0 {
		t.Fatalf("an ungrounded issuance records no resolution: %+v err=%v", records, err)
	}
}

func TestIssueGrantRefusesItsOwnDeclaration(t *testing.T) {
	// The capability cannot map its own declaration: that would make the
	// declaration say whatever the capability needed it to say.
	dir := t.TempDir()
	declaration := declarationFor(t, lineageCapability, "migration-workdir")
	err := runIssueGrantCommand(issueGrantOptions{
		Capability:      string(lineageCapability),
		Scope:           filepath.Join(dir, "workdir", "observed.json"),
		Issuer:          string(lineageCapability),
		Reason:          "the capability decided for itself",
		RunKey:          "run-1",
		OutPath:         filepath.Join(dir, "grants.json"),
		LogPath:         filepath.Join(dir, "issued.jsonl"),
		DeclarationPath: declaration,
		DeclaredScope:   "migration-workdir",
		ResolutionRoot:  filepath.Join(dir, "workdir"),
	})
	if err == nil {
		t.Fatal("a capability must not resolve its own declaration")
	}
}

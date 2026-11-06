package main

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/access"
)

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

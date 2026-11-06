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
	grants, err := access.ParseGrants(raw)
	if err != nil {
		t.Fatalf("the grants file must be readable by the adapters: %v", err)
	}
	if len(grants) != 1 || grants[0].Scope != filepath.Join(dir, "observed.json") {
		t.Fatalf("unexpected grants: %+v", grants)
	}

	records, err := (access.IssueFile{Path: log}).Issues()
	if err != nil || len(records) != 1 || records[0].Basis == "" {
		t.Fatalf("the issuance must be recorded: %+v err=%v", records, err)
	}
}

func TestIssueGrantRequiresItsFlags(t *testing.T) {
	if err := runIssueGrantCommand(issueGrantOptions{Capability: "tei.demo"}); err == nil {
		t.Fatal("missing flags must be reported")
	}
}

func TestIssueGrantRejectsANonCanonicalCapability(t *testing.T) {
	dir := t.TempDir()
	err := runIssueGrantCommand(issueGrantOptions{
		Capability: "tei.frozen-world.lineage",
		Scope:      filepath.Join(dir, "observed.json"),
		OutPath:    filepath.Join(dir, "grants.json"),
		LogPath:    filepath.Join(dir, "issued.jsonl"),
	})
	if err == nil {
		t.Fatal("a non-canonical capability must be rejected")
	}
}

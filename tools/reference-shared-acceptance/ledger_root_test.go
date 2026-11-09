package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The Decision and Confirmation ledgers are supporting records, but writing or
// reading them through an outward symlink would place them outside the
// Workspace while the output still names a Workspace-relative path. These tests
// keep the ledger directories and a linked Decision inside the pinned root.
func TestDecisionLedgerOutwardSymlinkIsNotFollowed(t *testing.T) {
	root := scratchWorkspace(t)
	outside := t.TempDir()
	ledger := filepath.Join(root, ".tei", "decisions")
	if err := os.RemoveAll(ledger); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, ledger); err != nil {
		t.Skipf("ledger symlink unavailable: %v", err)
	}

	if _, err := recordDecision(root, Decision{Schema: decisionSchema, Decision: "rejected"}); err == nil {
		t.Fatal("outward Decision ledger symlink accepted a record write")
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("Decision record was written outside the Workspace: %v", entries)
	}
	if _, err := loadDecisionRecords(root, nil); err == nil {
		t.Fatal("outward Decision ledger symlink was read as a Workspace ledger")
	}
}

func TestConfirmationLedgerOutwardSymlinkIsNotFollowed(t *testing.T) {
	root := scratchWorkspace(t)
	outside := t.TempDir()
	ledger := filepath.Join(root, ".tei", "confirmations")
	if err := os.RemoveAll(ledger); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, ledger); err != nil {
		t.Skipf("ledger symlink unavailable: %v", err)
	}

	if _, err := recordConfirmation(root, humanInput("q1")); err == nil {
		t.Fatal("outward Confirmation ledger symlink accepted a record write")
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("Confirmation record was written outside the Workspace: %v", entries)
	}
	if _, _, err := readConfirmations(root); err == nil {
		t.Fatal("outward Confirmation ledger symlink was read as a Workspace ledger")
	}
}

func TestLinkedDecisionOutwardSymlinkIsNotRead(t *testing.T) {
	root := scratchWorkspace(t)
	d := rejectViaCLI(t, root, unitPriceCandidate(revisionOfDir(t, root)), "今回は見送る")
	recordPath := filepath.Join(root, filepath.FromSlash(d.RecordPath))
	content, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), filepath.Base(recordPath))
	if err := os.WriteFile(outside, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(recordPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, recordPath); err != nil {
		t.Skipf("linked Decision symlink unavailable: %v", err)
	}

	_, err = recordConfirmation(root, rejectedProposalInput(d, proposalBasis))
	if refusalCode(err) != "LINKED_DECISION_INVALID" {
		t.Fatalf("linked Decision outside the Workspace was not refused: %v", err)
	}
}

// Source acceptance is authoritative and is not rolled back when the
// supporting Decision ledger cannot be written. The command must still report
// the accepted outcome, mark the missing record, and exit nonzero so a caller
// does not treat the ledger as durable.
func TestAcceptReportsSourceChangedWhenDecisionLedgerWriteFails(t *testing.T) {
	root := scratchWorkspace(t)
	before := revisionOfDir(t, root)
	candidatePath := candidateFile(t, unitPriceCandidate(before))
	review := reviewFingerprintViaCLI(t, root, candidatePath)

	ledger := filepath.Join(root, ".tei", "decisions")
	outside := t.TempDir()
	// The outward link is installed only after Acceptance has written and
	// re-read the Source: the Preview overlay copy intentionally rejects any
	// symlink in the Workspace, so installing it earlier would fail Preview
	// rather than exercise the Decision record publication.
	afterWritesHook = func(hookedRoot string) {
		if hookedRoot != root {
			return
		}
		if err := os.RemoveAll(ledger); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, ledger); err != nil {
			t.Skipf("ledger symlink unavailable: %v", err)
		}
	}
	defer func() { afterWritesHook = nil }()

	code, out, errOut := cliRun(
		t, "accept", "-workspace", root, "-candidate", candidatePath,
		"-expected-review-fingerprint", review,
	)
	if code == 0 {
		t.Fatalf("accept reported success although the Decision ledger failed: out=%s err=%s", out, errOut)
	}
	var decision Decision
	if err := json.Unmarshal([]byte(out), &decision); err != nil {
		t.Fatalf("accepted Decision was not emitted (code=%d): %v: out=%q err=%q", code, err, out, errOut)
	}
	if decision.Decision != "accepted" || !decision.SourceChanged {
		t.Fatalf("accepted Source outcome was lost: %+v", decision)
	}
	if decision.RecordPath != "" {
		t.Fatalf("missing ledger record was reported as written: %s", decision.RecordPath)
	}
	var warnings []string
	for _, w := range decision.Warnings {
		warnings = append(warnings, w.Code)
	}
	if !hasCode(warnings, "DECISION_RECORD_FAILED") {
		t.Fatalf("ledger failure was not reported as a warning: %+v", decision.Warnings)
	}
	if after := revisionOfDir(t, root); after == before {
		t.Fatal("Source was not accepted after the ledger failure")
	}
	if entries, err := os.ReadDir(outside); err != nil || len(entries) != 0 {
		t.Fatalf("Decision record escaped to the linked directory: %v %v", entries, err)
	}
}

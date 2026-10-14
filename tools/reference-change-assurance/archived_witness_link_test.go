package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const syntheticWitnessHead = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func setupSyntheticArchivedWitness(t *testing.T) (beforeRoot, afterRoot, receiptDir string, claims map[string]map[string]any) {
	t.Helper()
	reference := filepath.Join("..", "..", "reference", "workspace")
	beforeRoot, afterRoot, receiptDir = t.TempDir(), t.TempDir(), t.TempDir()
	for _, root := range []string{beforeRoot, afterRoot} {
		if err := os.CopyFS(root, os.DirFS(reference)); err != nil {
			t.Fatal(err)
		}
	}
	// Intentionally change managed bytes but not the Canonical World: this
	// is the actual R1/R2 blind spot in a World-only generation fingerprint.
	semantic := filepath.Join(afterRoot, "semantic", "sales", "order.tei")
	body, err := os.ReadFile(semantic)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(semantic, append(body, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	diff, err := reviewWorkspaceDiff(beforeRoot, afterRoot, "order-entry-reference", workspaceKey("application", "order_entry"))
	if err != nil {
		t.Fatal(err)
	}
	if diff.BeforeSourceRevision == diff.AfterSourceRevision || len(diff.SourceChanges) != 1 ||
		len(diff.Impact.Changes) != 0 || diff.Decision.Action != "unresolved" {
		t.Fatalf("expected source-only R0/R1 drift: %+v", diff)
	}
	claims = make(map[string]map[string]any)
	for name, nature := range witnessNatures {
		claims[name] = map[string]any{"evidenceNature": nature, "integrationHead": syntheticWitnessHead}
	}
	claims["workspace-r1.json"]["sourceBefore"] = diff.BeforeSourceRevision
	claims["workspace-r1.json"]["sourceAfter"] = diff.AfterSourceRevision
	claims["workspace-r1.json"]["decisionModelRevisionAfter"] = diff.AfterSourceRevision
	claims["workspace-r1.json"]["humanApproval"] = false
	claims["target-launch.json"]["workspaceSourceRevisionBeforeLaunch"] = diff.AfterSourceRevision
	claims["target-launch.json"]["targetBuildRevision"] = syntheticWitnessHead
	claims["target-launch.json"]["application"] = "order_entry"
	claims["target-launch.json"]["runtimeAuthorityVerified"] = false
	for _, name := range []string{"web-positive.json", "web-negative-unit-price.json", "n11-unobserved-control.json"} {
		claims[name]["acceptedSourceRevisionClaim"] = diff.AfterSourceRevision
	}
	claims["web-positive.json"]["observationNature"] = "synthetic"
	claims["web-positive.json"]["independentDatabaseRead"] = false
	claims["n11-unobserved-control.json"]["independentDatabaseRead"] = false
	claims["n11-unobserved-control.json"]["executionErrorNature"] = "synthetic-read-unavailable"
	claims["n11-unobserved-control.json"]["commandCompleted"] = true
	claims["n11-unobserved-control.json"]["observationReadAttempted"] = true
	claims["n11-unobserved-control.json"]["observationResultRecorded"] = false
	claims["stale-source-control.json"]["acceptedSourceRevisionClaim"] = diff.AfterSourceRevision
	claims["stale-source-control.json"]["actuallyLoadedWorkspaceSourceRevision"] = diff.BeforeSourceRevision
	claims["stale-source-control.json"]["mismatchDetectedByExternalTestHarness"] = true
	claims["stale-source-control.json"]["webServerRejectsMismatchedLabels"] = false
	writeSyntheticArchivedWitness(t, receiptDir, claims)
	return
}

func writeSyntheticArchivedWitness(t *testing.T, dir string, claims map[string]map[string]any) {
	t.Helper()
	for name, value := range claims {
		body, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), body, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func hasWitnessDiagnostic(report ArchivedWitnessLinkReport, code string) bool {
	for _, diagnostic := range report.Diagnostics {
		if diagnostic == code {
			return true
		}
	}
	return false
}

func TestArchivedWitnessLinksEightSyntheticReceiptsToSourceRevisionsButCannotQualify(t *testing.T) {
	before, after, dir, _ := setupSyntheticArchivedWitness(t)
	report, err := reviewArchivedWitnessLink(before, after, "order-entry-reference", workspaceKey("application", "order_entry"), dir)
	if err != nil {
		t.Fatal(err)
	}
	if report.LinkStatus != "declared-links-consistent" || len(report.Diagnostics) != 0 ||
		len(report.Files) != 8 || report.IntegrationHead != syntheticWitnessHead ||
		report.Qualification != "unresolved" || report.ExecutionAllowed ||
		report.Diff.Decision.Action != "unresolved" {
		t.Fatalf("synthetic claims incorrectly promoted or lost: %+v", report)
	}
	for _, f := range report.Files {
		if !strings.HasPrefix(f.Fingerprint, "sha256:") {
			t.Fatalf("receipt file without byte digest: %+v", f)
		}
	}
}

func TestArchivedWitnessMismatchedClaimsNeverQualify(t *testing.T) {
	cases := []struct {
		name string
		edit func(map[string]map[string]any)
		want string
	}{
		{"lineage changed", func(v map[string]map[string]any) { v["workspace-r1.json"]["sourceAfter"] = "sha256:wrong" }, "LINEAGE_SOURCE_REVISION_MISMATCH"},
		{"target loaded stale", func(v map[string]map[string]any) { v["target-launch.json"]["workspaceSourceRevisionBeforeLaunch"] = v["workspace-r1.json"]["sourceBefore"] }, "TARGET_LAUNCH_SOURCE_OR_BUILD_MISMATCH"},
		{"target revision mismatched", func(v map[string]map[string]any) { v["target-launch.json"]["targetBuildRevision"] = strings.Repeat("b", 40) }, "TARGET_LAUNCH_SOURCE_OR_BUILD_MISMATCH"},
		{"web accepted stale", func(v map[string]map[string]any) { v["web-positive.json"]["acceptedSourceRevisionClaim"] = v["workspace-r1.json"]["sourceBefore"] }, "WEB_AND_OBSERVATION_R1_CLAIMS_MISMATCH"},
		{"N11 actually observed", func(v map[string]map[string]any) { v["n11-unobserved-control.json"]["observationResultRecorded"] = true }, "N11_UNOBSERVED_CONTROL_INCOMPLETE"},
		{"N11 claims real database", func(v map[string]map[string]any) { v["n11-unobserved-control.json"]["independentDatabaseRead"] = true }, "SYNTHETIC_OR_AUTHORITY_LIMITS_NOT_EXPLICIT"},
		{"stale control lies", func(v map[string]map[string]any) { v["stale-source-control.json"]["actuallyLoadedWorkspaceSourceRevision"] = v["workspace-r1.json"]["sourceAfter"] }, "STALE_SOURCE_CONTROL_NOT_SEPARATELY_OBSERVED"},
		{"integration head differs", func(v map[string]map[string]any) { v["web-positive.json"]["integrationHead"] = strings.Repeat("b", 40) }, "INTEGRATION_HEAD_DIFFERS:web-positive.json"},
		{"wrong witness kind", func(v map[string]map[string]any) { v["web-positive.json"]["evidenceNature"] = "actual-independent-db" }, "UNEXPECTED_WITNESS_NATURE:web-positive.json"},
		{"false human approval", func(v map[string]map[string]any) { v["workspace-r1.json"]["humanApproval"] = true }, "SYNTHETIC_OR_AUTHORITY_LIMITS_NOT_EXPLICIT"},
	} 
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before, after, dir, records := setupSyntheticArchivedWitness(t)
			tc.edit(records)
			writeSyntheticArchivedWitness(t, dir, records)
			report, err := reviewArchivedWitnessLink(before, after, "order-entry-reference", workspaceKey("application", "order_entry"), dir)
			if err != nil {
				t.Fatal(err)
			}
			if !hasWitnessDiagnostic(report, tc.want) || report.LinkStatus != "unresolved" ||
				report.Qualification != "unresolved" || report.ExecutionAllowed {
				t.Fatalf("bad receipt claim was accepted: %+v", report)
			}
		})
	}
}

func TestArchivedWitnessRejectsUnchangedSourceAndUntrustedReceiptFiles(t *testing.T) {
	before, after, dir, _ := setupSyntheticArchivedWitness(t)
	report, err := reviewArchivedWitnessLink(before, before, "order-entry-reference", workspaceKey("application", "order_entry"), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !hasWitnessDiagnostic(report, "NO_OBSERVED_R0_R1_SOURCE_CHANGE") ||
		report.LinkStatus != "unresolved" || report.ExecutionAllowed {
		t.Fatalf("unchanged Source accepted as R1: %+v", report)
	}
	if err := os.Remove(filepath.Join(dir, "n11-unobserved-control.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := reviewArchivedWitnessLink(before, after, "order-entry-reference", workspaceKey("application", "order_entry"), dir); err == nil {
		t.Fatal("missing N11 receipt silently accepted")
	}
}

func TestArchivedWitnessRejectsDuplicateOrOversizeReceipts(t *testing.T) {
	before, after, dir, _ := setupSyntheticArchivedWitness(t)
	path := filepath.Join(dir, "web-positive.json")
	if err := os.WriteFile(path, []byte(`{"evidenceNature":"a","EvidenceNature":"b"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := reviewArchivedWitnessLink(before, after, "order-entry-reference", workspaceKey("application", "order_entry"), dir); err == nil {
		t.Fatal("duplicate JSON alias accepted")
	}
	if err := os.WriteFile(path, []byte(strings.Repeat(" ", (1<<20)+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := reviewArchivedWitnessLink(before, after, "order-entry-reference", workspaceKey("application", "order_entry"), dir); err == nil {
		t.Fatal("oversized receipt accepted")
	}
}

func TestArchivedWitnessRejectsOutOfRootSymlink(t *testing.T) {
	before, after, dir, _ := setupSyntheticArchivedWitness(t)
	path := filepath.Join(dir, "web-positive.json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "private.json")
	if err := os.WriteFile(outside, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	if _, err := reviewArchivedWitnessLink(before, after, "order-entry-reference", workspaceKey("application", "order_entry"), dir); err == nil {
		t.Fatal("receipt symlink outside root was followed")
	}
}

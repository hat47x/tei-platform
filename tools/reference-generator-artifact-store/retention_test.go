package generatorartifactstore

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func putCandidate(t *testing.T, storeDir, candidateID string) Receipt {
	t.Helper()
	receiptPath, _ := writeSelectionReceipt(t)
	_, _, _, candidates := selectionPaths(t)
	receipt, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: candidateID, CandidatesDir: candidates, StoreDir: storeDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed {
		t.Fatalf("put %s failed: %+v", candidateID, receipt)
	}
	return receipt
}

func TestListAndPruneExplicitKeep(t *testing.T) {
	storeDir := t.TempDir()
	template := putCandidate(t, storeDir, "template")
	alternative := putCandidate(t, storeDir, "alternative")
	list, err := List(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	if !list.Passed || len(list.Objects) != 2 {
		t.Fatalf("unexpected list: %+v", list)
	}
	prune, err := Prune(PruneOptions{StoreDir: storeDir, Keep: []string{template.SourceDigest}})
	if err != nil {
		t.Fatal(err)
	}
	if !prune.Passed || len(prune.Kept) != 1 || len(prune.Deleted) != 1 || prune.Deleted[0] != alternative.SourceDigest {
		t.Fatalf("unexpected prune: %+v", prune)
	}
	if _, err := Get(storeDir, template.SourceDigest); err != nil {
		t.Fatalf("kept object is unreadable: %v", err)
	}
	if _, err := Get(storeDir, alternative.SourceDigest); err == nil {
		t.Fatal("pruned object is still readable")
	}
	if !prune.AuditRecorded || prune.AuditPath == "" {
		t.Fatalf("prune audit was not recorded: %+v", prune)
	}
	audit, err := Audit(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	if !audit.Passed || len(audit.Records) != 1 {
		t.Fatalf("unexpected audit: %+v", audit)
	}
	record := audit.Records[0]
	if record.Operation != "prune" || record.DryRun || record.RecordedAt == "" ||
		len(record.Keep) != 1 || len(record.Deleted) != 1 || record.Deleted[0] != alternative.SourceDigest {
		t.Fatalf("unexpected audit record: %+v", record)
	}
	if record.PreviousDigest != "" || record.RecordDigest == "" {
		t.Fatalf("unexpected audit chain fields: %+v", record)
	}
}

func TestAuditChainLinksRecords(t *testing.T) {
	storeDir := t.TempDir()
	template := putCandidate(t, storeDir, "template")
	putCandidate(t, storeDir, "alternative")
	if _, err := Prune(PruneOptions{StoreDir: storeDir, Keep: []string{template.SourceDigest}}); err != nil {
		t.Fatal(err)
	}
	second, err := Prune(PruneOptions{StoreDir: storeDir, Keep: []string{template.SourceDigest}})
	if err != nil {
		t.Fatal(err)
	}
	if !second.Passed {
		t.Fatalf("second prune failed: %+v", second)
	}
	audit, err := Audit(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	if !audit.Passed || len(audit.Records) != 2 {
		t.Fatalf("unexpected audit: %+v", audit)
	}
	if audit.Records[1].PreviousDigest != audit.Records[0].RecordDigest {
		t.Fatalf("audit chain is not linked: %+v", audit.Records)
	}
}

func TestAuditChainDetectsTamper(t *testing.T) {
	storeDir := t.TempDir()
	template := putCandidate(t, storeDir, "template")
	putCandidate(t, storeDir, "alternative")
	if _, err := Prune(PruneOptions{StoreDir: storeDir, Keep: []string{template.SourceDigest}}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(storeDir, "prune-audit.jsonl")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record AuditRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		t.Fatal(err)
	}
	record.Deleted = append(record.Deleted, "sha256:"+strings.Repeat("0", 64))
	tampered, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(tampered, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	audit, err := Audit(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	if audit.Passed || !strings.Contains(audit.Reason, "digest") {
		t.Fatalf("tampered audit was accepted: %+v", audit)
	}
}

func TestPruneRefusesCorruptAudit(t *testing.T) {
	storeDir := t.TempDir()
	template := putCandidate(t, storeDir, "template")
	if err := os.WriteFile(filepath.Join(storeDir, "prune-audit.jsonl"), []byte("not-json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prune, err := Prune(PruneOptions{StoreDir: storeDir, Keep: []string{template.SourceDigest}})
	if err != nil {
		t.Fatal(err)
	}
	if prune.Passed || !strings.Contains(prune.Reason, "audit") {
		t.Fatalf("corrupt audit was ignored: %+v", prune)
	}
	if _, err := Get(storeDir, template.SourceDigest); err != nil {
		t.Fatalf("object was deleted despite corrupt audit: %v", err)
	}
}

func TestPruneDryRunDoesNotDelete(t *testing.T) {
	storeDir := t.TempDir()
	template := putCandidate(t, storeDir, "template")
	alternative := putCandidate(t, storeDir, "alternative")
	prune, err := Prune(PruneOptions{
		StoreDir: storeDir, Keep: []string{template.SourceDigest}, DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !prune.Passed || len(prune.WouldDelete) != 1 || prune.WouldDelete[0] != alternative.SourceDigest || len(prune.Deleted) != 0 {
		t.Fatalf("unexpected dry-run: %+v", prune)
	}
	if _, err := Get(storeDir, alternative.SourceDigest); err != nil {
		t.Fatalf("dry-run deleted an object: %v", err)
	}
}

func TestPruneRefusesMissingKeep(t *testing.T) {
	storeDir := t.TempDir()
	putCandidate(t, storeDir, "template")
	missing := "sha256:" + strings.Repeat("0", 64)
	prune, err := Prune(PruneOptions{StoreDir: storeDir, Keep: []string{missing}})
	if err != nil {
		t.Fatal(err)
	}
	if prune.Passed || len(prune.MissingKeep) != 1 || prune.MissingKeep[0] != missing {
		t.Fatalf("missing keep was accepted: %+v", prune)
	}
}

func TestPruneRefusesTamperedObject(t *testing.T) {
	storeDir := t.TempDir()
	put := putCandidate(t, storeDir, "template")
	if err := os.WriteFile(filepath.Join(put.ObjectPath, "artifact"), []byte("tampered"), 0o755); err != nil {
		t.Fatal(err)
	}
	prune, err := Prune(PruneOptions{StoreDir: storeDir, Keep: []string{put.SourceDigest}})
	if err != nil {
		t.Fatal(err)
	}
	if prune.Passed || !strings.Contains(prune.Reason, "unverified") {
		t.Fatalf("tampered object was pruned: %+v", prune)
	}
}

func TestCheckpointSummarizesAudit(t *testing.T) {
	empty, err := Checkpoint(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !empty.Passed || empty.RecordCount != 0 || empty.HeadDigest != "" || empty.AuditDigest == "" {
		t.Fatalf("unexpected empty checkpoint: %+v", empty)
	}

	storeDir := t.TempDir()
	template := putCandidate(t, storeDir, "template")
	putCandidate(t, storeDir, "alternative")
	if _, err := Prune(PruneOptions{StoreDir: storeDir, Keep: []string{template.SourceDigest}}); err != nil {
		t.Fatal(err)
	}
	checkpoint, err := Checkpoint(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	audit, err := Audit(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	if !checkpoint.Passed || checkpoint.RecordCount != 1 || checkpoint.HeadDigest != audit.Records[0].RecordDigest {
		t.Fatalf("unexpected checkpoint: %+v", checkpoint)
	}
	raw, err := os.ReadFile(filepath.Join(storeDir, "prune-audit.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.AuditDigest != digest(raw) {
		t.Fatalf("audit digest mismatch: %s", checkpoint.AuditDigest)
	}
}

func TestCheckpointRejectsTamperedAudit(t *testing.T) {
	storeDir := t.TempDir()
	template := putCandidate(t, storeDir, "template")
	putCandidate(t, storeDir, "alternative")
	if _, err := Prune(PruneOptions{StoreDir: storeDir, Keep: []string{template.SourceDigest}}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(storeDir, "prune-audit.jsonl")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, []byte("tampered\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	checkpoint, err := Checkpoint(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.Passed || checkpoint.Reason == "" {
		t.Fatalf("tampered audit produced a passing checkpoint: %+v", checkpoint)
	}
}

func TestVerifyCheckpointMatchesStore(t *testing.T) {
	storeDir := t.TempDir()
	checkpoint, err := Checkpoint(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	checkpointPath := filepath.Join(t.TempDir(), "checkpoint.json")
	raw, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(checkpointPath, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	verification, err := VerifyCheckpoint(storeDir, checkpointPath)
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Passed || verification.Expected.RecordCount != verification.Actual.RecordCount {
		t.Fatalf("checkpoint did not verify: %+v", verification)
	}
}

func TestVerifyCheckpointDetectsMismatch(t *testing.T) {
	storeDir := t.TempDir()
	checkpoint, err := Checkpoint(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint.HeadDigest = "sha256:" + strings.Repeat("0", 64)
	checkpointPath := filepath.Join(t.TempDir(), "checkpoint.json")
	raw, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(checkpointPath, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	verification, err := VerifyCheckpoint(storeDir, checkpointPath)
	if err != nil {
		t.Fatal(err)
	}
	if verification.Passed || !strings.Contains(verification.Reason, "head digest") {
		t.Fatalf("mismatched checkpoint was accepted: %+v", verification)
	}
}

func TestVerifyCheckpointRejectsInvalidCheckpoint(t *testing.T) {
	storeDir := t.TempDir()
	path := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := os.WriteFile(path, []byte(`{"schema":"other","passed":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	verification, err := VerifyCheckpoint(storeDir, path)
	if err != nil {
		t.Fatal(err)
	}
	if verification.Passed || verification.Reason == "" {
		t.Fatalf("invalid checkpoint was accepted: %+v", verification)
	}
}

func TestAuditRejectsMalformedRecord(t *testing.T) {
	storeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(storeDir, "prune-audit.jsonl"), []byte("not-json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	audit, err := Audit(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	if audit.Passed || !strings.Contains(audit.Reason, "invalid") {
		t.Fatalf("malformed audit was accepted: %+v", audit)
	}
}

func TestPruneRequiresKeepSet(t *testing.T) {
	prune, err := Prune(PruneOptions{StoreDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if prune.Passed || !strings.Contains(prune.Reason, "keep set") {
		t.Fatalf("empty keep set was accepted: %+v", prune)
	}
}

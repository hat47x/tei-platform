package generatorartifactstore

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	generatorselection "tei/tools/reference-generator-selection"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path is unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func selectionPaths(t *testing.T) (workspace, contract, fixtures, candidates string) {
	t.Helper()
	base := filepath.Join(repoRoot(t), "tools", "reference-generator-selection", "testdata")
	return filepath.Join(repoRoot(t), "reference", "workspace"),
		filepath.Join(base, "line-amount", "contract.json"),
		filepath.Join(base, "line-amount", "fixtures.json"),
		filepath.Join(base, "candidates")
}

var (
	selectionReceiptOnce sync.Once
	selectionReceiptPath string
	selectionReceiptRaw  []byte
	selectionReceiptErr  error
)

func writeSelectionReceipt(t *testing.T) (string, generatorselection.Receipt) {
	t.Helper()
	selectionReceiptOnce.Do(func() {
		workspace, contract, fixtures, candidates := selectionPaths(t)
		selection, err := generatorselection.Select(context.Background(), workspace, contract, fixtures, candidates)
		if err != nil {
			selectionReceiptErr = err
			return
		}
		raw, err := json.MarshalIndent(selection, "", "  ")
		if err != nil {
			selectionReceiptErr = err
			return
		}
		selectionReceiptRaw = append(raw, '\n')
		dir, err := os.MkdirTemp("", "tei-artifact-selection-*")
		if err != nil {
			selectionReceiptErr = err
			return
		}
		selectionReceiptPath = filepath.Join(dir, "selection.json")
		selectionReceiptErr = os.WriteFile(selectionReceiptPath, selectionReceiptRaw, 0o600)
	})
	if selectionReceiptErr != nil {
		t.Fatal(selectionReceiptErr)
	}
	var selection generatorselection.Receipt
	if err := json.Unmarshal(selectionReceiptRaw, &selection); err != nil {
		t.Fatal(err)
	}
	return selectionReceiptPath, selection
}

func TestPutAndGetArtifact(t *testing.T) {
	receiptPath, selection := writeSelectionReceipt(t)
	_, _, _, candidates := selectionPaths(t)
	storeDir := t.TempDir()
	put, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: "template", CandidatesDir: candidates, StoreDir: storeDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !put.Passed || put.Reused || put.SourceDigest == "" || put.BinaryDigest == "" || put.ObjectPath == "" {
		t.Fatalf("artifact put failed: %+v", put)
	}
	if _, err := os.Stat(filepath.Join(put.ObjectPath, "source")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(put.ObjectPath, "artifact")); err != nil {
		t.Fatal(err)
	}
	get, err := Get(storeDir, put.SourceDigest)
	if err != nil {
		t.Fatal(err)
	}
	if !get.Passed || get.SourceDigest != put.SourceDigest || get.BinaryDigest != put.BinaryDigest {
		t.Fatalf("artifact get failed: %+v", get)
	}
	again, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: "template", CandidatesDir: candidates, StoreDir: storeDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !again.Passed || !again.Reused {
		t.Fatalf("existing artifact was not reused: %+v", again)
	}
	var found bool
	for _, candidate := range selection.Candidates {
		if candidate.ID == "template" {
			found = candidate.SourceDigest == put.SourceDigest && candidate.BinaryDigest == put.BinaryDigest
		}
	}
	if !found {
		t.Fatal("stored digests do not match the selection receipt")
	}
}

func TestPutRejectsReceiptDigestMismatch(t *testing.T) {
	_, selection := writeSelectionReceipt(t)
	_, _, _, candidates := selectionPaths(t)
	for i := range selection.Candidates {
		if selection.Candidates[i].ID == "template" {
			selection.Candidates[i].BinaryDigest = "sha256:" + strings.Repeat("0", 64)
		}
	}
	raw, err := json.MarshalIndent(selection, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(t.TempDir(), "tampered.json")
	if err := os.WriteFile(receiptPath, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: "template", CandidatesDir: candidates, StoreDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || !strings.Contains(receipt.Reason, "binary digest") {
		t.Fatalf("tampered receipt was accepted: %+v", receipt)
	}
}

func TestGetRejectsTamperedArtifact(t *testing.T) {
	receiptPath, _ := writeSelectionReceipt(t)
	_, _, _, candidates := selectionPaths(t)
	storeDir := t.TempDir()
	put, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: "template", CandidatesDir: candidates, StoreDir: storeDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !put.Passed || put.ObjectPath == "" {
		t.Fatalf("artifact put failed before tamper test: %+v", put)
	}
	if err := os.WriteFile(filepath.Join(put.ObjectPath, "artifact"), []byte("tampered"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Get(storeDir, put.SourceDigest)
	if err != nil {
		t.Fatal(err)
	}
	if got.Passed || !strings.Contains(got.Reason, "artifact digest") {
		t.Fatalf("tampered artifact was accepted: %+v", got)
	}
}

func TestPutAndGetWasiArtifact(t *testing.T) {
	workspace, contract, fixtures, _ := selectionPaths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	if _, err := generatorselection.GenerateVersion(contract, "wasi-template", "1",
		filepath.Join(candidates, "wasi-template")); err != nil {
		t.Fatal(err)
	}
	selection, err := generatorselection.Select(context.Background(), workspace, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if !selection.Passed || len(selection.Candidates) != 1 || !selection.Candidates[0].Passed {
		t.Fatalf("wasi candidate did not pass selection: %+v", selection.Candidates)
	}
	raw, err := json.MarshalIndent(selection, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(t.TempDir(), "selection.json")
	if err := os.WriteFile(receiptPath, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	storeDir := t.TempDir()
	put, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: selection.Candidates[0].ID,
		CandidatesDir: candidates, StoreDir: storeDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !put.Passed || put.ObjectPath == "" {
		t.Fatalf("wasi artifact was not stored: %+v", put)
	}
	rawObject, err := os.ReadFile(filepath.Join(put.ObjectPath, "object.json"))
	if err != nil {
		t.Fatal(err)
	}
	var object Object
	if err := json.Unmarshal(rawObject, &object); err != nil {
		t.Fatal(err)
	}
	if object.Language != generatorselection.LanguageWasi {
		t.Fatalf("stored language=%s want wasi", object.Language)
	}
	got, err := Get(storeDir, put.SourceDigest)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Passed || got.BinaryDigest != put.BinaryDigest {
		t.Fatalf("wasi artifact was not read back: %+v", got)
	}
}

func TestPutRejectsUnknownCandidate(t *testing.T) {
	receiptPath, _ := writeSelectionReceipt(t)
	_, _, _, candidates := selectionPaths(t)
	if _, err := Put(context.Background(), StoreOptions{
		ReceiptPath: receiptPath, CandidateID: "missing", CandidatesDir: candidates, StoreDir: t.TempDir(),
	}); err == nil {
		t.Fatal("unknown candidate was accepted")
	}
}

package generatorattestation

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

func findSSHKeygen(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("ssh-keygen is not available")
	}
	return path
}

func selectionPaths(t *testing.T) (workspace, contract, fixtures, candidates string) {
	t.Helper()
	base := filepath.Join(repoRoot(t), "tools", "reference-generator-selection", "testdata")
	return filepath.Join(repoRoot(t), "reference", "workspace"),
		filepath.Join(base, "line-amount", "contract.json"),
		filepath.Join(base, "line-amount", "fixtures.json"),
		filepath.Join(base, "candidates")
}

func writeSelectionReceipt(t *testing.T) (receiptPath string) {
	t.Helper()
	workspace, contract, fixtures, candidates := selectionPaths(t)
	receipt, err := generatorselection.Select(context.Background(), workspace, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "selection.json")
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBuildStatementIsDeterministic(t *testing.T) {
	receiptPath := writeSelectionReceipt(t)
	statement, encoded, err := BuildStatementFromSelection(receiptPath, "template")
	if err != nil {
		t.Fatal(err)
	}
	if statement.Schema != StatementSchema || statement.CandidateID != "template" || statement.SourceDigest == "" || statement.ReceiptDigest == "" {
		t.Fatalf("statement is incomplete: %+v", statement)
	}
	again, againEncoded, err := BuildStatementFromSelection(receiptPath, "template")
	if err != nil {
		t.Fatal(err)
	}
	if statement != again || string(encoded) != string(againEncoded) {
		t.Fatal("statement is not deterministic")
	}
}

func TestBuildStatementRejectsUnknownCandidate(t *testing.T) {
	receiptPath := writeSelectionReceipt(t)
	if _, _, err := BuildStatementFromSelection(receiptPath, "missing"); err == nil {
		t.Fatal("unknown candidate was accepted")
	}
}

func TestSignAndVerifyStatement(t *testing.T) {
	keygen := findSSHKeygen(t)
	receiptPath := writeSelectionReceipt(t)
	_, encoded, err := BuildStatementFromSelection(receiptPath, "template")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	statementPath := filepath.Join(dir, "statement.json")
	if err := os.WriteFile(statementPath, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(dir, "key")
	keyCmd := exec.Command(keygen, "-q", "-t", "ed25519", "-N", "", "-f", keyPath)
	if output, err := keyCmd.CombinedOutput(); err != nil {
		t.Fatalf("key generation failed: %v: %s", err, output)
	}
	signaturePath := filepath.Join(dir, "statement.sig")
	signReceipt, err := Sign(context.Background(), statementPath, keyPath, signaturePath, Namespace)
	if err != nil {
		t.Fatal(err)
	}
	if signReceipt.SignatureDigest == "" || signReceipt.SignaturePath != signaturePath {
		t.Fatalf("signature receipt is incomplete: %+v", signReceipt)
	}
	publicKey, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	allowed := filepath.Join(dir, "allowed_signers")
	if err := os.WriteFile(allowed, []byte("signer "+strings.TrimSpace(string(publicKey))+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	verifyReceipt, err := Verify(context.Background(), statementPath, signaturePath, allowed, "signer", Namespace)
	if err != nil {
		t.Fatal(err)
	}
	if !verifyReceipt.Verified || verifyReceipt.Principal != "signer" || verifyReceipt.GrantsAuthority || verifyReceipt.VerifiesTrustRoot {
		t.Fatalf("valid signature was not verified: %+v", verifyReceipt)
	}
	if err := os.WriteFile(statementPath, append(encoded, ' '), 0o644); err != nil {
		t.Fatal(err)
	}
	tampered, err := Verify(context.Background(), statementPath, signaturePath, allowed, "signer", Namespace)
	if err != nil {
		t.Fatal(err)
	}
	if tampered.Verified || tampered.Reason == "" {
		t.Fatalf("tampered statement was accepted: %+v", tampered)
	}
	if err := os.WriteFile(statementPath, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	wrongIdentity, err := Verify(context.Background(), statementPath, signaturePath, allowed, "other", Namespace)
	if err != nil {
		t.Fatal(err)
	}
	if wrongIdentity.Verified {
		t.Fatalf("signature was accepted for an unknown identity: %+v", wrongIdentity)
	}
}

func TestSignRequiresExistingKey(t *testing.T) {
	dir := t.TempDir()
	statementPath := filepath.Join(dir, "statement.json")
	if err := os.WriteFile(statementPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Sign(context.Background(), statementPath, filepath.Join(dir, "missing-key"), "", Namespace); err == nil {
		t.Fatal("signature with a missing key was accepted")
	}
}

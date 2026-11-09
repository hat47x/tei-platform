package targetprofile

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func signPruneApproval(t *testing.T, approvalPath string) (signaturePath, allowedSigners string) {
	t.Helper()
	keygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("ssh-keygen is not available")
	}
	dir := filepath.Dir(approvalPath)
	keyPath := filepath.Join(dir, "approval-key")
	if output, err := exec.Command(keygen, "-q", "-t", "ed25519", "-N", "", "-f", keyPath).CombinedOutput(); err != nil {
		t.Fatalf("key generation failed: %v: %s", err, output)
	}
	if output, err := exec.Command(keygen, "-Y", "sign", "-f", keyPath, "-n", historyPruneNamespace, approvalPath).CombinedOutput(); err != nil {
		t.Fatalf("approval signing failed: %v: %s", err, output)
	}
	publicKey, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	allowedSigners = filepath.Join(dir, "allowed_signers")
	if err := os.WriteFile(allowedSigners, []byte("signer "+strings.TrimSpace(string(publicKey))+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return approvalPath + ".sig", allowedSigners
}

func writePruneApproval(t *testing.T, root string, keepLast int) string {
	t.Helper()
	approval, err := BuildHistoryPruneApproval(root, keepLast)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(approval, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "approval.json")
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPruneHistoryApproved(t *testing.T) {
	root := t.TempDir()
	writeHistoryArtifact(t, root, "one", time.Unix(100, 0))
	writeHistoryArtifact(t, root, "two", time.Unix(200, 0))
	writeHistoryArtifact(t, root, "three", time.Unix(300, 0))
	approvalPath := writePruneApproval(t, root, 2)
	signaturePath, allowedSigners := signPruneApproval(t, approvalPath)
	receipt, err := PruneHistoryApproved(context.Background(), root, 2, false, approvalPath, signaturePath, allowedSigners, "signer")
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || len(receipt.Deleted) != 1 || receipt.AuditPath == "" {
		t.Fatalf("approved prune failed: %+v", receipt)
	}
}

func TestPruneHistoryApprovedRejectsMismatch(t *testing.T) {
	root := t.TempDir()
	writeHistoryArtifact(t, root, "one", time.Unix(100, 0))
	writeHistoryArtifact(t, root, "two", time.Unix(200, 0))
	writeHistoryArtifact(t, root, "three", time.Unix(300, 0))
	approvalPath := writePruneApproval(t, root, 2)
	signaturePath, allowedSigners := signPruneApproval(t, approvalPath)
	receipt, err := PruneHistoryApproved(context.Background(), root, 1, false, approvalPath, signaturePath, allowedSigners, "signer")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || !strings.Contains(receipt.Reason, "keepLast") {
		t.Fatalf("mismatched approval was accepted: %+v", receipt)
	}
	entries, err := os.ReadDir(filepath.Join(root, "history"))
	if err != nil || len(entries) != 3 {
		t.Fatalf("mismatched approval changed history: %v %d", err, len(entries))
	}
}

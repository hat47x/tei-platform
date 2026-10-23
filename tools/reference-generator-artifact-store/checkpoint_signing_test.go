package generatorartifactstore

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckpointCanBeSignedAndVerified(t *testing.T) {
	keygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("ssh-keygen is not available")
	}
	checkpoint, err := Checkpoint(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	dir := t.TempDir()
	checkpointPath := filepath.Join(dir, "checkpoint.json")
	if err := os.WriteFile(checkpointPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(dir, "key")
	keyCmd := exec.Command(keygen, "-q", "-t", "ed25519", "-N", "", "-f", keyPath)
	if output, err := keyCmd.CombinedOutput(); err != nil {
		t.Fatalf("key generation failed: %v: %s", err, output)
	}
	sign := exec.Command(keygen, "-Y", "sign", "-f", keyPath, "-n", "tei-audit", checkpointPath)
	if output, err := sign.CombinedOutput(); err != nil {
		t.Fatalf("checkpoint signing failed: %v: %s", err, output)
	}
	signaturePath := checkpointPath + ".sig"
	publicKey, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	allowed := filepath.Join(dir, "allowed_signers")
	if err := os.WriteFile(allowed, []byte("signer "+strings.TrimSpace(string(publicKey))+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	verify := exec.Command(keygen, "-Y", "verify", "-f", allowed, "-I", "signer", "-n", "tei-audit", "-s", signaturePath)
	verify.Stdin = bytes.NewReader(raw)
	if output, err := verify.CombinedOutput(); err != nil {
		t.Fatalf("checkpoint verification failed: %v: %s", err, output)
	}
	tampered := append(append([]byte{}, raw...), ' ')
	verifyTampered := exec.Command(keygen, "-Y", "verify", "-f", allowed, "-I", "signer", "-n", "tei-audit", "-s", signaturePath)
	verifyTampered.Stdin = bytes.NewReader(tampered)
	if err := verifyTampered.Run(); err == nil {
		t.Fatal("tampered checkpoint signature was accepted")
	}
}

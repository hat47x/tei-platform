// Package generatorartifactstore stores Go generator candidate sources and
// binaries in a content-addressed local directory. It verifies the receipt,
// source, and binary digests; it does not sign, attest, or grant authority.
package generatorartifactstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	generatorselection "tei/tools/reference-generator-selection"
)

const (
	Schema       = "tei.reference.generator-artifact-store/v0"
	ObjectSchema = "tei.reference.generator-artifact-object/v0"
)

type StoreOptions struct {
	ReceiptPath   string
	CandidateID   string
	CandidatesDir string
	StoreDir      string
}

type Object struct {
	Schema          string `json:"schema"`
	CandidateID     string `json:"candidateId"`
	Language        string `json:"language"`
	ContractID      string `json:"contractId"`
	ContractVersion string `json:"contractVersion"`
	SourceDigest    string `json:"sourceDigest"`
	BinaryDigest    string `json:"binaryDigest"`
	ReceiptDigest   string `json:"receiptDigest"`
}

type Receipt struct {
	Schema           string   `json:"schema"`
	CandidateID      string   `json:"candidateId"`
	SourceDigest     string   `json:"sourceDigest"`
	BinaryDigest     string   `json:"binaryDigest"`
	ObjectPath       string   `json:"objectPath"`
	Reused           bool     `json:"reused"`
	Passed           bool     `json:"passed"`
	Reason           string   `json:"reason,omitempty"`
	GrantsAuthority  bool     `json:"grantsAuthority"`
	ProvesRealTarget bool     `json:"provesRealTarget"`
	Notes            []string `json:"notes"`
}

func digest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func digestHex(value string) (string, error) {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return "", fmt.Errorf("invalid sha256 digest %q", value)
	}
	return strings.TrimPrefix(value, "sha256:"), nil
}

// Put builds the selected Go candidate, verifies its source and binary against
// the selection receipt, and stores both under objects/<source hex>.
func Put(ctx context.Context, opts StoreOptions) (Receipt, error) {
	receipt := Receipt{
		Schema: Schema, CandidateID: opts.CandidateID, GrantsAuthority: false, ProvesRealTarget: false,
		Notes: []string{
			"選抜receiptのsource/binary digestを再計算し、content-addressedに保存する。",
			"署名・attestation・Authority・実Targetでの実行を証明しない。",
		},
	}
	rawReceipt, err := os.ReadFile(opts.ReceiptPath)
	if err != nil {
		return receipt, err
	}
	var selection generatorselection.Receipt
	if err := json.Unmarshal(rawReceipt, &selection); err != nil {
		return receipt, fmt.Errorf("%s: %w", opts.ReceiptPath, err)
	}
	var candidate *generatorselection.CandidateReceipt
	for i := range selection.Candidates {
		if selection.Candidates[i].ID == opts.CandidateID {
			candidate = &selection.Candidates[i]
			break
		}
	}
	if candidate == nil {
		return receipt, fmt.Errorf("candidate %q is not in the receipt", opts.CandidateID)
	}
	if candidate.Language != generatorselection.LanguageGo {
		receipt.Reason = "artifact store supports only Go candidates in this reference"
		return receipt, nil
	}
	receipt.SourceDigest = candidate.SourceDigest
	receipt.BinaryDigest = candidate.BinaryDigest
	sourceHex, err := digestHex(candidate.SourceDigest)
	if err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	candidateDir := filepath.Join(opts.CandidatesDir, candidate.ID)
	sourcePath := filepath.Join(candidateDir, "main.go")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		receipt.Reason = "candidate source is unreadable: " + err.Error()
		return receipt, nil
	}
	if digest(source) != candidate.SourceDigest {
		receipt.Reason = "candidate source digest does not match the receipt"
		return receipt, nil
	}
	tempDir, err := os.MkdirTemp("", "tei-artifact-store-*")
	if err != nil {
		return receipt, err
	}
	defer os.RemoveAll(tempDir)
	binaryPath := filepath.Join(tempDir, "artifact")
	if err := buildGoCandidate(ctx, candidateDir, binaryPath); err != nil {
		receipt.Reason = "candidate build failed: " + err.Error()
		return receipt, nil
	}
	binary, err := os.ReadFile(binaryPath)
	if err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	if digest(binary) != candidate.BinaryDigest {
		receipt.Reason = "rebuilt binary digest does not match the receipt"
		return receipt, nil
	}
	objectPath := filepath.Join(opts.StoreDir, "objects", sourceHex)
	if existing, err := os.ReadFile(filepath.Join(objectPath, "artifact")); err == nil {
		if digest(existing) == candidate.BinaryDigest {
			receipt.ObjectPath = objectPath
			receipt.Reused = true
			receipt.Passed = true
			return receipt, nil
		}
	}
	object := Object{
		Schema: ObjectSchema, CandidateID: candidate.ID, Language: candidate.Language,
		ContractID: selection.ContractID, ContractVersion: selection.ContractVersion,
		SourceDigest: candidate.SourceDigest, BinaryDigest: candidate.BinaryDigest,
		ReceiptDigest: digest(rawReceipt),
	}
	encodedObject, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		return receipt, err
	}
	if err := os.MkdirAll(objectPath, 0o755); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	if err := os.WriteFile(filepath.Join(objectPath, "source"), source, 0o644); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	if err := os.WriteFile(filepath.Join(objectPath, "artifact"), binary, 0o755); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	if err := os.WriteFile(filepath.Join(objectPath, "object.json"), append(encodedObject, '\n'), 0o644); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	receipt.ObjectPath = objectPath
	receipt.Passed = true
	return receipt, nil
}

// Get reads and re-verifies an object by its source digest.
func Get(storeDir, sourceDigest string) (Receipt, error) {
	receipt := Receipt{
		Schema: Schema, SourceDigest: sourceDigest, GrantsAuthority: false, ProvesRealTarget: false,
		Notes: []string{"content-addressed objectを読み、source/artifact digestを再計算する。"},
	}
	hexDigest, err := digestHex(sourceDigest)
	if err != nil {
		return receipt, err
	}
	objectPath := filepath.Join(storeDir, "objects", hexDigest)
	rawObject, err := os.ReadFile(filepath.Join(objectPath, "object.json"))
	if err != nil {
		return receipt, err
	}
	var object Object
	if err := json.Unmarshal(rawObject, &object); err != nil {
		return receipt, err
	}
	if object.SourceDigest != sourceDigest {
		receipt.Reason = "object metadata source digest does not match the requested digest"
		return receipt, nil
	}
	source, err := os.ReadFile(filepath.Join(objectPath, "source"))
	if err != nil {
		return receipt, err
	}
	artifact, err := os.ReadFile(filepath.Join(objectPath, "artifact"))
	if err != nil {
		return receipt, err
	}
	if digest(source) != object.SourceDigest {
		receipt.Reason = "stored source digest does not match"
		return receipt, nil
	}
	if digest(artifact) != object.BinaryDigest {
		receipt.Reason = "stored artifact digest does not match"
		return receipt, nil
	}
	receipt.CandidateID = object.CandidateID
	receipt.BinaryDigest = object.BinaryDigest
	receipt.ObjectPath = objectPath
	receipt.Passed = true
	return receipt, nil
}

func buildGoCandidate(ctx context.Context, dir, binaryPath string) error {
	build := exec.CommandContext(ctx, "go", "build", "-trimpath", "-buildvcs=false", "-o", binaryPath, "main.go")
	build.Dir = dir
	out, err := build.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return nil
}

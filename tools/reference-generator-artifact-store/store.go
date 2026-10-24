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
	switch candidate.Language {
	case generatorselection.LanguageGo, generatorselection.LanguageJava, generatorselection.LanguageWasm,
		generatorselection.LanguageWasi, generatorselection.LanguagePython, generatorselection.LanguageJavaScript:
	default:
		receipt.Reason = "unsupported candidate language in this reference"
		return receipt, nil
	}
	sourceOnly := candidate.Language == generatorselection.LanguagePython || candidate.Language == generatorselection.LanguageJavaScript
	receipt.SourceDigest = candidate.SourceDigest
	receipt.BinaryDigest = candidate.BinaryDigest
	sourceHex, err := digestHex(candidate.SourceDigest)
	if err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	candidateDir := filepath.Join(opts.CandidatesDir, candidate.ID)
	sourceName := "main.go"
	switch candidate.Language {
	case generatorselection.LanguageJava:
		sourceName = "Main.java"
	case generatorselection.LanguagePython:
		sourceName = "main.py"
	case generatorselection.LanguageJavaScript:
		sourceName = "main.js"
	}
	sourcePath := filepath.Join(candidateDir, sourceName)
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
	var binary []byte
	if !sourceOnly {
		binaryPath := filepath.Join(tempDir, "artifact")
		if err := buildArtifact(ctx, candidate.Language, candidateDir, binaryPath); err != nil {
			receipt.Reason = "candidate build failed: " + err.Error()
			return receipt, nil
		}
		binary, err = os.ReadFile(binaryPath)
		if err != nil {
			receipt.Reason = err.Error()
			return receipt, nil
		}
		if digest(binary) != candidate.BinaryDigest {
			receipt.Reason = "rebuilt binary digest does not match the receipt"
			return receipt, nil
		}
	} else if candidate.BinaryDigest != "" {
		receipt.Reason = "source-only candidate unexpectedly declares a binary digest"
		return receipt, nil
	}
	objectPath := filepath.Join(opts.StoreDir, "objects", sourceHex)
	if existing, err := os.ReadFile(filepath.Join(objectPath, "artifact")); err == nil {
		existingMatches := digest(existing) == candidate.BinaryDigest
		if candidate.BinaryDigest == "" {
			existingMatches = len(existing) == 0
		}
		if existingMatches {
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
	if object.BinaryDigest == "" {
		if len(artifact) != 0 {
			receipt.Reason = "source-only artifact is not empty"
			return receipt, nil
		}
	} else if digest(artifact) != object.BinaryDigest {
		receipt.Reason = "stored artifact digest does not match"
		return receipt, nil
	}
	receipt.CandidateID = object.CandidateID
	receipt.BinaryDigest = object.BinaryDigest
	receipt.ObjectPath = objectPath
	receipt.Passed = true
	return receipt, nil
}

func buildArtifact(ctx context.Context, language, dir, binaryPath string) error {
	if language == generatorselection.LanguageJava {
		javac, err := exec.LookPath("javac")
		if err != nil {
			return fmt.Errorf("javac is unavailable: %w", err)
		}
		classesDir, err := os.MkdirTemp("", "tei-artifact-java-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(classesDir)
		compile := exec.CommandContext(ctx, javac, "-d", classesDir, "Main.java")
		compile.Dir = dir
		if out, err := compile.CombinedOutput(); err != nil {
			return fmt.Errorf("%v: %s", err, out)
		}
		classBytes, err := os.ReadFile(filepath.Join(classesDir, "Main.class"))
		if err != nil {
			return err
		}
		return os.WriteFile(binaryPath, classBytes, 0o755)
	}
	build := exec.CommandContext(ctx, "go", "build", "-trimpath", "-buildvcs=false", "-o", binaryPath, "main.go")
	build.Dir = dir
	switch language {
	case generatorselection.LanguageGo:
	case generatorselection.LanguageWasm:
		build.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	case generatorselection.LanguageWasi:
		build.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	default:
		return fmt.Errorf("unsupported artifact language %q", language)
	}
	out, err := build.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return nil
}

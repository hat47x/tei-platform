package generatorartifactstore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	ListSchema       = "tei.reference.generator-artifact-list/v0"
	PruneSchema      = "tei.reference.generator-artifact-prune/v0"
	AuditSchema      = "tei.reference.generator-artifact-prune-audit/v0"
	CheckpointSchema = "tei.reference.generator-artifact-audit-checkpoint/v0"
)

type StoredObject struct {
	SourceDigest  string `json:"sourceDigest"`
	BinaryDigest  string `json:"binaryDigest"`
	CandidateID   string `json:"candidateId"`
	Path          string `json:"path"`
	SourceBytes   int64  `json:"sourceBytes"`
	ArtifactBytes int64  `json:"artifactBytes"`
	Verified      bool   `json:"verified"`
	Reason        string `json:"reason,omitempty"`
}

type ListReceipt struct {
	Schema   string         `json:"schema"`
	StoreDir string         `json:"storeDir"`
	Objects  []StoredObject `json:"objects"`
	Passed   bool           `json:"passed"`
	Reason   string         `json:"reason,omitempty"`
}

type PruneOptions struct {
	StoreDir string
	Keep     []string
	DryRun   bool
}

type PruneReceipt struct {
	Schema           string   `json:"schema"`
	StoreDir         string   `json:"storeDir"`
	DryRun           bool     `json:"dryRun"`
	Keep             []string `json:"keep"`
	Kept             []string `json:"kept"`
	Deleted          []string `json:"deleted,omitempty"`
	WouldDelete      []string `json:"wouldDelete,omitempty"`
	MissingKeep      []string `json:"missingKeep,omitempty"`
	AuditPath        string   `json:"auditPath,omitempty"`
	AuditRecorded    bool     `json:"auditRecorded"`
	Passed           bool     `json:"passed"`
	Reason           string   `json:"reason,omitempty"`
	GrantsAuthority  bool     `json:"grantsAuthority"`
	ProvesRealTarget bool     `json:"provesRealTarget"`
	Notes            []string `json:"notes"`
}

type AuditRecord struct {
	Schema         string   `json:"schema"`
	Operation      string   `json:"operation"`
	RecordedAt     string   `json:"recordedAt"`
	DryRun         bool     `json:"dryRun"`
	Keep           []string `json:"keep"`
	Kept           []string `json:"kept"`
	Deleted        []string `json:"deleted,omitempty"`
	WouldDelete    []string `json:"wouldDelete,omitempty"`
	Passed         bool     `json:"passed"`
	Reason         string   `json:"reason,omitempty"`
	PreviousDigest string   `json:"previousDigest,omitempty"`
	RecordDigest   string   `json:"recordDigest"`
}

type AuditReceipt struct {
	Schema   string        `json:"schema"`
	StoreDir string        `json:"storeDir"`
	Records  []AuditRecord `json:"records"`
	Passed   bool          `json:"passed"`
	Reason   string        `json:"reason,omitempty"`
}

// List reads every object directory and re-verifies its source and artifact
// digests. A malformed or tampered object makes the list fail closed.
func List(storeDir string) (ListReceipt, error) {
	receipt := ListReceipt{
		Schema: ListSchema, StoreDir: storeDir, Objects: []StoredObject{}, Passed: true,
	}
	objectsDir := filepath.Join(storeDir, "objects")
	entries, err := os.ReadDir(objectsDir)
	if os.IsNotExist(err) {
		return receipt, nil
	}
	if err != nil {
		return receipt, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		objectPath := filepath.Join(objectsDir, entry.Name())
		object := StoredObject{Path: objectPath}
		rawObject, err := os.ReadFile(filepath.Join(objectPath, "object.json"))
		if err != nil {
			object.Reason = "object metadata is unreadable: " + err.Error()
			receipt.Passed = false
			receipt.Objects = append(receipt.Objects, object)
			continue
		}
		var metadata Object
		if err := json.Unmarshal(rawObject, &metadata); err != nil {
			object.Reason = "object metadata is invalid: " + err.Error()
			receipt.Passed = false
			receipt.Objects = append(receipt.Objects, object)
			continue
		}
		object.SourceDigest = metadata.SourceDigest
		object.BinaryDigest = metadata.BinaryDigest
		object.CandidateID = metadata.CandidateID
		source, sourceErr := os.ReadFile(filepath.Join(objectPath, "source"))
		artifact, artifactErr := os.ReadFile(filepath.Join(objectPath, "artifact"))
		switch {
		case sourceErr != nil:
			object.Reason = "source is unreadable: " + sourceErr.Error()
		case artifactErr != nil:
			object.Reason = "artifact is unreadable: " + artifactErr.Error()
		case digest(source) != metadata.SourceDigest:
			object.Reason = "stored source digest does not match"
		case digest(artifact) != metadata.BinaryDigest:
			object.Reason = "stored artifact digest does not match"
		case filepath.Base(objectPath) != trimDigest(metadata.SourceDigest):
			object.Reason = "object path does not match the source digest"
		default:
			object.Verified = true
			object.SourceBytes = int64(len(source))
			object.ArtifactBytes = int64(len(artifact))
		}
		if !object.Verified {
			receipt.Passed = false
		}
		receipt.Objects = append(receipt.Objects, object)
	}
	sort.Slice(receipt.Objects, func(i, j int) bool {
		return receipt.Objects[i].SourceDigest < receipt.Objects[j].SourceDigest
	})
	if !receipt.Passed {
		receipt.Reason = "one or more objects failed verification"
	}
	return receipt, nil
}

// Prune deletes objects that are not in an explicit keep set. It refuses to
// delete anything when the keep set is empty, has an unknown digest, or the
// store contains an unverified object. It does not choose a retention period.
func Prune(opts PruneOptions) (PruneReceipt, error) {
	receipt := PruneReceipt{
		Schema: PruneSchema, StoreDir: opts.StoreDir, DryRun: opts.DryRun,
		Keep: []string{}, Kept: []string{}, Deleted: []string{}, WouldDelete: []string{}, MissingKeep: []string{},
		GrantsAuthority: false, ProvesRealTarget: false,
		Notes: []string{
			"明示したkeep setだけを残し、それ以外のobjectを削除する参照。",
			"保持期間・監査アクセス権・削除承認は決めない。",
			"空のkeep set、未知のkeep digest、検証失敗objectがある場合は削除しない。",
		},
	}
	if len(opts.Keep) == 0 {
		receipt.Reason = "keep set must not be empty"
		return receipt, nil
	}
	keep := map[string]bool{}
	for _, sourceDigest := range opts.Keep {
		if _, err := digestHex(sourceDigest); err != nil {
			receipt.Reason = err.Error()
			return receipt, nil
		}
		if !keep[sourceDigest] {
			keep[sourceDigest] = true
			receipt.Keep = append(receipt.Keep, sourceDigest)
		}
	}
	sort.Strings(receipt.Keep)
	list, err := List(opts.StoreDir)
	if err != nil {
		return receipt, err
	}
	if !list.Passed {
		receipt.Reason = "store contains an unverified object; refusing to prune"
		return receipt, nil
	}
	records, err := readAudit(opts.StoreDir)
	if err != nil {
		receipt.Reason = "existing audit log is invalid; refusing to prune: " + err.Error()
		return receipt, nil
	}
	previousDigest := ""
	if len(records) > 0 {
		previousDigest = records[len(records)-1].RecordDigest
	}
	found := map[string]bool{}
	for _, object := range list.Objects {
		if keep[object.SourceDigest] {
			found[object.SourceDigest] = true
			receipt.Kept = append(receipt.Kept, object.SourceDigest)
			continue
		}
		if opts.DryRun {
			receipt.WouldDelete = append(receipt.WouldDelete, object.SourceDigest)
			continue
		}
		if err := os.RemoveAll(object.Path); err != nil {
			receipt.Reason = "deleting " + object.SourceDigest + ": " + err.Error()
			return receipt, nil
		}
		receipt.Deleted = append(receipt.Deleted, object.SourceDigest)
	}
	for _, sourceDigest := range receipt.Keep {
		if !found[sourceDigest] {
			receipt.MissingKeep = append(receipt.MissingKeep, sourceDigest)
		}
	}
	if len(receipt.MissingKeep) > 0 {
		receipt.Reason = "keep set contains a digest that is not in the store"
		return receipt, nil
	}
	sort.Strings(receipt.Kept)
	sort.Strings(receipt.Deleted)
	sort.Strings(receipt.WouldDelete)
	receipt.AuditPath = filepath.Join(opts.StoreDir, "prune-audit.jsonl")
	if err := appendPruneAudit(opts.StoreDir, previousDigest, receipt); err != nil {
		receipt.Reason = "prune completed but audit append failed: " + err.Error()
		return receipt, nil
	}
	receipt.AuditRecorded = true
	receipt.Passed = true
	return receipt, nil
}

func appendPruneAudit(storeDir, previousDigest string, receipt PruneReceipt) error {
	path := filepath.Join(storeDir, "prune-audit.jsonl")
	record := AuditRecord{
		Schema: AuditSchema, Operation: "prune", RecordedAt: time.Now().UTC().Format(time.RFC3339Nano),
		DryRun: receipt.DryRun, Keep: receipt.Keep, Kept: receipt.Kept,
		Deleted: receipt.Deleted, WouldDelete: receipt.WouldDelete, Passed: true,
		PreviousDigest: previousDigest,
	}
	record.RecordDigest = auditRecordDigest(record)
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(raw, '\n')); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func auditRecordDigest(record AuditRecord) string {
	record.RecordDigest = ""
	raw, err := json.Marshal(record)
	if err != nil {
		return ""
	}
	return digest(raw)
}

// Audit reads the local hash-chained prune audit log. It verifies JSON syntax,
// record schema, each record digest, and the previous-record link. It does not
// verify identity, approval, or a signature.
func Audit(storeDir string) (AuditReceipt, error) {
	receipt := AuditReceipt{
		Schema: AuditSchema, StoreDir: storeDir, Records: []AuditRecord{}, Passed: true,
	}
	records, err := readAudit(storeDir)
	if err != nil {
		receipt.Passed = false
		receipt.Reason = err.Error()
		return receipt, nil
	}
	receipt.Records = records
	return receipt, nil
}

func readAudit(storeDir string) ([]AuditRecord, error) {
	raw, err := os.ReadFile(filepath.Join(storeDir, "prune-audit.jsonl"))
	if os.IsNotExist(err) {
		return []AuditRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	records := []AuditRecord{}
	previous := ""
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var record AuditRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, fmt.Errorf("audit line is invalid: %w", err)
		}
		if record.Schema != AuditSchema || record.Operation != "prune" || record.RecordedAt == "" {
			return nil, fmt.Errorf("audit record is incomplete")
		}
		if record.PreviousDigest != previous {
			return nil, fmt.Errorf("audit chain is broken at %s", record.RecordedAt)
		}
		if record.RecordDigest == "" || record.RecordDigest != auditRecordDigest(record) {
			return nil, fmt.Errorf("audit record digest does not match at %s", record.RecordedAt)
		}
		previous = record.RecordDigest
		records = append(records, record)
	}
	return records, nil
}

func trimDigest(value string) string {
	if len(value) > len("sha256:") {
		return value[len("sha256:"):]
	}
	return value
}

type CheckpointReceipt struct {
	Schema           string `json:"schema"`
	StoreDir         string `json:"storeDir"`
	RecordCount      int    `json:"recordCount"`
	HeadDigest       string `json:"headDigest,omitempty"`
	AuditDigest      string `json:"auditDigest"`
	Passed           bool   `json:"passed"`
	Reason           string `json:"reason,omitempty"`
	GrantsAuthority  bool   `json:"grantsAuthority"`
	ProvesRealTarget bool   `json:"provesRealTarget"`
}

// Checkpoint summarizes the audit chain without signing it. The caller can
// sign the emitted JSON or a file containing it with an external tool.
func Checkpoint(storeDir string) (CheckpointReceipt, error) {
	receipt := CheckpointReceipt{
		Schema: CheckpointSchema, StoreDir: storeDir, GrantsAuthority: false, ProvesRealTarget: false,
	}
	records, err := readAudit(storeDir)
	if err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	raw, err := os.ReadFile(filepath.Join(storeDir, "prune-audit.jsonl"))
	if os.IsNotExist(err) {
		raw = nil
	} else if err != nil {
		return receipt, err
	}
	receipt.RecordCount = len(records)
	receipt.AuditDigest = digest(raw)
	if len(records) > 0 {
		receipt.HeadDigest = records[len(records)-1].RecordDigest
	}
	receipt.Passed = true
	return receipt, nil
}

type CheckpointVerification struct {
	Schema         string            `json:"schema"`
	StoreDir       string            `json:"storeDir"`
	CheckpointPath string            `json:"checkpointPath"`
	Expected       CheckpointReceipt `json:"expected"`
	Actual         CheckpointReceipt `json:"actual"`
	Passed         bool              `json:"passed"`
	Reason         string            `json:"reason,omitempty"`
}

const CheckpointVerificationSchema = "tei.reference.generator-artifact-checkpoint-verification/v0"

// VerifyCheckpoint compares a stored checkpoint file with the current audit
// chain. It checks semantic equality, not a signature or the signer's identity.
func VerifyCheckpoint(storeDir, checkpointPath string) (CheckpointVerification, error) {
	verification := CheckpointVerification{
		Schema: CheckpointVerificationSchema, StoreDir: storeDir, CheckpointPath: checkpointPath,
	}
	raw, err := os.ReadFile(checkpointPath)
	if err != nil {
		return verification, err
	}
	var expected CheckpointReceipt
	if err := json.Unmarshal(raw, &expected); err != nil {
		return verification, err
	}
	verification.Expected = expected
	if expected.Schema != CheckpointSchema || !expected.Passed {
		verification.Reason = "stored checkpoint is not a passing checkpoint"
		return verification, nil
	}
	actual, err := Checkpoint(storeDir)
	if err != nil {
		return verification, err
	}
	verification.Actual = actual
	if !actual.Passed {
		verification.Reason = "current audit checkpoint is not passing: " + actual.Reason
		return verification, nil
	}
	switch {
	case expected.RecordCount != actual.RecordCount:
		verification.Reason = "record count differs from the stored checkpoint"
	case expected.HeadDigest != actual.HeadDigest:
		verification.Reason = "head digest differs from the stored checkpoint"
	case expected.AuditDigest != actual.AuditDigest:
		verification.Reason = "audit digest differs from the stored checkpoint"
	default:
		verification.Passed = true
	}
	return verification, nil
}

type SignedCheckpointVerification struct {
	CheckpointVerification
	SignaturePath     string `json:"signaturePath"`
	Identity          string `json:"identity"`
	SignatureVerified bool   `json:"signatureVerified"`
}

// VerifySignedCheckpoint verifies a detached ssh-keygen signature over the
// checkpoint bytes, then compares the checkpoint with the current audit chain.
// It does not verify the signer's authority or trust root.
func VerifySignedCheckpoint(ctx context.Context, storeDir, checkpointPath, signaturePath, allowedSigners, identity, namespace string) (SignedCheckpointVerification, error) {
	verification := SignedCheckpointVerification{
		CheckpointVerification: CheckpointVerification{
			Schema: CheckpointVerificationSchema, StoreDir: storeDir, CheckpointPath: checkpointPath,
		},
		SignaturePath: signaturePath, Identity: identity,
	}
	if namespace == "" {
		namespace = "tei-audit"
	}
	raw, err := os.ReadFile(checkpointPath)
	if err != nil {
		return verification, err
	}
	verify := exec.CommandContext(ctx, "ssh-keygen", "-Y", "verify",
		"-f", allowedSigners, "-I", identity, "-n", namespace, "-s", signaturePath)
	verify.Stdin = strings.NewReader(string(raw))
	if output, err := verify.CombinedOutput(); err != nil {
		verification.Reason = "checkpoint signature verification failed: " + strings.TrimSpace(string(output))
		return verification, nil
	}
	verification.SignatureVerified = true
	semantic, err := VerifyCheckpoint(storeDir, checkpointPath)
	if err != nil {
		return verification, err
	}
	verification.CheckpointVerification = semantic
	if !semantic.Passed {
		return verification, nil
	}
	verification.Passed = true
	return verification, nil
}

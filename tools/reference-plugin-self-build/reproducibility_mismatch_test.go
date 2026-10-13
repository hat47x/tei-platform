package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelfBuildMismatchEvidenceRetainsDigestsWithoutAcceptingArtifact(t *testing.T) {
	first := "sha256:" + strings.Repeat("a", 64)
	second := "sha256:" + strings.Repeat("b", 64)
	err := fmt.Errorf("transform bootstrap: %w", nonReproducibleSelfBuild(
		"transform-seed-builder", first, second,
	))
	var out bytes.Buffer
	matched, emitErr := emitSelfBuildMismatchEvidence(&out, err)
	if emitErr != nil || !matched {
		t.Fatalf("wrapped mismatch not emitted: matched=%t err=%v", matched, emitErr)
	}
	var receipt selfBuildMismatchEvidence
	if err := json.Unmarshal(out.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Schema != selfBuildMismatchSchema ||
		receipt.Stage != "transform-seed-builder" ||
		receipt.Outcome != "reproducibility-failure" ||
		receipt.FirstArtifactDigest != first ||
		receipt.SecondArtifactDigest != second ||
		receipt.Reproducible ||
		receipt.ArtifactDigest != "" ||
		receipt.ArtifactAccepted ||
		receipt.ArtifactExportAllowed ||
		receipt.ArtifactLockAllowed {
		t.Fatalf("mismatch was promoted to an accepted build: %+v", receipt)
	}
	if len(receipt.AuthorityLimits) == 0 || strings.Contains(out.String(), "lockedBuild") {
		t.Fatalf("mismatch lost its authority boundary: %s", out.String())
	}
}

func TestSelfBuildExecutionErrorDoesNotMasqueradeAsReproducibilityMismatch(t *testing.T) {
	var out bytes.Buffer
	matched, err := emitSelfBuildMismatchEvidence(&out, errors.New("compiler exited before producing outputs"))
	if err != nil || matched || out.Len() != 0 {
		t.Fatalf("execution failure was treated as output comparison: matched=%t err=%v out=%q", matched, err, out.String())
	}
}

type failedEvidenceWriter struct{}

func (failedEvidenceWriter) Write([]byte) (int, error) {
	return 0, errors.New("stdout unavailable")
}

func TestSelfBuildMismatchEvidenceWriteFailureIsReported(t *testing.T) {
	matched, err := emitSelfBuildMismatchEvidence(
		failedEvidenceWriter{},
		nonReproducibleSelfBuild("process-plugin-binary", "sha256:first", "sha256:second"),
	)
	if !matched || err == nil {
		t.Fatalf("failed output was silently ignored: matched=%t err=%v", matched, err)
	}
}

func TestSelfBuildFileComparisonKeepsMismatchAsTypedDomainEvidence(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "builder-first")
	second := filepath.Join(dir, "builder-second")
	if err := os.WriteFile(first, []byte("real first build"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("real second build"), 0o600); err != nil {
		t.Fatal(err)
	}
	a, b, err := compareSelfBuildFileDigests("native-plugin-builder-binary", first, second)
	var mismatch *selfBuildMismatchError
	if !errors.As(err, &mismatch) || a == "" || b == "" || a == b {
		t.Fatalf("distinct files did not create typed mismatch: first=%s second=%s error=%v", a, b, err)
	}
	if mismatch.Evidence.Stage != "native-plugin-builder-binary" ||
		mismatch.Evidence.FirstArtifactDigest != a ||
		mismatch.Evidence.SecondArtifactDigest != b ||
		mismatch.Evidence.ArtifactDigest != "" ||
		mismatch.Evidence.ArtifactAccepted ||
		mismatch.Evidence.ArtifactExportAllowed ||
		mismatch.Evidence.ArtifactLockAllowed {
		t.Fatalf("typed mismatch changed Plugin acceptance authority: %+v", mismatch.Evidence)
	}
	if err := os.WriteFile(second, []byte("real first build"), 0o600); err != nil {
		t.Fatal(err)
	}
	a, b, err = compareSelfBuildFileDigests("native-plugin-builder-binary", first, second)
	if err != nil || a == "" || a != b {
		t.Fatalf("matching real files rejected: a=%s b=%s err=%v", a, b, err)
	}
}

func TestSelfBuildMissingFileNeverEmitsMismatchReceipt(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "builder-first")
	second := filepath.Join(dir, "builder-second")
	if err := os.WriteFile(first, []byte("only first completed"), 0o600); err != nil {
		t.Fatal(err)
	}
	a, b, err := compareSelfBuildFileDigests("process-plugin-binary", first, second)
	if err == nil || a == "" || b != "" {
		t.Fatalf("missing output did not fail closed: a=%s b=%s err=%v", a, b, err)
	}
	var receipt bytes.Buffer
	matched, emitErr := emitSelfBuildMismatchEvidence(&receipt, err)
	if matched || emitErr != nil || receipt.Len() != 0 {
		t.Fatalf("missing output was misclassified as reproducibility mismatch: receipt=%q err=%v", receipt.String(), emitErr)
	}
}

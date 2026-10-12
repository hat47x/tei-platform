package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

package gobuilder

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareArtifactFilesPreservesActualOutputsWithoutAuthority(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "first"), filepath.Join(dir, "second")
	first := []byte("real first build output")
	if err := os.WriteFile(a, first, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, first, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(first)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	same, err := CompareArtifactFiles(a, b)
	if err != nil || same.FirstDigest != digest || same.SecondDigest != digest ||
		!same.Reproducible || same.CandidateDigest != digest {
		t.Fatalf("actual equal outputs not recorded: %+v: %v", same, err)
	}
	if err := os.WriteFile(b, []byte("modified second build output"), 0o600); err != nil {
		t.Fatal(err)
	}
	divergent, err := CompareArtifactFiles(a, b)
	if err != nil || divergent.FirstDigest != digest ||
		divergent.SecondDigest == digest || !strings.HasPrefix(divergent.SecondDigest, "sha256:") ||
		divergent.Reproducible || divergent.CandidateDigest != "" {
		t.Fatalf("divergence was promoted to an accepted candidate: %+v: %v", divergent, err)
	}
	if after, err := os.ReadFile(a); err != nil || string(after) != string(first) {
		t.Fatalf("comparing altered or exported the build output: %q: %v", after, err)
	}
}

func TestCompareArtifactFilesSeparatesMissingOutputFromMismatchEvidence(t *testing.T) {
	dir := t.TempDir()
	first, second := filepath.Join(dir, "first"), filepath.Join(dir, "second")
	cmp, err := CompareArtifactFiles(first, second)
	if err == nil || !strings.Contains(err.Error(), "first Go build artifact") ||
		cmp.FirstDigest != "" || cmp.SecondDigest != "" || cmp.CandidateDigest != "" || cmp.Reproducible {
		t.Fatalf("missing first output falsely became a comparison: %+v: %v", cmp, err)
	}
	if err := os.WriteFile(first, []byte("completed first output"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmp, err = CompareArtifactFiles(first, second)
	if err == nil || !strings.Contains(err.Error(), "second Go build artifact") ||
		cmp.FirstDigest == "" || cmp.SecondDigest != "" || cmp.CandidateDigest != "" || cmp.Reproducible {
		t.Fatalf("missing second output falsely became reproducibility Evidence: %+v: %v", cmp, err)
	}
	cmp, err = CompareArtifactFiles(first, dir)
	if err == nil || !strings.Contains(err.Error(), "not a regular file") ||
		cmp.CandidateDigest != "" || cmp.Reproducible {
		t.Fatalf("directory was treated as a build output: %+v: %v", cmp, err)
	}
}

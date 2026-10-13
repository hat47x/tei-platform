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

func TestCompareArtifactFilesRejectsSameFileAsTwoBuildAttempts(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first")
	if err := os.WriteFile(first, []byte("independently built?"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmp, err := CompareArtifactFiles(first, first)
	if err == nil || !strings.Contains(err.Error(), "not two independent outputs") ||
		cmp.FirstDigest == "" || cmp.SecondDigest != cmp.FirstDigest ||
		cmp.Reproducible || cmp.CandidateDigest != "" {
		t.Fatalf("single file was promoted to two reproducible builds: %+v: %v", cmp, err)
	}

	// A different pathname is not evidence of an independent build when
	// two paths share the same underlying inode/file identity.
	hardLink := filepath.Join(dir, "hardlink-second")
	if err := os.Link(first, hardLink); err != nil {
		t.Skipf("hard links unavailable on this filesystem: %v", err)
	}
	cmp, err = CompareArtifactFiles(first, hardLink)
	if err == nil || !strings.Contains(err.Error(), "not two independent outputs") ||
		cmp.CandidateDigest != "" || cmp.Reproducible {
		t.Fatalf("hard-link alias was promoted to reproducible build: %+v: %v", cmp, err)
	}
}

func TestCompareArtifactFilesRejectsSymlinkToOtherwiseValidArtifact(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first")
	second := filepath.Join(dir, "second")
	alias := filepath.Join(dir, "alias-second")
	if err := os.WriteFile(first, []byte("same bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("same bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(second, alias); err != nil {
		t.Skipf("symlink creation unavailable on this host: %v", err)
	}
	cmp, err := CompareArtifactFiles(first, alias)
	if err == nil || !strings.Contains(err.Error(), "not a regular file") ||
		cmp.FirstDigest == "" || cmp.SecondDigest != "" ||
		cmp.Reproducible || cmp.CandidateDigest != "" {
		t.Fatalf("symlinked artifact was treated as independent direct output: %+v: %v", cmp, err)
	}
	// Reproducibility remains observable for two genuinely separate files.
	cmp, err = CompareArtifactFiles(first, second)
	if err != nil || !cmp.Reproducible || cmp.CandidateDigest != cmp.FirstDigest {
		t.Fatalf("two direct regular outputs were incorrectly refused: %+v: %v", cmp, err)
	}
}

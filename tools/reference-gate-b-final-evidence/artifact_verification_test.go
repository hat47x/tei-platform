package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func retainedArtifactFixture(t *testing.T) (laneArtifactStore, []laneEvidence) {
	t.Helper()
	store := laneArtifactStore{dir: t.TempDir()}
	lanes := make([]laneEvidence, 0, 3)
	for _, spec := range laneSpecs(false) {
		content := []byte(fmt.Sprintf("{\n  \"evidenceNature\": %q,\n  \"repositoryRevision\": \"R\",\n  \"reproducibleRevision\": true\n}\n", spec.ExpectedNature))
		filename, err := store.writeLane(spec, content)
		if err != nil {
			t.Fatalf("writeLane(%s) error=%v", spec.Name, err)
		}
		lanes = append(lanes, laneEvidence{
			Name:                        spec.Name,
			Passed:                      true,
			ArtifactFile:                filename,
			EvidenceArtifactFingerprint: evidenceArtifactFingerprint(content),
		})
	}
	return store, lanes
}

func TestVerifyRetainedLanesAcceptsExactSameRunArtifacts(t *testing.T) {
	store, lanes := retainedArtifactFixture(t)
	if err := store.verifyRetainedLanes(lanes); err != nil {
		t.Fatalf("verifyRetainedLanes() error=%v", err)
	}
}

func TestVerifyRetainedLanesRejectsModifiedArtifact(t *testing.T) {
	store, lanes := retainedArtifactFixture(t)
	path := filepath.Join(store.dir, lanes[1].ArtifactFile)
	if err := os.WriteFile(path, []byte("{\"tampered\":true}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.verifyRetainedLanes(lanes); err == nil {
		t.Fatal("保存後にbytesが変わったartifactをretained=true相当で受理しています")
	}
}

func TestVerifyRetainedLanesRejectsMissingArtifact(t *testing.T) {
	store, lanes := retainedArtifactFixture(t)
	path := filepath.Join(store.dir, lanes[2].ArtifactFile)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := store.verifyRetainedLanes(lanes); err == nil {
		t.Fatal("final確認時に存在しないartifactをretained=true相当で受理しています")
	}
}

func TestVerifyRetainedLanesRejectsSymlinkArtifact(t *testing.T) {
	store, lanes := retainedArtifactFixture(t)
	path := filepath.Join(store.dir, lanes[0].ArtifactFile)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	replacement := filepath.Join(store.dir, "replacement.json")
	if err := os.WriteFile(replacement, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(replacement, path); err != nil {
		t.Skipf("symlink test unavailable: %v", err)
	}
	if err := store.verifyRetainedLanes(lanes); err == nil {
		t.Fatal("symlinkへ置換されたartifactをretained=true相当で受理しています")
	}
}

func TestVerifyRetainedLanesRejectsUnexpectedArtifactFilename(t *testing.T) {
	store, lanes := retainedArtifactFixture(t)
	lanes[0].ArtifactFile = "other.json"
	if err := store.verifyRetainedLanes(lanes); err == nil {
		t.Fatal("laneに対応しないartifact filenameを受理しています")
	}
}

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGuardRejectsStaleReferenceSourceBeforeNavigation(t *testing.T) {
	f, r, s, w := sample()
	p := build(f, r, s, w, "procedure.json", "review.json", "set.json")
	dir := t.TempDir()
	path := filepath.Join(dir, "procedure.json")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := snapshot(path)
	if err != nil {
		t.Fatal(err)
	}
	valid, err := navigateGuarded(p, "step:BP-04", []navigationSnapshot{before})
	if err != nil || valid.Status != "reference-only" {
		t.Fatalf("valid source navigation failed: %v / %+v", err, valid)
	}
	if err := os.WriteFile(path, []byte("concurrent editor updated reference"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := navigateGuarded(p, "step:BP-04", []navigationSnapshot{before}); err == nil ||
		!strings.Contains(err.Error(), "DFD_REFERENCE_SOURCE_CHANGED_AFTER_PROJECTION") {
		t.Fatalf("stale source was navigable without detection: %v", err)
	}
	if err := verifySnapshots([]navigationSnapshot{before}); err == nil {
		t.Fatal("stale review source was accepted as stable")
	}
}

func TestVerifySnapshotsFailsClosedOnMissingArtifact(t *testing.T) {
	if err := verifySnapshots([]navigationSnapshot{{File: filepath.Join(t.TempDir(), "gone.json"), SHA256: "sha256:missing"}}); err == nil {
		t.Fatal("removed reference artifact was accepted")
	}
}

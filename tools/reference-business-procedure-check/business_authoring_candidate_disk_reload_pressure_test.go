package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/workspace"
)

func TestBusinessCandidateOnlyDiskReloadPreservesNormalizedMeaningWithoutLegacyRead(t *testing.T) {
	repoRoot := repoPath()
	candidate, err := migrateCurrentReferenceAuthoringProbe(t, repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	want, err := deriveBusinessAuthoringBundleProbe(businessAuthoringBundleProbe(candidate))
	if err != nil {
		t.Fatal(err)
	}

	diskRoot := t.TempDir()
	procedurePaths := make([]string, 0, len(candidate.Procedures))
	for _, procedure := range candidate.Procedures {
		path := filepath.Join(diskRoot, "business", "procedures", procedure.Procedure.ID+".json")
		writeBusinessCandidateDiskSourceProbe(t, path, procedure)
		procedurePaths = append(procedurePaths, path)
	}
	handoffPath := filepath.Join(diskRoot, "business", "handoffs.json")
	writeBusinessCandidateDiskSourceProbe(t, handoffPath, candidate.Handoffs)

	// A legacy-looking file may still exist outside the candidate source set.
	// The candidate-only reload path below never reads it.
	legacyDecoy := filepath.Join(diskRoot, "reference", "business-system-design", "legacy-o1-overlay.json")
	if err := os.MkdirAll(filepath.Dir(legacyDecoy), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyDecoy, []byte(`{"legacy":"must-not-be-read-by-candidate-reload"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	reloaded := businessAuthoringBundleProbe{}
	for _, path := range procedurePaths {
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		procedure, decodeErr := decodeInlineO1FixtureProbe(file)
		closeErr := file.Close()
		if decodeErr != nil {
			t.Fatalf("candidate Procedure reload %s: %v", path, decodeErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		reloaded.Procedures = append(reloaded.Procedures, procedure)
	}

	handoffFile, err := os.Open(handoffPath)
	if err != nil {
		t.Fatal(err)
	}
	reloaded.Handoffs, err = decodeStandaloneHandoffSourceProbe(handoffFile)
	closeErr := handoffFile.Close()
	if err != nil {
		t.Fatalf("candidate Handoff reload: %v", err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}

	got, err := deriveBusinessAuthoringBundleProbe(reloaded)
	if err != nil {
		t.Fatalf("candidate-only disk sources could not normalize without legacy input: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("candidate-only disk reload changed normalized Business meaning: got=%+v want=%+v", got, want)
	}

	loaded, err := workspace.Load(filepath.Join(repoRoot, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateD4O1NormalizedBusinessModelProbe(loaded.World, got); err != nil {
		t.Fatalf("candidate-only disk reload does not conform to current World: %v", err)
	}
}

func TestBusinessCandidateOnlyDiskReloadFailsClosedOnUnknownCandidateField(t *testing.T) {
	repoRoot := repoPath()
	candidate, err := migrateCurrentReferenceAuthoringProbe(t, repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.Procedures) == 0 {
		t.Fatal("test setup: candidate Procedure is missing")
	}

	diskRoot := t.TempDir()
	encoded, err := json.Marshal(candidate.Procedures[0])
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatal(err)
	}
	object["legacyFallback"] = true
	mutated, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(diskRoot, "business", "procedures", "unknown-field.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, mutated, 0o644); err != nil {
		t.Fatal(err)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	_, decodeErr := decodeInlineO1FixtureProbe(file)
	closeErr := file.Close()
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	if decodeErr == nil {
		t.Fatal("candidate-only disk reload silently accepted an unknown legacyFallback field")
	}
}

func writeBusinessCandidateDiskSourceProbe(t *testing.T, path string, value any) {
	t.Helper()
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
}

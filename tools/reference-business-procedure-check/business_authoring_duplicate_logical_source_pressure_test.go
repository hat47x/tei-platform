package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestBusinessCandidateDiscoveryRejectsDuplicateLogicalProcedureAcrossDifferentPaths(t *testing.T) {
	repoRoot := repoPath()
	candidate, err := migrateCurrentReferenceAuthoringProbe(t, repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.Procedures) == 0 {
		t.Fatal("test setup: candidate Procedure is missing")
	}

	diskRoot := t.TempDir()
	first := filepath.Join(diskRoot, "business", "procedures", "order-entry.json")
	second := filepath.Join(diskRoot, "business", "procedures", "archive", "same-logical-procedure.json")
	writeBusinessCandidateDiskSourceProbe(t, first, candidate.Procedures[0])
	writeBusinessCandidateDiskSourceProbe(t, second, candidate.Procedures[0])

	paths := []string{first, second}
	if _, err := loadUniqueBusinessProcedureCandidatesProbe(paths); err == nil {
		t.Fatal("different physical paths with the same Business Procedure identity were silently accepted")
	}
}

func TestBusinessCandidateDiscoveryRejectsConflictingDuplicateLogicalProcedureWithoutPathPrecedence(t *testing.T) {
	repoRoot := repoPath()
	candidate, err := migrateCurrentReferenceAuthoringProbe(t, repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.Procedures) == 0 || len(candidate.Procedures[0].Steps) == 0 {
		t.Fatal("test setup: candidate Procedure or Step is missing")
	}

	diskRoot := t.TempDir()
	first := filepath.Join(diskRoot, "business", "procedures", "a-authority.json")
	second := filepath.Join(diskRoot, "business", "procedures", "z-authority.json")
	writeBusinessCandidateDiskSourceProbe(t, first, candidate.Procedures[0])

	// Keep the same logical Procedure identity but change accepted Business meaning.
	// A loader must reject the duplicate authority instead of resolving the conflict
	// by lexical path order, first-writer-wins, or last-writer-wins.
	candidate.Procedures[0].Steps[0].BusinessAction = "conflicting duplicate Business meaning"
	writeBusinessCandidateDiskSourceProbe(t, second, candidate.Procedures[0])

	for _, paths := range [][]string{{first, second}, {second, first}} {
		if _, err := loadUniqueBusinessProcedureCandidatesProbe(paths); err == nil {
			t.Fatalf("conflicting duplicate Business Procedure authority was silently resolved by input/path order: %v", paths)
		}
	}
}

func TestBusinessCandidateDiscoveryDoesNotUsePathAsProcedureIdentity(t *testing.T) {
	repoRoot := repoPath()
	candidate, err := migrateCurrentReferenceAuthoringProbe(t, repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.Procedures) == 0 {
		t.Fatal("test setup: candidate Procedure is missing")
	}

	diskRoot := t.TempDir()
	path := filepath.Join(diskRoot, "business", "procedures", "arbitrary-physical-name.json")
	writeBusinessCandidateDiskSourceProbe(t, path, candidate.Procedures[0])

	loaded, err := loadUniqueBusinessProcedureCandidatesProbe([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 {
		t.Fatalf("loaded Procedure count=%d want=1", len(loaded))
	}
	if loaded[0].Procedure.ID != candidate.Procedures[0].Procedure.ID {
		t.Fatalf("physical path changed Business Procedure identity: got=%q want=%q", loaded[0].Procedure.ID, candidate.Procedures[0].Procedure.ID)
	}
}

func loadUniqueBusinessProcedureCandidatesProbe(paths []string) ([]inlineO1FixtureProbe, error) {
	ordered := append([]string(nil), paths...)
	sort.Strings(ordered)
	seen := map[string]string{}
	loaded := make([]inlineO1FixtureProbe, 0, len(ordered))

	for _, path := range ordered {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		procedure, decodeErr := decodeInlineO1FixtureProbe(file)
		closeErr := file.Close()
		if decodeErr != nil {
			return nil, fmt.Errorf("decode Business Procedure candidate %s: %w", path, decodeErr)
		}
		if closeErr != nil {
			return nil, closeErr
		}
		logicalID := procedure.Procedure.ID
		if logicalID == "" {
			return nil, fmt.Errorf("Business Procedure candidate %s has empty logical identity", path)
		}
		if previous, duplicate := seen[logicalID]; duplicate {
			return nil, fmt.Errorf("duplicate Business Procedure logical identity %q at %s and %s", logicalID, previous, path)
		}
		seen[logicalID] = path
		loaded = append(loaded, procedure)
	}
	return loaded, nil
}

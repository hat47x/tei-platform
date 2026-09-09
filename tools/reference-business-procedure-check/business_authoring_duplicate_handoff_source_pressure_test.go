package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestBusinessCandidateDiscoveryRejectsDuplicateStandaloneHandoffAcrossDifferentPaths(t *testing.T) {
	repoRoot := repoPath()
	candidate, err := migrateCurrentReferenceAuthoringProbe(t, repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.Handoffs.Handoffs) == 0 {
		t.Fatal("test setup: candidate Handoff is missing")
	}

	diskRoot := t.TempDir()
	first := filepath.Join(diskRoot, "business", "handoffs", "a-authority.json")
	second := filepath.Join(diskRoot, "business", "handoffs", "archive", "same-handoff.json")
	writeBusinessCandidateDiskSourceProbe(t, first, candidate.Handoffs)
	writeBusinessCandidateDiskSourceProbe(t, second, candidate.Handoffs)

	for _, paths := range [][]string{{first, second}, {second, first}} {
		if _, err := loadUniqueStandaloneHandoffSourcesProbe(paths); err == nil {
			t.Fatalf("duplicate standalone Handoff authority was silently resolved by input/path order: %v", paths)
		}
	}
}

func TestBusinessCandidateDiscoveryDoesNotInventEndpointOnlyHandoffIdentity(t *testing.T) {
	repoRoot := repoPath()
	candidate, err := migrateCurrentReferenceAuthoringProbe(t, repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.Handoffs.Handoffs) == 0 {
		t.Fatal("test setup: candidate Handoff is missing")
	}

	firstSource := candidate.Handoffs
	firstSource.Handoffs = append([]referenceSetHandoff(nil), candidate.Handoffs.Handoffs[:1]...)
	secondSource := candidate.Handoffs
	secondSource.Handoffs = append([]referenceSetHandoff(nil), candidate.Handoffs.Handoffs[:1]...)
	secondSource.Handoffs[0].Condition = "follow-up remains required because unresolved external work continues"

	diskRoot := t.TempDir()
	first := filepath.Join(diskRoot, "business", "handoffs", "a-authority.json")
	second := filepath.Join(diskRoot, "business", "handoffs", "z-authority.json")
	writeBusinessCandidateDiskSourceProbe(t, first, firstSource)
	writeBusinessCandidateDiskSourceProbe(t, second, secondSource)

	wantConditions := map[string]struct{}{
		firstSource.Handoffs[0].Condition:  {},
		secondSource.Handoffs[0].Condition: {},
	}
	if len(wantConditions) != 2 {
		t.Fatal("test setup: Handoff conditions must be distinct")
	}

	for _, paths := range [][]string{{first, second}, {second, first}} {
		loaded, err := loadUniqueStandaloneHandoffSourcesProbe(paths)
		if err != nil {
			t.Fatalf("different H1/R3 meaning sharing endpoints was rejected by invented endpoint-only or path/input-order authority: paths=%v err=%v", paths, err)
		}
		if len(loaded.Handoffs) != 2 {
			t.Fatalf("loaded Handoff count=%d want=2 paths=%v", len(loaded.Handoffs), paths)
		}

		gotConditions := map[string]int{}
		for _, item := range loaded.Handoffs {
			gotConditions[item.Condition]++
		}
		if len(gotConditions) != 2 {
			t.Fatalf("same endpoint pairのcondition meaningをpath/input orderで潰しました: paths=%v conditions=%v", paths, gotConditions)
		}
		for condition := range wantConditions {
			if gotConditions[condition] != 1 {
				t.Fatalf("expected Handoff condition meaningをexactly once保持できません: paths=%v condition=%q got=%v", paths, condition, gotConditions)
			}
		}
	}
}

func loadUniqueStandaloneHandoffSourcesProbe(paths []string) (standaloneHandoffSourceProbe, error) {
	ordered := append([]string(nil), paths...)
	sort.Strings(ordered)
	seen := map[string]string{}
	merged := standaloneHandoffSourceProbe{
		SchemaVersion: standaloneHandoffSchemaVersion,
		ArtifactRole:  standaloneHandoffArtifactRole,
		Canonical:     false,
	}

	for _, path := range ordered {
		file, err := os.Open(path)
		if err != nil {
			return standaloneHandoffSourceProbe{}, err
		}
		source, decodeErr := decodeStandaloneHandoffSourceProbe(file)
		closeErr := file.Close()
		if decodeErr != nil {
			return standaloneHandoffSourceProbe{}, fmt.Errorf("decode standalone Handoff candidate %s: %w", path, decodeErr)
		}
		if closeErr != nil {
			return standaloneHandoffSourceProbe{}, closeErr
		}

		for _, item := range source.Handoffs {
			key := normalizedHandoffKey(canonicalBusinessHandoffProbe{
				From: canonicalBusinessHandoffEndpointProbe{
					Procedure: item.From.Procedure,
					Step:      item.From.Step,
				},
				To: canonicalBusinessHandoffEndpointProbe{
					Procedure: item.To.Procedure,
					Step:      item.To.Step,
				},
				Condition: item.Condition,
			})
			if previous, duplicate := seen[key]; duplicate {
				return standaloneHandoffSourceProbe{}, fmt.Errorf("duplicate standalone Business Handoff meaning %q at %s and %s", key, previous, path)
			}
			seen[key] = path
			merged.Handoffs = append(merged.Handoffs, item)
		}
	}

	return merged, nil
}

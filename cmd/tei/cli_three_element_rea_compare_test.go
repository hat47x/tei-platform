package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/reacandidate"
)

// A typed CandidateSet is a read-only projection, not an acceptance record.
// These fixtures exercise the actual CLI dispatcher and shared Go comparator.
func comparisonCLICandidateSet(attempt string, ids ...string) reacandidate.CandidateSet {
	set := reacandidate.CandidateSet{
		SchemaVersion:   "tei.rea.candidate-set/v0",
		Effect:          reacandidate.EffectReadOnly,
		Qualification:   "complete",
		AuthorityLimits: reacandidate.CandidateAuthorityLimits(),
	}
	set.Attempt.EvidenceID = attempt
	set.Attempt.Provider.ID = "rea-test-provider"
	set.Attempt.Provider.Version = "6.8.0"
	set.Attempt.Operation = "analyze_javascript_application"
	set.Attempt.SubjectDigest = "sha256:synthetic-subject"
	for _, id := range ids {
		set.Candidates = append(set.Candidates, reacandidate.StructureCandidate{
			CandidateID: id,
			Kind:        reacandidate.CandidateKindCrossReference,
			Facts: reacandidate.StructuralFacts{
				ExternalIdentity:     id,
				ExternalOccurrenceID: id,
				ObservedKind:         "imports",
				Direction: &reacandidate.Direction{
					FromExternalID: "node-order",
					ToExternalID:   "node-line",
				},
			},
			SourceRevision: reacandidate.SourceRevision{
				ArtifactPath: "src/order.js",
				Fingerprint:  "sha256:synthetic-source-revision",
				Binding:      reacandidate.SourceBindingProviderReported,
			},
			ReviewStatus: reacandidate.ReviewStatusRequiresOwner,
			Resolution:   reacandidate.ResolutionUnresolved,
		})
	}
	set.Fingerprint = reacandidate.SetFingerprint(set)
	return set
}

func writeComparisonCLICandidateSet(t *testing.T, path string, set reacandidate.CandidateSet) {
	t.Helper()
	raw, err := json.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func runComparisonCLI(t *testing.T, before, after string) (int, reaIntakeResponse) {
	t.Helper()
	var out strings.Builder
	code, err := runThreeElementCommand([]string{"rea-compare", "-before", before, "-after", after}, &out)
	if err != nil {
		t.Fatal(err)
	}
	return code, decodeReaIntake(t, out.String())
}

func TestThreeElementREACompareKeepsParallelRelationsUnpaired(t *testing.T) {
	dir := t.TempDir()
	beforePath := filepath.Join(dir, "before.json")
	afterPath := filepath.Join(dir, "after.json")
	writeComparisonCLICandidateSet(t, beforePath, comparisonCLICandidateSet("attempt-old", "old-1", "old-2"))
	writeComparisonCLICandidateSet(t, afterPath, comparisonCLICandidateSet("attempt-new", "new-2", "new-1"))
	exit, response := runComparisonCLI(t, beforePath, afterPath)
	if exit != 0 || !response.OK || response.SchemaVersion != "tei.cli.three-element.rea-compare/v0" {
		t.Fatalf("CLI compare envelope: exit=%d response=%+v", exit, response)
	}
	result, ok := response.Result.(map[string]any)
	if !ok {
		t.Fatalf("CLI did not return a review result: %T", response.Result)
	}
	packet, ok := result["reviewPacket"].(map[string]any)
	if !ok || packet["acceptedMeaning"] != false || packet["ownerDecision"] != "pending" ||
		packet["effect"] != "read-only" || packet["comparedWithBefore"] != true {
		t.Fatalf("comparison falsely accepted meaning: %+v", packet)
	}
	comparison, ok := packet["comparison"].(map[string]any)
	if !ok {
		t.Fatalf("comparison is missing from packet: %+v", packet)
	}
	entries, ok := comparison["entries"].([]any)
	if !ok || len(entries) != 4 {
		t.Fatalf("old/new relations not preserved: %+v", comparison)
	}
	oldCount, newCount := 0, 0
	for _, raw := range entries {
		entry, ok := raw.(map[string]any)
		if !ok || entry["status"] != reacandidate.ComparisonAmbiguousStructural {
			t.Fatalf("invented a positional relation pairing: %+v", raw)
		}
		_, oldSource := entry["beforeSource"]
		_, newSource := entry["afterSource"]
		_, oldAttempt := entry["beforeAttemptId"]
		_, newAttempt := entry["afterAttemptId"]
		switch {
		case oldSource && !newSource && oldAttempt && !newAttempt:
			oldCount++
		case newSource && !oldSource && newAttempt && !oldAttempt:
			newCount++
		default:
			t.Fatalf("comparison JSON invented a counterpart: %+v", entry)
		}
	}
	if oldCount != 2 || newCount != 2 {
		t.Fatalf("before/after candidates = %d/%d, want 2/2", oldCount, newCount)
	}
	summary, ok := result["summary"].([]any)
	if !ok {
		t.Fatalf("summary missing: %+v", result)
	}
	var lines []string
	for _, line := range summary {
		value, ok := line.(string)
		if !ok {
			t.Fatalf("unexpected summary value: %T", line)
		}
		lines = append(lines, value)
	}
	if !strings.Contains(strings.Join(lines, "\n"), "comparison: ambiguous-structural-match=4") {
		t.Fatalf("summary hid ambiguous relations: %v", lines)
	}
}

// Unique structural-key matches are only review hints. A reused ID with
// changed observed facts must never be mislabeled as unchanged.
func TestThreeElementREACompareUniqueAndChangedFacts(t *testing.T) {
	dir := t.TempDir()
	beforePath := filepath.Join(dir, "before.json")
	afterPath := filepath.Join(dir, "after.json")
	before := comparisonCLICandidateSet("attempt-old", "old")
	writeComparisonCLICandidateSet(t, beforePath, before)
	for _, tc := range []struct {
		name   string
		after  reacandidate.CandidateSet
		status string
	}{
		{
			name:   "unique-renumbering",
			after:  comparisonCLICandidateSet("attempt-new", "new"),
			status: reacandidate.ComparisonRenumbered,
		},
		{
			name: "changed-observed-facts",
			after: func() reacandidate.CandidateSet {
				set := comparisonCLICandidateSet("attempt-new", "old")
				set.Candidates[0].Facts.ObservedKind = "calls"
				set.Fingerprint = reacandidate.SetFingerprint(set)
				return set
			}(),
			status: reacandidate.ComparisonObservedFactsChanged,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writeComparisonCLICandidateSet(t, afterPath, tc.after)
			exit, response := runComparisonCLI(t, beforePath, afterPath)
			if exit != 0 || !response.OK {
				t.Fatalf("comparison was refused: exit=%d result=%+v", exit, response)
			}
			packet := response.Result.(map[string]any)["reviewPacket"].(map[string]any)
			entries := packet["comparison"].(map[string]any)["entries"].([]any)
			if len(entries) != 1 || entries[0].(map[string]any)["status"] != tc.status {
				t.Fatalf("%s has unexpected comparison: %+v", tc.name, entries)
			}
			if packet["acceptedMeaning"] != false {
				t.Fatal("review comparison acquired owner authority")
			}
		})
	}
}

func TestThreeElementREACompareRejectsUnqualifiedInputs(t *testing.T) {
	dir := t.TempDir()
	beforePath := filepath.Join(dir, "before.json")
	afterPath := filepath.Join(dir, "after.json")
	valid := comparisonCLICandidateSet("attempt-old", "edge-1")
	writeComparisonCLICandidateSet(t, beforePath, valid)
	cases := []struct {
		name  string
		value reacandidate.CandidateSet
	}{
		{name: "unrecognized-schema", value: func() reacandidate.CandidateSet {
			v := valid
			v.SchemaVersion = "tei.rea.candidate-set/v999"
			v.Fingerprint = reacandidate.SetFingerprint(v)
			return v
		}()},
		{name: "write-effect", value: func() reacandidate.CandidateSet {
			v := valid
			v.Effect = "write"
			v.Fingerprint = reacandidate.SetFingerprint(v)
			return v
		}()},
		{name: "tampered-fingerprint", value: func() reacandidate.CandidateSet {
			v := valid
			v.Fingerprint = "sha256:wrong"
			return v
		}()},
		{name: "duplicate-id", value: func() reacandidate.CandidateSet {
			v := valid
			v.Candidates = append(append([]reacandidate.StructureCandidate(nil), v.Candidates...), v.Candidates[0])
			v.Fingerprint = reacandidate.SetFingerprint(v)
			return v
		}()},
		{name: "accepted-status", value: func() reacandidate.CandidateSet {
			v := valid
			v.Candidates = append([]reacandidate.StructureCandidate(nil), v.Candidates...)
			v.Candidates[0].ReviewStatus = "accepted"
			v.Fingerprint = reacandidate.SetFingerprint(v)
			return v
		}()},
		{name: "removed-authority", value: func() reacandidate.CandidateSet {
			v := valid
			v.AuthorityLimits = nil
			return v
		}()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writeComparisonCLICandidateSet(t, afterPath, tc.value)
			exit, response := runComparisonCLI(t, beforePath, afterPath)
			if exit != 2 || response.OK || response.Error == nil ||
				response.Error.Code != "AFTER_CANDIDATE_SET_INVALID" || response.Result != nil {
				t.Fatalf("invalid input incorrectly accepted: exit=%d response=%+v", exit, response)
			}
		})
	}
	t.Run("legacy-review-packet", func(t *testing.T) {
		if err := os.WriteFile(afterPath, []byte(`{"schemaVersion":"tei.rea.candidate-review/v0","effect":"read-only"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		exit, response := runComparisonCLI(t, beforePath, afterPath)
		if exit != 2 || response.Error == nil || response.Error.Code != "AFTER_CANDIDATE_SET_INVALID" {
			t.Fatalf("legacy ReviewPacket accepted as typed set: %+v", response)
		}
	})
	t.Run("malformed", func(t *testing.T) {
		if err := os.WriteFile(afterPath, []byte("{not json"), 0o600); err != nil {
			t.Fatal(err)
		}
		exit, response := runComparisonCLI(t, beforePath, afterPath)
		if exit != 2 || response.Error == nil || response.Error.Code != "AFTER_CANDIDATE_SET_INVALID" {
			t.Fatalf("malformed input accepted: %+v", response)
		}
	})
	t.Run("null", func(t *testing.T) {
		if err := os.WriteFile(afterPath, []byte("null"), 0o600); err != nil {
			t.Fatal(err)
		}
		exit, response := runComparisonCLI(t, beforePath, afterPath)
		if exit != 2 || response.Error == nil || response.Error.Code != "AFTER_CANDIDATE_SET_INVALID" {
			t.Fatalf("null input accepted: %+v", response)
		}
	})
	t.Run("missing-before", func(t *testing.T) {
		exit, response := runComparisonCLI(t, filepath.Join(dir, "not-present.json"), beforePath)
		if exit != 2 || response.Error == nil || response.Error.Code != "BEFORE_CANDIDATE_SET_INVALID" {
			t.Fatalf("missing previous set accepted: %+v", response)
		}
	})
	t.Run("missing-flags", func(t *testing.T) {
		var out strings.Builder
		exit, err := runThreeElementCommand([]string{"rea-compare", "-after", afterPath}, &out)
		if err != nil || exit != 2 {
			t.Fatalf("missing flag: exit=%d err=%v out=%s", exit, err, out.String())
		}
		response := decodeReaIntake(t, out.String())
		if response.Error == nil || response.Error.Code != "INVALID_ARGUMENT" {
			t.Fatalf("missing flag was accepted: %+v", response)
		}
	})
}

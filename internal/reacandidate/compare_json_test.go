package reacandidate

import (
	"encoding/json"
	"testing"
)

// A source on the absent side of an ambiguous structural-key candidate must
// NOT become an empty object on the wire. The counterpart was not observed.
func TestComparisonJSONAmbiguousRelationsOmitUnprovenCounterparts(t *testing.T) {
	beforeFixture := defaultEvidence()
	parallel := beforeFixture.edges[0]
	parallel.id = "json-older-second-relation"
	beforeFixture.edges = append(beforeFixture.edges, parallel)
	before := mustGenerate(t, beforeFixture)

	afterFixture := defaultEvidence()
	afterFixture.evidenceID = "ev-json-parallel-after"
	first := afterFixture.edges[0]
	first.id = "json-new-first-relation"
	second := first
	second.id = "json-new-second-relation"
	afterFixture.edges = []edgeFixture{second, first}
	after := mustGenerate(t, afterFixture)

	comparison := CompareCandidateSets(before, after)
	packet := BuildReviewPacket(after, nil, nil, &comparison)
	raw, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		AcceptedMeaning bool `json:"acceptedMeaning"`
		OwnerDecision string `json:"ownerDecision"`
		Comparison struct {
			Entries []map[string]json.RawMessage `json:"entries"`
		} `json:"comparison"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if wire.AcceptedMeaning || wire.OwnerDecision != "pending" {
		t.Fatalf("JSON ReviewPacket acquired owner acceptance: %+v", wire)
	}
	oldCount, newCount := 0, 0
	for _, entry := range wire.Comparison.Entries {
		var status string
		if err := json.Unmarshal(entry["status"], &status); err != nil {
			t.Fatalf("entry missing status: %v", err)
		}
		if status != ComparisonAmbiguousStructural {
			continue
		}
		_, oldSource := entry["beforeSource"]
		_, newSource := entry["afterSource"]
		_, oldAttempt := entry["beforeAttemptId"]
		_, newAttempt := entry["afterAttemptId"]
		switch {
		case oldSource && oldAttempt && !newSource && !newAttempt:
			oldCount++
		case newSource && newAttempt && !oldSource && !oldAttempt:
			newCount++
		default:
			t.Fatalf("JSON fabricated or lost a counterpart: %s", mustMarshalEntry(t, entry))
		}
	}
	if oldCount != 2 || newCount != 2 {
		t.Fatalf("ambiguous JSON entries old=%d new=%d, want 2+2", oldCount, newCount)
	}
}

// A genuine unique renumbering has both sources; a structurally unmatched
// addition/removal keeps only the source that was observed.
func TestComparisonJSONPreservesPresentSidesOnly(t *testing.T) {
	baseline := mustGenerate(t, defaultEvidence())
	t.Run("unique-renumbering", func(t *testing.T) {
		fixture := defaultEvidence()
		fixture.evidenceID = "ev-json-unique"
		fixture.edges[0].id = "json-unique-renumber"
		comparison := CompareCandidateSets(baseline, mustGenerate(t, fixture))
		matched := 0
		for _, entry := range comparison.Entries {
			if entry.Status != ComparisonRenumbered {
				continue
			}
			matched++
			wire := marshalEntry(t, entry)
			if _, ok := wire["beforeSource"]; !ok {
				t.Fatalf("renumbering lost the before source: %v", wire)
			}
			if _, ok := wire["afterSource"]; !ok {
				t.Fatalf("renumbering lost the after source: %v", wire)
			}
		}
		if matched != 1 {
			t.Fatalf("renumbered entries = %d, want 1", matched)
		}
	})
	t.Run("added-and-removed", func(t *testing.T) {
		fixture := defaultEvidence()
		fixture.evidenceID = "ev-json-no-relations"
		fixture.edges = nil
		withoutRelation := mustGenerate(t, fixture)
		for _, tc := range []struct{
			status string
			comparison CandidateSetComparison
			present string
			absent string
		}{
			{ComparisonRemoved, CompareCandidateSets(baseline, withoutRelation), "beforeSource", "afterSource"},
			{ComparisonAdded, CompareCandidateSets(withoutRelation, baseline), "afterSource", "beforeSource"},
		} {
			matched := 0
			for _, entry := range tc.comparison.Entries {
				if entry.Status != tc.status {
					continue
				}
				matched++
				wire := marshalEntry(t, entry)
				if _, ok := wire[tc.present]; !ok {
					t.Fatalf("%s did not serialize %s", tc.status, tc.present)
				}
				if _, ok := wire[tc.absent]; ok {
					t.Fatalf("%s serialized absent %s: %s", tc.status, tc.absent, mustMarshalEntry(t, wire))
				}
			}
			if matched != 1 {
				t.Fatalf("%s count=%d, want 1", tc.status, matched)
			}
		}
	})
}

func marshalEntry(t *testing.T, entry CandidateComparisonEntry) map[string]json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]json.RawMessage
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func mustMarshalEntry(t *testing.T, entry map[string]json.RawMessage) string {
	t.Helper()
	raw, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

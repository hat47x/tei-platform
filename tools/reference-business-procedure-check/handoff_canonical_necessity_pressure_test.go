package main

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestProcedureBodiesAloneCannotObserveAcceptedHandoffConditionChange(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}

	beforeBodies, err := canonicalProcedureBodiesForHandoffPressure(fixtures)
	if err != nil {
		t.Fatal(err)
	}
	changed := set
	changed.Handoffs = append([]referenceSetHandoff(nil), set.Handoffs...)
	changed.Handoffs[0].Condition = "follow-up is required because unresolved delivery work remains"
	afterBodies, err := canonicalProcedureBodiesForHandoffPressure(fixtures)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(beforeBodies, afterBodies) {
		t.Fatal("handoff-only changeでProcedure body meaningまで変化しました")
	}
}

func TestNormalizedHandoffMeaningObservesAcceptedConditionChange(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}

	before, err := deriveCanonicalBusinessHandoffProbes(set, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	changed := set
	changed.Handoffs = append([]referenceSetHandoff(nil), set.Handoffs...)
	changed.Handoffs[0].Condition = "follow-up is required because unresolved delivery work remains"
	after, err := deriveCanonicalBusinessHandoffProbes(changed, fixtures)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(before, after) {
		t.Fatal("accepted handoff condition changeをnormalized Business meaningから失いました")
	}
}

func TestHandoffEndpointChangeIsReferentialChangeWithoutProcedureBodyChange(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}

	beforeBodies, err := canonicalProcedureBodiesForHandoffPressure(fixtures)
	if err != nil {
		t.Fatal(err)
	}
	beforeHandoffs, err := deriveCanonicalBusinessHandoffProbes(set, fixtures)
	if err != nil {
		t.Fatal(err)
	}

	changed := set
	changed.Handoffs = append([]referenceSetHandoff(nil), set.Handoffs...)
	changed.Handoffs[0].From.Step = "BP-08"
	afterHandoffs, err := deriveCanonicalBusinessHandoffProbes(changed, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	afterBodies, err := canonicalProcedureBodiesForHandoffPressure(fixtures)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(beforeBodies, afterBodies) {
		t.Fatal("handoff endpoint changeでProcedure body meaningを捏造変更しました")
	}
	if reflect.DeepEqual(beforeHandoffs, afterHandoffs) {
		t.Fatal("handoff endpoint changeをreferential Business meaning差として観測できません")
	}
}

func canonicalProcedureBodiesForHandoffPressure(fixtures map[string]fixture) ([]canonicalBusinessProcedureMeaningWithoutRealizationProbe, error) {
	ids := make([]string, 0, len(fixtures))
	for id := range fixtures {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	result := make([]canonicalBusinessProcedureMeaningWithoutRealizationProbe, 0, len(ids))
	for _, id := range ids {
		meaning, err := deriveCanonicalBusinessProcedureMeaningWithoutRealizationProbe(fixtures[id])
		if err != nil {
			return nil, err
		}
		result = append(result, meaning)
	}
	return result, nil
}

package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestThinCanonicalBusinessMeaningPreservesTriggerAndActorCandidateStatus(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	meaning, err := deriveCanonicalBusinessProcedureMeaningProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	if meaning.Trigger.Status != "candidate" || meaning.Trigger.Text != "an order-handling need enters the sales work context" {
		t.Fatalf("trigger candidate meaningを保持できません: %+v", meaning.Trigger)
	}
	if meaning.Actor.Status != "candidate" || meaning.Actor.Label != "sales operator" {
		t.Fatalf("actor candidate meaningを保持できません: %+v", meaning.Actor)
	}
}

func TestTriggerEpistemicStatusChangeIsBusinessMeaningChangeNotIdentityChange(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := deriveCanonicalBusinessProcedureMeaningProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	changed.Procedure.Trigger.Status = "unresolved"
	after, err := deriveCanonicalBusinessProcedureMeaningProbe(changed)
	if err != nil {
		t.Fatal(err)
	}

	if before.ID != after.ID {
		t.Fatal("trigger epistemic status changeでProcedure identityまで変化しました")
	}
	if reflect.DeepEqual(before, after) {
		t.Fatal("trigger candidate -> unresolvedをBusiness semantic changeとして観測できません")
	}
	if after.Trigger.Status != "unresolved" {
		t.Fatalf("unresolved trigger statusを別状態へ丸めました: %+v", after.Trigger)
	}
}

func TestActorEpistemicStatusAndLabelChangeDoNotCreateAuthorizationMeaning(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := deriveCanonicalBusinessProcedureMeaningProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	changed.Procedure.Actor.Status = "unresolved"
	changed.Procedure.Actor.Label = "sales role to be confirmed"
	after, err := deriveCanonicalBusinessProcedureMeaningProbe(changed)
	if err != nil {
		t.Fatal(err)
	}

	if before.ID != after.ID {
		t.Fatal("actor candidate changeでProcedure identityまで変化しました")
	}
	if reflect.DeepEqual(before, after) {
		t.Fatal("actor status / label changeをBusiness semantic changeとして観測できません")
	}
	if after.Actor.Status != "unresolved" || after.Actor.Label != "sales role to be confirmed" {
		t.Fatalf("actor unresolved meaningを保持できません: %+v", after.Actor)
	}

	beforePatterns := allPatternRefs(before)
	afterPatterns := allPatternRefs(after)
	if !reflect.DeepEqual(beforePatterns, afterPatterns) {
		t.Fatalf("actor候補変更からApplication Function meaningを暗黙変更しました: before=%v after=%v", beforePatterns, afterPatterns)
	}
}

func TestCandidateRefShapeContainsNoAuthorizationFields(t *testing.T) {
	candidateType := reflect.TypeOf(candidateRef{})
	want := []string{"Status", "Text", "Label"}
	if got := fieldNames(candidateType); !reflect.DeepEqual(got, want) {
		t.Fatalf("candidateRefのfield境界が変化しました: got=%v want=%v", got, want)
	}
	for i := 0; i < candidateType.NumField(); i++ {
		name := strings.ToLower(candidateType.Field(i).Name)
		for _, forbidden := range []string{"role", "principal", "permission", "authority", "credential"} {
			if strings.Contains(name, forbidden) {
				t.Fatalf("Business actor / trigger candidateへAuthorization field %qが侵入しました", candidateType.Field(i).Name)
			}
		}
	}
}

func TestThinCanonicalMeaningDoesNotInventResolvedStatus(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	f.Procedure.Trigger.Status = "unresolved"
	f.Procedure.Actor.Status = "unresolved"
	meaning, err := deriveCanonicalBusinessProcedureMeaningProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	if meaning.Trigger.Status != "unresolved" || meaning.Actor.Status != "unresolved" {
		t.Fatalf("unresolvedを既知状態へ丸めました: trigger=%+v actor=%+v", meaning.Trigger, meaning.Actor)
	}
}

func TestEpistemicStatusVocabularyIsNotFrozenByCurrentReference(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}

	// This Reference proves preservation, not a universal enum. A future accepted
	// authoring contract may refine the vocabulary. The thin normalization must
	// not silently map an explicit source status to today's only observed value.
	f.Procedure.Trigger.Status = "needs-domain-review"
	meaning, err := deriveCanonicalBusinessProcedureMeaningProbe(f)
	if err != nil {
		t.Fatal(err)
	}
	if meaning.Trigger.Status != "needs-domain-review" {
		t.Fatalf("explicit epistemic statusをcurrent Reference vocabularyへ丸めました: %+v", meaning.Trigger)
	}
}

func allPatternRefs(meaning canonicalBusinessProcedureMeaningProbe) []string {
	var result []string
	for _, item := range meaning.Steps {
		result = append(result, item.ApplicationPatterns...)
	}
	return result
}

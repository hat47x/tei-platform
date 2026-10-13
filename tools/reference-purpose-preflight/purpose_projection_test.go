package main

import (
	"reflect"
	"testing"
)

func purposeRequirementByName(p PurposeProjection, name string) (PurposeObligation, bool) {
	for _, value := range p.Obligations {
		if value.Requirement == name {
			return value, true
		}
	}
	return PurposeObligation{}, false
}

func TestPurposeProjectionSelectsOnlyRelevantObligations(t *testing.T) {
	assessment := WorkspaceAssessment{
		SubjectKind:  "semantic-validation", SubjectID: "sales.order_line.line_number_positive",
		SourceStatus: "traceable-source-declaration", RuleMeaning: "canonical-validation-declared",
		Witnesses:    []SourceWitness{{Path: "semantic/sales/validation.tei", Fingerprint: "sha256:source", Role: "definition"}},
	}
	var lengths []int
	for _, purpose := range []string{"describe", "simulate", "execute-preflight"} {
		got, err := ProjectForPurpose(assessment, purpose)
		if err != nil {
			t.Fatal(err)
		}
		lengths = append(lengths, len(got.Obligations))
		if got.ExecutionAllowed {
			t.Fatalf("purpose %s granted execution", purpose)
		}
		source, ok := purposeRequirementByName(got, "typed-source")
		if !ok || source.Status != "observed" {
			t.Fatalf("qualified source not observed: %+v", got)
		}
		meaning, ok := purposeRequirementByName(got, "business-meaning")
		if !ok || meaning.Status != "declared" {
			t.Fatalf("actual canonical declaration was not represented: %+v", got)
		}
		_, hasEffect := purposeRequirementByName(got, "effect-fence")
		if hasEffect != (purpose == "execute-preflight") {
			t.Fatalf("effect fence required for wrong purpose: %+v", got)
		}
		_, hasFixture := purposeRequirementByName(got, "evaluation-fixture")
		if hasFixture != (purpose != "describe") {
			t.Fatalf("evaluation fixture required for wrong purpose: %+v", got)
		}
	}
	if !reflect.DeepEqual(lengths, []int{2, 3, 10}) {
		t.Fatalf("unexpected purpose-specific obligation counts: %v", lengths)
	}
}

func TestMatchedDecisionCannotDischargeHumanOrRuntimeObligations(t *testing.T) {
	assessment := WorkspaceAssessment{
		SubjectKind:           "semantic-validation", SubjectID: "sales.order_line.line_number_positive",
		SourceStatus:          "traceable-source-declaration", RuleMeaning: "canonical-validation-declared",
		Witnesses:             []SourceWitness{{Path: "semantic/sales/validation.tei", Fingerprint: "sha256:source", Role: "definition"}},
		DecisionHistoryStatus: "reported-source-matched",
	}
	p, err := ProjectForPurpose(assessment, "execute-preflight")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"human-acceptance", "runtime-business-validation", "host-authorization", "effect-fence", "business-effective-scope"} {
		ob, exists := purposeRequirementByName(p, name)
		if !exists {
			t.Fatalf("missing obligation %q", name)
		}
		if ob.Status == "satisfied" || ob.Status == "observed" || ob.Status == "declared" {
			t.Fatalf("history upgraded an unverified condition: %+v", ob)
		}
	}
	if got, ok := purposeRequirementByName(p, "human-acceptance"); !ok || got.Status != "unverified" {
		t.Fatalf("history wrongly established human approval: %+v", p)
	}
	// The typed subject is a Validation, not an executable Operation with
	// an authoritatively classified effect. Do not invent extra runtime gates.
	for _, name := range []string{"business-effective-scope", "provider-contract", "effect-fence"} {
		if got, ok := purposeRequirementByName(p, name); !ok || got.Status != "conditional" {
			t.Fatalf("%s should remain dependent on the actual Operation contract: %+v", name, p)
		}
	}
}

func TestUnresolvedSubjectNotPromotedForAnyPurpose(t *testing.T) {
	assessment := WorkspaceAssessment{SubjectKind: "application-pattern", SubjectID: "order_entry.submit_order"}
	for _, purpose := range []string{"describe", "simulate", "execute-preflight"} {
		result, err := ProjectForPurpose(assessment, purpose)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"typed-source", "business-meaning"} {
			ob, ok := purposeRequirementByName(result, name)
			if !ok || ob.Status != "unresolved" {
				t.Fatalf("%s %s unexpectedly resolved: %+v", purpose, name, result)
			}
		}
		if result.ExecutionAllowed {
			t.Fatal("unresolved subject granted execution")
		}
	}
}

func TestPurposeProjectionRejectsUnknownInput(t *testing.T) {
	report := WorkspaceAssessment{SubjectKind: "semantic-validation", SubjectID: "sales.order_line.line_number_positive"}
	for _, purpose := range []string{"", "deploy", "execute", "simulate-and-execute"} {
		if _, err := ProjectForPurpose(report, purpose); err == nil {
			t.Fatalf("unknown purpose accepted: %q", purpose)
		}
	}
	if _, err := ProjectForPurpose(WorkspaceAssessment{SubjectKind: "semantic-validation"}, "describe"); err == nil {
		t.Fatal("missing typed subject ID was accepted")
	}
}

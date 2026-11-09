package evaluationcontext

import (
	"reflect"
	"testing"
)

func elapsed(version string) Input {
	return Input{
		Kind: KindElapsedDays, RuleVersion: version,
		StartDate: "2026-10-12", EvaluationDate: "2026-10-14", EvaluationDateProvided: true,
	}
}

func hasUnknown(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// The rule version is part of the explanation, and the same fixed input and
// context reproduce the same result.
func TestRuleVersionIsCarriedAndReevaluationIsStable(t *testing.T) {
	in := elapsed("v2")
	first := Evaluate(in)
	second := Evaluate(in)
	if !first.Accepted || first.Code != CodeDays || first.Value != "2" || first.RuleVersion != "v2" {
		t.Fatalf("first=%+v", first)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("re-evaluation changed: %+v != %+v", first, second)
	}

	changed := elapsed("v3")
	third := Evaluate(changed)
	if third.Value != first.Value {
		t.Fatalf("value changed with only the rule version: %q != %q", third.Value, first.Value)
	}
	if third.RuleVersion == first.RuleVersion {
		t.Fatalf("rule version was not carried: %+v", third)
	}
}

func TestProviderImpliedClockIsUnproven(t *testing.T) {
	in := elapsed("v2")
	in.ProviderImpliedClock = true
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeContextUnproven {
		t.Fatalf("report=%+v want CONTEXT_UNPROVEN", report)
	}
	if !hasUnknown(report.Unknown, "providerImpliedClock") {
		t.Fatalf("unknown=%v", report.Unknown)
	}
	if report.RuleVersion != "v2" {
		t.Fatalf("rule version=%q", report.RuleVersion)
	}
}

func TestProviderImpliedLocaleIsUnproven(t *testing.T) {
	in := elapsed("v2")
	in.ProviderImpliedLocale = true
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeContextUnproven {
		t.Fatalf("report=%+v want CONTEXT_UNPROVEN", report)
	}
	if !hasUnknown(report.Unknown, "providerImpliedLocale") {
		t.Fatalf("unknown=%v", report.Unknown)
	}
}

func TestDeclaredContextHasNoUnknown(t *testing.T) {
	report := Evaluate(elapsed("v2"))
	if len(report.Unknown) != 0 {
		t.Fatalf("unknown=%v want empty", report.Unknown)
	}
}

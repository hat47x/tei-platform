package main

import (
	"testing"

	"tei/internal/buildintent"
	"tei/internal/plugin"
)

// These tests pin the parameter comparison (SB-C0 §4.17's remaining field).
//
// Parameters are compared by name because both sides sort by name and neither
// changes a value: buildintent.NormalizeIntent sorts the declaration's and
// plugin.NormalizeBuildPlan sorts the plan's. This file checks that claim as
// well as the comparison itself, so the comparison cannot silently depend on an
// ordering property that stops holding.

// Both normalisers sort parameters by name and leave values untouched.
func TestNormalisersOnlyReorderParameters(t *testing.T) {
	declared := []plugin.BuildParameter{
		{Name: "optimization", Value: "size"},
		{Name: "debug", Value: "true"},
	}
	intent := declarationFor(t, func(i *buildintent.BuildIntent) {
		i.Parameters = append([]plugin.BuildParameter(nil), declared...)
	})
	normalizedIntent := buildintent.NormalizeIntent(intent)
	if len(normalizedIntent.Parameters) != len(declared) {
		t.Fatalf("normalising the declaration changed the parameter count: %d", len(normalizedIntent.Parameters))
	}
	for i := 0; i+1 < len(normalizedIntent.Parameters); i++ {
		if normalizedIntent.Parameters[i].Name > normalizedIntent.Parameters[i+1].Name {
			t.Fatalf("the declaration's parameters are not sorted by name: %+v", normalizedIntent.Parameters)
		}
	}
	values := map[string]string{}
	for _, p := range normalizedIntent.Parameters {
		values[p.Name] = p.Value
	}
	for _, want := range declared {
		if values[want.Name] != want.Value {
			t.Fatalf("normalising the declaration changed %q: %q -> %q", want.Name, want.Value, values[want.Name])
		}
	}

	plan := fixturePlan(t)
	plan.Parameters = append([]plugin.BuildParameter(nil), declared...)
	normalizedPlan := plugin.NormalizeBuildPlan(plan)
	for i := 0; i+1 < len(normalizedPlan.Parameters); i++ {
		if normalizedPlan.Parameters[i].Name > normalizedPlan.Parameters[i+1].Name {
			t.Fatalf("the plan's parameters are not sorted by name: %+v", normalizedPlan.Parameters)
		}
	}
	planValues := map[string]string{}
	for _, p := range normalizedPlan.Parameters {
		planValues[p.Name] = p.Value
	}
	for _, want := range declared {
		if planValues[want.Name] != want.Value {
			t.Fatalf("normalising the plan changed %q: %q -> %q", want.Name, want.Value, planValues[want.Name])
		}
	}
}

// Parameters that agree, in different declaration orders, compare equal.
func TestCompareIntentFieldsComparesParametersByName(t *testing.T) {
	intent := declarationFor(t, func(i *buildintent.BuildIntent) {
		i.Parameters = []plugin.BuildParameter{
			{Name: "optimization", Value: "size"},
			{Name: "debug", Value: "true"},
		}
	})
	plan := fixturePlan(t)
	// The plan lists them in the other order. Because comparison is by name, the
	// order must not matter.
	plan.Parameters = []plugin.BuildParameter{
		{Name: "debug", Value: "true"},
		{Name: "optimization", Value: "size"},
	}
	if err := compareIntentFields(intent, plan); err != nil {
		t.Fatalf("the same parameters in a different order were refused: %v", err)
	}
}

// A different value is refused and named.
func TestCompareIntentFieldsRefusesADifferentParameterValue(t *testing.T) {
	intent := declarationFor(t, func(i *buildintent.BuildIntent) {
		i.Parameters = []plugin.BuildParameter{{Name: "optimization", Value: "size"}}
	})
	plan := fixturePlan(t)
	plan.Parameters = []plugin.BuildParameter{{Name: "optimization", Value: "speed"}}
	err := compareIntentFields(intent, plan)
	if err == nil {
		t.Fatal("a different parameter value was accepted")
	}
	if !containsAll(err.Error(), "optimization", "size", "speed") {
		t.Fatalf("the mismatch did not name the parameter and both values: %v", err)
	}
}

// A parameter the plan does not have is refused.
func TestCompareIntentFieldsRefusesAnUnnamedParameter(t *testing.T) {
	intent := declarationFor(t, func(i *buildintent.BuildIntent) {
		i.Parameters = []plugin.BuildParameter{{Name: "extra", Value: "1"}}
	})
	if err := compareIntentFields(intent, fixturePlan(t)); err == nil {
		t.Fatal("a parameter missing from the plan was accepted")
	}
}

// A parameter the declaration does not have is refused, even when every named
// parameter agrees.
func TestCompareIntentFieldsRefusesAnExtraPlanParameter(t *testing.T) {
	plan := fixturePlan(t)
	plan.Parameters = []plugin.BuildParameter{{Name: "extra", Value: "1"}}
	if err := compareIntentFields(declarationFor(t, nil), plan); err == nil {
		t.Fatal("a plan parameter the declaration never named was accepted")
	}
}

// A declaration with no parameters and a plan with none agrees.
func TestCompareIntentFieldsAcceptsNoParameters(t *testing.T) {
	if err := compareIntentFields(declarationFor(t, nil), fixturePlan(t)); err != nil {
		t.Fatalf("an agreement with no parameters was refused: %v", err)
	}
}

func containsAll(haystack string, needles ...string) bool {
	for _, needle := range needles {
		found := false
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

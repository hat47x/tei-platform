package rulebinding

import (
	"reflect"
	"testing"
)

// The D4 fixture binds one rule `sales.line-amount` to two targets:
// sales.order-line.amount and query.order-list.line-total.

func impactTargets() []Target {
	return []Target{
		{ID: "sales.order-line.amount", Type: "amount"},
		{ID: "query.order-list.line-total", Type: "amount"},
		{ID: "query.order-list.tax", Type: "amount"},
	}
}

func impactBindings() []Binding {
	return []Binding{
		{RuleID: "sales.line-amount", TargetID: "sales.order-line.amount"},
		{RuleID: "sales.line-amount", TargetID: "query.order-list.line-total"},
		{RuleID: "sales.line-tax", TargetID: "query.order-list.tax"},
	}
}

func impactsByTarget(report ImpactReport) map[string]BindingImpact {
	out := map[string]BindingImpact{}
	for _, impact := range report.Impacts {
		out[impact.TargetID] = impact
	}
	return out
}

func TestImpactRemovedRuleLeavesEveryBindingUnresolved(t *testing.T) {
	report := PlanImpact(ImpactInput{
		Targets:  impactTargets(),
		Bindings: impactBindings(),
		Changes:  []RuleChange{{RuleID: "sales.line-amount", Change: ChangeRemoved}},
	})
	byTarget := impactsByTarget(report)
	for _, target := range []string{"sales.order-line.amount", "query.order-list.line-total"} {
		if byTarget[target].Status != BindingUnresolved {
			t.Errorf("%s = %q, want %q", target, byTarget[target].Status, BindingUnresolved)
		}
	}
	if byTarget["query.order-list.tax"].Status != BindingUnchanged {
		t.Errorf("unrelated binding changed: %+v", byTarget["query.order-list.tax"])
	}
	if report.Accepted {
		t.Error("a removed rule must not be accepted")
	}
	if report.Summary.Unresolved != 2 {
		t.Errorf("summary = %+v", report.Summary)
	}
}

func TestImpactTypeChangeRequiresReacceptanceNotSilentRebinding(t *testing.T) {
	report := PlanImpact(ImpactInput{
		Targets:  impactTargets(),
		Bindings: impactBindings(),
		Changes:  []RuleChange{{RuleID: "sales.line-amount", Change: ChangeTypeChanged, NewType: "text"}},
	})
	byTarget := impactsByTarget(report)
	for _, target := range []string{"sales.order-line.amount", "query.order-list.line-total"} {
		impact := byTarget[target]
		if impact.Status != BindingReacceptance {
			t.Errorf("%s = %q, want %q", target, impact.Status, BindingReacceptance)
		}
		if impact.Reason == "" {
			t.Errorf("%s has no reason", target)
		}
	}
	if report.Accepted || report.Applied {
		t.Errorf("report must not be accepted or applied: %+v", report)
	}
}

func TestImpactContractChangeRequiresReacceptance(t *testing.T) {
	report := PlanImpact(ImpactInput{
		Targets:  impactTargets(),
		Bindings: impactBindings(),
		Changes:  []RuleChange{{RuleID: "sales.line-amount", Change: ChangeContractChange}},
	})
	byTarget := impactsByTarget(report)
	if byTarget["sales.order-line.amount"].Status != BindingReacceptance {
		t.Fatalf("status = %q, want %q", byTarget["sales.order-line.amount"].Status, BindingReacceptance)
	}
}

func TestImpactVersionChangeIsRevalidationNotUnresolved(t *testing.T) {
	report := PlanImpact(ImpactInput{
		Targets:  impactTargets(),
		Bindings: impactBindings(),
		Changes:  []RuleChange{{RuleID: "sales.line-amount", Change: ChangeVersionChanged, NewVersion: "2"}},
	})
	byTarget := impactsByTarget(report)
	impact := byTarget["sales.order-line.amount"]
	if impact.Status != BindingRevalidation {
		t.Fatalf("status = %q, want %q", impact.Status, BindingRevalidation)
	}
	if impact.Status == BindingUnresolved {
		t.Fatal("a version change is not an unresolved binding")
	}
	if report.Summary.Revalidation != 2 {
		t.Errorf("summary = %+v", report.Summary)
	}
	if report.Accepted {
		t.Error("a revalidation candidate must not be accepted automatically")
	}
}

func TestImpactUnchangedRuleKeepsBindingsAccepted(t *testing.T) {
	report := PlanImpact(ImpactInput{
		Targets:  impactTargets(),
		Bindings: impactBindings()[:2],
	})
	if !report.Accepted {
		t.Fatalf("report = %+v want accepted", report)
	}
	for _, impact := range report.Impacts {
		if impact.Status != BindingUnchanged {
			t.Errorf("impact = %+v", impact)
		}
	}
	if report.Summary.Unchanged != 2 {
		t.Errorf("summary = %+v", report.Summary)
	}
}

func TestImpactOneRuleReuseReportsEachBindingOnce(t *testing.T) {
	report := PlanImpact(ImpactInput{
		Targets:  impactTargets(),
		Bindings: impactBindings()[:2],
		Changes:  []RuleChange{{RuleID: "sales.line-amount", Change: ChangeRemoved}},
	})
	if len(report.Impacts) != 2 {
		t.Fatalf("impacts = %+v want two distinct bindings", report.Impacts)
	}
	seen := map[string]bool{}
	for _, impact := range report.Impacts {
		if seen[impact.TargetID] {
			t.Fatalf("duplicate target %q", impact.TargetID)
		}
		seen[impact.TargetID] = true
	}
}

func TestImpactUnknownChangeKindFailsClosed(t *testing.T) {
	report := PlanImpact(ImpactInput{
		Targets:  impactTargets(),
		Bindings: impactBindings()[:1],
		Changes:  []RuleChange{{RuleID: "sales.line-amount", Change: "renamed-in-place"}},
	})
	if report.Impacts[0].Status != BindingReacceptance {
		t.Fatalf("status = %q, want %q", report.Impacts[0].Status, BindingReacceptance)
	}
	if report.Accepted {
		t.Error("an unknown change kind must not be accepted")
	}
}

func TestImpactMissingTargetIsDistinct(t *testing.T) {
	report := PlanImpact(ImpactInput{
		Targets:  impactTargets(),
		Bindings: []Binding{{RuleID: "sales.line-amount", TargetID: "sales.order-line.gone"}},
		Changes:  []RuleChange{{RuleID: "sales.line-amount", Change: ChangeRemoved}},
	})
	if report.Impacts[0].Status != BindingTargetMissing {
		t.Fatalf("status = %q, want %q", report.Impacts[0].Status, BindingTargetMissing)
	}
}

func TestImpactIsDeterministicAndOrdered(t *testing.T) {
	in := ImpactInput{
		Targets: impactTargets(),
		Bindings: []Binding{
			{RuleID: "sales.line-tax", TargetID: "query.order-list.tax"},
			{RuleID: "sales.line-amount", TargetID: "query.order-list.line-total"},
			{RuleID: "sales.line-amount", TargetID: "sales.order-line.amount"},
		},
		Changes: []RuleChange{
			{RuleID: "sales.line-tax", Change: ChangeVersionChanged, NewVersion: "3"},
			{RuleID: "sales.line-amount", Change: ChangeRemoved},
		},
	}
	first := PlanImpact(in)
	second := PlanImpact(in)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("impact is not deterministic: %+v != %+v", first, second)
	}
	want := []string{"sales.line-amount|query.order-list.line-total", "sales.line-amount|sales.order-line.amount", "sales.line-tax|query.order-list.tax"}
	for i, impact := range first.Impacts {
		got := impact.RuleID + "|" + impact.TargetID
		if got != want[i] {
			t.Fatalf("impact[%d] = %q, want %q", i, got, want[i])
		}
	}
}

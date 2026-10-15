// E37 reuses the main-integrated project-owned Go advisory and the existing
// Composition Registry. Provider selection is only a structural candidate:
// this test does not grant a Service Contract, Runtime Authority or Action.
package main

import (
	"testing"

	"tei/internal/composition"
	customeradvice "tei/plugins/reference-project-customer-advice"
	handoff "tei/reference/composition-evidence-handoff"
)

// Matches the reference-only, descriptive project Bridge declaration from
// tools/reference-project-advice-lit/contract_bridge.go. It is NOT an
// accepted #951 contract fingerprint, nor a new Project Profile or Lock.
func e37ReferenceContract() composition.Contract {
	return composition.Contract{
		ID:      "tei.reference.project.customer-advice",
		Version: "v0",
		Shape:   "customer-code-note-to-advisory-v0",
	}
}

func e37Provider(id, scope string) composition.Provider {
	return composition.Provider{
		ID: id, Scope: scope, Offers: e37ReferenceContract(),
	}
}

func e37Requirement(scope, provider string) composition.WiringRequirement {
	return composition.WiringRequirement{
		Scope: scope, ProviderID: provider,
		Contract: e37ReferenceContract(), SlotID: "reference-advice",
	}
}

func e37Registry(t *testing.T) *composition.Registry {
	t.Helper()
	r := composition.NewRegistry()
	_, err := r.Apply(0, []composition.Provider{
		e37Provider("case-a-advice", "case-a"),
		e37Provider("case-b-advice", "case-b"),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestE37OneProjectOwnedGoLogicCanServeTwoIsolatedSelections(t *testing.T) {
	r := e37Registry(t)
	for _, tc := range []struct {
		scope, provider, customer, expectedKind string
	}{
		{"case-a", "case-a-advice", "VIP-123", "review_suggested"},
		{"case-b", "case-b-advice", "NORMAL-5", "no_suggestion"},
	} {
		t.Run(tc.scope, func(t *testing.T) {
			set := r.InspectWiringSet([]composition.WiringRequirement{
				e37Requirement(tc.scope, tc.provider),
			})
			if set.Verdict != composition.ProposalCandidate ||
				!r.ProposalCurrent(set) || len(set.Findings) != 1 ||
				set.Findings[0].Selected == nil ||
				set.Findings[0].Selected.ProviderID != tc.provider {
				t.Fatalf("project choice was not structurally selected: %+v", set)
			}
			result, err := customeradvice.Evaluate(customeradvice.Input{
				CustomerCode: tc.customer,
			})
			if err != nil || !result.Advisory || result.Kind != tc.expectedKind {
				t.Fatalf("actual shared advisory failed: %+v %v", result, err)
			}
			// Structural selection and successful Go logic still lack
			// independent #951 evidence. No Host admission occurs here.
			report := handoff.ReconcileCurrent(r, set, nil)
			if report.Blocked || len(report.Subjects) != 1 ||
				report.Subjects[0].State != handoff.ReferenceMissing {
				t.Fatalf("source-only choice was promoted to conformant: %+v", report)
			}
		})
	}
}

func TestE37AmbiguousProvidersRequireAnExplicitProjectDecision(t *testing.T) {
	r := e37Registry(t)
	s := r.Snapshot()
	if _, err := r.Apply(s.Revision, []composition.Provider{
		e37Provider("case-a-alternate", "case-a"),
	}, nil); err != nil {
		t.Fatal(err)
	}
	implicit := r.InspectWiring(e37Requirement("case-a", ""))
	if implicit.Verdict != composition.ProposalConflict ||
		implicit.Reason != composition.ReasonAmbiguousProviders ||
		implicit.Selected != nil {
		t.Fatalf("project-specific selection was guessed: %+v", implicit)
	}
	pinned := r.InspectWiring(e37Requirement("case-a", "case-a-advice"))
	if pinned.Verdict != composition.ProposalCandidate ||
		pinned.Selected == nil || pinned.Selected.ProviderID != "case-a-advice" {
		t.Fatalf("explicit project choice was not kept: %+v", pinned)
	}
	other := r.InspectWiring(e37Requirement("case-b", ""))
	if other.Verdict != composition.ProposalCandidate ||
		other.Selected == nil || other.Selected.ProviderID != "case-b-advice" {
		t.Fatalf("unrelated project became ambiguous: %+v", other)
	}
}

func TestE37ProviderChangeInvalidatesOnlyAffectedProject(t *testing.T) {
	r := e37Registry(t)
	caseA := r.InspectWiringSet([]composition.WiringRequirement{
		e37Requirement("case-a", "case-a-advice"),
	})
	caseB := r.InspectWiringSet([]composition.WiringRequirement{
		e37Requirement("case-b", "case-b-advice"),
	})
	if !r.ProposalCurrent(caseA) || !r.ProposalCurrent(caseB) {
		t.Fatal("initial projects did not have current structural proposals")
	}
	s := r.Snapshot()
	if _, err := r.Apply(s.Revision, []composition.Provider{{
		ID: "case-a-advice", Scope: "case-a",
		Offers: composition.Contract{
			ID: e37ReferenceContract().ID, Version: "test-updated",
			Shape: "changed-project-a-only",
		},
	}}, nil); err != nil {
		t.Fatal(err)
	}
	if r.ProposalCurrent(caseA) {
		t.Fatal("changed project A continued using stale selection")
	}
	if !r.ProposalCurrent(caseB) {
		t.Fatal("project A change invalidated project B")
	}
	if report := handoff.ReconcileCurrent(r, caseA, nil); !report.Blocked {
		t.Fatal("stale project selection was reconciled as current")
	}
	if report := handoff.ReconcileCurrent(r, caseB, nil); report.Blocked ||
		len(report.Subjects) != 1 || report.Subjects[0].State != handoff.ReferenceMissing {
		t.Fatalf("project B should remain independently reviewable: %+v", report)
	}
}

func TestE37CannotAccidentallyCrossProjectOrConflateScopedBatches(t *testing.T) {
	r := e37Registry(t)
	cross := r.InspectWiring(e37Requirement("case-a", "case-b-advice"))
	if cross.Verdict == composition.ProposalCandidate ||
		cross.Selected != nil {
		t.Fatalf("foreign project provider was selected: %+v", cross)
	}
	both := r.InspectWiringSet([]composition.WiringRequirement{
		e37Requirement("case-a", "case-a-advice"),
		e37Requirement("case-b", "case-b-advice"),
	})
	if both.Verdict == composition.ProposalCandidate {
		t.Fatalf("cross-project batch gained one review scope: %+v", both)
	}
}

func TestE37RealProjectGoAdviceDoesNotBecomeBusinessRejection(t *testing.T) {
	result, err := customeradvice.Evaluate(customeradvice.Input{
		CustomerCode: "VIP-999", Note: "",
	})
	if err != nil || !result.Advisory || result.Kind != "review_suggested" {
		t.Fatalf("actual project-owned advisory failed: %+v %v", result, err)
	}
	if result.Schema != "tei.reference.project-customer-advice/v0" ||
		result.Message == "" {
		t.Fatalf("advice response identity changed: %+v", result)
	}
	// The result has no approval/denial or effect-bearing fields. An order
	// validation cannot be inferred from the "review_suggested" label.
	if _, err := customeradvice.Evaluate(customeradvice.Input{
		CustomerCode: "",
	}); err == nil {
		t.Fatal("malformed project advisory input was silently accepted")
	}
}

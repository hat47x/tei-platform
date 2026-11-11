package composition

import "testing"

// ProposalReason is why a proposal was refused. A caller shows it to an operator,
// so a reason that no test names is a message nothing proves can happen (#4032).
func TestProposalReasonsAreCovered(t *testing.T) {
	reasons := []ProposalReason{
		ReasonStructurallyEligible, ReasonInvalidRequirement, ReasonNoKnownProvider,
		ReasonAmbiguousProviders, ReasonExplicitMismatch, ReasonDependencyIneligible,
		ReasonMixedScopes, ReasonInconsistentSlot,
	}
	want := []string{
		"structurally-eligible", "invalid-requirement", "no-known-provider",
		"ambiguous-providers", "explicit-provider-mismatch", "dependency-ineligible",
		"mixed-scopes", "inconsistent-slot",
	}
	if len(reasons) != len(want) {
		t.Fatalf("a reason was dropped: %d values", len(reasons))
	}
	seen := map[string]bool{}
	for i, reason := range reasons {
		if string(reason) != want[i] {
			t.Fatalf("reasons[%d]=%q want %q", i, reason, want[i])
		}
		if seen[string(reason)] {
			t.Fatalf("reason %q is declared twice", reason)
		}
		seen[string(reason)] = true
	}
}

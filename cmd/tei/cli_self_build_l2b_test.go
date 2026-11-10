package main

import (
	"strings"
	"testing"
)

// These tests pin the L2-b predicate (§4.23) as a set of closed questions about
// the observations an attempt recorded. Each case removes exactly one condition,
// so a failure names the condition rather than only "the claim is false".

// satisfied builds the report of an attempt that met every condition.
func satisfiedL2B() selfBuildOuterEvidence {
	return selfBuildOuterEvidence{
		PlanReceipt: &PlanReceipt{
			State:     intentResolutionResolved,
			DerivedBy: derivedByEntryInProcess,
		},
		IntentBinding: &selfBuildIntentBinding{
			CheckPerformed: true,
			CheckMatched:   true,
		},
		PlanCrossCheck: &planCrossCheck{State: planCrossCheckMatched},
		PlanArtifactBinding: &planArtifactBinding{
			State: planArtifactBound,
			// The per-observation flag stays false: no single observation is the
			// claim, so the predicate must not read it.
			L2BClaimed: false,
		},
		ArtifactReproducibility: &selfBuildReproducibility{State: reproducibilityReproduced},
		BuilderBinding:          &selfBuildBuilderBinding{State: builderBindingBound, Matched: true},
	}
}

func TestAssessL2BClaimsWhenEveryConditionHolds(t *testing.T) {
	claim := assessL2B(satisfiedL2B())
	if !claim.Claimed {
		t.Fatalf("the claim was not made despite every condition holding: unmet=%v", claim.Unmet)
	}
	if len(claim.Unmet) != 0 {
		t.Fatalf("unmet=%v", claim.Unmet)
	}
	if claim.Boundary == "" {
		t.Fatal("the claim has no boundary")
	}
}

func TestAssessL2BNamesEveryMissingCondition(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*selfBuildOuterEvidence)
		want   string
	}{
		{
			name:   "no plan derived",
			mutate: func(r *selfBuildOuterEvidence) { r.PlanReceipt = nil },
			want:   "no plan was derived",
		},
		{
			name: "plan derived by a child, not this entry",
			mutate: func(r *selfBuildOuterEvidence) {
				r.PlanReceipt.DerivedBy = ""
				r.PlanReceipt.ResolverPath = "/some/resolver"
			},
			want: "not by this entry",
		},
		{
			name: "derivation did not resolve",
			mutate: func(r *selfBuildOuterEvidence) {
				r.PlanReceipt.State = intentResolutionRefused
				r.PlanReceipt.FailureCause = "corrupt catalog"
			},
			want: "did not resolve",
		},
		{
			name:   "no declaration binding",
			mutate: func(r *selfBuildOuterEvidence) { r.IntentBinding = nil },
			want:   "no declaration binding",
		},
		{
			name:   "byte check did not run",
			mutate: func(r *selfBuildOuterEvidence) { r.IntentBinding.CheckPerformed = false },
			want:   "byte check did not run",
		},
		{
			name:   "builder read other bytes",
			mutate: func(r *selfBuildOuterEvidence) { r.IntentBinding.CheckMatched = false },
			want:   "different declaration bytes",
		},
		{
			name:   "plan cross-check missing",
			mutate: func(r *selfBuildOuterEvidence) { r.PlanCrossCheck = nil },
			want:   "not matched against the entry",
		},
		{
			name:   "plan cross-check mismatched",
			mutate: func(r *selfBuildOuterEvidence) { r.PlanCrossCheck.State = planCrossCheckMismatched },
			want:   "not matched against the entry",
		},
		{
			name:   "artifact not bound",
			mutate: func(r *selfBuildOuterEvidence) { r.PlanArtifactBinding = nil },
			want:   "not bound to the artifact",
		},
		{
			name:   "artifact not reproduced",
			mutate: func(r *selfBuildOuterEvidence) { r.ArtifactReproducibility = nil },
			want:   "not reproduced from the plan",
		},
		{
			name:   "reproduction unperformed",
			mutate: func(r *selfBuildOuterEvidence) { r.ArtifactReproducibility.State = reproducibilityUnperformed },
			want:   "not reproduced from the plan",
		},
		{
			name:   "started file never compared with the declared builder",
			mutate: func(r *selfBuildOuterEvidence) { r.BuilderBinding = nil },
			want:   "not shown to be the builder the plan declares (unperformed)",
		},
		{
			name:   "started file is not the declared builder",
			mutate: func(r *selfBuildOuterEvidence) { r.BuilderBinding.State = builderBindingMismatched },
			want:   "not shown to be the builder the plan declares (mismatched)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report := satisfiedL2B()
			tc.mutate(&report)
			claim := assessL2B(report)
			if claim.Claimed {
				t.Fatal("the claim was made despite a missing condition")
			}
			joined := strings.Join(claim.Unmet, "\n")
			if !strings.Contains(joined, tc.want) {
				t.Fatalf("unmet does not name %q:\n%s", tc.want, joined)
			}
		})
	}
}

// An empty attempt must report every condition, not just the first: a reader
// needs the whole list to know what to change.
func TestAssessL2BOnAnEmptyAttemptReportsEveryCondition(t *testing.T) {
	claim := assessL2B(selfBuildOuterEvidence{})
	if claim.Claimed {
		t.Fatal("an empty attempt claimed L2-b")
	}
	// Six independent conditions: the five of §4.23 and the builder binding of
	// §4.47. The three facts about the plan receipt answer one question with
	// mutually exclusive answers, so an attempt with no receipt reports one entry
	// for it, not three.
	if len(claim.Unmet) != 6 {
		t.Fatalf("unmet has %d entries, want 6:\n%s", len(claim.Unmet), strings.Join(claim.Unmet, "\n"))
	}
}

// The per-observation flag must never be the input to the claim: if it were, the
// claim would depend on a field whose purpose is to say "not yet".
func TestAssessL2BIgnoresThePerObservationFlag(t *testing.T) {
	report := satisfiedL2B()
	report.PlanArtifactBinding.L2BClaimed = true
	if claim := assessL2B(report); !claim.Claimed {
		t.Fatalf("a true per-observation flag changed the outcome: %v", claim.Unmet)
	}

	report = satisfiedL2B()
	report.PlanArtifactBinding.L2BClaimed = false
	if claim := assessL2B(report); !claim.Claimed {
		t.Fatalf("a false per-observation flag changed the outcome: %v", claim.Unmet)
	}
}

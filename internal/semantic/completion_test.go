package semantic

import (
	"reflect"
	"testing"
)

func completionCapability(policy EmptyPolicy) Capability {
	return Capability{ID: "sales.submit", EmptyPolicy: policy}
}

func TestEvaluateCompletionHoldsOnlyWithCoveredTargets(t *testing.T) {
	capability := completionCapability("")
	held := CompletionEvidence{
		Targets:  []TargetResult{{ID: "L1", Holds: true}, {ID: "L2", Holds: true}},
		Coverage: CoverageComplete,
	}
	decision := EvaluateCompletion(capability, EmptyPolicyUniversal, held)
	if decision.State != CompletionHolds {
		t.Fatalf("complete covered targets=%+v", decision)
	}
	if decision.EmptyPolicy != EmptyPolicyRequiresEvidence || decision.PolicySource != "default" {
		t.Fatalf("policy decision=%+v", decision)
	}

	partial := held
	partial.Coverage = CoveragePartial
	if decision := EvaluateCompletion(capability, EmptyPolicyUniversal, partial); decision.State != CompletionUnverified {
		t.Fatalf("partial coverage=%+v", decision)
	}

	violating := CompletionEvidence{
		Targets:  []TargetResult{{ID: "L1", Holds: true}, {ID: "L2", Holds: false}},
		Coverage: CoveragePartial,
	}
	decision = EvaluateCompletion(capability, EmptyPolicyUniversal, violating)
	if decision.State != CompletionFails || !reflect.DeepEqual(decision.Violations, []string{"L2"}) {
		t.Fatalf("violation decision=%+v", decision)
	}
}

func TestEvaluateCompletionEmptySetPolicies(t *testing.T) {
	cases := []struct {
		name       string
		capability Capability
		kind       EmptyPolicyCondition
		evidence   CompletionEvidence
		want       CompletionState
	}{
		{
			name:       "universal empty without evidence is unverified",
			capability: completionCapability(""),
			kind:       EmptyPolicyUniversal,
			evidence:   CompletionEvidence{Coverage: CoverageComplete},
			want:       CompletionUnverified,
		},
		{
			name:       "universal empty with independent evidence holds",
			capability: completionCapability(""),
			kind:       EmptyPolicyUniversal,
			evidence:   CompletionEvidence{Coverage: CoverageComplete, EmptyJustified: true},
			want:       CompletionHolds,
		},
		{
			name:       "allow override on empty holds",
			capability: completionCapability(EmptyPolicyAllow),
			kind:       EmptyPolicyUniversal,
			evidence:   CompletionEvidence{Coverage: CoverageComplete},
			want:       CompletionHolds,
		},
		{
			name:       "allow on empty without coverage is unverified",
			capability: completionCapability(EmptyPolicyAllow),
			kind:       EmptyPolicyUniversal,
			evidence:   CompletionEvidence{Coverage: CoverageUnestablished},
			want:       CompletionUnverified,
		},
		{
			name:       "deny override on empty fails",
			capability: completionCapability(EmptyPolicyDeny),
			kind:       EmptyPolicyUniversal,
			evidence:   CompletionEvidence{Coverage: CoverageComplete},
			want:       CompletionFails,
		},
		{
			name:       "deny on unobserved empty is unverified",
			capability: completionCapability(EmptyPolicyDeny),
			kind:       EmptyPolicyUniversal,
			evidence:   CompletionEvidence{Coverage: CoverageUnestablished},
			want:       CompletionUnverified,
		},
		{
			name:       "deny on partially observed empty is unverified",
			capability: completionCapability(EmptyPolicyDeny),
			kind:       EmptyPolicyUniversal,
			evidence:   CompletionEvidence{Coverage: CoveragePartial},
			want:       CompletionUnverified,
		},
		{
			name:       "existence default deny on unobserved empty is unverified",
			capability: completionCapability(""),
			kind:       EmptyPolicyExistence,
			evidence:   CompletionEvidence{Coverage: CoverageUnestablished},
			want:       CompletionUnverified,
		},
		{
			name:       "existence default deny on empty fails",
			capability: completionCapability(""),
			kind:       EmptyPolicyExistence,
			evidence:   CompletionEvidence{Coverage: CoverageComplete},
			want:       CompletionFails,
		},
		{
			name:       "absence default allow on empty holds",
			capability: completionCapability(""),
			kind:       EmptyPolicyAbsence,
			evidence:   CompletionEvidence{Coverage: CoverageComplete},
			want:       CompletionHolds,
		},
		{
			name:       "declared allow overrides the existence default",
			capability: completionCapability(EmptyPolicyAllow),
			kind:       EmptyPolicyExistence,
			evidence:   CompletionEvidence{Coverage: CoverageComplete},
			want:       CompletionHolds,
		},
		{
			name:       "handoff without confirmation is unverified",
			capability: completionCapability(""),
			kind:       EmptyPolicyHandoff,
			evidence: CompletionEvidence{
				Targets: []TargetResult{{ID: "L1", Holds: true}}, Coverage: CoverageComplete,
			},
			want: CompletionUnverified,
		},
		{
			name:       "handoff with confirmation holds",
			capability: completionCapability(""),
			kind:       EmptyPolicyHandoff,
			evidence: CompletionEvidence{
				Targets: []TargetResult{{ID: "L1", Holds: true}}, Coverage: CoverageComplete,
				HandoffConfirmed: true,
			},
			want: CompletionHolds,
		},
		{
			name:       "unknown condition has no default",
			capability: completionCapability(""),
			kind:       EmptyPolicyCondition("custom"),
			evidence:   CompletionEvidence{Coverage: CoverageComplete},
			want:       CompletionUnverified,
		},
		{
			name:       "conflicting evidence is a contradiction",
			capability: completionCapability(""),
			kind:       EmptyPolicyUniversal,
			evidence: CompletionEvidence{
				Targets: []TargetResult{{ID: "L1", Holds: true}}, Coverage: CoverageComplete, Conflicting: true,
			},
			want: CompletionContradiction,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			decision := EvaluateCompletion(testCase.capability, testCase.kind, testCase.evidence)
			if decision.State != testCase.want {
				t.Fatalf("state=%s reason=%q, want %s", decision.State, decision.Reason, testCase.want)
			}
		})
	}
}

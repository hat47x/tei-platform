package main

import (
	"reflect"
	"testing"
)

const (
	actorContextR3NClassAuthorityUnresolved               = "authority-unresolved"
	actorContextR3NClassRuntimeContextUnresolved          = "runtime-context-unresolved"
	actorContextR3NClassEvaluationNotAttempted            = "evaluation-not-attempted"
	actorContextR3NClassEvaluationObservationMismatch     = "evaluation-observation-mismatch"
	actorContextR3NClassEvaluatedDenied                   = "evaluated-denied"
	actorContextR3NClassCurrentContextEvaluationUnresolved = "current-context-evaluation-unresolved"
)

type actorContextR3NStopInputProbe struct {
	BusinessAddress   string
	BlockingReasons   []string
	EvaluationStatus  string
	ObservationOutcome string
}

type actorContextR3NStopTaxonomyProbe struct {
	BusinessAddress         string
	StopStage               string
	StopClass               string
	MachineReasons          []string
	ResolutionNeedClasses []string
	RuntimeOutcome          string
	DownstreamNotAttempted  []string
	AuthorityLimits         []string
}

func TestActorContextR3NRequirementPrincipalActionAndEvaluationStopsRemainDistinct(t *testing.T) {
	cases := []struct {
		name         string
		input        actorContextR3NStopInputProbe
		wantStage    string
		wantClass    string
		wantNeeds  []string
		wantReason   string
	}{
		{
			name: "requirement",
			input: actorContextR3NStopInputProbe{
				BusinessAddress: "order-entry.order-handling/BP-02",
				BlockingReasons: []string{"authorization-requirement-unresolved"},
			},
			wantStage:   "authorization-requirement-authority",
			wantClass:   actorContextR3NClassAuthorityUnresolved,
			wantNeeds: []string{"authorization-requirement-authority"},
			wantReason:  "authorization-requirement-unresolved",
		},
		{
			name: "principal",
			input: actorContextR3NStopInputProbe{
				BusinessAddress: "order-entry.order-handling/BP-06",
				BlockingReasons: []string{"actor-principal-unresolved"},
			},
			wantStage:   "actor-principal",
			wantClass:   actorContextR3NClassRuntimeContextUnresolved,
			wantNeeds: []string{"runtime-principal"},
			wantReason:  "actor-principal-unresolved",
		},
		{
			name: "resource-authority",
			input: actorContextR3NStopInputProbe{
				BusinessAddress: "order-entry.order-handling/BP-06",
				BlockingReasons: []string{"authorization-resource-authority-unresolved"},
			},
			wantStage:   "authorization-resource-authority",
			wantClass:   actorContextR3NClassAuthorityUnresolved,
			wantNeeds:   []string{"authorization-resource-authority"},
			wantReason:  "authorization-resource-authority-unresolved",
		},
		{
			name: "action-authority",
			input: actorContextR3NStopInputProbe{
				BusinessAddress: "order-entry.order-handling/BP-06",
				BlockingReasons: []string{"authorization-action-authority-unresolved"},
			},
			wantStage:   "authorization-action-authority",
			wantClass:   actorContextR3NClassAuthorityUnresolved,
			wantNeeds: []string{"authorization-action-authority"},
			wantReason:  "authorization-action-authority-unresolved",
		},
		{
			name: "evaluation-not-attempted",
			input: actorContextR3NStopInputProbe{
				BusinessAddress: "order-entry.order-handling/BP-06",
				BlockingReasons: []string{"authorization-not-evaluated"},
				EvaluationStatus: actorContextR3IEvaluationNotEvaluated,
			},
			wantStage:  "authorization-evaluation",
			wantClass:  actorContextR3NClassEvaluationNotAttempted,
			wantNeeds:  []string{"authorization-evaluation"},
			wantReason: "authorization-not-evaluated",
		},
	}

	seen := map[string]bool{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := actorContextR3NClassifyStop(tc.input)
			if got.StopStage != tc.wantStage || got.StopClass != tc.wantClass {
				t.Fatalf("R3N collapsed distinct stop semantics: got=%+v wantStage=%q wantClass=%q",
					got, tc.wantStage, tc.wantClass)
			}
			if !reflect.DeepEqual(got.ResolutionNeedClasses, tc.wantNeeds) {
				t.Fatalf("R3N mapped resolution need incorrectly: got=%v want=%v",
					got.ResolutionNeedClasses, tc.wantNeeds)
			}
			if !reflect.DeepEqual(got.MachineReasons, []string{tc.wantReason}) {
				t.Fatalf("R3N lost exact machine-readable reason: %+v", got.MachineReasons)
			}
			key := got.StopStage + "/" + got.StopClass
			if seen[key] {
				t.Fatalf("R3N produced duplicate taxonomy key for semantically distinct case: %s", key)
			}
			seen[key] = true
		})
	}
}

func TestActorContextR3NEarliestBlockerWinsWithoutPromotingDeferredStages(t *testing.T) {
	got := actorContextR3NClassifyStop(actorContextR3NStopInputProbe{
		BusinessAddress: "order-entry.order-handling/BP-02",
		BlockingReasons: []string{
			"authorization-not-evaluated",
			"authorization-action-authority-unresolved",
			"authorization-requirement-unresolved",
		},
	})

	if got.StopStage != "authorization-requirement-authority" ||
		got.StopClass != actorContextR3NClassAuthorityUnresolved {
		t.Fatalf("R3N did not preserve the earliest blocker: %+v", got)
	}
	if !reflect.DeepEqual(got.MachineReasons, []string{"authorization-requirement-unresolved"}) {
		t.Fatalf("R3N promoted deferred blockers into the current stop reason: %+v", got.MachineReasons)
	}
	if !reflect.DeepEqual(got.ResolutionNeedClasses, []string{"authorization-requirement-authority"}) {
		t.Fatalf("R3N inferred downstream missing authority from an earlier stop: %+v", got)
	}
}

func TestActorContextR3NDeniedObservationIsRuntimeOutcomeNotMissingAuthority(t *testing.T) {
	got := actorContextR3NClassifyStop(actorContextR3NStopInputProbe{
		BusinessAddress:    "order-entry.order-handling/BP-06",
		EvaluationStatus:   actorContextR3IEvaluationCurrentObserved,
		ObservationOutcome: "denied",
	})

	if got.StopStage != "authorization-evaluation" ||
		got.StopClass != actorContextR3NClassEvaluatedDenied ||
		got.RuntimeOutcome != "denied" {
		t.Fatalf("R3N did not preserve evaluated denial as a distinct runtime outcome: %+v", got)
	}
	if len(got.ResolutionNeedClasses) != 0 {
		t.Fatalf("evaluated denial was misrepresented as a resolution need: %+v", got.ResolutionNeedClasses)
	}
	if !reflect.DeepEqual(got.MachineReasons, []string{"authorization-evaluation-denied"}) {
		t.Fatalf("R3N lost the derived denial reason: %+v", got.MachineReasons)
	}
}

func TestActorContextR3NObservationMismatchAndCrossContextReuseRemainDistinct(t *testing.T) {
	mismatch := actorContextR3NClassifyStop(actorContextR3NStopInputProbe{
		BusinessAddress:  "order-entry.order-handling/BP-06",
		BlockingReasons:  []string{"authorization-evaluation-observation-mismatch"},
		EvaluationStatus: actorContextR3IEvaluationObservationMismatch,
	})
	reuse := actorContextR3NClassifyStop(actorContextR3NStopInputProbe{
		BusinessAddress:  "order-entry.follow-up/FU-00",
		BlockingReasons:  []string{"authorization-current-context-evaluation-unresolved"},
		EvaluationStatus: actorContextR3IEvaluationPriorContextReuseUnresolved,
	})

	if mismatch.StopClass != actorContextR3NClassEvaluationObservationMismatch {
		t.Fatalf("Observation mismatch was collapsed: %+v", mismatch)
	}
	if !reflect.DeepEqual(mismatch.ResolutionNeedClasses, []string{"authorization-evaluation-observation"}) {
		t.Fatalf("Observation mismatch lost its resolution need: %+v", mismatch)
	}
	if reuse.StopClass != actorContextR3NClassCurrentContextEvaluationUnresolved {
		t.Fatalf("cross-context reuse uncertainty was collapsed: %+v", reuse)
	}
	if !reflect.DeepEqual(reuse.ResolutionNeedClasses, []string{"authorization-current-context-evaluation"}) {
		t.Fatalf("cross-context reuse uncertainty lost its resolution need: %+v", reuse)
	}
	if mismatch.StopClass == reuse.StopClass {
		t.Fatalf("distinct runtime evaluation states became the same taxonomy class")
	}
	if mismatch.RuntimeOutcome == "denied" || reuse.RuntimeOutcome == "denied" {
		t.Fatalf("non-denial runtime states were rewritten as denied: mismatch=%+v reuse=%+v", mismatch, reuse)
	}
}

func TestActorContextR3NStopTaxonomyHasNoRoutingOrAuthorityShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3NStopTaxonomyProbe{})
	for _, forbidden := range []string{
		"HandoffOwner",
		"SuggestedAction",
		"RecommendedAction",
		"Authorized",
		"Allowed",
		"Approved",
		"CanExecute",
		"Confidence",
		"GenericError",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3N taxonomy introduced forbidden shortcut field %q", forbidden)
		}
	}
}

func actorContextR3NClassifyStop(input actorContextR3NStopInputProbe) actorContextR3NStopTaxonomyProbe {
	result := actorContextR3NStopTaxonomyProbe{
		BusinessAddress: input.BusinessAddress,
		AuthorityLimits: []string{
			"distinct-stop-states-do-not-collapse-into-generic-authorization-error",
			"resolution-need-class-does-not-imply-missing-value-or-owner",
			"authority-unresolved-does-not-equal-runtime-denied",
			"observation-mismatch-does-not-equal-denied",
			"prior-context-reuse-unresolved-does-not-equal-not-evaluated",
			"stop-taxonomy-does-not-select-handoff-owner-or-next-action",
		},
	}

	type stopRule struct {
		reason     string
		stage      string
		class      string
		missing    string
		downstream []string
	}
	rules := []stopRule{
		{
			reason:  "authorization-requirement-unresolved",
			stage:   "authorization-requirement-authority",
			class:   actorContextR3NClassAuthorityUnresolved,
			missing: "authorization-requirement-authority",
			downstream: []string{
				"authorization-principal-resolution",
				"authorization-target-resolution",
				"authorization-evaluation",
				"protected-operation-invocation",
				"business-completion",
			},
		},
		{
			reason:  "actor-principal-unresolved",
			stage:   "actor-principal",
			class:   actorContextR3NClassRuntimeContextUnresolved,
			missing: "runtime-principal",
			downstream: []string{
				"authorization-evaluation",
				"protected-operation-invocation",
				"business-completion",
			},
		},
		{
			reason:  "authorization-resource-authority-unresolved",
			stage:   "authorization-resource-authority",
			class:   actorContextR3NClassAuthorityUnresolved,
			missing: "authorization-resource-authority",
			downstream: []string{
				"authorization-evaluation",
				"protected-operation-invocation",
				"business-completion",
			},
		},
		{
			reason:  "authorization-resource-unresolved",
			stage:   "authorization-resource-authority",
			class:   actorContextR3NClassAuthorityUnresolved,
			missing: "authorization-resource-authority",
			downstream: []string{
				"authorization-evaluation",
				"protected-operation-invocation",
				"business-completion",
			},
		},
		{
			reason:  "authorization-action-authority-unresolved",
			stage:   "authorization-action-authority",
			class:   actorContextR3NClassAuthorityUnresolved,
			missing: "authorization-action-authority",
			downstream: []string{
				"authorization-evaluation",
				"protected-operation-invocation",
				"business-completion",
			},
		},
		{
			reason:  "authorization-action-unresolved",
			stage:   "authorization-action-authority",
			class:   actorContextR3NClassAuthorityUnresolved,
			missing: "authorization-action-authority",
			downstream: []string{
				"authorization-evaluation",
				"protected-operation-invocation",
				"business-completion",
			},
		},
		{
			reason:  "authorization-evaluation-observation-mismatch",
			stage:   "authorization-evaluation",
			class:   actorContextR3NClassEvaluationObservationMismatch,
			missing: "authorization-evaluation-observation",
			downstream: []string{
				"protected-operation-invocation",
				"business-completion",
			},
		},
		{
			reason:  "authorization-current-context-evaluation-unresolved",
			stage:   "authorization-evaluation",
			class:   actorContextR3NClassCurrentContextEvaluationUnresolved,
			missing: "authorization-current-context-evaluation",
			downstream: []string{
				"protected-operation-invocation",
				"business-completion",
			},
		},
		{
			reason:  "authorization-not-evaluated",
			stage:   "authorization-evaluation",
			class:   actorContextR3NClassEvaluationNotAttempted,
			missing: "authorization-evaluation",
			downstream: []string{
				"protected-operation-invocation",
				"business-completion",
			},
		},
	}

	reasonSet := map[string]bool{}
	for _, reason := range input.BlockingReasons {
		reasonSet[reason] = true
	}
	for _, rule := range rules {
		if !reasonSet[rule.reason] {
			continue
		}
		result.StopStage = rule.stage
		result.StopClass = rule.class
		result.MachineReasons = []string{rule.reason}
		if rule.missing != "" {
			result.ResolutionNeedClasses = []string{rule.missing}
		}
		result.DownstreamNotAttempted = actorContextR0SortedUnique(rule.downstream)
		return result
	}

	if input.EvaluationStatus == actorContextR3IEvaluationCurrentObserved &&
		input.ObservationOutcome == "denied" {
		result.StopStage = "authorization-evaluation"
		result.StopClass = actorContextR3NClassEvaluatedDenied
		result.MachineReasons = []string{"authorization-evaluation-denied"}
		result.RuntimeOutcome = "denied"
		result.DownstreamNotAttempted = []string{
			"business-completion",
			"protected-operation-invocation",
		}
	}

	return result
}

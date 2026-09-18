package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3XStatusNoContextIdentityChange            = "no-context-identity-change"
	actorContextR3XStatusObservationContextMismatch          = "observation-context-mismatch"
	actorContextR3XStatusRequirementTransitionUnresolved     = "requirement-transition-authority-unresolved"
	actorContextR3XStatusTupleCompatibleFreshnessUnresolved  = "tuple-compatible-freshness-authority-unresolved"
	actorContextR3XStatusRuntimeOutcomeChanged               = "runtime-outcome-changed-same-context"

	actorContextR3XObservationTupleMatch    = "observation-tuple-match"
	actorContextR3XObservationTupleMismatch = "observation-tuple-mismatch"
)

type actorContextR3XAuthorizationContextProbe struct {
	RequirementStatus string
	BusinessAddress   string
	SubjectID         string
	Action            string
	Resource          string
}

type actorContextR3XContextDeltaProbe struct {
	Before                      actorContextR3XAuthorizationContextProbe
	After                       actorContextR3XAuthorizationContextProbe
	ChangedProjectionCategories []string
	StaticSemanticChanged       bool
	RuntimeOutcomeBefore        string
	RuntimeOutcomeAfter         string
}

type actorContextR3XContextImpactProbe struct {
	Status                           string
	ChangedTupleComponents           []string
	RequirementApplicabilityChanged  bool
	RuntimeOutcomeChanged            bool
	ObservationTupleStatus                string
	FreshnessAuthorityStatus         string
	AuthorityLimits                  []string
}

func TestActorContextR3XActionChangeMakesPriorObservationContextMismatch(t *testing.T) {
	before := actorContextR3XBaseContext()
	after := before
	after.Action = "browse"

	got, err := actorContextR3XAnalyzeContextChange(
		actorContextR3XBaseObservation("allowed"),
		actorContextR3XContextDeltaProbe{Before: before, After: after},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != actorContextR3XStatusObservationContextMismatch ||
		!reflect.DeepEqual(got.ChangedTupleComponents, []string{"action"}) ||
		got.ObservationTupleStatus != actorContextR3XObservationTupleMismatch {
		t.Fatalf("R3X action change did not invalidate prior Observation context: %+v", got)
	}
}

func TestActorContextR3XOtherTupleComponentChangesRemainExplicit(t *testing.T) {
	cases := map[string]struct {
		mutate func(*actorContextR3XAuthorizationContextProbe)
		want   string
	}{
		"business-address": {
			mutate: func(c *actorContextR3XAuthorizationContextProbe) {
				c.BusinessAddress = "order-entry.follow-up/FU-00"
			},
			want: "business-address",
		},
		"subject": {
			mutate: func(c *actorContextR3XAuthorizationContextProbe) {
				c.SubjectID = "different-subject"
			},
			want: "subject-id",
		},
		"resource": {
			mutate: func(c *actorContextR3XAuthorizationContextProbe) {
				c.Resource = "order_entry.order_record"
			},
			want: "resource",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			before := actorContextR3XBaseContext()
			after := before
			tc.mutate(&after)
			got, err := actorContextR3XAnalyzeContextChange(
				actorContextR3XBaseObservation("allowed"),
				actorContextR3XContextDeltaProbe{Before: before, After: after},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != actorContextR3XStatusObservationContextMismatch ||
				!reflect.DeepEqual(got.ChangedTupleComponents, []string{tc.want}) {
				t.Fatalf("%s change was not preserved as tuple mismatch: %+v", name, got)
			}
		})
	}
}

func TestActorContextR3XRequirementChangeIsNotTupleMismatch(t *testing.T) {
	before := actorContextR3XBaseContext()
	after := before
	after.RequirementStatus = actorContextR3BAuthorizationUnresolved

	got, err := actorContextR3XAnalyzeContextChange(
		actorContextR3XBaseObservation("allowed"),
		actorContextR3XContextDeltaProbe{Before: before, After: after},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != actorContextR3XStatusRequirementTransitionUnresolved ||
		!got.RequirementApplicabilityChanged ||
		len(got.ChangedTupleComponents) != 0 {
		t.Fatalf("R3X requirement transition was confused with tuple change: %+v", got)
	}
	if got.ObservationTupleStatus == actorContextR3XObservationTupleMismatch {
		t.Fatalf("R3X requirement transition fabricated Observation tuple mismatch: %+v", got)
	}
}

func TestActorContextR3XProjectionCategoryChangeDoesNotInferTupleComponent(t *testing.T) {
	before := actorContextR3XBaseContext()
	got, err := actorContextR3XAnalyzeContextChange(
		actorContextR3XBaseObservation("allowed"),
		actorContextR3XContextDeltaProbe{
			Before: before,
			After:  before,
			ChangedProjectionCategories: []string{
				actorContextR3PCategoryMachineReasons,
				actorContextR3PCategorySourceRefs,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ChangedTupleComponents) != 0 {
		t.Fatalf("R3X inferred tuple component from projection category change: %+v", got)
	}
	if got.Status != actorContextR3XStatusTupleCompatibleFreshnessUnresolved {
		t.Fatalf("R3X treated projection change as freshness proof: %+v", got)
	}
}

func TestActorContextR3XStaticSemanticChangeWithSameTupleLeavesFreshnessAuthorityUnresolved(t *testing.T) {
	context := actorContextR3XBaseContext()
	got, err := actorContextR3XAnalyzeContextChange(
		actorContextR3XBaseObservation("allowed"),
		actorContextR3XContextDeltaProbe{
			Before:                context,
			After:                 context,
			StaticSemanticChanged: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != actorContextR3XStatusTupleCompatibleFreshnessUnresolved ||
		got.FreshnessAuthorityStatus != "unresolved" {
		t.Fatalf("R3X inferred reuse/invalidation from same-tuple static change: %+v", got)
	}
}

func TestActorContextR3XRuntimeOutcomeChangeDoesNotChangeContextIdentity(t *testing.T) {
	context := actorContextR3XBaseContext()
	got, err := actorContextR3XAnalyzeContextChange(
		actorContextR3XBaseObservation("allowed"),
		actorContextR3XContextDeltaProbe{
			Before:               context,
			After:                context,
			RuntimeOutcomeBefore: "allowed",
			RuntimeOutcomeAfter:  "denied",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != actorContextR3XStatusRuntimeOutcomeChanged ||
		!got.RuntimeOutcomeChanged ||
		len(got.ChangedTupleComponents) != 0 {
		t.Fatalf("R3X outcome change mutated context identity: %+v", got)
	}
}

func TestActorContextR3XShapeHasNoReuseOrAutomaticReevaluationShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3XContextImpactProbe{})
	for _, forbidden := range []string{
		"Reusable",
		"Valid",
		"CacheValid",
		"ShouldReevaluate",
		"AutoReauthorize",
		"ReevaluationRequired",
		"Authorized",
		"Allowed",
		"CanExecute",
		"ExpiresAt",
		"TTL",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3X context impact introduced forbidden shortcut %q", forbidden)
		}
	}
}

func actorContextR3XBaseContext() actorContextR3XAuthorizationContextProbe {
	return actorContextR3XAuthorizationContextProbe{
		RequirementStatus: actorContextR3BAuthorizationRequired,
		BusinessAddress:   "order-entry.order-handling/BP-06",
		SubjectID:         "subject-order-viewer",
		Action:            "execute",
		Resource:          "order_entry.submit_order",
	}
}

func actorContextR3XBaseObservation(outcome string) actorContextR3IEvaluationObservationProbe {
	context := actorContextR3XBaseContext()
	return actorContextR3IEvaluationObservationProbe{
		BusinessAddress: context.BusinessAddress,
		SubjectID:       context.SubjectID,
		Action:          context.Action,
		Resource:        context.Resource,
		Outcome:         outcome,
	}
}

func actorContextR3XAnalyzeContextChange(
	prior actorContextR3IEvaluationObservationProbe,
	delta actorContextR3XContextDeltaProbe,
) (actorContextR3XContextImpactProbe, error) {
	if err := actorContextR3XObservationMatchesContext(prior, delta.Before); err != nil {
		return actorContextR3XContextImpactProbe{}, err
	}

	changed := []string{}
	if delta.Before.BusinessAddress != delta.After.BusinessAddress {
		changed = append(changed, "business-address")
	}
	if delta.Before.SubjectID != delta.After.SubjectID {
		changed = append(changed, "subject-id")
	}
	if delta.Before.Action != delta.After.Action {
		changed = append(changed, "action")
	}
	if delta.Before.Resource != delta.After.Resource {
		changed = append(changed, "resource")
	}
	sort.Strings(changed)

	result := actorContextR3XContextImpactProbe{
		ChangedTupleComponents:          changed,
		RequirementApplicabilityChanged: delta.Before.RequirementStatus != delta.After.RequirementStatus,
		RuntimeOutcomeChanged:           delta.RuntimeOutcomeBefore != "" &&
			delta.RuntimeOutcomeAfter != "" &&
			delta.RuntimeOutcomeBefore != delta.RuntimeOutcomeAfter,
		FreshnessAuthorityStatus: "unresolved",
		AuthorityLimits: []string{
			"projection-category-change-does-not-identify-authorization-tuple-component",
			"tuple-match-does-not-authorize-reuse",
			"tuple-match-does-not-prove-evaluation-freshness",
			"context-mismatch-does-not-trigger-automatic-reauthorization",
			"requirement-transition-is-not-observation-tuple-mismatch",
			"runtime-outcome-is-not-context-identity",
		},
	}

	if len(changed) > 0 {
		result.Status = actorContextR3XStatusObservationContextMismatch
		result.ObservationTupleStatus = actorContextR3XObservationTupleMismatch
		return result, nil
	}
	if result.RequirementApplicabilityChanged {
		result.Status = actorContextR3XStatusRequirementTransitionUnresolved
		result.ObservationTupleStatus = actorContextR3XObservationTupleMatch
		return result, nil
	}
	if result.RuntimeOutcomeChanged {
		result.Status = actorContextR3XStatusRuntimeOutcomeChanged
		result.ObservationTupleStatus = actorContextR3XObservationTupleMatch
		return result, nil
	}
	if delta.StaticSemanticChanged || len(delta.ChangedProjectionCategories) > 0 {
		result.Status = actorContextR3XStatusTupleCompatibleFreshnessUnresolved
		result.ObservationTupleStatus = actorContextR3XObservationTupleMatch
		return result, nil
	}

	result.Status = actorContextR3XStatusNoContextIdentityChange
	result.ObservationTupleStatus = actorContextR3XObservationTupleMatch
	return result, nil
}

func actorContextR3XObservationMatchesContext(
	observation actorContextR3IEvaluationObservationProbe,
	context actorContextR3XAuthorizationContextProbe,
) error {
	if observation.BusinessAddress != context.BusinessAddress ||
		observation.SubjectID != context.SubjectID ||
		observation.Action != context.Action ||
		observation.Resource != context.Resource {
		return fmt.Errorf("R3X prior Observation does not match before-context tuple")
	}
	if observation.Outcome != "allowed" && observation.Outcome != "denied" {
		return fmt.Errorf("R3X prior Observation outcome is not evaluable")
	}
	return nil
}

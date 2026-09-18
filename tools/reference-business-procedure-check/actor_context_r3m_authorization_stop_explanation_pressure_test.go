package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

const (
	actorContextR3MStageAuthorizationActionAuthority = "authorization-action-authority"
	actorContextR3MDownstreamNotAttempted            = "not-attempted"
)

type actorContextR3MStopExplanationProbe struct {
	BusinessAddress         string
	StopStage               string
	RequirementStatus       string
	KnownResource           string
	KnownResourceSource     string
	UnresolvedAuthority     string
	BlockingReason          string
	AuthorizationEvaluation string
	AuthorizationInvocation string
	SourceRefs              []string
	AuthorityLimits         []string
}

func TestActorContextR3MCurrentBP06ExplainsUnresolvedActionAuthorityWithoutInventingDenial(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	requirement, err := deriveActorContextR3BAuthorizationRequirementProbe(f, "BP-06")
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := readApplicationPatternTargetOverlayProbe(
		filepath.Join(root, "reference", "business-system-design", "order-entry-protected-target-overlay.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	targets := actorContextR3BTargetsForStep(overlay, "BP-06")
	if len(targets) != 1 {
		t.Fatalf("test setup changed: BP-06 protected targets=%v", targets)
	}

	gate := actorContextR3JDeriveActionAuthorityGate(
		requirement.Status,
		targets[0],
		"reference-pressure-overlay:BP-06",
		true,
		true,
		true,
		nil,
	)
	got := actorContextR3MExplainAuthorizationStop(
		"order-entry.order-handling/BP-06",
		gate,
		[]string{
			"reference/business-system-design/order-entry-business-procedure.json",
			"reference/business-system-design/order-entry-protected-target-overlay.json",
			"plan/ACTOR_CONTEXT_R3J_AUTHORIZATION_ACTION_AUTHORITY_GATE_2026-09.md",
		},
	)

	if got.StopStage != actorContextR3MStageAuthorizationActionAuthority ||
		got.UnresolvedAuthority != "authorization-action-authority" ||
		got.BlockingReason != "authorization-action-authority-unresolved" {
		t.Fatalf("R3M did not explain the exact current stop boundary: %+v", got)
	}
	if got.RequirementStatus != actorContextR3BAuthorizationRequired ||
		got.KnownResource != "order_entry.submit_order" ||
		got.KnownResourceSource == "" {
		t.Fatalf("R3M lost known context while explaining the stop: %+v", got)
	}
	if got.AuthorizationEvaluation != actorContextR3MDownstreamNotAttempted ||
		got.AuthorizationInvocation != actorContextR3MDownstreamNotAttempted {
		t.Fatalf("R3M fabricated downstream evaluation/invocation: %+v", got)
	}
	if len(got.SourceRefs) != 3 {
		t.Fatalf("R3M lost drill-down refs: %+v", got.SourceRefs)
	}
}

func TestActorContextR3MStopExplanationHasNoInventedHandoffOwnerOrAuthorityShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3MStopExplanationProbe{})
	for _, forbidden := range []string{
		"HandoffOwner",
		"Owner",
		"Approver",
		"Team",
		"Authorized",
		"Allowed",
		"Approved",
		"CanExecute",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3M stop explanation introduced forbidden inferred/shortcut field %q", forbidden)
		}
	}
}

func TestActorContextR3MUnresolvedAuthorityIsNotDeniedOutcome(t *testing.T) {
	gate := actorContextR3JActionAuthorityGateProbe{
		RequirementStatus:     actorContextR3BAuthorizationRequired,
		Resource:              "order_entry.submit_order",
		ResourceSource:        "reference-pressure-overlay:BP-06",
		ActionAuthorityStatus: actorContextR3JActionAuthorityUnresolved,
		BlockingReasons:       []string{"authorization-action-authority-unresolved"},
	}
	got := actorContextR3MExplainAuthorizationStop(
		"order-entry.order-handling/BP-06",
		gate,
		[]string{"test-only:source"},
	)

	for _, forbiddenValue := range []string{"denied", "unauthorized", "not-allowed", "failed"} {
		if got.AuthorizationEvaluation == forbiddenValue ||
			got.AuthorizationInvocation == forbiddenValue ||
			got.BlockingReason == forbiddenValue {
			t.Fatalf("unresolved authority was rewritten as runtime outcome %q: %+v", forbiddenValue, got)
		}
	}
}

func actorContextR3MExplainAuthorizationStop(
	businessAddress string,
	gate actorContextR3JActionAuthorityGateProbe,
	sourceRefs []string,
) actorContextR3MStopExplanationProbe {
	result := actorContextR3MStopExplanationProbe{
		BusinessAddress:         businessAddress,
		RequirementStatus:       gate.RequirementStatus,
		KnownResource:           gate.Resource,
		KnownResourceSource:     gate.ResourceSource,
		AuthorizationEvaluation: actorContextR3MDownstreamNotAttempted,
		AuthorizationInvocation: actorContextR3MDownstreamNotAttempted,
		SourceRefs:              actorContextR0SortedUnique(sourceRefs),
		AuthorityLimits: []string{
			"stop-explanation-is-not-authority",
			"unresolved-authority-is-not-denial",
			"blocking-reason-does-not-identify-handoff-owner",
			"not-attempted-is-not-failed",
		},
	}

	if gate.RequirementStatus == actorContextR3BAuthorizationRequired &&
		gate.ActionAuthorityStatus == actorContextR3JActionAuthorityUnresolved {
		result.StopStage = actorContextR3MStageAuthorizationActionAuthority
		result.UnresolvedAuthority = "authorization-action-authority"
		result.BlockingReason = "authorization-action-authority-unresolved"
	}
	return result
}

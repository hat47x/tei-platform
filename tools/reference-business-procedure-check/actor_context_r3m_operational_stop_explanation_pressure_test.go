package main

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3MHandoffRecipientUnresolved = "unresolved"
)

type actorContextR3MAuthorityFactProbe struct {
	Kind      string
	Value     string
	SourceRef string
}

type actorContextR3MOperationalStopExplanationProbe struct {
	BusinessAddress        string
	StopStage              string
	BlockingReasons        []string
	KnownAuthorityFacts    []actorContextR3MAuthorityFactProbe
	ResolutionNeedClasses  []string
	SourceRefs             []string
	DownstreamNotAttempted []string
	ResolutionRequested    bool
	HandoffRecipientStatus string
	AuthorityLimits        []string
}

func TestActorContextR3MExplainsCurrentBP06ActionAuthorityStopWithoutInventingValueOrOwner(t *testing.T) {
	root := repoPath()
	fixturePath := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	requirement, err := deriveActorContextR3BAuthorizationRequirementProbe(f, "BP-06")
	if err != nil {
		t.Fatal(err)
	}
	if requirement.Status != actorContextR3BAuthorizationRequired {
		t.Fatalf("test setup changed: BP-06 requirement=%+v", requirement)
	}

	overlay, err := readApplicationPatternTargetOverlayProbe(
		filepath.Join(root, "reference", "business-system-design", "order-entry-protected-target-overlay.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	targets := actorContextR3BTargetsForStep(overlay, "BP-06")
	if !reflect.DeepEqual(targets, []string{"order_entry.submit_order"}) {
		t.Fatalf("test setup changed: BP-06 target=%v", targets)
	}

	actionGate := actorContextR3JDeriveActionAuthorityGate(
		requirement.Status,
		targets[0],
		"reference-pressure-overlay:BP-06",
		true,
		true,
		true,
		nil,
	)
	if !reflect.DeepEqual(actionGate.BlockingReasons, []string{"authorization-action-authority-unresolved"}) {
		t.Fatalf("test setup changed: BP-06 action gate=%+v", actionGate)
	}

	got := actorContextR3MBuildStopExplanation(
		"order-entry.order-handling/BP-06",
		"authorization-action-authority",
		actionGate.BlockingReasons,
		[]actorContextR3MAuthorityFactProbe{
			{
				Kind:      "authorization-requirement",
				Value:     actorContextR3BAuthorizationRequired,
				SourceRef: requirement.RequirementSource,
			},
			{
				Kind:      "protected-resource",
				Value:     targets[0],
				SourceRef: "reference-pressure-overlay:BP-06",
			},
		},
		[]string{
			"reference/business-system-design/order-entry-business-procedure.json",
			"reference/business-system-design/order-entry-protected-target-overlay.json",
			"tools/reference-business-procedure-check/actor_context_r3j_authorization_action_authority_gate_pressure_test.go",
		},
	)

	if got.StopStage != "authorization-action-authority" {
		t.Fatalf("R3M changed the stop stage: %+v", got)
	}
	if !reflect.DeepEqual(got.BlockingReasons, []string{"authorization-action-authority-unresolved"}) {
		t.Fatalf("R3M blurred the exact blocker vocabulary: %+v", got.BlockingReasons)
	}
	if !reflect.DeepEqual(got.ResolutionNeedClasses, []string{"authorization-action-authority"}) {
		t.Fatalf("R3M did not expose the resolution need class precisely: %+v", got.ResolutionNeedClasses)
	}
	if got.HandoffRecipientStatus != actorContextR3MHandoffRecipientUnresolved || !got.ResolutionRequested {
		t.Fatalf("R3M should request resolution without inventing a recipient: %+v", got)
	}
	if !reflect.DeepEqual(got.DownstreamNotAttempted, []string{
		"authorization-invocation",
		"business-completion",
		"protected-operation-invocation",
	}) {
		t.Fatalf("R3M lost downstream not-attempted state: %+v", got.DownstreamNotAttempted)
	}

	for _, fact := range got.KnownAuthorityFacts {
		if fact.Value == "execute" {
			t.Fatalf("R3M promoted the likely Legacy/pattern/wording hint into missing action authority: %+v", fact)
		}
	}
}

func TestActorContextR3MPreservesKnownFactProvenance(t *testing.T) {
	got := actorContextR3MBuildStopExplanation(
		"order-entry.order-handling/BP-06",
		"authorization-action-authority",
		[]string{"authorization-action-authority-unresolved"},
		[]actorContextR3MAuthorityFactProbe{
			{
				Kind:      "authorization-requirement",
				Value:     "required",
				SourceRef: "business-procedure-security-boundary",
			},
			{
				Kind:      "protected-resource",
				Value:     "order_entry.submit_order",
				SourceRef: "reference-pressure-overlay:BP-06",
			},
		},
		nil,
	)

	if len(got.KnownAuthorityFacts) != 2 {
		t.Fatalf("R3M lost known authority facts: %+v", got.KnownAuthorityFacts)
	}
	for _, fact := range got.KnownAuthorityFacts {
		if fact.Kind == "" || fact.Value == "" || fact.SourceRef == "" {
			t.Fatalf("R3M emitted an authority fact without provenance: %+v", fact)
		}
	}
}

func TestActorContextR3MEarliestBlockerDoesNotBecomeDeferredAuthorizationBlocker(t *testing.T) {
	requirementStop := actorContextR3MBuildStopExplanation(
		"order-entry.order-handling/BP-02",
		"authorization-requirement-authority",
		[]string{"authorization-requirement-unresolved"},
		nil,
		nil,
	)
	if !reflect.DeepEqual(requirementStop.BlockingReasons, []string{"authorization-requirement-unresolved"}) {
		t.Fatalf("R3M changed the earliest requirement blocker: %+v", requirementStop.BlockingReasons)
	}
	if actorContextR0Contains(requirementStop.BlockingReasons, "authorization-action-authority-unresolved") ||
		actorContextR0Contains(requirementStop.BlockingReasons, "authorization-not-evaluated") {
		t.Fatalf("R3M promoted downstream gaps into current blockers: %+v", requirementStop.BlockingReasons)
	}
	if !reflect.DeepEqual(requirementStop.ResolutionNeedClasses, []string{"authorization-requirement-authority"}) {
		t.Fatalf("R3M mapped the requirement stop to the wrong resolution need class: %+v", requirementStop)
	}
}

func TestActorContextR3MExplanationShapeContainsNoAuthorityOrHandoffShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3MOperationalStopExplanationProbe{})
	for _, forbidden := range []string{
		"HandoffOwner",
		"SuggestedAction",
		"RecommendedAction",
		"Approved",
		"Safe",
		"CanExecute",
		"Authorized",
		"Allowed",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3M explanation introduced forbidden shortcut field %q", forbidden)
		}
	}
}

func actorContextR3MBuildStopExplanation(
	businessAddress string,
	stopStage string,
	blockingReasons []string,
	knownFacts []actorContextR3MAuthorityFactProbe,
	sourceRefs []string,
) actorContextR3MOperationalStopExplanationProbe {
	result := actorContextR3MOperationalStopExplanationProbe{
		BusinessAddress:        businessAddress,
		StopStage:              stopStage,
		BlockingReasons:        actorContextR0SortedUnique(append([]string(nil), blockingReasons...)),
		KnownAuthorityFacts:    append([]actorContextR3MAuthorityFactProbe(nil), knownFacts...),
		SourceRefs:             actorContextR0SortedUnique(append([]string(nil), sourceRefs...)),
		ResolutionRequested:    len(blockingReasons) > 0,
		HandoffRecipientStatus: actorContextR3MHandoffRecipientUnresolved,
		AuthorityLimits: []string{
			"explanation-is-derived-not-authoritative",
			"resolution-need-class-does-not-imply-missing-value-or-owner",
			"stop-reason-does-not-imply-handoff-owner",
			"business-actor-label-does-not-imply-handoff-owner",
			"corroborating-policy-does-not-fill-missing-authority",
			"downstream-not-attempted-does-not-become-current-blocker",
		},
	}

	needs := []string{}
	downstream := []string{}
	for _, reason := range result.BlockingReasons {
		switch reason {
		case "authorization-requirement-unresolved":
			needs = append(needs, "authorization-requirement-authority")
			downstream = append(downstream,
				"authorization-invocation",
				"protected-operation-invocation",
				"business-completion",
			)
		case "actor-principal-unresolved":
			needs = append(needs, "runtime-principal")
			downstream = append(downstream,
				"authorization-invocation",
				"protected-operation-invocation",
				"business-completion",
			)
		case "authorization-resource-unresolved", "authorization-resource-authority-unresolved":
			needs = append(needs, "authorization-resource-authority")
			downstream = append(downstream,
				"authorization-invocation",
				"protected-operation-invocation",
				"business-completion",
			)
		case "authorization-action-unresolved", "authorization-action-authority-unresolved":
			needs = append(needs, "authorization-action-authority")
			downstream = append(downstream,
				"authorization-invocation",
				"protected-operation-invocation",
				"business-completion",
			)
		case "authorization-not-evaluated":
			needs = append(needs, "authorization-evaluation")
			downstream = append(downstream,
				"protected-operation-invocation",
				"business-completion",
			)
		case "authorization-evaluation-observation-mismatch":
			needs = append(needs, "authorization-evaluation-observation")
			downstream = append(downstream,
				"protected-operation-invocation",
				"business-completion",
			)
		case "authorization-evaluation-freshness-authority-unresolved":
			needs = append(needs, "authorization-evaluation-freshness-authority")
			downstream = append(downstream,
				"protected-operation-invocation",
				"business-completion",
			)
		case "authorization-current-context-evaluation-unresolved":
			needs = append(needs, "authorization-current-context-evaluation")
			downstream = append(downstream,
				"protected-operation-invocation",
				"business-completion",
			)
		}
	}
	result.ResolutionNeedClasses = actorContextR0SortedUnique(needs)
	result.DownstreamNotAttempted = actorContextR0SortedUnique(downstream)

	sort.Slice(result.KnownAuthorityFacts, func(i, j int) bool {
		if result.KnownAuthorityFacts[i].Kind == result.KnownAuthorityFacts[j].Kind {
			return result.KnownAuthorityFacts[i].SourceRef < result.KnownAuthorityFacts[j].SourceRef
		}
		return result.KnownAuthorityFacts[i].Kind < result.KnownAuthorityFacts[j].Kind
	})
	return result
}

package main

import (
	"reflect"
	"sort"
	"testing"
)

type actorContextR3UAggregatedImpactProbe struct {
	Audience          string
	Purpose           string
	Reasons           []actorContextR3TReevaluationReasonProbe
	UnselectedChanges []actorContextR3TObservedChangeProbe
	AuthorityLimits   []string
}

func TestActorContextR3UOneCategoryPreservesMultipleIndependentReasons(t *testing.T) {
	input := []actorContextR3TImpactExplanationProbe{
		{
			Reasons: []actorContextR3TReevaluationReasonProbe{
				{
					Category:         actorContextR3PCategoryMachineReasons,
					ChangeClass:      actorContextR3TChangeReferenceInput,
					DependencyKey:    "reference:business-procedure-reference:order-entry-business-procedure.json",
					DependencyReason: actorContextR3SReasonAuthorizationRequirementSource,
				},
				{
					Category:         actorContextR3PCategoryMachineReasons,
					ChangeClass:      actorContextR3TChangeReferenceInput,
					DependencyKey:    "reference:protected-target-overlay:order-entry-protected-target-overlay.json",
					DependencyReason: actorContextR3SReasonProtectedResourceSource,
				},
			},
		},
	}

	got := actorContextR3UAggregateImpacts(
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
		input,
	)
	if len(got.Reasons) != 2 {
		t.Fatalf("R3U collapsed independent machine-reason causes: %+v", got.Reasons)
	}
	for _, reason := range got.Reasons {
		if reason.Category != actorContextR3PCategoryMachineReasons {
			t.Fatalf("R3U moved reason to a different category: %+v", reason)
		}
	}
}

func TestActorContextR3UPreservesIndependentFreshnessPlanesTogether(t *testing.T) {
	input := []actorContextR3TImpactExplanationProbe{
		{
			Reasons: []actorContextR3TReevaluationReasonProbe{
				{
					Category:         actorContextR3PCategoryKnownAuthorityFacts,
					ChangeClass:      actorContextR3TChangeSemanticSource,
					DependencyKey:    "semantic:application-pattern:order_entry.submit_order",
					DependencyReason: actorContextR3TReasonSemanticSubjectMeaning,
				},
				{
					Category:         actorContextR3PCategorySourceRefs,
					ChangeClass:      actorContextR3TChangeLocatorRefresh,
					DependencyKey:    "semantic:application-pattern:order_entry.submit_order",
					DependencyReason: actorContextR3SReasonSourceIndexDefinitionLocator,
				},
				{
					Category:         actorContextR3PCategoryRuntimeOutcome,
					ChangeClass:      actorContextR3TChangeRuntimeObservation,
					DependencyKey:    "runtime:BP-06:subject:execute:order_entry.submit_order:allowed",
					DependencyReason: actorContextR3SReasonExactRuntimeObservation,
				},
				{
					Category:         actorContextR3PCategoryRuntimeOutcome,
					ChangeClass:      actorContextR3TChangeProfile,
					DependencyKey:    "profile:human:orientation",
					DependencyReason: actorContextR3TReasonProfileSelection,
				},
			},
		},
	}

	got := actorContextR3UAggregateImpacts("test-audience", "test-purpose", input)
	for _, class := range []string{
		actorContextR3TChangeSemanticSource,
		actorContextR3TChangeLocatorRefresh,
		actorContextR3TChangeRuntimeObservation,
		actorContextR3TChangeProfile,
	} {
		if !actorContextR3UHasChangeClass(got.Reasons, class) {
			t.Fatalf("R3U lost independent change class %q: %+v", class, got.Reasons)
		}
	}
}

func TestActorContextR3UDeduplicatesOnlyExactReasonIdentity(t *testing.T) {
	exact := actorContextR3TReevaluationReasonProbe{
		Category:         actorContextR3PCategoryMachineReasons,
		ChangeClass:      actorContextR3TChangeReferenceInput,
		DependencyKey:    "reference:business-procedure-reference:order-entry-business-procedure.json",
		DependencyReason: actorContextR3SReasonAuthorizationRequirementSource,
	}
	differentReason := exact
	differentReason.DependencyReason = actorContextR3SReasonBusinessAddressSource
	differentCategory := exact
	differentCategory.Category = actorContextR3PCategoryStopIdentity

	got := actorContextR3UAggregateImpacts(
		"human",
		"orientation",
		[]actorContextR3TImpactExplanationProbe{
			{Reasons: []actorContextR3TReevaluationReasonProbe{exact, exact}},
			{Reasons: []actorContextR3TReevaluationReasonProbe{differentReason, differentCategory}},
		},
	)

	if len(got.Reasons) != 3 {
		t.Fatalf("R3U over/under-deduplicated reasons: %+v", got.Reasons)
	}
}

func TestActorContextR3UInputOrderDoesNotCreatePrecedence(t *testing.T) {
	a := actorContextR3TImpactExplanationProbe{
		Reasons: []actorContextR3TReevaluationReasonProbe{
			{
				Category:         actorContextR3PCategoryMachineReasons,
				ChangeClass:      actorContextR3TChangeReferenceInput,
				DependencyKey:    "reference:a",
				DependencyReason: actorContextR3SReasonAuthorizationRequirementSource,
			},
		},
	}
	b := actorContextR3TImpactExplanationProbe{
		Reasons: []actorContextR3TReevaluationReasonProbe{
			{
				Category:         actorContextR3PCategorySourceRefs,
				ChangeClass:      actorContextR3TChangeLocatorRefresh,
				DependencyKey:    "semantic:b",
				DependencyReason: actorContextR3SReasonSourceIndexDefinitionLocator,
			},
		},
	}

	first := actorContextR3UAggregateImpacts("human", "orientation", []actorContextR3TImpactExplanationProbe{a, b})
	second := actorContextR3UAggregateImpacts("human", "orientation", []actorContextR3TImpactExplanationProbe{b, a})

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3U input order changed normalized aggregation: first=%+v second=%+v", first, second)
	}
}

func TestActorContextR3USelectedReasonRemovesExactUnselectedDuplicate(t *testing.T) {
	change := actorContextR3TObservedChangeProbe{
		ChangeClass:   actorContextR3TChangeReferenceInput,
		DependencyKey: "reference:a",
	}
	reason := actorContextR3TReevaluationReasonProbe{
		Category:         actorContextR3PCategoryMachineReasons,
		ChangeClass:      change.ChangeClass,
		DependencyKey:    change.DependencyKey,
		DependencyReason: actorContextR3SReasonAuthorizationRequirementSource,
	}

	got := actorContextR3UAggregateImpacts(
		"human",
		"orientation",
		[]actorContextR3TImpactExplanationProbe{
			{UnselectedChanges: []actorContextR3TObservedChangeProbe{change}},
			{Reasons: []actorContextR3TReevaluationReasonProbe{reason}},
		},
	)
	if len(got.Reasons) != 1 || len(got.UnselectedChanges) != 0 {
		t.Fatalf("R3U double-counted selected change as unselected: %+v", got)
	}
}

func TestActorContextR3UAggregationHasNoWinnerOrPriorityShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3UAggregatedImpactProbe{})
	for _, forbidden := range []string{
		"PrimaryReason",
		"Winner",
		"Priority",
		"Severity",
		"FirstCause",
		"RootCause",
		"AutoRebuild",
		"NextAction",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3U aggregation introduced forbidden ranking/action shortcut %q", forbidden)
		}
	}
}

func actorContextR3UAggregateImpacts(
	audience string,
	purpose string,
	inputs []actorContextR3TImpactExplanationProbe,
) actorContextR3UAggregatedImpactProbe {
	var reasons []actorContextR3TReevaluationReasonProbe
	var unselected []actorContextR3TObservedChangeProbe
	for _, input := range inputs {
		reasons = append(reasons, input.Reasons...)
		unselected = append(unselected, input.UnselectedChanges...)
	}

	reasons = actorContextR3TUniqueReasons(reasons)
	unselected = actorContextR3TUniqueChanges(unselected)

	selectedChangeKeys := map[string]bool{}
	for _, reason := range reasons {
		selectedChangeKeys[reason.ChangeClass+"|"+reason.DependencyKey] = true
	}
	filtered := unselected[:0]
	for _, change := range unselected {
		if selectedChangeKeys[change.ChangeClass+"|"+change.DependencyKey] {
			continue
		}
		filtered = append(filtered, change)
	}
	unselected = filtered

	sort.Slice(unselected, func(i, j int) bool {
		if unselected[i].ChangeClass != unselected[j].ChangeClass {
			return unselected[i].ChangeClass < unselected[j].ChangeClass
		}
		if unselected[i].DependencyKey != unselected[j].DependencyKey {
			return unselected[i].DependencyKey < unselected[j].DependencyKey
		}
		return unselected[i].CategoryHint < unselected[j].CategoryHint
	})

	return actorContextR3UAggregatedImpactProbe{
		Audience:          audience,
		Purpose:           purpose,
		Reasons:           reasons,
		UnselectedChanges: unselected,
		AuthorityLimits: []string{
			"stable-order-is-not-priority",
			"multi-reason-aggregation-does-not-select-winner",
			"reason-union-does-not-trigger-rebuild",
			"unselected-change-does-not-mean-globally-irrelevant",
		},
	}
}

func actorContextR3UHasChangeClass(
	values []actorContextR3TReevaluationReasonProbe,
	changeClass string,
) bool {
	for _, value := range values {
		if value.ChangeClass == changeClass {
			return true
		}
	}
	return false
}

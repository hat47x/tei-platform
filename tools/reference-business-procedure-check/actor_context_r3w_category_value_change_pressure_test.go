package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3WStatusChanged   = "changed-after-reevaluation"
	actorContextR3WStatusUnchanged = "unchanged-after-reevaluation"
)

type actorContextR3WCategoryResultProbe struct {
	Category          string
	Status            string
	BeforeFingerprint string
	AfterFingerprint  string
	Reasons           []actorContextR3TReevaluationReasonProbe
}

type actorContextR3WResultProbe struct {
	Categories        []actorContextR3WCategoryResultProbe
	UnselectedChanges []actorContextR3TObservedChangeProbe
	AuthorityLimits   []string
}

func TestActorContextR3WReasonCanReevaluateToUnchangedCategory(t *testing.T) {
	core := actorContextR3QCurrentBP06Core(t)
	view := actorContextR3WHumanView(t, core)
	impact := actorContextR3UAggregateImpacts(
		view.Audience,
		view.Purpose,
		[]actorContextR3TImpactExplanationProbe{{Reasons: []actorContextR3TReevaluationReasonProbe{{
			Category:         actorContextR3PCategoryMachineReasons,
			ChangeClass:      actorContextR3TChangeReferenceInput,
			DependencyKey:    "reference:business-procedure-reference:test",
			DependencyReason: actorContextR3SReasonAuthorizationRequirementSource,
		}}}},
	)

	got, err := actorContextR3WEvaluateSameProfile(core, actorContextR3QCloneCore(core), view, impact)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Categories) != 1 ||
		got.Categories[0].Status != actorContextR3WStatusUnchanged ||
		got.Categories[0].Category != actorContextR3PCategoryMachineReasons {
		t.Fatalf("R3W equated reason with changed value: %+v", got)
	}
}

func TestActorContextR3WChangedCategoryPreservesMultipleReasons(t *testing.T) {
	before := actorContextR3QCurrentBP06Core(t)
	after := actorContextR3QCloneCore(before)
	after.MachineReasons = append(after.MachineReasons, "test-only:additional-machine-reason")
	view := actorContextR3WHumanView(t, before)

	impact := actorContextR3UAggregateImpacts(
		view.Audience,
		view.Purpose,
		[]actorContextR3TImpactExplanationProbe{{Reasons: []actorContextR3TReevaluationReasonProbe{
			{
				Category:         actorContextR3PCategoryMachineReasons,
				ChangeClass:      actorContextR3TChangeReferenceInput,
				DependencyKey:    "reference:business-procedure-reference:test",
				DependencyReason: actorContextR3SReasonAuthorizationRequirementSource,
			},
			{
				Category:         actorContextR3PCategoryMachineReasons,
				ChangeClass:      actorContextR3TChangeReferenceInput,
				DependencyKey:    "reference:protected-target-overlay:test",
				DependencyReason: actorContextR3SReasonProtectedResourceSource,
			},
		}}},
	)

	got, err := actorContextR3WEvaluateSameProfile(before, after, view, impact)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Categories) != 1 {
		t.Fatalf("R3W duplicated category for multiple reasons: %+v", got.Categories)
	}
	category := got.Categories[0]
	if category.Status != actorContextR3WStatusChanged || len(category.Reasons) != 2 {
		t.Fatalf("R3W lost changed status or independent reasons: %+v", category)
	}
	if category.BeforeFingerprint == category.AfterFingerprint {
		t.Fatalf("changed category kept identical fingerprint: %+v", category)
	}
}

func TestActorContextR3WLocatorReasonNeedsActualSourceRefChange(t *testing.T) {
	before := actorContextR3QCurrentBP06Core(t)
	view := actorContextR3WHumanView(t, before)
	impact := actorContextR3UAggregateImpacts(
		view.Audience,
		view.Purpose,
		[]actorContextR3TImpactExplanationProbe{{Reasons: []actorContextR3TReevaluationReasonProbe{{
			Category:         actorContextR3PCategorySourceRefs,
			ChangeClass:      actorContextR3TChangeLocatorRefresh,
			DependencyKey:    "semantic:application-pattern:order_entry.submit_order",
			DependencyReason: actorContextR3SReasonSourceIndexDefinitionLocator,
		}}}},
	)

	unchanged, err := actorContextR3WEvaluateSameProfile(before, actorContextR3QCloneCore(before), view, impact)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Categories[0].Status != actorContextR3WStatusUnchanged {
		t.Fatalf("locator reason alone changed source-refs: %+v", unchanged)
	}

	after := actorContextR3QCloneCore(before)
	after.SourceRefs = actorContextR0SortedUnique(append(
		after.SourceRefs,
		"workspace:applications/order-entry/patterns/moved-submit-order.tei",
	))
	changed, err := actorContextR3WEvaluateSameProfile(before, after, view, impact)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Categories[0].Status != actorContextR3WStatusChanged {
		t.Fatalf("actual source-ref change was not detected: %+v", changed)
	}
}

func TestActorContextR3WRuntimeReasonNeedsActualRuntimeOutcomeChange(t *testing.T) {
	before := actorContextR3QCurrentBP06Core(t)
	view := actorContextR3WHumanView(t, before)
	impact := actorContextR3UAggregateImpacts(
		view.Audience,
		view.Purpose,
		[]actorContextR3TImpactExplanationProbe{{Reasons: []actorContextR3TReevaluationReasonProbe{{
			Category:         actorContextR3PCategoryRuntimeOutcome,
			ChangeClass:      actorContextR3TChangeRuntimeObservation,
			DependencyKey:    "runtime:BP-06:test-observation",
			DependencyReason: actorContextR3SReasonExactRuntimeObservation,
		}}}},
	)

	after := actorContextR3QCloneCore(before)
	after.RuntimeOutcome = "denied"
	got, err := actorContextR3WEvaluateSameProfile(before, after, view, impact)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Categories) != 1 ||
		got.Categories[0].Category != actorContextR3PCategoryRuntimeOutcome ||
		got.Categories[0].Status != actorContextR3WStatusChanged {
		t.Fatalf("R3W lost actual runtime outcome change: %+v", got)
	}
}

func TestActorContextR3WSelectedCategoryChangeWithoutReasonFailsClosed(t *testing.T) {
	before := actorContextR3QCurrentBP06Core(t)
	after := actorContextR3QCloneCore(before)
	after.ResolutionNeedClasses = []string{"authorization-resource-authority"}
	view := actorContextR3WHumanView(t, before)

	if _, err := actorContextR3WEvaluateSameProfile(
		before,
		after,
		view,
		actorContextR3UAggregatedImpactProbe{},
	); err == nil {
		t.Fatal("R3W accepted unexplained selected category change")
	}
}

func TestActorContextR3WOmittedCategoryChangeDoesNotBecomeSelectedResult(t *testing.T) {
	before := actorContextR3QCurrentBP06Core(t)
	after := actorContextR3QCloneCore(before)
	after.KnownAuthorityFacts = append(
		after.KnownAuthorityFacts,
		actorContextR3MAuthorityFactProbe{
			Kind:      "test-only-omitted-fact",
			Value:     "changed",
			SourceRef: "test-only:omitted",
		},
	)
	view := actorContextR3WHumanView(t, before)

	got, err := actorContextR3WEvaluateSameProfile(
		before,
		after,
		view,
		actorContextR3UAggregatedImpactProbe{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Categories) != 0 {
		t.Fatalf("R3W promoted omitted category change into selected result: %+v", got.Categories)
	}
}

func TestActorContextR3WProfileChangeRequiresSelectionRebuildFirst(t *testing.T) {
	core := actorContextR3QCurrentBP06Core(t)
	view := actorContextR3WHumanView(t, core)
	impact := actorContextR3UAggregateImpacts(
		view.Audience,
		view.Purpose,
		[]actorContextR3TImpactExplanationProbe{{Reasons: []actorContextR3TReevaluationReasonProbe{{
			Category:         actorContextR3PCategoryRuntimeOutcome,
			ChangeClass:      actorContextR3TChangeProfile,
			DependencyKey:    "profile:human:orientation",
			DependencyReason: actorContextR3TReasonProfileSelection,
		}}}},
	)
	if _, err := actorContextR3WEvaluateSameProfile(core, core, view, impact); err == nil {
		t.Fatal("R3W compared category values under changed profile selection")
	}
}

func TestActorContextR3WResultHasNoGlobalStaleOrAutomaticActionShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3WResultProbe{})
	for _, forbidden := range []string{
		"Stale",
		"Fresh",
		"RefreshAll",
		"RebuildAll",
		"AutoRebuild",
		"AutoReevaluate",
		"RuntimeReevaluationRequired",
		"Authorized",
		"CanExecute",
		"NextAction",
		"Winner",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3W result introduced forbidden shortcut field %q", forbidden)
		}
	}
}

func actorContextR3WHumanView(
	t *testing.T,
	core actorContextR3OStopProjectionCoreProbe,
) actorContextR3PBoundedStopViewProbe {
	t.Helper()
	view, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	return view
}

func actorContextR3WEvaluateSameProfile(
	before actorContextR3OStopProjectionCoreProbe,
	after actorContextR3OStopProjectionCoreProbe,
	view actorContextR3PBoundedStopViewProbe,
	impact actorContextR3UAggregatedImpactProbe,
) (actorContextR3WResultProbe, error) {
	if err := actorContextR3PValidateCategoryPartition(
		view.SelectedCategories,
		view.OmittedCategories,
	); err != nil {
		return actorContextR3WResultProbe{}, err
	}

	reasonsByCategory := map[string][]actorContextR3TReevaluationReasonProbe{}
	for _, reason := range impact.Reasons {
		if reason.ChangeClass == actorContextR3TChangeProfile {
			return actorContextR3WResultProbe{}, fmt.Errorf(
				"R3W profile change requires bounded selection rebuild",
			)
		}
		if !actorContextR0Contains(view.SelectedCategories, reason.Category) {
			return actorContextR3WResultProbe{}, fmt.Errorf(
				"R3W reason category %q is not selected by %s/%s",
				reason.Category,
				view.Audience,
				view.Purpose,
			)
		}
		reasonsByCategory[reason.Category] = append(
			reasonsByCategory[reason.Category],
			reason,
		)
	}

	results := []actorContextR3WCategoryResultProbe{}
	for _, category := range actorContextR0SortedUnique(view.SelectedCategories) {
		beforePayload, err := actorContextR3QCategoryPayload(before, category)
		if err != nil {
			return actorContextR3WResultProbe{}, err
		}
		afterPayload, err := actorContextR3QCategoryPayload(after, category)
		if err != nil {
			return actorContextR3WResultProbe{}, err
		}
		beforeFingerprint, err := actorContextR3QHash(beforePayload)
		if err != nil {
			return actorContextR3WResultProbe{}, err
		}
		afterFingerprint, err := actorContextR3QHash(afterPayload)
		if err != nil {
			return actorContextR3WResultProbe{}, err
		}
		reasons := actorContextR3TUniqueReasons(reasonsByCategory[category])

		if beforeFingerprint != afterFingerprint && len(reasons) == 0 {
			return actorContextR3WResultProbe{}, fmt.Errorf(
				"R3W unexplained selected category change %q",
				category,
			)
		}
		if len(reasons) == 0 {
			continue
		}

		status := actorContextR3WStatusUnchanged
		if beforeFingerprint != afterFingerprint {
			status = actorContextR3WStatusChanged
		}
		results = append(results, actorContextR3WCategoryResultProbe{
			Category:          category,
			Status:            status,
			BeforeFingerprint: beforeFingerprint,
			AfterFingerprint:  afterFingerprint,
			Reasons:           reasons,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Category < results[j].Category
	})

	return actorContextR3WResultProbe{
		Categories:        results,
		UnselectedChanges: actorContextR3TUniqueChanges(impact.UnselectedChanges),
		AuthorityLimits: []string{
			"re-evaluation-reason-does-not-prove-value-change",
			"category-value-change-does-not-trigger-automatic-rebuild",
			"unexplained-selected-category-change-fails-closed",
			"runtime-reevaluation-authority-remains-separate",
		},
	}, nil
}

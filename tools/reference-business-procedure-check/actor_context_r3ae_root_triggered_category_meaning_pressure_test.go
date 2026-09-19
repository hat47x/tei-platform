package main

import (
	"reflect"
	"sort"
	"testing"
)

type actorContextR3AEResultProbe struct {
	Categories          []actorContextR3WCategoryResultProbe
	UnselectedChanges   []actorContextR3TObservedChangeProbe
	ResidualRootChanges []actorContextR3ACRootChangeProbe
	AuthorityLimits     []string
}

func TestActorContextR3AERequirementRootReasonCanReevaluateFreshnessStopToUnchanged(t *testing.T) {
	drilldown := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)
	afterRoots := actorContextR3ACCloneRoots(drilldown.Roots)
	idx := actorContextR3ADRootIndexByClaim(afterRoots, "authorization-requirement-source")
	if idx < 0 {
		t.Fatal("R3AE setup lost requirement root")
	}
	afterRoots[idx].Fingerprint += ":changed"
	assessment := actorContextR3ACClassifyRootChanges(drilldown.Roots, afterRoots)

	before := actorContextR3ABFreshnessCore(t)
	after := actorContextR3QCloneCore(before)

	humanView := actorContextR3ADFreshnessView(t, actorContextR3PAudienceHuman)
	humanBridge := actorContextR3ADExplainRootChanges(humanView, drilldown.Roots, assessment)
	human, err := actorContextR3AEEvaluate(before, after, humanView, humanBridge)
	if err != nil {
		t.Fatal(err)
	}
	humanMachine := actorContextR3AECategory(human.Categories, actorContextR3PCategoryMachineReasons)
	if humanMachine == nil || humanMachine.Status != actorContextR3WStatusUnchanged {
		t.Fatalf("R3AE equated requirement root change with changed freshness meaning: %+v", human)
	}

	agentView := actorContextR3ADFreshnessView(t, actorContextR3PAudienceAgent)
	agentBridge := actorContextR3ADExplainRootChanges(agentView, drilldown.Roots, assessment)
	agent, err := actorContextR3AEEvaluate(before, after, agentView, agentBridge)
	if err != nil {
		t.Fatal(err)
	}
	agentMachine := actorContextR3AECategory(agent.Categories, actorContextR3PCategoryMachineReasons)
	if agentMachine == nil || !reflect.DeepEqual(*humanMachine, *agentMachine) {
		t.Fatalf("Human / Agent shared machine-reason result diverged: human=%+v agent=%+v", humanMachine, agentMachine)
	}
	agentFacts := actorContextR3AECategory(agent.Categories, actorContextR3PCategoryKnownAuthorityFacts)
	if agentFacts == nil || agentFacts.Status != actorContextR3WStatusUnchanged {
		t.Fatalf("Agent selected known-authority-facts did not re-evaluate independently: %+v", agent)
	}
	if actorContextR3AECategory(human.Categories, actorContextR3PCategoryKnownAuthorityFacts) != nil {
		t.Fatalf("Human bounded view received omitted known-authority-facts result: %+v", human)
	}
}

func TestActorContextR3AEPriorObservationRevisionKeepsFreshnessReasonUnchangedWithoutRuntimeOutcome(t *testing.T) {
	drilldown := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)
	afterRoots := actorContextR3ACCloneRoots(drilldown.Roots)
	idx := actorContextR3ACRootIndexByRole(afterRoots, actorContextR3ABRoleRuntimeObservation)
	if idx < 0 {
		t.Fatal("R3AE setup lost prior Observation root")
	}
	afterRoots[idx].Fingerprint += ":new-observation"
	afterRoots[idx].ClaimKind = "prior-authorization-observation:denied"
	assessment := actorContextR3ACClassifyRootChanges(drilldown.Roots, afterRoots)

	before := actorContextR3ABFreshnessCore(t)
	after := actorContextR3QCloneCore(before)
	view := actorContextR3ADFreshnessView(t, actorContextR3PAudienceAgent)
	bridge := actorContextR3ADExplainRootChanges(view, drilldown.Roots, assessment)

	got, err := actorContextR3AEEvaluate(before, after, view, bridge)
	if err != nil {
		t.Fatal(err)
	}
	machine := actorContextR3AECategory(got.Categories, actorContextR3PCategoryMachineReasons)
	if machine == nil || machine.Status != actorContextR3WStatusUnchanged {
		t.Fatalf("prior Observation revision changed freshness stop without changed derived meaning: %+v", got)
	}
	if actorContextR3AECategory(got.Categories, actorContextR3PCategoryRuntimeOutcome) != nil {
		t.Fatalf("prior Observation revision became current runtime-outcome result: %+v", got)
	}
	if actorContextR3AECategory(got.Categories, actorContextR3PCategoryKnownAuthorityFacts) != nil {
		t.Fatalf("runtime Observation revision became authority-fact result: %+v", got)
	}
}

func TestActorContextR3AEOneTestContextRootCanYieldMixedChangedAndUnchangedCategories(t *testing.T) {
	drilldown := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceHuman)
	afterRoots := actorContextR3ACCloneRoots(drilldown.Roots)
	idx := actorContextR3ACRootIndexByRole(afterRoots, actorContextR3ABRoleTestOnlyContext)
	if idx < 0 {
		t.Fatal("R3AE setup lost test-only context root")
	}
	afterRoots[idx].Fingerprint += ":changed"
	assessment := actorContextR3ACClassifyRootChanges(drilldown.Roots, afterRoots)

	before := actorContextR3ABFreshnessCore(t)
	after := actorContextR3QCloneCore(before)
	after.MachineReasons = []string{"test-only:freshness-context-reclassified"}

	view := actorContextR3ADFreshnessView(t, actorContextR3PAudienceHuman)
	bridge := actorContextR3ADExplainRootChanges(view, drilldown.Roots, assessment)
	got, err := actorContextR3AEEvaluate(before, after, view, bridge)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		actorContextR3PCategoryStopIdentity:    actorContextR3WStatusUnchanged,
		actorContextR3PCategoryMachineReasons:  actorContextR3WStatusChanged,
		actorContextR3PCategoryResolutionNeeds: actorContextR3WStatusUnchanged,
	}
	for category, status := range want {
		result := actorContextR3AECategory(got.Categories, category)
		if result == nil || result.Status != status {
			t.Fatalf("R3AE one-root mixed result mismatch for %s: got=%+v want=%q all=%+v",
				category, result, status, got.Categories)
		}
	}
}

func TestActorContextR3AELocatorReasonRequiresActualSourceRefValueChange(t *testing.T) {
	drilldown := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceHuman)
	afterRoots := actorContextR3ACCloneRoots(drilldown.Roots)
	idx := actorContextR3ADRootIndexByClaim(afterRoots, "protected-resource-source")
	if idx < 0 {
		t.Fatal("R3AE setup lost protected-resource root")
	}
	afterRoots[idx].Locator += ".moved"
	assessment := actorContextR3ACClassifyRootChanges(drilldown.Roots, afterRoots)

	before := actorContextR3ABFreshnessCore(t)
	view := actorContextR3ADFreshnessView(t, actorContextR3PAudienceHuman)
	bridge := actorContextR3ADExplainRootChanges(view, drilldown.Roots, assessment)

	unchanged, err := actorContextR3AEEvaluate(
		before,
		actorContextR3QCloneCore(before),
		view,
		bridge,
	)
	if err != nil {
		t.Fatal(err)
	}
	sourceRefs := actorContextR3AECategory(unchanged.Categories, actorContextR3PCategorySourceRefs)
	if sourceRefs == nil || sourceRefs.Status != actorContextR3WStatusUnchanged {
		t.Fatalf("locator reason alone changed source-refs meaning: %+v", unchanged)
	}

	after := actorContextR3QCloneCore(before)
	after.SourceRefs = actorContextR0SortedUnique(append(
		after.SourceRefs,
		"reference:test-only/moved-protected-target",
	))
	changed, err := actorContextR3AEEvaluate(before, after, view, bridge)
	if err != nil {
		t.Fatal(err)
	}
	sourceRefs = actorContextR3AECategory(changed.Categories, actorContextR3PCategorySourceRefs)
	if sourceRefs == nil || sourceRefs.Status != actorContextR3WStatusChanged {
		t.Fatalf("actual source-ref value change was not detected after locator reason: %+v", changed)
	}
}

func TestActorContextR3AEResidualCorroborationCannotExplainSelectedCategoryChange(t *testing.T) {
	drilldown := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)
	afterRoots := actorContextR3ACCloneRoots(drilldown.Roots)
	idx := actorContextR3ACRootIndexByRole(afterRoots, actorContextR3ABRoleCorroboration)
	if idx < 0 {
		t.Fatal("R3AE setup lost corroboration root")
	}
	afterRoots[idx].Fingerprint += ":changed"
	assessment := actorContextR3ACClassifyRootChanges(drilldown.Roots, afterRoots)

	before := actorContextR3ABFreshnessCore(t)
	view := actorContextR3ADFreshnessView(t, actorContextR3PAudienceAgent)
	bridge := actorContextR3ADExplainRootChanges(view, drilldown.Roots, assessment)

	unchanged, err := actorContextR3AEEvaluate(
		before,
		actorContextR3QCloneCore(before),
		view,
		bridge,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(unchanged.Categories) != 0 ||
		len(unchanged.ResidualRootChanges) != 1 ||
		unchanged.ResidualRootChanges[0].ChangeClass != actorContextR3ACChangeCorroboration {
		t.Fatalf("R3AE did not preserve residual corroboration without selected category impact: %+v", unchanged)
	}

	after := actorContextR3QCloneCore(before)
	after.MachineReasons = append(after.MachineReasons, "test-only:unexplained-change")
	if _, err := actorContextR3AEEvaluate(before, after, view, bridge); err == nil {
		t.Fatal("R3AE allowed residual corroboration change to justify unexplained selected category change")
	}
}

func TestActorContextR3AEResultHasNoAutomaticFreshnessOrResponseShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3AEResultProbe{})
	for _, forbidden := range []string{
		"Fresh",
		"Stale",
		"Safe",
		"Reusable",
		"CacheValid",
		"ShouldReevaluate",
		"ReevaluationRequired",
		"RetryRequired",
		"AutoReauthorize",
		"AutoRebuild",
		"NextAction",
		"HandoffOwner",
		"Authorized",
		"Allowed",
		"Winner",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3AE result introduced forbidden automatic freshness/action shortcut %q", forbidden)
		}
	}
}

func actorContextR3AEEvaluate(
	before actorContextR3OStopProjectionCoreProbe,
	after actorContextR3OStopProjectionCoreProbe,
	view actorContextR3PBoundedStopViewProbe,
	bridge actorContextR3ADBridgeResultProbe,
) (actorContextR3AEResultProbe, error) {
	impact := actorContextR3UAggregateImpacts(
		view.Audience,
		view.Purpose,
		[]actorContextR3TImpactExplanationProbe{bridge.Impact},
	)
	categoryResult, err := actorContextR3WEvaluateSameProfile(before, after, view, impact)
	if err != nil {
		return actorContextR3AEResultProbe{}, err
	}

	residual := append([]actorContextR3ACRootChangeProbe(nil), bridge.ResidualRootChanges...)
	sort.Slice(residual, func(i, j int) bool {
		if residual[i].RootIdentity != residual[j].RootIdentity {
			return residual[i].RootIdentity < residual[j].RootIdentity
		}
		if residual[i].RootRole != residual[j].RootRole {
			return residual[i].RootRole < residual[j].RootRole
		}
		return residual[i].ChangeClass < residual[j].ChangeClass
	})

	return actorContextR3AEResultProbe{
		Categories:          append([]actorContextR3WCategoryResultProbe(nil), categoryResult.Categories...),
		UnselectedChanges:   append([]actorContextR3TObservedChangeProbe(nil), categoryResult.UnselectedChanges...),
		ResidualRootChanges: residual,
		AuthorityLimits: []string{
			"root-change-reason-does-not-prove-category-value-change",
			"unchanged-after-reevaluation-does-not-prove-freshness-or-reuse",
			"residual-root-change-does-not-author-selected-category-change",
			"category-value-change-does-not-select-response-action",
		},
	}, nil
}

func actorContextR3AECategory(
	values []actorContextR3WCategoryResultProbe,
	category string,
) *actorContextR3WCategoryResultProbe {
	for i := range values {
		if values[i].Category == category {
			return &values[i]
		}
	}
	return nil
}

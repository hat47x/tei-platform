package main

import (
	"reflect"
	"testing"
)

func TestActorContextR3AERequirementRootReasonCanReevaluateToUnchangedMeaning(t *testing.T) {
	baseDrill := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceHuman)
	afterRoots := actorContextR3ACCloneRoots(baseDrill.Roots)
	idx := actorContextR3ADRootIndexByClaim(afterRoots, "authorization-requirement-source")
	if idx < 0 {
		t.Fatal("R3AE setup lost authorization requirement root")
	}
	afterRoots[idx].Fingerprint += ":changed"

	assessment := actorContextR3ACClassifyRootChanges(baseDrill.Roots, afterRoots)
	view := actorContextR3ADFreshnessView(t, actorContextR3PAudienceHuman)
	bridge := actorContextR3ADExplainRootChanges(view, baseDrill.Roots, assessment)
	impact := actorContextR3UAggregateImpacts(
		view.Audience,
		view.Purpose,
		[]actorContextR3TImpactExplanationProbe{bridge.Impact},
	)

	before := actorContextR3ABFreshnessCore(t)
	after := actorContextR3QCloneCore(before)
	got, err := actorContextR3WEvaluateSameProfile(before, after, view, impact)
	if err != nil {
		t.Fatal(err)
	}

	result := actorContextR3AECategory(got, actorContextR3PCategoryMachineReasons)
	if result == nil || result.Status != actorContextR3WStatusUnchanged {
		t.Fatalf("root change reason was equated with changed machine meaning: %+v", got)
	}
}

func TestActorContextR3AESameRootReasonCanLeadToChangedMeaningWhenFingerprintActuallyChanges(t *testing.T) {
	baseDrill := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceHuman)
	afterRoots := actorContextR3ACCloneRoots(baseDrill.Roots)
	idx := actorContextR3ADRootIndexByClaim(afterRoots, "authorization-requirement-source")
	if idx < 0 {
		t.Fatal("R3AE setup lost authorization requirement root")
	}
	afterRoots[idx].Fingerprint += ":changed"

	assessment := actorContextR3ACClassifyRootChanges(baseDrill.Roots, afterRoots)
	view := actorContextR3ADFreshnessView(t, actorContextR3PAudienceHuman)
	bridge := actorContextR3ADExplainRootChanges(view, baseDrill.Roots, assessment)
	impact := actorContextR3UAggregateImpacts(
		view.Audience,
		view.Purpose,
		[]actorContextR3TImpactExplanationProbe{bridge.Impact},
	)

	before := actorContextR3ABFreshnessCore(t)
	after := actorContextR3QCloneCore(before)
	after.MachineReasons = actorContextR0SortedUnique(append(
		after.MachineReasons,
		"test-only:additional-freshness-machine-reason",
	))

	got, err := actorContextR3WEvaluateSameProfile(before, after, view, impact)
	if err != nil {
		t.Fatal(err)
	}
	result := actorContextR3AECategory(got, actorContextR3PCategoryMachineReasons)
	if result == nil || result.Status != actorContextR3WStatusChanged {
		t.Fatalf("actual machine meaning change was not detected: %+v", got)
	}
	if result.BeforeFingerprint == result.AfterFingerprint {
		t.Fatalf("changed category retained identical fingerprint: %+v", result)
	}
}

func TestActorContextR3AEPriorObservationRevisionDoesNotCreateRuntimeOutcomeResult(t *testing.T) {
	baseDrill := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceHuman)
	afterRoots := actorContextR3ACCloneRoots(baseDrill.Roots)
	idx := actorContextR3ACRootIndexByRole(afterRoots, actorContextR3ABRoleRuntimeObservation)
	if idx < 0 {
		t.Fatal("R3AE setup lost prior runtime Observation root")
	}
	afterRoots[idx].Fingerprint += ":denied"
	afterRoots[idx].ClaimKind = "prior-authorization-observation:denied"

	assessment := actorContextR3ACClassifyRootChanges(baseDrill.Roots, afterRoots)
	view := actorContextR3ADFreshnessView(t, actorContextR3PAudienceHuman)
	bridge := actorContextR3ADExplainRootChanges(view, baseDrill.Roots, assessment)
	impact := actorContextR3UAggregateImpacts(
		view.Audience,
		view.Purpose,
		[]actorContextR3TImpactExplanationProbe{bridge.Impact},
	)

	before := actorContextR3ABFreshnessCore(t)
	after := actorContextR3QCloneCore(before)
	got, err := actorContextR3WEvaluateSameProfile(before, after, view, impact)
	if err != nil {
		t.Fatal(err)
	}

	machine := actorContextR3AECategory(got, actorContextR3PCategoryMachineReasons)
	if machine == nil || machine.Status != actorContextR3WStatusUnchanged {
		t.Fatalf("freshness stop did not remain stable after prior Observation revision: %+v", got)
	}
	if actorContextR3AECategory(got, actorContextR3PCategoryRuntimeOutcome) != nil {
		t.Fatalf("prior Observation revision fabricated runtime-outcome category result: %+v", got)
	}
	if before.RuntimeOutcome != "" || after.RuntimeOutcome != "" {
		t.Fatalf("R3AE setup fabricated current runtime outcome: before=%q after=%q",
			before.RuntimeOutcome, after.RuntimeOutcome)
	}
}

func TestActorContextR3AELocatorReasonNeedsActualSourceRefValueChange(t *testing.T) {
	baseDrill := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceHuman)
	afterRoots := actorContextR3ACCloneRoots(baseDrill.Roots)
	idx := actorContextR3ADRootIndexByClaim(afterRoots, "protected-resource-source")
	if idx < 0 {
		t.Fatal("R3AE setup lost protected-resource root")
	}
	oldLocator := afterRoots[idx].Locator
	newLocator := oldLocator + ".moved"
	afterRoots[idx].Locator = newLocator

	assessment := actorContextR3ACClassifyRootChanges(baseDrill.Roots, afterRoots)
	view := actorContextR3ADFreshnessView(t, actorContextR3PAudienceHuman)
	bridge := actorContextR3ADExplainRootChanges(view, baseDrill.Roots, assessment)
	impact := actorContextR3UAggregateImpacts(
		view.Audience,
		view.Purpose,
		[]actorContextR3TImpactExplanationProbe{bridge.Impact},
	)

	before := actorContextR3ABFreshnessCore(t)
	unchangedAfter := actorContextR3QCloneCore(before)
	unchanged, err := actorContextR3WEvaluateSameProfile(before, unchangedAfter, view, impact)
	if err != nil {
		t.Fatal(err)
	}
	sourceRefs := actorContextR3AECategory(unchanged, actorContextR3PCategorySourceRefs)
	if sourceRefs == nil || sourceRefs.Status != actorContextR3WStatusUnchanged {
		t.Fatalf("locator reason alone changed source-refs meaning: %+v", unchanged)
	}

	changedAfter := actorContextR3QCloneCore(before)
	changedAfter.SourceRefs = actorContextR3AEReplaceString(
		changedAfter.SourceRefs,
		oldLocator,
		newLocator,
	)
	if reflect.DeepEqual(before.SourceRefs, changedAfter.SourceRefs) {
		t.Fatalf("test setup could not replace source ref %q in %+v", oldLocator, before.SourceRefs)
	}
	changed, err := actorContextR3WEvaluateSameProfile(before, changedAfter, view, impact)
	if err != nil {
		t.Fatal(err)
	}
	sourceRefs = actorContextR3AECategory(changed, actorContextR3PCategorySourceRefs)
	if sourceRefs == nil || sourceRefs.Status != actorContextR3WStatusChanged {
		t.Fatalf("actual locator value change was not detected: %+v", changed)
	}
}

func TestActorContextR3AEAgentOnlyKnownFactChangeDoesNotStaleHumanSelectedResult(t *testing.T) {
	baseDrill := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)
	afterRoots := actorContextR3ACCloneRoots(baseDrill.Roots)
	idx := actorContextR3ADRootIndexByClaim(afterRoots, "authorization-requirement-source")
	if idx < 0 {
		t.Fatal("R3AE setup lost authorization requirement root")
	}
	afterRoots[idx].Fingerprint += ":changed"
	assessment := actorContextR3ACClassifyRootChanges(baseDrill.Roots, afterRoots)

	before := actorContextR3ABFreshnessCore(t)
	after := actorContextR3QCloneCore(before)
	after.KnownAuthorityFacts = append(after.KnownAuthorityFacts, actorContextR3MAuthorityFactProbe{
		Kind:      "test-only:refreshed-authority-fact",
		Value:     "changed",
		SourceRef: "test-only:r3ae",
	})

	humanView := actorContextR3ADFreshnessView(t, actorContextR3PAudienceHuman)
	humanBridge := actorContextR3ADExplainRootChanges(humanView, baseDrill.Roots, assessment)
	humanImpact := actorContextR3UAggregateImpacts(
		humanView.Audience,
		humanView.Purpose,
		[]actorContextR3TImpactExplanationProbe{humanBridge.Impact},
	)
	humanResult, err := actorContextR3WEvaluateSameProfile(before, after, humanView, humanImpact)
	if err != nil {
		t.Fatal(err)
	}
	if actorContextR3AECategory(humanResult, actorContextR3PCategoryKnownAuthorityFacts) != nil {
		t.Fatalf("Human omitted known-authority-facts became selected result: %+v", humanResult)
	}

	agentView := actorContextR3ADFreshnessView(t, actorContextR3PAudienceAgent)
	agentBridge := actorContextR3ADExplainRootChanges(agentView, baseDrill.Roots, assessment)
	agentImpact := actorContextR3UAggregateImpacts(
		agentView.Audience,
		agentView.Purpose,
		[]actorContextR3TImpactExplanationProbe{agentBridge.Impact},
	)
	agentResult, err := actorContextR3WEvaluateSameProfile(before, after, agentView, agentImpact)
	if err != nil {
		t.Fatal(err)
	}
	knownFacts := actorContextR3AECategory(agentResult, actorContextR3PCategoryKnownAuthorityFacts)
	if knownFacts == nil || knownFacts.Status != actorContextR3WStatusChanged {
		t.Fatalf("Agent selected known-authority-facts change was not detected: %+v", agentResult)
	}
}

func TestActorContextR3AEResidualRootChangeCannotExplainUnrelatedSelectedValueChange(t *testing.T) {
	baseDrill := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceHuman)
	afterRoots := actorContextR3ACCloneRoots(baseDrill.Roots)
	idx := actorContextR3ACRootIndexByRole(afterRoots, actorContextR3ABRoleCorroboration)
	if idx < 0 {
		t.Fatal("R3AE setup lost corroboration root")
	}
	afterRoots[idx].Fingerprint += ":changed"
	assessment := actorContextR3ACClassifyRootChanges(baseDrill.Roots, afterRoots)

	view := actorContextR3ADFreshnessView(t, actorContextR3PAudienceHuman)
	bridge := actorContextR3ADExplainRootChanges(view, baseDrill.Roots, assessment)
	if len(bridge.ResidualRootChanges) != 1 || len(bridge.Impact.Reasons) != 0 {
		t.Fatalf("test setup requires residual-only root change: %+v", bridge)
	}
	impact := actorContextR3UAggregateImpacts(
		view.Audience,
		view.Purpose,
		[]actorContextR3TImpactExplanationProbe{bridge.Impact},
	)

	before := actorContextR3ABFreshnessCore(t)
	after := actorContextR3QCloneCore(before)
	after.MachineReasons = append(after.MachineReasons, "test-only:unexplained-change")

	if _, err := actorContextR3WEvaluateSameProfile(before, after, view, impact); err == nil {
		t.Fatal("R3AE allowed residual unmapped root change to explain selected category value change")
	}
}

func TestActorContextR3AEUsesExistingR3WOutcomeWithoutAutomaticActionShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3WResultProbe{})
	for _, forbidden := range []string{
		"Stale",
		"Fresh",
		"ShouldReevaluate",
		"ReevaluationRequired",
		"AutoRebuild",
		"AutoReauthorize",
		"RetryRequired",
		"CacheValid",
		"NextAction",
		"HandoffOwner",
		"Winner",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3AE/R3W outcome introduced forbidden automatic action %q", forbidden)
		}
	}
}

func actorContextR3AECategory(
	result actorContextR3WResultProbe,
	category string,
) *actorContextR3WCategoryResultProbe {
	for i := range result.Categories {
		if result.Categories[i].Category == category {
			return &result.Categories[i]
		}
	}
	return nil
}

func actorContextR3AEReplaceString(
	values []string,
	before string,
	after string,
) []string {
	out := append([]string(nil), values...)
	replaced := false
	for i := range out {
		if out[i] == before {
			out[i] = after
			replaced = true
		}
	}
	if !replaced {
		return out
	}
	return actorContextR0SortedUnique(out)
}

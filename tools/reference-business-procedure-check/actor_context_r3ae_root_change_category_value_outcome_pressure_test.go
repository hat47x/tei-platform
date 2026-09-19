package main

import (
	"reflect"
	"testing"
)

func TestActorContextR3AERequirementRootReasonCanReevaluateToUnchangedOrChangedMeaning(t *testing.T) {
	base, reasons, bindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceHuman)
	afterRoots := actorContextR3ACCloneRoots(base)
	idx := actorContextR3ADRootIndexByClaim(afterRoots, "authorization-requirement-source")
	if idx < 0 {
		t.Fatal("R3AE setup lost Authorization requirement root")
	}
	afterRoots[idx].Fingerprint += ":changed"

	deltas := actorContextR3ACClassifyRootChanges(base, afterRoots)
	bridge := actorContextR3ADExplainBoundRootChanges(reasons, deltas.Changes, bindings)
	view := actorContextR3AEView(t, actorContextR3PAudienceHuman)

	before := actorContextR3ABFreshnessCore(t)
	unchangedAfter := actorContextR3QCloneCore(before)
	unchanged, err := actorContextR3AEEvaluate(before, unchangedAfter, view, bridge)
	if err != nil {
		t.Fatal(err)
	}
	machine := actorContextR3AECategory(unchanged, actorContextR3PCategoryMachineReasons)
	if machine == nil || machine.Status != actorContextR3WStatusUnchanged {
		t.Fatalf("requirement root reason was mistaken for category value change: %+v", unchanged)
	}
	stopIdentity := actorContextR3AECategory(unchanged, actorContextR3PCategoryStopIdentity)
	if stopIdentity == nil || stopIdentity.Status != actorContextR3WStatusUnchanged {
		t.Fatalf("Business-address dependency did not retain unchanged stop identity: %+v", unchanged)
	}

	changedAfter := actorContextR3QCloneCore(before)
	changedAfter.MachineReasons = append(
		changedAfter.MachineReasons,
		"test-only:r3ae-additional-machine-reason",
	)
	changed, err := actorContextR3AEEvaluate(before, changedAfter, view, bridge)
	if err != nil {
		t.Fatal(err)
	}
	machine = actorContextR3AECategory(changed, actorContextR3PCategoryMachineReasons)
	if machine == nil || machine.Status != actorContextR3WStatusChanged {
		t.Fatalf("actual machine-reason meaning change was not detected: %+v", changed)
	}
	stopIdentity = actorContextR3AECategory(changed, actorContextR3PCategoryStopIdentity)
	if stopIdentity == nil || stopIdentity.Status != actorContextR3WStatusUnchanged {
		t.Fatalf("unrelated stop identity changed with machine-reason value: %+v", changed)
	}
}

func TestActorContextR3AEPriorObservationRevisionRemainsResidualWithoutCategoryOutcome(t *testing.T) {
	base, reasons, bindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceAgent)
	afterRoots := actorContextR3ACCloneRoots(base)
	idx := actorContextR3ACRootIndexByRole(afterRoots, actorContextR3ABRoleRuntimeObservation)
	if idx < 0 {
		t.Fatal("R3AE setup lost prior runtime Observation root")
	}
	afterRoots[idx].Fingerprint += ":denied"
	afterRoots[idx].ClaimKind = "prior-authorization-observation:denied"

	deltas := actorContextR3ACClassifyRootChanges(base, afterRoots)
	bridge := actorContextR3ADExplainBoundRootChanges(reasons, deltas.Changes, bindings)
	if len(bridge.Impact.Reasons) != 0 ||
		len(bridge.ResidualRootDeltas) != 1 ||
		bridge.ResidualRootDeltas[0].Reason != actorContextR3ADResidualNoBinding {
		t.Fatalf("prior Observation revision should remain no-binding residual: %+v", bridge)
	}

	view := actorContextR3AEView(t, actorContextR3PAudienceAgent)
	before := actorContextR3ABFreshnessCore(t)
	after := actorContextR3QCloneCore(before)
	got, err := actorContextR3AEEvaluate(before, after, view, bridge)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Categories) != 0 {
		t.Fatalf("prior Observation residual fabricated selected category outcome: %+v", got)
	}
	if actorContextR3AECategory(got, actorContextR3PCategoryRuntimeOutcome) != nil {
		t.Fatalf("prior Observation revision fabricated runtime-outcome result: %+v", got)
	}
	if before.RuntimeOutcome != "" || after.RuntimeOutcome != "" {
		t.Fatalf("R3AE setup fabricated current runtime outcome: before=%q after=%q",
			before.RuntimeOutcome, after.RuntimeOutcome)
	}
}

func TestActorContextR3AELocatorReasonNeedsActualSourceRefValueChange(t *testing.T) {
	base, reasons, bindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceHuman)
	afterRoots := actorContextR3ACCloneRoots(base)
	idx := actorContextR3ADRootIndexByClaim(afterRoots, "protected-resource-source")
	if idx < 0 {
		t.Fatal("R3AE setup lost protected-resource root")
	}
	oldLocator := afterRoots[idx].Locator
	newLocator := oldLocator + ".moved"
	afterRoots[idx].Locator = newLocator

	deltas := actorContextR3ACClassifyRootChanges(base, afterRoots)
	bridge := actorContextR3ADExplainBoundRootChanges(reasons, deltas.Changes, bindings)
	view := actorContextR3AEView(t, actorContextR3PAudienceHuman)

	before := actorContextR3ABFreshnessCore(t)
	unchangedAfter := actorContextR3QCloneCore(before)
	unchanged, err := actorContextR3AEEvaluate(before, unchangedAfter, view, bridge)
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
	changed, err := actorContextR3AEEvaluate(before, changedAfter, view, bridge)
	if err != nil {
		t.Fatal(err)
	}
	sourceRefs = actorContextR3AECategory(changed, actorContextR3PCategorySourceRefs)
	if sourceRefs == nil || sourceRefs.Status != actorContextR3WStatusChanged {
		t.Fatalf("actual locator value change was not detected: %+v", changed)
	}
}

func TestActorContextR3AEAgentOnlyKnownFactChangeDoesNotBecomeHumanSelectedResult(t *testing.T) {
	agentBase, agentReasons, agentBindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceAgent)
	_, humanReasons, humanBindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceHuman)
	afterRoots := actorContextR3ACCloneRoots(agentBase)
	idx := actorContextR3ADRootIndexByClaim(afterRoots, "authorization-requirement-source")
	if idx < 0 {
		t.Fatal("R3AE setup lost Authorization requirement root")
	}
	afterRoots[idx].Fingerprint += ":changed"
	deltas := actorContextR3ACClassifyRootChanges(agentBase, afterRoots)

	before := actorContextR3ABFreshnessCore(t)
	after := actorContextR3QCloneCore(before)
	after.KnownAuthorityFacts = append(after.KnownAuthorityFacts, actorContextR3MAuthorityFactProbe{
		Kind:      "test-only:refreshed-authority-fact",
		Value:     "changed",
		SourceRef: "test-only:r3ae",
	})

	humanView := actorContextR3AEView(t, actorContextR3PAudienceHuman)
	humanBridge := actorContextR3ADExplainBoundRootChanges(
		humanReasons,
		deltas.Changes,
		humanBindings,
	)
	humanResult, err := actorContextR3AEEvaluate(before, after, humanView, humanBridge)
	if err != nil {
		t.Fatal(err)
	}
	if actorContextR3AECategory(humanResult, actorContextR3PCategoryKnownAuthorityFacts) != nil {
		t.Fatalf("Human omitted known-authority-facts became selected result: %+v", humanResult)
	}

	agentView := actorContextR3AEView(t, actorContextR3PAudienceAgent)
	agentBridge := actorContextR3ADExplainBoundRootChanges(
		agentReasons,
		deltas.Changes,
		agentBindings,
	)
	agentResult, err := actorContextR3AEEvaluate(before, after, agentView, agentBridge)
	if err != nil {
		t.Fatal(err)
	}
	knownFacts := actorContextR3AECategory(agentResult, actorContextR3PCategoryKnownAuthorityFacts)
	if knownFacts == nil || knownFacts.Status != actorContextR3WStatusChanged {
		t.Fatalf("Agent selected known-authority-facts change was not detected: %+v", agentResult)
	}
}

func TestActorContextR3AEResidualCorroborationCannotExplainSelectedValueChange(t *testing.T) {
	base, reasons, bindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceHuman)
	afterRoots := actorContextR3ACCloneRoots(base)
	idx := actorContextR3ACRootIndexByRole(afterRoots, actorContextR3ABRoleCorroboration)
	if idx < 0 {
		t.Fatal("R3AE setup lost corroboration root")
	}
	afterRoots[idx].Fingerprint += ":changed"

	deltas := actorContextR3ACClassifyRootChanges(base, afterRoots)
	bridge := actorContextR3ADExplainBoundRootChanges(reasons, deltas.Changes, bindings)
	if len(bridge.ResidualRootDeltas) != 1 ||
		bridge.ResidualRootDeltas[0].Reason != actorContextR3ADResidualNoBinding ||
		len(bridge.Impact.Reasons) != 0 {
		t.Fatalf("test setup requires corroboration residual only: %+v", bridge)
	}

	view := actorContextR3AEView(t, actorContextR3PAudienceHuman)
	before := actorContextR3ABFreshnessCore(t)
	after := actorContextR3QCloneCore(before)
	after.MachineReasons = append(after.MachineReasons, "test-only:unexplained-change")

	if _, err := actorContextR3AEEvaluate(before, after, view, bridge); err == nil {
		t.Fatal("R3AE allowed residual corroboration delta to explain selected category change")
	}
}

func TestActorContextR3AEBoundRootSetResidualDoesNotBecomeCategoryOutcome(t *testing.T) {
	base, reasons, bindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceHuman)
	afterRoots := actorContextR3ACCloneRoots(base)
	idx := actorContextR3ADRootIndexByClaim(afterRoots, "authorization-requirement-source")
	if idx < 0 {
		t.Fatal("R3AE setup lost Authorization requirement root")
	}
	afterRoots = append(afterRoots[:idx], afterRoots[idx+1:]...)

	deltas := actorContextR3ACClassifyRootChanges(base, afterRoots)
	bridge := actorContextR3ADExplainBoundRootChanges(reasons, deltas.Changes, bindings)
	if len(bridge.ResidualRootDeltas) != 1 ||
		bridge.ResidualRootDeltas[0].Reason != actorContextR3ADResidualNoExactClass {
		t.Fatalf("bound root-set delta should remain unmapped residual: %+v", bridge)
	}

	view := actorContextR3AEView(t, actorContextR3PAudienceHuman)
	before := actorContextR3ABFreshnessCore(t)
	got, err := actorContextR3AEEvaluate(
		before,
		actorContextR3QCloneCore(before),
		view,
		bridge,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Categories) != 0 {
		t.Fatalf("unmapped root-set residual fabricated category outcome: %+v", got)
	}
}

func TestActorContextR3AEUsesR3WOutcomeWithoutAutomaticActionShortcut(t *testing.T) {
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

func actorContextR3AEView(
	t *testing.T,
	audience string,
) actorContextR3PBoundedStopViewProbe {
	t.Helper()
	core := actorContextR3ABFreshnessCore(t)
	purpose := map[string]string{
		actorContextR3PAudienceHuman: actorContextR3PPurposeOrientation,
		actorContextR3PAudienceAgent: actorContextR3PPurposeResolutionInspection,
	}[audience]
	view, err := actorContextR3PSelectBoundedStopView(core, audience, purpose)
	if err != nil {
		t.Fatal(err)
	}
	return view
}

func actorContextR3AEEvaluate(
	before actorContextR3OStopProjectionCoreProbe,
	after actorContextR3OStopProjectionCoreProbe,
	view actorContextR3PBoundedStopViewProbe,
	bridge actorContextR3ADBridgeResultProbe,
) (actorContextR3WResultProbe, error) {
	impact := actorContextR3UAggregateImpacts(
		view.Audience,
		view.Purpose,
		[]actorContextR3TImpactExplanationProbe{bridge.Impact},
	)
	return actorContextR3WEvaluateSameProfile(before, after, view, impact)
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

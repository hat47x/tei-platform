package main

import (
	"reflect"
	"sort"
	"testing"
)

type actorContextR3AFCategoryOutcomeProbe struct {
	Category          string
	Status            string
	BeforeFingerprint string
	AfterFingerprint  string
	Reasons           []actorContextR3TReevaluationReasonProbe
}

type actorContextR3AFExplanationProbe struct {
	Audience                    string
	Purpose                     string
	FreshnessStopReason         string
	ResolutionNeedClasses       []string
	ReevaluationAuthorityStatus string
	ReuseAuthorityStatus        string
	CategoryOutcomes            []actorContextR3AFCategoryOutcomeProbe
	UnselectedChanges           []actorContextR3TObservedChangeProbe
	ResidualRootDeltas          []actorContextR3ADResidualRootDeltaProbe
	AuthorityLimits             []string
}

func TestActorContextR3AFUnchangedCategoryDoesNotResolveFreshnessAuthority(t *testing.T) {
	base, reasons, bindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceHuman)
	afterRoots := actorContextR3ACCloneRoots(base)
	idx := actorContextR3ADRootIndexByClaim(afterRoots, "authorization-requirement-source")
	if idx < 0 {
		t.Fatal("R3AF setup lost Authorization requirement root")
	}
	afterRoots[idx].Fingerprint += ":changed"

	deltas := actorContextR3ACClassifyRootChanges(base, afterRoots)
	bridge := actorContextR3ADExplainBoundRootChanges(reasons, deltas.Changes, bindings)
	view := actorContextR3AEView(t, actorContextR3PAudienceHuman)

	before := actorContextR3ABFreshnessCore(t)
	result, err := actorContextR3AEEvaluate(
		before,
		actorContextR3QCloneCore(before),
		view,
		bridge,
	)
	if err != nil {
		t.Fatal(err)
	}

	explanation := actorContextR3AFBuildExplanation(
		view,
		result,
		bridge,
		actorContextR3AFFreshnessAuthority(),
	)
	machine := actorContextR3AFCategory(explanation, actorContextR3PCategoryMachineReasons)
	if machine == nil || machine.Status != actorContextR3WStatusUnchanged {
		t.Fatalf("R3AF lost unchanged category outcome: %+v", explanation)
	}
	if machine.BeforeFingerprint == "" || machine.AfterFingerprint == "" ||
		machine.BeforeFingerprint != machine.AfterFingerprint {
		t.Fatalf("R3AF unchanged outcome lost exact fingerprint identity: %+v", machine)
	}
	if len(machine.Reasons) == 0 ||
		machine.Reasons[0].DependencyReason != actorContextR3SReasonAuthorizationRequirementSource {
		t.Fatalf("R3AF unchanged outcome lost exact reason identity: %+v", machine)
	}
	if explanation.ReevaluationAuthorityStatus != actorContextR3YReevaluationUnresolved ||
		explanation.ReuseAuthorityStatus != actorContextR3YReuseUnresolved {
		t.Fatalf("unchanged category outcome resolved freshness authority: %+v", explanation)
	}
	if !reflect.DeepEqual(explanation.ResolutionNeedClasses, []string{
		"authorization-evaluation-freshness-authority",
	}) {
		t.Fatalf("R3AF lost unresolved freshness resolution need: %+v", explanation)
	}
}

func TestActorContextR3AFChangedCategoryStillDoesNotSelectResponseAction(t *testing.T) {
	base, reasons, bindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceHuman)
	afterRoots := actorContextR3ACCloneRoots(base)
	idx := actorContextR3ADRootIndexByClaim(afterRoots, "authorization-requirement-source")
	if idx < 0 {
		t.Fatal("R3AF setup lost Authorization requirement root")
	}
	afterRoots[idx].Fingerprint += ":changed"
	deltas := actorContextR3ACClassifyRootChanges(base, afterRoots)
	bridge := actorContextR3ADExplainBoundRootChanges(reasons, deltas.Changes, bindings)
	view := actorContextR3AEView(t, actorContextR3PAudienceHuman)

	before := actorContextR3ABFreshnessCore(t)
	after := actorContextR3QCloneCore(before)
	after.MachineReasons = actorContextR0SortedUnique(append(
		after.MachineReasons,
		"test-only:r3af-category-meaning-changed",
	))
	result, err := actorContextR3AEEvaluate(before, after, view, bridge)
	if err != nil {
		t.Fatal(err)
	}

	explanation := actorContextR3AFBuildExplanation(
		view,
		result,
		bridge,
		actorContextR3AFFreshnessAuthority(),
	)
	machine := actorContextR3AFCategory(explanation, actorContextR3PCategoryMachineReasons)
	if machine == nil || machine.Status != actorContextR3WStatusChanged {
		t.Fatalf("R3AF lost changed category outcome: %+v", explanation)
	}
	if machine.BeforeFingerprint == "" || machine.AfterFingerprint == "" ||
		machine.BeforeFingerprint == machine.AfterFingerprint {
		t.Fatalf("R3AF changed outcome lost before/after fingerprint delta: %+v", machine)
	}
	if explanation.ReevaluationAuthorityStatus != actorContextR3YReevaluationUnresolved ||
		explanation.ReuseAuthorityStatus != actorContextR3YReuseUnresolved {
		t.Fatalf("changed category outcome fabricated freshness authority decision: %+v", explanation)
	}
}

func TestActorContextR3AFPriorObservationResidualStaysSeparateFromCategoryOutcome(t *testing.T) {
	base, reasons, bindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceAgent)
	afterRoots := actorContextR3ACCloneRoots(base)
	idx := actorContextR3ACRootIndexByRole(afterRoots, actorContextR3ABRoleRuntimeObservation)
	if idx < 0 {
		t.Fatal("R3AF setup lost prior runtime Observation root")
	}
	afterRoots[idx].Fingerprint += ":denied"
	afterRoots[idx].ClaimKind = "prior-authorization-observation:denied"

	deltas := actorContextR3ACClassifyRootChanges(base, afterRoots)
	bridge := actorContextR3ADExplainBoundRootChanges(reasons, deltas.Changes, bindings)
	if len(bridge.ResidualRootDeltas) != 1 ||
		bridge.ResidualRootDeltas[0].Reason != actorContextR3ADResidualNoBinding {
		t.Fatalf("R3AF setup requires no-binding prior Observation residual: %+v", bridge)
	}

	view := actorContextR3AEView(t, actorContextR3PAudienceAgent)
	before := actorContextR3ABFreshnessCore(t)
	result, err := actorContextR3AEEvaluate(
		before,
		actorContextR3QCloneCore(before),
		view,
		bridge,
	)
	if err != nil {
		t.Fatal(err)
	}
	explanation := actorContextR3AFBuildExplanation(
		view,
		result,
		bridge,
		actorContextR3AFFreshnessAuthority(),
	)

	if len(explanation.CategoryOutcomes) != 0 {
		t.Fatalf("prior Observation residual fabricated category outcome: %+v", explanation)
	}
	if len(explanation.ResidualRootDeltas) != 1 ||
		explanation.ResidualRootDeltas[0].Delta.RootRole != actorContextR3ABRoleRuntimeObservation {
		t.Fatalf("R3AF lost prior Observation residual detail: %+v", explanation)
	}
	if actorContextR3AFCategory(explanation, actorContextR3PCategoryRuntimeOutcome) != nil {
		t.Fatalf("prior Observation residual became current runtime outcome: %+v", explanation)
	}
}

func TestActorContextR3AFHumanAndAgentShareMachineOutcomeWhileAgentMayAddFacts(t *testing.T) {
	agentBase, agentReasons, agentBindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceAgent)
	_, humanReasons, humanBindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceHuman)
	afterRoots := actorContextR3ACCloneRoots(agentBase)
	idx := actorContextR3ADRootIndexByClaim(afterRoots, "authorization-requirement-source")
	if idx < 0 {
		t.Fatal("R3AF setup lost Authorization requirement root")
	}
	afterRoots[idx].Fingerprint += ":changed"
	deltas := actorContextR3ACClassifyRootChanges(agentBase, afterRoots)

	before := actorContextR3ABFreshnessCore(t)
	after := actorContextR3QCloneCore(before)
	after.KnownAuthorityFacts = append(after.KnownAuthorityFacts, actorContextR3MAuthorityFactProbe{
		Kind:      "test-only:r3af-refreshed-fact",
		Value:     "changed",
		SourceRef: "test-only:r3af",
	})

	humanView := actorContextR3AEView(t, actorContextR3PAudienceHuman)
	humanBridge := actorContextR3ADExplainBoundRootChanges(humanReasons, deltas.Changes, humanBindings)
	humanResult, err := actorContextR3AEEvaluate(before, after, humanView, humanBridge)
	if err != nil {
		t.Fatal(err)
	}
	human := actorContextR3AFBuildExplanation(
		humanView, humanResult, humanBridge, actorContextR3AFFreshnessAuthority(),
	)

	agentView := actorContextR3AEView(t, actorContextR3PAudienceAgent)
	agentBridge := actorContextR3ADExplainBoundRootChanges(agentReasons, deltas.Changes, agentBindings)
	agentResult, err := actorContextR3AEEvaluate(before, after, agentView, agentBridge)
	if err != nil {
		t.Fatal(err)
	}
	agent := actorContextR3AFBuildExplanation(
		agentView, agentResult, agentBridge, actorContextR3AFFreshnessAuthority(),
	)

	humanMachine := actorContextR3AFCategory(human, actorContextR3PCategoryMachineReasons)
	agentMachine := actorContextR3AFCategory(agent, actorContextR3PCategoryMachineReasons)
	if humanMachine == nil || agentMachine == nil || !reflect.DeepEqual(*humanMachine, *agentMachine) {
		t.Fatalf("Human / Agent shared machine outcome diverged: human=%+v agent=%+v", humanMachine, agentMachine)
	}
	if actorContextR3AFCategory(human, actorContextR3PCategoryKnownAuthorityFacts) != nil {
		t.Fatalf("Human explanation exposed omitted known-authority-facts: %+v", human)
	}
	agentFacts := actorContextR3AFCategory(agent, actorContextR3PCategoryKnownAuthorityFacts)
	if agentFacts == nil || agentFacts.Status != actorContextR3WStatusChanged {
		t.Fatalf("Agent explanation lost selected known-authority-facts outcome: %+v", agent)
	}
}

func TestActorContextR3AFKeepsUnselectedChangesSeparateFromCategoryOutcome(t *testing.T) {
	view := actorContextR3AEView(t, actorContextR3PAudienceHuman)
	unselected := actorContextR3TObservedChangeProbe{
		ChangeClass:   actorContextR3TChangeReferenceInput,
		DependencyKey: "reference:test-only:r3af-unselected",
	}
	result := actorContextR3WResultProbe{
		UnselectedChanges: []actorContextR3TObservedChangeProbe{unselected},
	}
	explanation := actorContextR3AFBuildExplanation(
		view,
		result,
		actorContextR3ADBridgeResultProbe{},
		actorContextR3AFFreshnessAuthority(),
	)
	if len(explanation.CategoryOutcomes) != 0 ||
		!reflect.DeepEqual(explanation.UnselectedChanges, []actorContextR3TObservedChangeProbe{unselected}) {
		t.Fatalf("R3AF collapsed unselected change into category outcome: %+v", explanation)
	}
}

func TestActorContextR3AFExplanationHasNoFreshSafeReuseOrActionShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3AFExplanationProbe{})
	for _, forbidden := range []string{
		"Fresh",
		"Stale",
		"Safe",
		"Reusable",
		"CacheValid",
		"ShouldReevaluate",
		"ReevaluationRequired",
		"ReusePermitted",
		"RetryRequired",
		"AutoReauthorize",
		"AutoRebuild",
		"ApprovalRequired",
		"NextAction",
		"HandoffOwner",
		"Authorized",
		"Allowed",
		"Winner",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3AF explanation introduced forbidden freshness/action shortcut %q", forbidden)
		}
	}
}

func actorContextR3AFFreshnessAuthority() actorContextR3YAuthorityResultProbe {
	return actorContextR3YAnalyzeFreshnessAuthority(
		actorContextR3YTupleMatchImpact(),
		actorContextR3YFreshnessPressureProbe{StaticSemanticChanged: true},
		nil,
	)
}

func actorContextR3AFBuildExplanation(
	view actorContextR3PBoundedStopViewProbe,
	categoryResult actorContextR3WResultProbe,
	bridge actorContextR3ADBridgeResultProbe,
	authority actorContextR3YAuthorityResultProbe,
) actorContextR3AFExplanationProbe {
	out := actorContextR3AFExplanationProbe{
		Audience:                    view.Audience,
		Purpose:                     view.Purpose,
		FreshnessStopReason:         actorContextR3AAFreshnessReason,
		ResolutionNeedClasses:       []string{"authorization-evaluation-freshness-authority"},
		ReevaluationAuthorityStatus: authority.ReevaluationAuthorityStatus,
		ReuseAuthorityStatus:        authority.ReuseAuthorityStatus,
		UnselectedChanges:           append([]actorContextR3TObservedChangeProbe(nil), categoryResult.UnselectedChanges...),
		ResidualRootDeltas:          append([]actorContextR3ADResidualRootDeltaProbe(nil), bridge.ResidualRootDeltas...),
		AuthorityLimits: []string{
			"unchanged-category-meaning-does-not-prove-authorization-freshness",
			"unchanged-category-meaning-does-not-authorize-prior-observation-reuse",
			"changed-category-meaning-does-not-select-response-action",
			"unselected-change-does-not-mean-resolved",
			"residual-root-delta-does-not-mean-resolved",
		},
	}

	for _, category := range categoryResult.Categories {
		out.CategoryOutcomes = append(out.CategoryOutcomes, actorContextR3AFCategoryOutcomeProbe{
			Category:          category.Category,
			Status:            category.Status,
			BeforeFingerprint: category.BeforeFingerprint,
			AfterFingerprint:  category.AfterFingerprint,
			Reasons:           actorContextR3TUniqueReasons(category.Reasons),
		})
	}
	sort.Slice(out.CategoryOutcomes, func(i, j int) bool {
		return out.CategoryOutcomes[i].Category < out.CategoryOutcomes[j].Category
	})
	sort.Slice(out.ResidualRootDeltas, func(i, j int) bool {
		left := out.ResidualRootDeltas[i]
		right := out.ResidualRootDeltas[j]
		if left.Delta.RootIdentity != right.Delta.RootIdentity {
			return left.Delta.RootIdentity < right.Delta.RootIdentity
		}
		if left.Delta.ChangeClass != right.Delta.ChangeClass {
			return left.Delta.ChangeClass < right.Delta.ChangeClass
		}
		return left.Reason < right.Reason
	})
	return out
}

func actorContextR3AFCategory(
	explanation actorContextR3AFExplanationProbe,
	category string,
) *actorContextR3AFCategoryOutcomeProbe {
	for i := range explanation.CategoryOutcomes {
		if explanation.CategoryOutcomes[i].Category == category {
			return &explanation.CategoryOutcomes[i]
		}
	}
	return nil
}

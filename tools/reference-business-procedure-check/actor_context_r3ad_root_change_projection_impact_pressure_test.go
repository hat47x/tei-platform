package main

import (
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3ADResidualNoBinding    = "projection-dependency-binding-unresolved"
	actorContextR3ADResidualNoExactClass = "r3t-change-class-unmapped"
)

type actorContextR3ADRootDependencyBindingProbe struct {
	RootAddress   string
	DependencyKey string
}

type actorContextR3ADResidualRootDeltaProbe struct {
	Delta  actorContextR3ACRootChangeProbe
	Reason string
}

type actorContextR3ADBridgeResultProbe struct {
	Impact             actorContextR3TImpactExplanationProbe
	ResidualRootDeltas []actorContextR3ADResidualRootDeltaProbe
	AuthorityLimits    []string
}

func TestActorContextR3ADBusinessReferenceContentDeltaUsesExistingDependencyReasons(t *testing.T) {
	base, humanReasons, humanBindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceHuman)
	_, agentReasons, agentBindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceAgent)
	after := actorContextR3ACCloneRoots(base)
	idx := actorContextR3ADRootIndexByClaim(after, "authorization-requirement-source")
	if idx < 0 {
		t.Fatal("R3AD setup lost Authorization requirement root")
	}
	after[idx].Fingerprint += ":changed"

	deltas := actorContextR3ACClassifyRootChanges(base, after)
	human := actorContextR3ADExplainBoundRootChanges(humanReasons, deltas.Changes, humanBindings)
	agent := actorContextR3ADExplainBoundRootChanges(agentReasons, deltas.Changes, agentBindings)

	for _, impact := range []actorContextR3ADBridgeResultProbe{human, agent} {
		if !actorContextR3THasReason(
			impact.Impact.Reasons,
			actorContextR3PCategoryMachineReasons,
			actorContextR3TChangeReferenceInput,
			actorContextR3SReasonAuthorizationRequirementSource,
		) {
			t.Fatalf("R3AD lost shared Authorization requirement reason: %+v", impact)
		}
		if !actorContextR3THasReason(
			impact.Impact.Reasons,
			actorContextR3PCategoryStopIdentity,
			actorContextR3TChangeReferenceInput,
			actorContextR3SReasonBusinessAddressSource,
		) {
			t.Fatalf("R3AD lost shared Business address reason: %+v", impact)
		}
		if len(impact.ResidualRootDeltas) != 0 {
			t.Fatalf("bound Reference delta remained residual: %+v", impact.ResidualRootDeltas)
		}
	}

	humanMachine := actorContextR3ADReasonsForCategory(human.Impact.Reasons, actorContextR3PCategoryMachineReasons)
	agentMachine := actorContextR3ADReasonsForCategory(agent.Impact.Reasons, actorContextR3PCategoryMachineReasons)
	if !reflect.DeepEqual(humanMachine, agentMachine) {
		t.Fatalf("shared Human / Agent machine reason identity diverged: human=%+v agent=%+v", humanMachine, agentMachine)
	}
	if actorContextR3THasCategory(human.Impact.Reasons, actorContextR3PCategoryKnownAuthorityFacts) {
		t.Fatalf("Human impact disclosed omitted known-authority-facts: %+v", human.Impact.Reasons)
	}
	if !actorContextR3THasReason(
		agent.Impact.Reasons,
		actorContextR3PCategoryKnownAuthorityFacts,
		actorContextR3TChangeReferenceInput,
		actorContextR3SReasonAuthorizationRequirementSource,
	) {
		t.Fatalf("Agent impact lost selected known-authority-facts reason: %+v", agent.Impact.Reasons)
	}
}

func TestActorContextR3ADOverlayLocatorOnlyDeltaAffectsSourceRefsOnly(t *testing.T) {
	base, reasons, bindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceHuman)
	after := actorContextR3ACCloneRoots(base)
	idx := actorContextR3ADRootIndexByClaim(after, "protected-resource-source")
	if idx < 0 {
		t.Fatal("R3AD setup lost protected-resource root")
	}
	after[idx].Locator += ".moved"

	deltas := actorContextR3ACClassifyRootChanges(base, after)
	got := actorContextR3ADExplainBoundRootChanges(reasons, deltas.Changes, bindings)

	if !actorContextR3THasReason(
		got.Impact.Reasons,
		actorContextR3PCategorySourceRefs,
		actorContextR3TChangeLocatorRefresh,
		actorContextR3SReasonDrillDownLocator,
	) {
		t.Fatalf("R3AD lost overlay locator-refresh reason: %+v", got)
	}
	if len(got.Impact.Reasons) != 1 {
		t.Fatalf("locator-only delta leaked into semantic categories: %+v", got.Impact.Reasons)
	}
}

func TestActorContextR3ADOverlayContentAndLocatorDeltaPreservesIndependentReasons(t *testing.T) {
	base, reasons, bindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceHuman)
	after := actorContextR3ACCloneRoots(base)
	idx := actorContextR3ADRootIndexByClaim(after, "protected-resource-source")
	if idx < 0 {
		t.Fatal("R3AD setup lost protected-resource root")
	}
	after[idx].Fingerprint += ":changed"
	after[idx].Locator += ".moved"

	deltas := actorContextR3ACClassifyRootChanges(base, after)
	got := actorContextR3ADExplainBoundRootChanges(reasons, deltas.Changes, bindings)

	for _, want := range []struct {
		category string
		class    string
		reason   string
	}{
		{actorContextR3PCategoryMachineReasons, actorContextR3TChangeReferenceInput, actorContextR3SReasonProtectedResourceSource},
		{actorContextR3PCategoryResolutionNeeds, actorContextR3TChangeReferenceInput, actorContextR3SReasonUnresolvedClassification},
		{actorContextR3PCategorySourceRefs, actorContextR3TChangeLocatorRefresh, actorContextR3SReasonDrillDownLocator},
	} {
		if !actorContextR3THasReason(got.Impact.Reasons, want.category, want.class, want.reason) {
			t.Fatalf("R3AD lost independent bounded reason %+v: %+v", want, got)
		}
	}
}

func TestActorContextR3ADPriorObservationDeltaRemainsNoBindingResidual(t *testing.T) {
	base, reasons, bindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceAgent)
	after := actorContextR3ACCloneRoots(base)
	idx := actorContextR3ACRootIndexByRole(after, actorContextR3ABRoleRuntimeObservation)
	if idx < 0 {
		t.Fatal("R3AD setup lost prior Observation root")
	}
	after[idx].Fingerprint += ":new-observation"
	after[idx].ClaimKind = "prior-authorization-observation:denied"

	deltas := actorContextR3ACClassifyRootChanges(base, after)
	got := actorContextR3ADExplainBoundRootChanges(reasons, deltas.Changes, bindings)

	if actorContextR3THasCategory(got.Impact.Reasons, actorContextR3PCategoryRuntimeOutcome) {
		t.Fatalf("R3AD promoted prior Observation delta to current runtime-outcome: %+v", got)
	}
	if len(got.ResidualRootDeltas) != 1 ||
		got.ResidualRootDeltas[0].Reason != actorContextR3ADResidualNoBinding ||
		got.ResidualRootDeltas[0].Delta.RootRole != actorContextR3ABRoleRuntimeObservation {
		t.Fatalf("R3AD did not preserve prior Observation as no-binding residual: %+v", got.ResidualRootDeltas)
	}
}

func TestActorContextR3ADTestContextAndCorroborationRemainNoBindingResiduals(t *testing.T) {
	base, reasons, bindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceAgent)
	after := actorContextR3ACCloneRoots(base)
	contextIndex := actorContextR3ACRootIndexByRole(after, actorContextR3ABRoleTestOnlyContext)
	policyIndex := actorContextR3ACRootIndexByRole(after, actorContextR3ABRoleCorroboration)
	if contextIndex < 0 || policyIndex < 0 {
		t.Fatal("R3AD setup lost context/corroboration roots")
	}
	after[contextIndex].Fingerprint += ":changed"
	after[policyIndex].Fingerprint += ":changed"

	deltas := actorContextR3ACClassifyRootChanges(base, after)
	got := actorContextR3ADExplainBoundRootChanges(reasons, deltas.Changes, bindings)

	if len(got.Impact.Reasons) != 0 {
		t.Fatalf("R3AD invented bounded reasons for unbound context/corroboration: %+v", got.Impact.Reasons)
	}
	if len(got.ResidualRootDeltas) != 2 {
		t.Fatalf("R3AD lost context/corroboration residuals: %+v", got.ResidualRootDeltas)
	}
	for _, residual := range got.ResidualRootDeltas {
		if residual.Reason != actorContextR3ADResidualNoBinding {
			t.Fatalf("R3AD misclassified unbound residual: %+v", residual)
		}
	}
}

func TestActorContextR3ADBoundReferenceRootSetChangeRemainsUnmappedResidual(t *testing.T) {
	base, reasons, bindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceHuman)
	after := actorContextR3ACCloneRoots(base)
	idx := actorContextR3ADRootIndexByClaim(after, "authorization-requirement-source")
	if idx < 0 {
		t.Fatal("R3AD setup lost Authorization requirement root")
	}
	after = append(after[:idx], after[idx+1:]...)

	deltas := actorContextR3ACClassifyRootChanges(base, after)
	got := actorContextR3ADExplainBoundRootChanges(reasons, deltas.Changes, bindings)

	if len(got.ResidualRootDeltas) != 1 ||
		got.ResidualRootDeltas[0].Reason != actorContextR3ADResidualNoExactClass ||
		got.ResidualRootDeltas[0].Delta.ChangeClass != actorContextR3ACChangeRootSet {
		t.Fatalf("R3AD coerced root-set change to nearest R3T class: %+v", got)
	}
}

func TestActorContextR3ADReferenceRoleWithoutBindingDoesNotInferDependency(t *testing.T) {
	base, reasons, _ := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceHuman)
	after := actorContextR3ACCloneRoots(base)
	idx := actorContextR3ADRootIndexByClaim(after, "authorization-requirement-source")
	if idx < 0 {
		t.Fatal("R3AD setup lost Authorization requirement root")
	}
	after[idx].Fingerprint += ":changed"

	deltas := actorContextR3ACClassifyRootChanges(base, after)
	got := actorContextR3ADExplainBoundRootChanges(reasons, deltas.Changes, nil)

	if len(got.Impact.Reasons) != 0 ||
		len(got.ResidualRootDeltas) != 1 ||
		got.ResidualRootDeltas[0].Reason != actorContextR3ADResidualNoBinding {
		t.Fatalf("R3AD inferred dependency from root role without explicit binding: %+v", got)
	}
}

func TestActorContextR3ADBindingWithoutSelectedReasonEdgeRemainsR3TUnselected(t *testing.T) {
	base, _, bindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceHuman)
	after := actorContextR3ACCloneRoots(base)
	idx := actorContextR3ADRootIndexByClaim(after, "authorization-requirement-source")
	if idx < 0 {
		t.Fatal("R3AD setup lost Authorization requirement root")
	}
	after[idx].Fingerprint += ":changed"

	deltas := actorContextR3ACClassifyRootChanges(base, after)
	got := actorContextR3ADExplainBoundRootChanges(
		actorContextR3SReasonEnvelopeProbe{
			Audience: actorContextR3PAudienceHuman,
			Purpose:  actorContextR3PPurposeOrientation,
		},
		deltas.Changes,
		bindings,
	)

	if len(got.Impact.Reasons) != 0 ||
		len(got.Impact.UnselectedChanges) != 1 ||
		len(got.ResidualRootDeltas) != 0 {
		t.Fatalf("R3AD binding without selected edge invented/dropped impact: %+v", got)
	}
}

func TestActorContextR3ADBridgeHasNoAutomaticResponseShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3ADBridgeResultProbe{})
	for _, forbidden := range []string{
		"Stale",
		"Fresh",
		"RebuildAll",
		"AutoRebuild",
		"AutoRefresh",
		"ShouldReevaluate",
		"ReevaluationRequired",
		"ReusePermitted",
		"RetryRequired",
		"NextAction",
		"HandoffOwner",
		"Authorized",
		"Allowed",
		"CanExecute",
		"Winner",
		"Priority",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3AD bridge introduced forbidden automatic response shortcut %q", forbidden)
		}
	}
}

func actorContextR3ADFreshnessFixture(
	t *testing.T,
	audience string,
) (
	[]actorContextR3ABProvenanceRootProbe,
	actorContextR3SReasonEnvelopeProbe,
	[]actorContextR3ADRootDependencyBindingProbe,
) {
	t.Helper()

	drilldown := actorContextR3ABBuildFreshnessDrilldown(t, audience)
	core := actorContextR3ABFreshnessCore(t)
	purpose := map[string]string{
		actorContextR3PAudienceHuman: actorContextR3PPurposeOrientation,
		actorContextR3PAudienceAgent: actorContextR3PPurposeResolutionInspection,
	}[audience]
	view, err := actorContextR3PSelectBoundedStopView(core, audience, purpose)
	if err != nil {
		t.Fatal(err)
	}
	deps := actorContextR3SCurrentBP06ReferenceEnvelope(t, repoPath())
	businessDep := actorContextR3ADReferenceDepByClass(t, deps, "business-procedure-reference")
	overlayDep := actorContextR3ADReferenceDepByClass(t, deps, "protected-target-overlay")
	businessKey := actorContextR3SReferenceKey(businessDep)
	overlayKey := actorContextR3SReferenceKey(overlayDep)

	edges := []actorContextR3SDependencyReasonEdgeProbe{
		{Category: actorContextR3PCategoryStopIdentity, DependencyKey: businessKey, ReasonCode: actorContextR3SReasonBusinessAddressSource},
		{Category: actorContextR3PCategoryMachineReasons, DependencyKey: businessKey, ReasonCode: actorContextR3SReasonAuthorizationRequirementSource},
		{Category: actorContextR3PCategoryMachineReasons, DependencyKey: overlayKey, ReasonCode: actorContextR3SReasonProtectedResourceSource},
		{Category: actorContextR3PCategoryResolutionNeeds, DependencyKey: overlayKey, ReasonCode: actorContextR3SReasonUnresolvedClassification},
		{Category: actorContextR3PCategorySourceRefs, DependencyKey: businessKey, ReasonCode: actorContextR3SReasonDrillDownLocator},
		{Category: actorContextR3PCategorySourceRefs, DependencyKey: overlayKey, ReasonCode: actorContextR3SReasonDrillDownLocator},
	}
	if audience == actorContextR3PAudienceAgent {
		edges = append(edges,
			actorContextR3SDependencyReasonEdgeProbe{
				Category: actorContextR3PCategoryKnownAuthorityFacts,
				DependencyKey: businessKey,
				ReasonCode: actorContextR3SReasonAuthorizationRequirementSource,
			},
			actorContextR3SDependencyReasonEdgeProbe{
				Category: actorContextR3PCategoryKnownAuthorityFacts,
				DependencyKey: overlayKey,
				ReasonCode: actorContextR3SReasonProtectedResourceSource,
			},
		)
	}
	reasons, err := actorContextR3SBuildReasonEnvelope(core, view, deps, edges)
	if err != nil {
		t.Fatal(err)
	}

	businessRoot := actorContextR3ADRootByClaim(t, drilldown.Roots, "authorization-requirement-source")
	overlayRoot := actorContextR3ADRootByClaim(t, drilldown.Roots, "protected-resource-source")
	bindings := []actorContextR3ADRootDependencyBindingProbe{
		{RootAddress: actorContextR3ACRootAddress(businessRoot), DependencyKey: businessKey},
		{RootAddress: actorContextR3ACRootAddress(overlayRoot), DependencyKey: overlayKey},
	}
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].RootAddress < bindings[j].RootAddress })

	return drilldown.Roots, reasons, bindings
}

func actorContextR3ADExplainBoundRootChanges(
	reasons actorContextR3SReasonEnvelopeProbe,
	rootChanges []actorContextR3ACRootChangeProbe,
	bindings []actorContextR3ADRootDependencyBindingProbe,
) actorContextR3ADBridgeResultProbe {
	bindingByRoot := map[string]string{}
	for _, binding := range bindings {
		if binding.RootAddress != "" && binding.DependencyKey != "" {
			bindingByRoot[binding.RootAddress] = binding.DependencyKey
		}
	}

	var observed []actorContextR3TObservedChangeProbe
	var residual []actorContextR3ADResidualRootDeltaProbe
	for _, change := range rootChanges {
		dependencyKey, bound := bindingByRoot[change.RootIdentity]
		if !bound {
			residual = append(residual, actorContextR3ADResidualRootDeltaProbe{
				Delta:  change,
				Reason: actorContextR3ADResidualNoBinding,
			})
			continue
		}
		changeClass, mapped := actorContextR3ADToR3TChangeClass(change.ChangeClass)
		if !mapped {
			residual = append(residual, actorContextR3ADResidualRootDeltaProbe{
				Delta:  change,
				Reason: actorContextR3ADResidualNoExactClass,
			})
			continue
		}
		observed = append(observed, actorContextR3TObservedChangeProbe{
			ChangeClass:   changeClass,
			DependencyKey: dependencyKey,
		})
	}
	sort.Slice(residual, func(i, j int) bool {
		if residual[i].Delta.RootIdentity != residual[j].Delta.RootIdentity {
			return residual[i].Delta.RootIdentity < residual[j].Delta.RootIdentity
		}
		if residual[i].Delta.ChangeClass != residual[j].Delta.ChangeClass {
			return residual[i].Delta.ChangeClass < residual[j].Delta.ChangeClass
		}
		return residual[i].Reason < residual[j].Reason
	})

	return actorContextR3ADBridgeResultProbe{
		Impact:             actorContextR3TExplainChanges(reasons, actorContextR3TUniqueChanges(observed)),
		ResidualRootDeltas: residual,
		AuthorityLimits: []string{
			"root-delta-does-not-imply-projection-dependency-change",
			"existing-r3r-r3s-dependency-key-is-reused-rather-than-parallel-root-key",
			"prior-observation-root-is-not-current-runtime-outcome-dependency",
			"test-only-context-and-corroboration-are-not-auto-bound",
			"unsupported-change-class-is-preserved-not-coerced",
			"binding-does-not-select-category-without-r3s-reason-edge",
			"residual-root-delta-does-not-mean-globally-irrelevant",
		},
	}
}

func actorContextR3ADToR3TChangeClass(changeClass string) (string, bool) {
	switch changeClass {
	case actorContextR3ACChangeReferenceEvidence:
		return actorContextR3TChangeReferenceInput, true
	case actorContextR3ACChangeLocatorRefresh:
		return actorContextR3TChangeLocatorRefresh, true
	case actorContextR3ACChangeRuntimeObservation:
		return actorContextR3TChangeRuntimeObservation, true
	default:
		return "", false
	}
}

func actorContextR3ADReferenceDepByClass(
	t *testing.T,
	deps actorContextR3RDependencyEnvelopeProbe,
	inputClass string,
) actorContextR3RReferenceArtifactDependencyProbe {
	t.Helper()
	for _, dep := range deps.ReferenceArtifacts {
		if dep.InputClass == inputClass {
			return dep
		}
	}
	t.Fatalf("R3AD setup missing Reference dependency class %q", inputClass)
	return actorContextR3RReferenceArtifactDependencyProbe{}
}

func actorContextR3ADRootByClaim(
	t *testing.T,
	roots []actorContextR3ABProvenanceRootProbe,
	claim string,
) actorContextR3ABProvenanceRootProbe {
	t.Helper()
	for _, root := range roots {
		if root.ClaimKind == claim {
			return root
		}
	}
	t.Fatalf("R3AD setup missing root claim %q", claim)
	return actorContextR3ABProvenanceRootProbe{}
}

func actorContextR3ADRootIndexByClaim(
	roots []actorContextR3ABProvenanceRootProbe,
	claim string,
) int {
	for i := range roots {
		if roots[i].ClaimKind == claim {
			return i
		}
	}
	return -1
}

func actorContextR3ADReasonsForCategory(
	values []actorContextR3TReevaluationReasonProbe,
	category string,
) []actorContextR3TReevaluationReasonProbe {
	out := []actorContextR3TReevaluationReasonProbe{}
	for _, value := range values {
		if value.Category == category {
			out = append(out, value)
		}
	}
	actorContextR3TSortReasons(out)
	return out
}

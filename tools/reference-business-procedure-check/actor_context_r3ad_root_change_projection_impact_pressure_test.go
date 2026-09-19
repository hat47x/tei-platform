package main

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

const (
	actorContextR3ADReasonRequirementRoot       = "freshness-requirement-root"
	actorContextR3ADReasonProtectedResourceRoot = "freshness-protected-resource-root"
	actorContextR3ADReasonPriorObservationRoot  = "freshness-prior-observation-root"
	actorContextR3ADReasonTestContextRoot       = "freshness-test-context-root"
	actorContextR3ADReasonRootLocator           = "provenance-root-locator"
)

type actorContextR3ADCategoryBindingProbe struct {
	Category         string
	DependencyReason string
}

type actorContextR3ADBridgeResultProbe struct {
	Impact              actorContextR3TImpactExplanationProbe
	ResidualRootChanges []actorContextR3ACRootChangeProbe
	AuthorityLimits     []string
}

func TestActorContextR3ADRequirementRootChangeKeepsSharedMachineReasonAcrossAudiences(t *testing.T) {
	base := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)
	after := actorContextR3ACCloneRoots(base.Roots)
	idx := actorContextR3ADRootIndexByClaim(after, "authorization-requirement-source")
	if idx < 0 {
		t.Fatal("R3AD setup lost Authorization requirement provenance root")
	}
	after[idx].Fingerprint += ":changed"
	assessment := actorContextR3ACClassifyRootChanges(base.Roots, after)

	human := actorContextR3ADFreshnessView(t, actorContextR3PAudienceHuman)
	agent := actorContextR3ADFreshnessView(t, actorContextR3PAudienceAgent)

	humanImpact := actorContextR3ADExplainRootChanges(human, base.Roots, assessment)
	agentImpact := actorContextR3ADExplainRootChanges(agent, base.Roots, assessment)

	humanMachine := actorContextR3ADReasonsForCategory(
		humanImpact.Impact.Reasons,
		actorContextR3PCategoryMachineReasons,
	)
	agentMachine := actorContextR3ADReasonsForCategory(
		agentImpact.Impact.Reasons,
		actorContextR3PCategoryMachineReasons,
	)
	if len(humanMachine) != 1 || !reflect.DeepEqual(humanMachine, agentMachine) {
		t.Fatalf("Human / Agent shared machine-reason impact diverged: human=%+v agent=%+v",
			humanMachine, agentMachine)
	}
	if humanMachine[0].ChangeClass != actorContextR3ACChangeReferenceEvidence ||
		humanMachine[0].DependencyReason != actorContextR3ADReasonRequirementRoot {
		t.Fatalf("R3AD requirement root change lost role-aware identity: %+v", humanMachine[0])
	}

	if actorContextR3ADHasCategory(humanImpact.Impact.Reasons, actorContextR3PCategoryKnownAuthorityFacts) {
		t.Fatalf("Human bounded view received omitted known-authority-facts impact: %+v", humanImpact)
	}
	if !actorContextR3ADHasCategory(agentImpact.Impact.Reasons, actorContextR3PCategoryKnownAuthorityFacts) {
		t.Fatalf("Agent bounded view lost selected known-authority-facts impact: %+v", agentImpact)
	}
}

func TestActorContextR3ADPriorObservationChangeDoesNotInventCurrentRuntimeOutcomeImpact(t *testing.T) {
	base := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)
	after := actorContextR3ACCloneRoots(base.Roots)
	idx := actorContextR3ACRootIndexByRole(after, actorContextR3ABRoleRuntimeObservation)
	if idx < 0 {
		t.Fatal("R3AD setup lost prior runtime Observation root")
	}
	after[idx].Fingerprint += ":new-observation"
	after[idx].ClaimKind = "prior-authorization-observation:denied"

	assessment := actorContextR3ACClassifyRootChanges(base.Roots, after)
	view := actorContextR3ADFreshnessView(t, actorContextR3PAudienceAgent)
	got := actorContextR3ADExplainRootChanges(view, base.Roots, assessment)

	if !actorContextR3ADHasReason(
		got.Impact.Reasons,
		actorContextR3PCategoryMachineReasons,
		actorContextR3ACChangeRuntimeObservation,
		actorContextR3ADReasonPriorObservationRoot,
	) {
		t.Fatalf("prior Observation change did not reach freshness machine-reason re-evaluation: %+v", got)
	}
	if actorContextR3ADHasCategory(got.Impact.Reasons, actorContextR3PCategoryRuntimeOutcome) {
		t.Fatalf("prior Observation revision was promoted to current runtime-outcome impact: %+v", got)
	}

	core := actorContextR3ABFreshnessCore(t)
	if core.RuntimeOutcome != "" {
		t.Fatalf("R3AD setup unexpectedly has current RuntimeOutcome: %+v", core)
	}
}

func TestActorContextR3ADTestOnlyContextChangeAffectsFreshnessStopMeaningWithoutAuthoringProductionAction(t *testing.T) {
	base := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceHuman)
	after := actorContextR3ACCloneRoots(base.Roots)
	idx := actorContextR3ACRootIndexByRole(after, actorContextR3ABRoleTestOnlyContext)
	if idx < 0 {
		t.Fatal("R3AD setup lost test-only Authorization context root")
	}
	after[idx].Fingerprint += ":changed"

	assessment := actorContextR3ACClassifyRootChanges(base.Roots, after)
	view := actorContextR3ADFreshnessView(t, actorContextR3PAudienceHuman)
	got := actorContextR3ADExplainRootChanges(view, base.Roots, assessment)

	for _, category := range []string{
		actorContextR3PCategoryStopIdentity,
		actorContextR3PCategoryMachineReasons,
		actorContextR3PCategoryResolutionNeeds,
	} {
		if !actorContextR3ADHasCategory(got.Impact.Reasons, category) {
			t.Fatalf("test-only context change did not re-evaluate selected freshness category %q: %+v",
				category, got)
		}
	}
	for _, reason := range got.Impact.Reasons {
		if reason.DependencyReason != actorContextR3ADReasonTestContextRoot {
			continue
		}
		if strings.Contains(reason.DependencyReason, "production-action-authority") {
			t.Fatalf("R3AD promoted test-only context change to production action authority: %+v", reason)
		}
	}
}

func TestActorContextR3ADCorroborationChangeRemainsUnselectedWithoutAuthorityBinding(t *testing.T) {
	base := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)
	after := actorContextR3ACCloneRoots(base.Roots)
	idx := actorContextR3ACRootIndexByRole(after, actorContextR3ABRoleCorroboration)
	if idx < 0 {
		t.Fatal("R3AD setup lost corroboration root")
	}
	after[idx].Fingerprint += ":changed"

	assessment := actorContextR3ACClassifyRootChanges(base.Roots, after)
	view := actorContextR3ADFreshnessView(t, actorContextR3PAudienceAgent)
	got := actorContextR3ADExplainRootChanges(view, base.Roots, assessment)

	if len(got.Impact.Reasons) != 0 || len(got.Impact.UnselectedChanges) != 0 {
		t.Fatalf("corroboration change was promoted into selected/unselected bounded impact: %+v", got)
	}
	if len(got.ResidualRootChanges) != 1 ||
		got.ResidualRootChanges[0].ChangeClass != actorContextR3ACChangeCorroboration {
		t.Fatalf("R3AD failed to retain unmapped corroboration delta as residual: %+v", got.ResidualRootChanges)
	}
}

func TestActorContextR3ADLocatorOnlyMoveAffectsSourceRefsOnly(t *testing.T) {
	base := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceHuman)
	after := actorContextR3ACCloneRoots(base.Roots)
	idx := actorContextR3ADRootIndexByClaim(after, "protected-resource-source")
	if idx < 0 {
		t.Fatal("R3AD setup lost protected-resource root")
	}
	after[idx].Locator += ".moved"

	assessment := actorContextR3ACClassifyRootChanges(base.Roots, after)
	view := actorContextR3ADFreshnessView(t, actorContextR3PAudienceHuman)
	got := actorContextR3ADExplainRootChanges(view, base.Roots, assessment)

	if len(got.Impact.Reasons) != 1 ||
		got.Impact.Reasons[0].Category != actorContextR3PCategorySourceRefs ||
		got.Impact.Reasons[0].ChangeClass != actorContextR3ACChangeLocatorRefresh ||
		got.Impact.Reasons[0].DependencyReason != actorContextR3ADReasonRootLocator {
		t.Fatalf("locator-only move leaked beyond source-refs: %+v", got)
	}
}

func TestActorContextR3ADUnknownAddedRootDoesNotBecomeGlobalImpact(t *testing.T) {
	base := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)
	after := actorContextR3ACCloneRoots(base.Roots)
	after = append(after, actorContextR3ABProvenanceRootProbe{
		Identity:    "future:unbound-root",
		Role:        "future-unbound-role",
		ClaimKind:   "future-unbound-claim",
		Locator:     "future/unbound",
		Fingerprint: "sha256:future",
	})

	assessment := actorContextR3ACClassifyRootChanges(base.Roots, after)
	view := actorContextR3ADFreshnessView(t, actorContextR3PAudienceAgent)
	got := actorContextR3ADExplainRootChanges(view, after, assessment)

	if len(got.Impact.Reasons) != 0 || len(got.Impact.UnselectedChanges) != 0 {
		t.Fatalf("unbound new root was expanded into bounded-category impact: %+v", got)
	}
	if len(got.ResidualRootChanges) != 1 ||
		got.ResidualRootChanges[0].ChangeClass != actorContextR3ACChangeRootSet {
		t.Fatalf("unbound root-set change was not retained as residual delta: %+v", got)
	}
}

func TestActorContextR3ADInputOrderDoesNotChangeNormalizedImpact(t *testing.T) {
	base := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)
	after := actorContextR3ACCloneRoots(base.Roots)

	req := actorContextR3ADRootIndexByClaim(after, "authorization-requirement-source")
	runtime := actorContextR3ACRootIndexByRole(after, actorContextR3ABRoleRuntimeObservation)
	if req < 0 || runtime < 0 {
		t.Fatal("R3AD setup lost required roots")
	}
	after[req].Fingerprint += ":changed"
	after[runtime].Fingerprint += ":changed"

	assessment := actorContextR3ACClassifyRootChanges(base.Roots, after)
	reversed := assessment
	reversed.Changes = append([]actorContextR3ACRootChangeProbe(nil), assessment.Changes...)
	for i, j := 0, len(reversed.Changes)-1; i < j; i, j = i+1, j-1 {
		reversed.Changes[i], reversed.Changes[j] = reversed.Changes[j], reversed.Changes[i]
	}

	view := actorContextR3ADFreshnessView(t, actorContextR3PAudienceAgent)
	left := actorContextR3ADExplainRootChanges(view, base.Roots, assessment)
	right := actorContextR3ADExplainRootChanges(view, base.Roots, reversed)
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("R3AD impact explanation became input-order dependent: left=%+v right=%+v", left, right)
	}
}

func TestActorContextR3ADBridgeHasNoAutomaticResponseShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3ADBridgeResultProbe{})
	for _, forbidden := range []string{
		"ShouldReevaluate",
		"ReevaluationRequired",
		"AutoReauthorize",
		"RetryRequired",
		"CacheValid",
		"AutoRefresh",
		"Winner",
		"NextAction",
		"HandoffOwner",
		"Authorized",
		"Allowed",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3AD bridge introduced forbidden automatic response shortcut %q", forbidden)
		}
	}
}

func actorContextR3ADFreshnessView(
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

func actorContextR3ADExplainRootChanges(
	view actorContextR3PBoundedStopViewProbe,
	roots []actorContextR3ABProvenanceRootProbe,
	assessment actorContextR3ACAssessmentProbe,
) actorContextR3ADBridgeResultProbe {
	result := actorContextR3ADBridgeResultProbe{
		AuthorityLimits: []string{
			"root-change-does-not-imply-all-categories-affected",
			"prior-observation-change-does-not-imply-current-runtime-outcome-change",
			"test-only-context-change-does-not-author-production-action",
			"corroboration-change-does-not-author-authority",
			"locator-change-does-not-imply-semantic-change",
			"affected-category-does-not-select-response-action",
		},
	}
	result.Impact.AuthorityLimits = append([]string(nil), result.AuthorityLimits...)

	selected := map[string]bool{}
	for _, category := range view.SelectedCategories {
		selected[category] = true
	}

	rootByAddress := map[string]actorContextR3ABProvenanceRootProbe{}
	for _, root := range roots {
		rootByAddress[actorContextR3ACRootAddress(root)] = root
	}

	for _, change := range assessment.Changes {
		root, found := rootByAddress[change.RootIdentity]
		if !found {
			result.ResidualRootChanges = append(result.ResidualRootChanges, change)
			continue
		}

		bindings := actorContextR3ADBindings(root, change.ChangeClass)
		if len(bindings) == 0 {
			result.ResidualRootChanges = append(result.ResidualRootChanges, change)
			continue
		}
		matched := false
		for _, binding := range bindings {
			if !selected[binding.Category] {
				continue
			}
			matched = true
			result.Impact.Reasons = append(result.Impact.Reasons, actorContextR3TReevaluationReasonProbe{
				Category:         binding.Category,
				ChangeClass:      change.ChangeClass,
				DependencyKey:    actorContextR3ADDependencyKey(change.RootIdentity),
				DependencyReason: binding.DependencyReason,
			})
		}
		if !matched {
			result.Impact.UnselectedChanges = append(result.Impact.UnselectedChanges, actorContextR3TObservedChangeProbe{
				ChangeClass:   change.ChangeClass,
				DependencyKey: actorContextR3ADDependencyKey(change.RootIdentity),
			})
		}
	}

	result.Impact.Reasons = actorContextR3TUniqueReasons(result.Impact.Reasons)
	result.Impact.UnselectedChanges = actorContextR3TUniqueChanges(result.Impact.UnselectedChanges)
	sort.Slice(result.ResidualRootChanges, func(i, j int) bool {
		if result.ResidualRootChanges[i].RootIdentity != result.ResidualRootChanges[j].RootIdentity {
			return result.ResidualRootChanges[i].RootIdentity < result.ResidualRootChanges[j].RootIdentity
		}
		if result.ResidualRootChanges[i].RootRole != result.ResidualRootChanges[j].RootRole {
			return result.ResidualRootChanges[i].RootRole < result.ResidualRootChanges[j].RootRole
		}
		return result.ResidualRootChanges[i].ChangeClass < result.ResidualRootChanges[j].ChangeClass
	})
	sort.Strings(result.AuthorityLimits)
	sort.Strings(result.Impact.AuthorityLimits)
	return result
}

func actorContextR3ADBindings(
	root actorContextR3ABProvenanceRootProbe,
	changeClass string,
) []actorContextR3ADCategoryBindingProbe {
	if changeClass == actorContextR3ACChangeLocatorRefresh {
		switch root.ClaimKind {
		case "authorization-requirement-source",
			"protected-resource-source",
			"prior-authorization-observation:allowed",
			"prior-authorization-observation:denied":
			return []actorContextR3ADCategoryBindingProbe{{
				Category:         actorContextR3PCategorySourceRefs,
				DependencyReason: actorContextR3ADReasonRootLocator,
			}}
		default:
			return nil
		}
	}

	contentLike := changeClass == actorContextR3ACChangeReferenceEvidence ||
		changeClass == actorContextR3ACChangeRuntimeObservation ||
		changeClass == actorContextR3ACChangeTestOnlyContext ||
		changeClass == actorContextR3ACChangeRootSet
	if !contentLike {
		return nil
	}

	switch {
	case root.ClaimKind == "authorization-requirement-source":
		return []actorContextR3ADCategoryBindingProbe{
			{Category: actorContextR3PCategoryMachineReasons, DependencyReason: actorContextR3ADReasonRequirementRoot},
			{Category: actorContextR3PCategoryKnownAuthorityFacts, DependencyReason: actorContextR3SReasonAuthorizationRequirementSource},
		}
	case root.ClaimKind == "protected-resource-source":
		return []actorContextR3ADCategoryBindingProbe{
			{Category: actorContextR3PCategoryMachineReasons, DependencyReason: actorContextR3ADReasonProtectedResourceRoot},
			{Category: actorContextR3PCategoryResolutionNeeds, DependencyReason: actorContextR3ADReasonProtectedResourceRoot},
			{Category: actorContextR3PCategoryKnownAuthorityFacts, DependencyReason: actorContextR3SReasonProtectedResourceSource},
		}
	case root.Role == actorContextR3ABRoleRuntimeObservation:
		return []actorContextR3ADCategoryBindingProbe{
			{Category: actorContextR3PCategoryMachineReasons, DependencyReason: actorContextR3ADReasonPriorObservationRoot},
		}
	case root.Role == actorContextR3ABRoleTestOnlyContext:
		return []actorContextR3ADCategoryBindingProbe{
			{Category: actorContextR3PCategoryStopIdentity, DependencyReason: actorContextR3ADReasonTestContextRoot},
			{Category: actorContextR3PCategoryMachineReasons, DependencyReason: actorContextR3ADReasonTestContextRoot},
			{Category: actorContextR3PCategoryResolutionNeeds, DependencyReason: actorContextR3ADReasonTestContextRoot},
		}
	default:
		return nil
	}
}

func actorContextR3ADDependencyKey(rootAddress string) string {
	return "provenance-root:" + rootAddress
}

func actorContextR3ADRootIndexByClaim(
	values []actorContextR3ABProvenanceRootProbe,
	claim string,
) int {
	for i := range values {
		if values[i].ClaimKind == claim {
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

func actorContextR3ADHasReason(
	values []actorContextR3TReevaluationReasonProbe,
	category string,
	changeClass string,
	dependencyReason string,
) bool {
	for _, value := range values {
		if value.Category == category &&
			value.ChangeClass == changeClass &&
			value.DependencyReason == dependencyReason {
			return true
		}
	}
	return false
}

func actorContextR3ADHasCategory(
	values []actorContextR3TReevaluationReasonProbe,
	category string,
) bool {
	for _, value := range values {
		if value.Category == category {
			return true
		}
	}
	return false
}

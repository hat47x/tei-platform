package main

import (
	"reflect"
	"sort"
	"testing"

	"tei/internal/sourceindex"
)

const (
	actorContextR3TChangeSemanticSource       = "semantic-source-changed"
	actorContextR3TChangeReferenceInput       = "reference-input-changed"
	actorContextR3TChangeLocatorRefresh       = "locator-refresh-required"
	actorContextR3TChangeRuntimeObservation   = "runtime-observation-changed"
	actorContextR3TChangeProfile              = "profile-changed"

	actorContextR3TReasonSemanticSubjectMeaning = "semantic-subject-meaning-source"
	actorContextR3TReasonProfileSelection       = "projection-profile-selection-changed"
)

type actorContextR3TObservedChangeProbe struct {
	ChangeClass   string
	DependencyKey string
	CategoryHint  string
}

type actorContextR3TReevaluationReasonProbe struct {
	Category         string
	ChangeClass      string
	DependencyKey    string
	DependencyReason string
}

type actorContextR3TImpactExplanationProbe struct {
	Reasons           []actorContextR3TReevaluationReasonProbe
	UnselectedChanges []actorContextR3TObservedChangeProbe
	AuthorityLimits   []string
}

func TestActorContextR3TReferenceContentChangeAffectsOnlyCompatibleSelectedReasons(t *testing.T) {
	core := actorContextR3QCurrentBP06Core(t)
	view, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	deps := actorContextR3SCurrentBP06ReferenceEnvelope(t, repoPath())

	businessKey := actorContextR3SReferenceKey(deps.ReferenceArtifacts[0])
	overlayKey := actorContextR3SReferenceKey(deps.ReferenceArtifacts[1])
	edges := []actorContextR3SDependencyReasonEdgeProbe{
		{
			Category:      actorContextR3PCategoryStopIdentity,
			DependencyKey: businessKey,
			ReasonCode:    actorContextR3SReasonBusinessAddressSource,
		},
		{
			Category:      actorContextR3PCategoryMachineReasons,
			DependencyKey: businessKey,
			ReasonCode:    actorContextR3SReasonAuthorizationRequirementSource,
		},
		{
			Category:      actorContextR3PCategoryMachineReasons,
			DependencyKey: overlayKey,
			ReasonCode:    actorContextR3SReasonProtectedResourceSource,
		},
		{
			Category:      actorContextR3PCategorySourceRefs,
			DependencyKey: businessKey,
			ReasonCode:    actorContextR3SReasonDrillDownLocator,
		},
	}
	reasons, err := actorContextR3SBuildReasonEnvelope(core, view, deps, edges)
	if err != nil {
		t.Fatal(err)
	}

	after := actorContextR3TCloneEnvelope(deps)
	after.ReferenceArtifacts[0].ContentFingerprint = "sha256:changed-business-reference"
	changes := actorContextR3TChangesFromEnvelopes(deps, after)
	got := actorContextR3TExplainChanges(reasons, changes)

	want := []actorContextR3TReevaluationReasonProbe{
		{
			Category:         actorContextR3PCategoryMachineReasons,
			ChangeClass:      actorContextR3TChangeReferenceInput,
			DependencyKey:    businessKey,
			DependencyReason: actorContextR3SReasonAuthorizationRequirementSource,
		},
		{
			Category:         actorContextR3PCategoryStopIdentity,
			ChangeClass:      actorContextR3TChangeReferenceInput,
			DependencyKey:    businessKey,
			DependencyReason: actorContextR3SReasonBusinessAddressSource,
		},
	}
	actorContextR3TSortReasons(want)
	if !reflect.DeepEqual(got.Reasons, want) {
		t.Fatalf("Reference content change affected wrong selected categories: got=%+v want=%+v", got.Reasons, want)
	}
	for _, reason := range got.Reasons {
		if reason.Category == actorContextR3PCategorySourceRefs {
			t.Fatalf("content-only change incorrectly became locator refresh impact: %+v", reason)
		}
	}
}

func TestActorContextR3TOmittedKnownFactChangeIsUnselectedForHumanButSelectedForAgent(t *testing.T) {
	core := actorContextR3QCurrentBP06Core(t)
	dep := actorContextR3RReferenceArtifactDependencyProbe{
		InputClass:          "test-only-extra-known-fact",
		ContentFingerprint: "sha256:before",
		Locator:             "test-only/known-fact.ref",
	}
	before := actorContextR3RDependencyEnvelopeProbe{
		ReferenceArtifacts: []actorContextR3RReferenceArtifactDependencyProbe{dep},
	}
	after := actorContextR3TCloneEnvelope(before)
	after.ReferenceArtifacts[0].ContentFingerprint = "sha256:after"
	change := actorContextR3TChangesFromEnvelopes(before, after)

	human, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	humanReasons, err := actorContextR3SBuildReasonEnvelope(core, human, before, nil)
	if err != nil {
		t.Fatal(err)
	}
	humanImpact := actorContextR3TExplainChanges(humanReasons, change)
	if len(humanImpact.Reasons) != 0 || len(humanImpact.UnselectedChanges) != 1 {
		t.Fatalf("omitted Human known-fact dependency became selected impact: %+v", humanImpact)
	}

	agent, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceAgent,
		actorContextR3PPurposeResolutionInspection,
	)
	if err != nil {
		t.Fatal(err)
	}
	edge := actorContextR3SDependencyReasonEdgeProbe{
		Category:      actorContextR3PCategoryKnownAuthorityFacts,
		DependencyKey: actorContextR3SReferenceKey(dep),
		ReasonCode:    actorContextR3SReasonAuthorizationRequirementSource,
	}
	agentReasons, err := actorContextR3SBuildReasonEnvelope(core, agent, before, []actorContextR3SDependencyReasonEdgeProbe{edge})
	if err != nil {
		t.Fatal(err)
	}
	agentImpact := actorContextR3TExplainChanges(agentReasons, change)
	if len(agentImpact.Reasons) != 1 ||
		agentImpact.Reasons[0].Category != actorContextR3PCategoryKnownAuthorityFacts ||
		agentImpact.Reasons[0].ChangeClass != actorContextR3TChangeReferenceInput {
		t.Fatalf("selected Agent known-fact change lost impact reason: %+v", agentImpact)
	}
}

func TestActorContextR3TSemanticContentAndLocatorChangesRemainDifferent(t *testing.T) {
	core := actorContextR3QCurrentBP06Core(t)
	core.SourceRefs = actorContextR0SortedUnique(append(core.SourceRefs, "workspace:order_entry.submit_order"))
	view, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	dep := actorContextR3RSemanticSubjectDependencyProbe{
		Subject: sourceindex.Subject{Kind: "application-pattern", ID: "order_entry.submit_order"},
		ContentFingerprint: "sha256:before",
		Locator:            "applications/order-entry/patterns/submit-order.tei",
		Role:               sourceindex.RoleDefinition,
	}
	before := actorContextR3RDependencyEnvelopeProbe{
		SemanticSubjects: []actorContextR3RSemanticSubjectDependencyProbe{dep},
	}
	semanticEdge := actorContextR3SDependencyReasonEdgeProbe{
		Category:      actorContextR3PCategoryMachineReasons,
		DependencyKey: actorContextR3SSemanticKey(dep),
		ReasonCode:    actorContextR3TReasonSemanticSubjectMeaning,
	}
	locatorEdge := actorContextR3SDependencyReasonEdgeProbe{
		Category:      actorContextR3PCategorySourceRefs,
		DependencyKey: actorContextR3SSemanticKey(dep),
		ReasonCode:    actorContextR3SReasonSourceIndexDefinitionLocator,
	}
	reasonEnvelope, err := actorContextR3SBuildReasonEnvelope(core, view, before, []actorContextR3SDependencyReasonEdgeProbe{
		semanticEdge,
		locatorEdge,
	})
	if err != nil {
		t.Fatal(err)
	}

	contentChanged := actorContextR3TCloneEnvelope(before)
	contentChanged.SemanticSubjects[0].ContentFingerprint = "sha256:after"
	contentImpact := actorContextR3TExplainChanges(
		reasonEnvelope,
		actorContextR3TChangesFromEnvelopes(before, contentChanged),
	)
	if !actorContextR3THasReason(
		contentImpact.Reasons,
		actorContextR3PCategoryMachineReasons,
		actorContextR3TChangeSemanticSource,
		actorContextR3TReasonSemanticSubjectMeaning,
	) {
		t.Fatalf("semantic subject content change lost machine-reason impact: %+v", contentImpact)
	}
	if actorContextR3THasCategory(contentImpact.Reasons, actorContextR3PCategorySourceRefs) {
		t.Fatalf("semantic content change incorrectly became locator-only impact: %+v", contentImpact)
	}

	locatorChanged := actorContextR3TCloneEnvelope(before)
	locatorChanged.SemanticSubjects[0].Locator = "applications/order-entry/interactions/submit-order.tei"
	locatorImpact := actorContextR3TExplainChanges(
		reasonEnvelope,
		actorContextR3TChangesFromEnvelopes(before, locatorChanged),
	)
	if !actorContextR3THasReason(
		locatorImpact.Reasons,
		actorContextR3PCategorySourceRefs,
		actorContextR3TChangeLocatorRefresh,
		actorContextR3SReasonSourceIndexDefinitionLocator,
	) {
		t.Fatalf("locator-only move lost source-ref refresh reason: %+v", locatorImpact)
	}
	if actorContextR3THasCategory(locatorImpact.Reasons, actorContextR3PCategoryMachineReasons) {
		t.Fatalf("locator-only move incorrectly staled semantic machine reason: %+v", locatorImpact)
	}
}

func TestActorContextR3TRuntimeObservationChangeAffectsRuntimeOutcomeOnly(t *testing.T) {
	core := actorContextR3RDeniedCoreForReason(t)
	view, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	allowed := actorContextR3RRuntimeObservationDependencyProbe{
		BusinessAddress: "order-entry.order-handling/BP-06",
		SubjectID:       "subject-order-viewer",
		Action:          "execute",
		Resource:        "order_entry.submit_order",
		Outcome:         "allowed",
	}
	before := actorContextR3RDependencyEnvelopeProbe{
		RuntimeObservations: []actorContextR3RRuntimeObservationDependencyProbe{allowed},
	}
	edge := actorContextR3SDependencyReasonEdgeProbe{
		Category:      actorContextR3PCategoryRuntimeOutcome,
		DependencyKey: actorContextR3SRuntimeKey(allowed),
		ReasonCode:    actorContextR3SReasonExactRuntimeObservation,
	}
	reasonEnvelope, err := actorContextR3SBuildReasonEnvelope(core, view, before, []actorContextR3SDependencyReasonEdgeProbe{edge})
	if err != nil {
		t.Fatal(err)
	}

	after := actorContextR3TCloneEnvelope(before)
	after.RuntimeObservations[0].Outcome = "denied"
	got := actorContextR3TExplainChanges(
		reasonEnvelope,
		actorContextR3TChangesFromEnvelopes(before, after),
	)
	if !actorContextR3THasReason(
		got.Reasons,
		actorContextR3PCategoryRuntimeOutcome,
		actorContextR3TChangeRuntimeObservation,
		actorContextR3SReasonExactRuntimeObservation,
	) {
		t.Fatalf("runtime Observation change lost runtime-outcome impact: %+v", got)
	}
	if len(got.Reasons) != 1 {
		t.Fatalf("runtime-only change leaked into static categories: %+v", got.Reasons)
	}
}

func TestActorContextR3TProfilePartitionChangeIsProfileReasonNotSourceChange(t *testing.T) {
	core := actorContextR3QCurrentBP06Core(t)
	before, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	after := before
	after.SelectedCategories = actorContextR3QRemoveString(
		after.SelectedCategories,
		actorContextR3PCategoryRuntimeOutcome,
	)
	after.OmittedCategories = append(after.OmittedCategories, actorContextR3PCategoryRuntimeOutcome)
	if err := actorContextR3PValidateCategoryPartition(after.SelectedCategories, after.OmittedCategories); err != nil {
		t.Fatal(err)
	}

	changes := actorContextR3TProfileChanges(before, after)
	got := actorContextR3TExplainChanges(
		actorContextR3SReasonEnvelopeProbe{Audience: before.Audience, Purpose: before.Purpose},
		changes,
	)
	if !actorContextR3THasReason(
		got.Reasons,
		actorContextR3PCategoryRuntimeOutcome,
		actorContextR3TChangeProfile,
		actorContextR3TReasonProfileSelection,
	) {
		t.Fatalf("profile selection change was not explained as profile change: %+v", got)
	}
	if len(got.Reasons) != 1 {
		t.Fatalf("profile-only change leaked into source/runtime reasons: %+v", got.Reasons)
	}
}

func TestActorContextR3TExplanationHasNoRebuildOrAuthorityShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3TImpactExplanationProbe{})
	for _, forbidden := range []string{
		"Stale",
		"Fresh",
		"RebuildAll",
		"AutoRebuild",
		"ImpactWinner",
		"Winner",
		"NextAction",
		"Confidence",
		"Authorized",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3T impact explanation introduced forbidden shortcut field %q", forbidden)
		}
	}
}

func actorContextR3TChangesFromEnvelopes(
	before actorContextR3RDependencyEnvelopeProbe,
	after actorContextR3RDependencyEnvelopeProbe,
) []actorContextR3TObservedChangeProbe {
	changes := []actorContextR3TObservedChangeProbe{}

	afterSemantic := map[sourceindex.Subject]actorContextR3RSemanticSubjectDependencyProbe{}
	for _, dep := range after.SemanticSubjects {
		afterSemantic[dep.Subject] = dep
	}
	for _, dep := range before.SemanticSubjects {
		next, ok := afterSemantic[dep.Subject]
		if !ok || dep.ContentFingerprint != next.ContentFingerprint || dep.Role != next.Role {
			changes = append(changes, actorContextR3TObservedChangeProbe{
				ChangeClass:   actorContextR3TChangeSemanticSource,
				DependencyKey: actorContextR3SSemanticKey(dep),
			})
		}
		if ok && dep.Locator != next.Locator {
			changes = append(changes, actorContextR3TObservedChangeProbe{
				ChangeClass:   actorContextR3TChangeLocatorRefresh,
				DependencyKey: actorContextR3SSemanticKey(dep),
			})
		}
	}

	afterReference := map[string]actorContextR3RReferenceArtifactDependencyProbe{}
	for _, dep := range after.ReferenceArtifacts {
		afterReference[dep.InputClass] = dep
	}
	for _, dep := range before.ReferenceArtifacts {
		next, ok := afterReference[dep.InputClass]
		if !ok || dep.ContentFingerprint != next.ContentFingerprint {
			changes = append(changes, actorContextR3TObservedChangeProbe{
				ChangeClass:   actorContextR3TChangeReferenceInput,
				DependencyKey: actorContextR3SReferenceKey(dep),
			})
		}
		if ok && dep.Locator != next.Locator {
			changes = append(changes, actorContextR3TObservedChangeProbe{
				ChangeClass:   actorContextR3TChangeLocatorRefresh,
				DependencyKey: actorContextR3SReferenceKey(dep),
			})
		}
	}

	afterRuntime := map[string]actorContextR3RRuntimeObservationDependencyProbe{}
	for _, dep := range after.RuntimeObservations {
		afterRuntime[actorContextR3TRuntimeIdentity(dep)] = dep
	}
	for _, dep := range before.RuntimeObservations {
		next, ok := afterRuntime[actorContextR3TRuntimeIdentity(dep)]
		if !ok || !reflect.DeepEqual(dep, next) {
			changes = append(changes, actorContextR3TObservedChangeProbe{
				ChangeClass:   actorContextR3TChangeRuntimeObservation,
				DependencyKey: actorContextR3SRuntimeKey(dep),
			})
		}
	}

	return actorContextR3TUniqueChanges(changes)
}

func actorContextR3TProfileChanges(
	before actorContextR3PBoundedStopViewProbe,
	after actorContextR3PBoundedStopViewProbe,
) []actorContextR3TObservedChangeProbe {
	beforeSelected := map[string]bool{}
	afterSelected := map[string]bool{}
	for _, category := range before.SelectedCategories {
		beforeSelected[category] = true
	}
	for _, category := range after.SelectedCategories {
		afterSelected[category] = true
	}

	all := map[string]bool{}
	for category := range beforeSelected {
		all[category] = true
	}
	for category := range afterSelected {
		all[category] = true
	}

	changes := []actorContextR3TObservedChangeProbe{}
	for category := range all {
		if beforeSelected[category] == afterSelected[category] {
			continue
		}
		changes = append(changes, actorContextR3TObservedChangeProbe{
			ChangeClass:   actorContextR3TChangeProfile,
			DependencyKey: "profile:" + before.Audience + ":" + before.Purpose,
			CategoryHint:  category,
		})
	}
	return actorContextR3TUniqueChanges(changes)
}

func actorContextR3TExplainChanges(
	reasons actorContextR3SReasonEnvelopeProbe,
	changes []actorContextR3TObservedChangeProbe,
) actorContextR3TImpactExplanationProbe {
	result := actorContextR3TImpactExplanationProbe{
		AuthorityLimits: []string{
			"impact-explanation-does-not-trigger-rebuild",
			"unselected-change-does-not-mean-globally-irrelevant",
			"affected-category-does-not-imply-authority-change",
			"change-class-does-not-imply-next-action",
		},
	}

	for _, change := range changes {
		if change.ChangeClass == actorContextR3TChangeProfile {
			result.Reasons = append(result.Reasons, actorContextR3TReevaluationReasonProbe{
				Category:         change.CategoryHint,
				ChangeClass:      change.ChangeClass,
				DependencyKey:    change.DependencyKey,
				DependencyReason: actorContextR3TReasonProfileSelection,
			})
			continue
		}

		matched := false
		for _, edge := range reasons.Edges {
			if edge.DependencyKey != change.DependencyKey ||
				!actorContextR3TReasonSupportsChange(edge.ReasonCode, change.ChangeClass) {
				continue
			}
			matched = true
			result.Reasons = append(result.Reasons, actorContextR3TReevaluationReasonProbe{
				Category:         edge.Category,
				ChangeClass:      change.ChangeClass,
				DependencyKey:    change.DependencyKey,
				DependencyReason: edge.ReasonCode,
			})
		}
		if !matched {
			result.UnselectedChanges = append(result.UnselectedChanges, change)
		}
	}

	result.Reasons = actorContextR3TUniqueReasons(result.Reasons)
	result.UnselectedChanges = actorContextR3TUniqueChanges(result.UnselectedChanges)
	return result
}

func actorContextR3TReasonSupportsChange(reason string, changeClass string) bool {
	switch changeClass {
	case actorContextR3TChangeSemanticSource:
		return reason == actorContextR3TReasonSemanticSubjectMeaning
	case actorContextR3TChangeReferenceInput:
		switch reason {
		case actorContextR3SReasonBusinessAddressSource,
			actorContextR3SReasonAuthorizationRequirementSource,
			actorContextR3SReasonProtectedResourceSource,
			actorContextR3SReasonUnresolvedClassification:
			return true
		default:
			return false
		}
	case actorContextR3TChangeLocatorRefresh:
		return reason == actorContextR3SReasonDrillDownLocator ||
			reason == actorContextR3SReasonSourceIndexDefinitionLocator
	case actorContextR3TChangeRuntimeObservation:
		return reason == actorContextR3SReasonExactRuntimeObservation
	default:
		return false
	}
}

func actorContextR3TRuntimeIdentity(dep actorContextR3RRuntimeObservationDependencyProbe) string {
	return dep.BusinessAddress + "|" + dep.SubjectID + "|" + dep.Action + "|" + dep.Resource
}

func actorContextR3TCloneEnvelope(
	in actorContextR3RDependencyEnvelopeProbe,
) actorContextR3RDependencyEnvelopeProbe {
	out := in
	out.SemanticSubjects = append([]actorContextR3RSemanticSubjectDependencyProbe(nil), in.SemanticSubjects...)
	out.ReferenceArtifacts = append([]actorContextR3RReferenceArtifactDependencyProbe(nil), in.ReferenceArtifacts...)
	out.RuntimeObservations = append([]actorContextR3RRuntimeObservationDependencyProbe(nil), in.RuntimeObservations...)
	return out
}

func actorContextR3TUniqueChanges(
	values []actorContextR3TObservedChangeProbe,
) []actorContextR3TObservedChangeProbe {
	seen := map[string]actorContextR3TObservedChangeProbe{}
	for _, value := range values {
		key := value.ChangeClass + "|" + value.DependencyKey + "|" + value.CategoryHint
		seen[key] = value
	}
	out := make([]actorContextR3TObservedChangeProbe, 0, len(seen))
	for _, value := range seen {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ChangeClass != out[j].ChangeClass {
			return out[i].ChangeClass < out[j].ChangeClass
		}
		if out[i].DependencyKey != out[j].DependencyKey {
			return out[i].DependencyKey < out[j].DependencyKey
		}
		return out[i].CategoryHint < out[j].CategoryHint
	})
	return out
}

func actorContextR3TUniqueReasons(
	values []actorContextR3TReevaluationReasonProbe,
) []actorContextR3TReevaluationReasonProbe {
	seen := map[string]actorContextR3TReevaluationReasonProbe{}
	for _, value := range values {
		key := value.Category + "|" + value.ChangeClass + "|" + value.DependencyKey + "|" + value.DependencyReason
		seen[key] = value
	}
	out := make([]actorContextR3TReevaluationReasonProbe, 0, len(seen))
	for _, value := range seen {
		out = append(out, value)
	}
	actorContextR3TSortReasons(out)
	return out
}

func actorContextR3TSortReasons(values []actorContextR3TReevaluationReasonProbe) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Category != values[j].Category {
			return values[i].Category < values[j].Category
		}
		if values[i].ChangeClass != values[j].ChangeClass {
			return values[i].ChangeClass < values[j].ChangeClass
		}
		if values[i].DependencyKey != values[j].DependencyKey {
			return values[i].DependencyKey < values[j].DependencyKey
		}
		return values[i].DependencyReason < values[j].DependencyReason
	})
}

func actorContextR3THasReason(
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

func actorContextR3THasCategory(
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

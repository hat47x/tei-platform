package main

import (
	"reflect"
	"sort"
	"testing"
)

type actorContextR6MPlaneCardinalityDeltaProbe struct {
	PlaneClass          string `json:"planeClass"`
	ScopeDisposition    string `json:"scopeDisposition"`
	BeforeObservedCount int    `json:"beforeObservedCount"`
	AfterObservedCount  int    `json:"afterObservedCount"`
	CountDelta          int    `json:"countDelta"`
}

type actorContextR6MSameScopeObservationDeltaProbe struct {
	ScopeIdentity  string                                      `json:"scopeIdentity"`
	BeforeAudience string                                      `json:"beforeAudience"`
	BeforePurpose  string                                      `json:"beforePurpose"`
	AfterAudience  string                                      `json:"afterAudience"`
	AfterPurpose   string                                      `json:"afterPurpose"`
	PlaneDeltas    []actorContextR6MPlaneCardinalityDeltaProbe `json:"planeDeltas"`
	AuthorityLimits []string                                   `json:"authorityLimits"`
}

func TestActorContextR6MHumanAgentDifferenceIsSameScopeObservationDelta(t *testing.T) {
	root := repoPath()
	qualified := actorContextR6CCurrentBP06QualifiedKnowledge(t, root)
	humanHandoff, agentHandoff, _ := actorContextR6IBP06AudienceHandoffs(t, root, qualified)

	humanMaterialized, err := actorContextR6IBuildBoundedMaterialization(qualified, humanHandoff)
	if err != nil {
		t.Fatal(err)
	}
	agentMaterialized, err := actorContextR6IBuildBoundedMaterialization(qualified, agentHandoff)
	if err != nil {
		t.Fatal(err)
	}

	expectations := actorContextR6LRoutedOnlyExpectations()
	human, err := actorContextR6LBindScope(
		actorContextR6LAssemblyFromR6I(humanMaterialized),
		expectations,
	)
	if err != nil {
		t.Fatal(err)
	}
	agent, err := actorContextR6LBindScope(
		actorContextR6LAssemblyFromR6I(agentMaterialized),
		expectations,
	)
	if err != nil {
		t.Fatal(err)
	}

	got, err := actorContextR6MCompareSameScopeObservations(human, agent)
	if err != nil {
		t.Fatal(err)
	}

	if got.ScopeIdentity != human.ScopeIdentity ||
		got.ScopeIdentity != agent.ScopeIdentity {
		t.Fatalf(
			"R6M same-scope comparison lost shared scope identity: got=%q human=%q agent=%q",
			got.ScopeIdentity,
			human.ScopeIdentity,
			agent.ScopeIdentity,
		)
	}

	routedKnowledge := actorContextR6MPlaneDelta(
		t,
		got.PlaneDeltas,
		actorContextR6KPlaneRoutedKnowledge,
	)
	if routedKnowledge.CountDelta != 0 ||
		routedKnowledge.BeforeObservedCount != 1 ||
		routedKnowledge.AfterObservedCount != 1 {
		t.Fatalf(
			"R6M Human/Agent comparison changed shared knowledge body cardinality: %+v",
			routedKnowledge,
		)
	}

	routedUse := actorContextR6MPlaneDelta(
		t,
		got.PlaneDeltas,
		actorContextR6KPlaneRoutedUse,
	)
	if routedUse.BeforeObservedCount != 1 ||
		routedUse.AfterObservedCount != 2 ||
		routedUse.CountDelta != 1 {
		t.Fatalf(
			"R6M lost bounded use cardinality delta under same scope: %+v",
			routedUse,
		)
	}

	for _, plane := range []string{
		actorContextR6KPlaneConsumerReference,
		actorContextR6KPlaneConsumerRuntime,
		actorContextR6KPlaneConsumerSemantic,
	} {
		delta := actorContextR6MPlaneDelta(t, got.PlaneDeltas, plane)
		if delta.CountDelta != 0 ||
			delta.ScopeDisposition != actorContextR6KScopeOutOfScope {
			t.Fatalf(
				"R6M same-scope comparison fabricated consumer-local observation delta for %q: %+v",
				plane,
				delta,
			)
		}
	}
}

func TestActorContextR6MRejectsComparisonAcrossDifferentScopeIdentity(t *testing.T) {
	first := actorContextR6LScopeBindingProbe{
		Audience:      actorContextR3PAudienceHuman,
		Purpose:       actorContextR3PPurposeOrientation,
		ScopeIdentity: "scope:test-only:first",
		Planes: []actorContextR6KPlaneObservationProbe{{
			PlaneClass:       actorContextR6KPlaneRoutedKnowledge,
			ScopeDisposition: actorContextR6KScopeRequired,
			ObservedCount:    1,
		}},
	}
	second := first
	second.ScopeIdentity = "scope:test-only:second"

	if _, err := actorContextR6MCompareSameScopeObservations(first, second); err == nil {
		t.Fatal("R6M compared cardinality across different semantic input scopes")
	}
}

func TestActorContextR6MRejectsDispositionDriftUnderClaimedSameScope(t *testing.T) {
	scopeID, err := actorContextR6LScopeIdentityFromExpectations(
		actorContextR6LRoutedOnlyExpectations(),
	)
	if err != nil {
		t.Fatal(err)
	}

	first := actorContextR6LScopeBindingProbe{
		Audience:      actorContextR3PAudienceHuman,
		Purpose:       actorContextR3PPurposeOrientation,
		ScopeIdentity: scopeID,
		Planes: []actorContextR6KPlaneObservationProbe{
			{
				PlaneClass:       actorContextR6KPlaneRoutedKnowledge,
				ScopeDisposition: actorContextR6KScopeRequired,
				ObservedCount:    1,
			},
		},
	}
	second := first
	second.Planes = []actorContextR6KPlaneObservationProbe{
		{
			PlaneClass:       actorContextR6KPlaneRoutedKnowledge,
			ScopeDisposition: actorContextR6KScopeOutOfScope,
			ObservedCount:    0,
		},
	}

	if _, err := actorContextR6MCompareSameScopeObservations(first, second); err == nil {
		t.Fatal("R6M accepted disposition drift under the same scope identity")
	}
}

func TestActorContextR6MPlaneOrderDoesNotChangeObservationDelta(t *testing.T) {
	root := repoPath()
	qualified := actorContextR6CCurrentBP06QualifiedKnowledge(t, root)
	humanHandoff, _, _ := actorContextR6IBP06AudienceHandoffs(t, root, qualified)
	materialized, err := actorContextR6IBuildBoundedMaterialization(qualified, humanHandoff)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := actorContextR6LBindScope(
		actorContextR6LAssemblyFromR6I(materialized),
		actorContextR6LRoutedOnlyExpectations(),
	)
	if err != nil {
		t.Fatal(err)
	}

	reordered := binding
	reordered.Planes = append([]actorContextR6KPlaneObservationProbe(nil), binding.Planes...)
	for i, j := 0, len(reordered.Planes)-1; i < j; i, j = i+1, j-1 {
		reordered.Planes[i], reordered.Planes[j] = reordered.Planes[j], reordered.Planes[i]
	}

	got, err := actorContextR6MCompareSameScopeObservations(binding, reordered)
	if err != nil {
		t.Fatal(err)
	}
	for _, delta := range got.PlaneDeltas {
		if delta.CountDelta != 0 ||
			delta.BeforeObservedCount != delta.AfterObservedCount {
			t.Fatalf(
				"R6M plane order changed normalized observation comparison: %+v",
				got.PlaneDeltas,
			)
		}
	}
}

func TestActorContextR6MObservationDeltaHasNoScopeFreshnessOrAuthorityShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR6MSameScopeObservationDeltaProbe{})
	for _, forbidden := range []string{
		"ScopeChanged",
		"SemanticChanged",
		"Complete",
		"Ready",
		"Fresh",
		"Stale",
		"Rebuild",
		"ShouldReevaluate",
		"Authorized",
		"CanExecute",
		"NextAction",
		"Priority",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf(
				"R6M same-scope observation delta introduced forbidden shortcut field %q",
				forbidden,
			)
		}
	}
}

func actorContextR6MCompareSameScopeObservations(
	before,
	after actorContextR6LScopeBindingProbe,
) (actorContextR6MSameScopeObservationDeltaProbe, error) {
	if before.ScopeIdentity == "" || after.ScopeIdentity == "" {
		return actorContextR6MSameScopeObservationDeltaProbe{}, errProcedureMeaning(
			"R6M same-scope observation comparison requires scope identity",
		)
	}
	if before.ScopeIdentity != after.ScopeIdentity {
		return actorContextR6MSameScopeObservationDeltaProbe{}, errProcedureMeaning(
			"R6M refuses observation comparison across different semantic input scopes",
		)
	}

	beforeByPlane, err := actorContextR6MPlaneObservationIndex(before.Planes)
	if err != nil {
		return actorContextR6MSameScopeObservationDeltaProbe{}, err
	}
	afterByPlane, err := actorContextR6MPlaneObservationIndex(after.Planes)
	if err != nil {
		return actorContextR6MSameScopeObservationDeltaProbe{}, err
	}

	if len(beforeByPlane) != len(afterByPlane) {
		return actorContextR6MSameScopeObservationDeltaProbe{}, errProcedureMeaning(
			"R6M same-scope observation sets cover different plane counts",
		)
	}

	deltas := make([]actorContextR6MPlaneCardinalityDeltaProbe, 0, len(beforeByPlane))
	for plane, beforeValue := range beforeByPlane {
		afterValue, ok := afterByPlane[plane]
		if !ok {
			return actorContextR6MSameScopeObservationDeltaProbe{}, errProcedureMeaning(
				"R6M same-scope after observation lost plane " + plane,
			)
		}
		if beforeValue.ScopeDisposition != afterValue.ScopeDisposition {
			return actorContextR6MSameScopeObservationDeltaProbe{}, errProcedureMeaning(
				"R6M same scope identity contains disposition drift for plane " + plane,
			)
		}
		deltas = append(deltas, actorContextR6MPlaneCardinalityDeltaProbe{
			PlaneClass:          plane,
			ScopeDisposition:    beforeValue.ScopeDisposition,
			BeforeObservedCount: beforeValue.ObservedCount,
			AfterObservedCount:  afterValue.ObservedCount,
			CountDelta:          afterValue.ObservedCount - beforeValue.ObservedCount,
		})
	}
	sort.Slice(deltas, func(i, j int) bool {
		return deltas[i].PlaneClass < deltas[j].PlaneClass
	})

	return actorContextR6MSameScopeObservationDeltaProbe{
		ScopeIdentity:  before.ScopeIdentity,
		BeforeAudience: before.Audience,
		BeforePurpose:  before.Purpose,
		AfterAudience:  after.Audience,
		AfterPurpose:   after.Purpose,
		PlaneDeltas:    deltas,
		AuthorityLimits: []string{
			"observation-cardinality-delta-is-only-valid-within-one-scope-identity",
			"cardinality-delta-does-not-imply-scope-meaning-change",
			"audience-purpose-difference-does-not-strengthen-input-authority",
			"same-scope-observation-delta-does-not-decide-freshness-or-reevaluation",
			"scope-declaration-change-must-be-handled-as-a-different-scope",
		},
	}, nil
}

func actorContextR6MPlaneObservationIndex(
	values []actorContextR6KPlaneObservationProbe,
) (map[string]actorContextR6KPlaneObservationProbe, error) {
	out := map[string]actorContextR6KPlaneObservationProbe{}
	for _, value := range values {
		if _, exists := out[value.PlaneClass]; exists {
			return nil, errProcedureMeaning(
				"R6M duplicate plane observation " + value.PlaneClass,
			)
		}
		out[value.PlaneClass] = value
	}
	return out, nil
}

func actorContextR6MPlaneDelta(
	t *testing.T,
	values []actorContextR6MPlaneCardinalityDeltaProbe,
	plane string,
) actorContextR6MPlaneCardinalityDeltaProbe {
	t.Helper()
	for _, value := range values {
		if value.PlaneClass == plane {
			return value
		}
	}
	t.Fatalf("R6M plane delta not found for %q: %+v", plane, values)
	return actorContextR6MPlaneCardinalityDeltaProbe{}
}

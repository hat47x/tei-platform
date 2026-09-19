package main

import (
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR6KPlaneRoutedKnowledge   = "routed-knowledge-body"
	actorContextR6KPlaneRoutedUse         = "routed-use-edge"
	actorContextR6KPlaneConsumerReference = "consumer-reference-use-edge"
	actorContextR6KPlaneConsumerRuntime   = "consumer-runtime-use-edge"
	actorContextR6KPlaneConsumerSemantic  = "consumer-semantic-use-edge"

	actorContextR6KScopeRequired   = "required"
	actorContextR6KScopeOutOfScope = "out-of-scope"
)

var actorContextR6KAllPlanes = []string{
	actorContextR6KPlaneRoutedKnowledge,
	actorContextR6KPlaneRoutedUse,
	actorContextR6KPlaneConsumerReference,
	actorContextR6KPlaneConsumerRuntime,
	actorContextR6KPlaneConsumerSemantic,
}

type actorContextR6KPlaneExpectationProbe struct {
	PlaneClass       string `json:"planeClass"`
	ScopeDisposition string `json:"scopeDisposition"`
}

type actorContextR6KPlaneObservationProbe struct {
	PlaneClass       string `json:"planeClass"`
	ScopeDisposition string `json:"scopeDisposition"`
	ObservedCount    int    `json:"observedCount"`
}

type actorContextR6KBoundedScopeWitnessProbe struct {
	Audience        string                                 `json:"audience"`
	Purpose         string                                 `json:"purpose"`
	Planes          []actorContextR6KPlaneObservationProbe `json:"planes"`
	AuthorityLimits []string                               `json:"authorityLimits"`
}

func TestActorContextR6KBP06HumanDistinguishesRequiredAndOutOfScopePlanes(t *testing.T) {
	root := repoPath()
	qualified := actorContextR6CCurrentBP06QualifiedKnowledge(t, root)
	origins, _, _ := actorContextR6JBP06HumanOriginSet(t, root, qualified)
	assembly, err := actorContextR6JAssembleBoundedInput(qualified, origins)
	if err != nil {
		t.Fatal(err)
	}

	got, err := actorContextR6KObserveScope(
		assembly,
		[]actorContextR6KPlaneExpectationProbe{
			{actorContextR6KPlaneRoutedKnowledge, actorContextR6KScopeRequired},
			{actorContextR6KPlaneRoutedUse, actorContextR6KScopeRequired},
			{actorContextR6KPlaneConsumerReference, actorContextR6KScopeRequired},
			{actorContextR6KPlaneConsumerRuntime, actorContextR6KScopeOutOfScope},
			{actorContextR6KPlaneConsumerSemantic, actorContextR6KScopeOutOfScope},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if actorContextR6KObservedCount(
		t, got.Planes, actorContextR6KPlaneRoutedKnowledge,
	) != 1 {
		t.Fatalf("R6K BP-06 Human lost routed knowledge body: %+v", got.Planes)
	}
	if actorContextR6KObservedCount(
		t, got.Planes, actorContextR6KPlaneRoutedUse,
	) != 3 {
		t.Fatalf("R6K BP-06 Human lost routed use edges: %+v", got.Planes)
	}
	if actorContextR6KObservedCount(
		t, got.Planes, actorContextR6KPlaneConsumerReference,
	) != 3 {
		t.Fatalf("R6K BP-06 Human lost consumer-reference use edges: %+v", got.Planes)
	}
	if actorContextR6KObservedCount(
		t, got.Planes, actorContextR6KPlaneConsumerRuntime,
	) != 0 ||
		actorContextR6KObservedCount(
			t, got.Planes, actorContextR6KPlaneConsumerSemantic,
		) != 0 {
		t.Fatalf("R6K BP-06 Human out-of-scope plane gained inputs: %+v", got.Planes)
	}
}

func TestActorContextR6KRuntimeOnlyScopeTreatsEmptyRoutedPlanesAsIntentional(t *testing.T) {
	root := repoPath()
	qualified := actorContextR6CCurrentBP06QualifiedKnowledge(t, root)
	origins := actorContextR6EDeniedRuntimeOriginSet(t, qualified)
	assembly, err := actorContextR6JAssembleBoundedInput(qualified, origins)
	if err != nil {
		t.Fatal(err)
	}

	got, err := actorContextR6KObserveScope(
		assembly,
		[]actorContextR6KPlaneExpectationProbe{
			{actorContextR6KPlaneRoutedKnowledge, actorContextR6KScopeOutOfScope},
			{actorContextR6KPlaneRoutedUse, actorContextR6KScopeOutOfScope},
			{actorContextR6KPlaneConsumerReference, actorContextR6KScopeOutOfScope},
			{actorContextR6KPlaneConsumerRuntime, actorContextR6KScopeRequired},
			{actorContextR6KPlaneConsumerSemantic, actorContextR6KScopeOutOfScope},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if actorContextR6KObservedCount(
		t, got.Planes, actorContextR6KPlaneConsumerRuntime,
	) != 1 {
		t.Fatalf("R6K runtime-only scope lost required Observation input: %+v", got.Planes)
	}
	for _, plane := range []string{
		actorContextR6KPlaneRoutedKnowledge,
		actorContextR6KPlaneRoutedUse,
		actorContextR6KPlaneConsumerReference,
		actorContextR6KPlaneConsumerSemantic,
	} {
		if actorContextR6KObservedCount(t, got.Planes, plane) != 0 {
			t.Fatalf("R6K runtime-only out-of-scope plane %q was populated: %+v", plane, got.Planes)
		}
	}

	if !reflect.DeepEqual(assembly.QualifiedNotMaterialized, []string{
		"business-procedure-reference|order-entry.order-handling",
		"semantic-capability|security.authorize",
	}) {
		t.Fatalf(
			"R6K intentional routed-plane exclusion lost qualified audit identities: %+v",
			assembly.QualifiedNotMaterialized,
		)
	}
}

func TestActorContextR6KRequiredPlaneMissingFailsClosed(t *testing.T) {
	assembly := actorContextR6JBoundedInputAssemblyProbe{
		Audience: actorContextR3PAudienceHuman,
		Purpose:  actorContextR3PPurposeOrientation,
	}

	_, err := actorContextR6KObserveScope(
		assembly,
		[]actorContextR6KPlaneExpectationProbe{
			{actorContextR6KPlaneRoutedKnowledge, actorContextR6KScopeRequired},
			{actorContextR6KPlaneRoutedUse, actorContextR6KScopeOutOfScope},
			{actorContextR6KPlaneConsumerReference, actorContextR6KScopeOutOfScope},
			{actorContextR6KPlaneConsumerRuntime, actorContextR6KScopeOutOfScope},
			{actorContextR6KPlaneConsumerSemantic, actorContextR6KScopeOutOfScope},
		},
	)
	if err == nil {
		t.Fatal("R6K accepted required routed knowledge plane with zero observed inputs")
	}
}

func TestActorContextR6KOutOfScopePlanePopulationFailsClosed(t *testing.T) {
	assembly := actorContextR6JBoundedInputAssemblyProbe{
		Audience: actorContextR3PAudienceHuman,
		Purpose:  actorContextR3PPurposeOrientation,
		ConsumerLocalUseEdges: []actorContextR6JConsumerLocalUseProbe{{
			OriginClass:        actorContextR6DOriginConsumerRuntime,
			ProjectionCategory: actorContextR3PCategoryRuntimeOutcome,
			DependencyKey:      "runtime:test-only",
			DependencyReason:   actorContextR3SReasonExactRuntimeObservation,
		}},
	}

	_, err := actorContextR6KObserveScope(
		assembly,
		[]actorContextR6KPlaneExpectationProbe{
			{actorContextR6KPlaneRoutedKnowledge, actorContextR6KScopeOutOfScope},
			{actorContextR6KPlaneRoutedUse, actorContextR6KScopeOutOfScope},
			{actorContextR6KPlaneConsumerReference, actorContextR6KScopeOutOfScope},
			{actorContextR6KPlaneConsumerRuntime, actorContextR6KScopeOutOfScope},
			{actorContextR6KPlaneConsumerSemantic, actorContextR6KScopeOutOfScope},
		},
	)
	if err == nil {
		t.Fatal("R6K accepted populated consumer-runtime plane declared out-of-scope")
	}
}

func TestActorContextR6KScopeDeclarationMustCoverEveryKnownPlaneExactlyOnce(t *testing.T) {
	assembly := actorContextR6JBoundedInputAssemblyProbe{
		Audience: actorContextR3PAudienceHuman,
		Purpose:  actorContextR3PPurposeOrientation,
	}

	incomplete := []actorContextR6KPlaneExpectationProbe{
		{actorContextR6KPlaneRoutedKnowledge, actorContextR6KScopeOutOfScope},
	}
	if _, err := actorContextR6KObserveScope(assembly, incomplete); err == nil {
		t.Fatal("R6K accepted scope declaration that left known planes implicit")
	}

	duplicate := []actorContextR6KPlaneExpectationProbe{
		{actorContextR6KPlaneRoutedKnowledge, actorContextR6KScopeOutOfScope},
		{actorContextR6KPlaneRoutedKnowledge, actorContextR6KScopeOutOfScope},
		{actorContextR6KPlaneRoutedUse, actorContextR6KScopeOutOfScope},
		{actorContextR6KPlaneConsumerReference, actorContextR6KScopeOutOfScope},
		{actorContextR6KPlaneConsumerRuntime, actorContextR6KScopeOutOfScope},
		{actorContextR6KPlaneConsumerSemantic, actorContextR6KScopeOutOfScope},
	}
	if _, err := actorContextR6KObserveScope(assembly, duplicate); err == nil {
		t.Fatal("R6K accepted duplicate plane expectation")
	}
}

func TestActorContextR6KScopeWitnessHasNoGlobalCompletenessOrAuthorityShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR6KBoundedScopeWitnessProbe{})
	for _, forbidden := range []string{
		"Complete",
		"Valid",
		"Ready",
		"Missing",
		"Fresh",
		"Stale",
		"Authorized",
		"CanExecute",
		"NextAction",
		"Owner",
		"Priority",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R6K scope witness introduced forbidden shortcut field %q", forbidden)
		}
	}
}

func actorContextR6KObserveScope(
	assembly actorContextR6JBoundedInputAssemblyProbe,
	expectations []actorContextR6KPlaneExpectationProbe,
) (actorContextR6KBoundedScopeWitnessProbe, error) {
	known := map[string]bool{}
	for _, plane := range actorContextR6KAllPlanes {
		known[plane] = true
	}

	expected := map[string]string{}
	for _, expectation := range expectations {
		if !known[expectation.PlaneClass] {
			return actorContextR6KBoundedScopeWitnessProbe{}, errProcedureMeaning(
				"R6K unknown input plane " + expectation.PlaneClass,
			)
		}
		if _, exists := expected[expectation.PlaneClass]; exists {
			return actorContextR6KBoundedScopeWitnessProbe{}, errProcedureMeaning(
				"R6K duplicate input plane expectation " + expectation.PlaneClass,
			)
		}
		switch expectation.ScopeDisposition {
		case actorContextR6KScopeRequired, actorContextR6KScopeOutOfScope:
		default:
			return actorContextR6KBoundedScopeWitnessProbe{}, errProcedureMeaning(
				"R6K unsupported scope disposition " + expectation.ScopeDisposition,
			)
		}
		expected[expectation.PlaneClass] = expectation.ScopeDisposition
	}
	if len(expected) != len(actorContextR6KAllPlanes) {
		return actorContextR6KBoundedScopeWitnessProbe{}, errProcedureMeaning(
			"R6K bounded input scope left known planes implicit",
		)
	}

	counts := map[string]int{
		actorContextR6KPlaneRoutedKnowledge: len(assembly.RoutedKnowledge),
		actorContextR6KPlaneRoutedUse:       len(assembly.RoutedUseEdges),
	}
	for _, edge := range assembly.ConsumerLocalUseEdges {
		switch edge.OriginClass {
		case actorContextR6DOriginConsumerReference:
			counts[actorContextR6KPlaneConsumerReference]++
		case actorContextR6DOriginConsumerRuntime:
			counts[actorContextR6KPlaneConsumerRuntime]++
		case actorContextR6DOriginConsumerSemantic:
			counts[actorContextR6KPlaneConsumerSemantic]++
		default:
			return actorContextR6KBoundedScopeWitnessProbe{}, errProcedureMeaning(
				"R6K assembly contains unsupported consumer-local origin " + edge.OriginClass,
			)
		}
	}

	planes := make([]actorContextR6KPlaneObservationProbe, 0, len(actorContextR6KAllPlanes))
	for _, plane := range actorContextR6KAllPlanes {
		disposition := expected[plane]
		count := counts[plane]
		if disposition == actorContextR6KScopeRequired && count == 0 {
			return actorContextR6KBoundedScopeWitnessProbe{}, errProcedureMeaning(
				"R6K required input plane has no observed input " + plane,
			)
		}
		if disposition == actorContextR6KScopeOutOfScope && count != 0 {
			return actorContextR6KBoundedScopeWitnessProbe{}, errProcedureMeaning(
				"R6K out-of-scope input plane was populated " + plane,
			)
		}
		planes = append(planes, actorContextR6KPlaneObservationProbe{
			PlaneClass:       plane,
			ScopeDisposition: disposition,
			ObservedCount:    count,
		})
	}
	sort.Slice(planes, func(i, j int) bool {
		return planes[i].PlaneClass < planes[j].PlaneClass
	})

	return actorContextR6KBoundedScopeWitnessProbe{
		Audience: assembly.Audience,
		Purpose:  assembly.Purpose,
		Planes:   planes,
		AuthorityLimits: []string{
			"required-zero-is-not-treated-as-intentional-absence",
			"out-of-scope-zero-is-not-treated-as-missing-input",
			"scope-declaration-does-not-create-global-completeness-status",
			"qualified-not-materialized-remains-separate-from-plane-scope",
			"scope-observation-does-not-decide-freshness-or-authority",
		},
	}, nil
}

func actorContextR6KObservedCount(
	t *testing.T,
	values []actorContextR6KPlaneObservationProbe,
	plane string,
) int {
	t.Helper()
	for _, value := range values {
		if value.PlaneClass == plane {
			return value.ObservedCount
		}
	}
	t.Fatalf("R6K plane observation not found for %q: %+v", plane, values)
	return 0
}

package main

import (
	"reflect"
	"testing"
)

func TestActorContextR6QRouterSubstitutionIsInvisibleToBoundedObservation(t *testing.T) {
	root := repoPath()

	exact := actorContextR6EBP06RoutingProbe(
		t,
		root,
		"exact-task-router",
		actorContextR6BSelectTaskHints,
	)
	broad := actorContextR6EBP06RoutingProbe(
		t,
		root,
		"broad-heterogeneous-router",
		actorContextR6BSelectAll,
	)

	if len(broad.Selected) <= len(exact.Selected) {
		t.Fatalf(
			"R6Q setup lost broad-router over-selection: exact=%d broad=%d",
			len(exact.Selected),
			len(broad.Selected),
		)
	}
	if len(broad.Rejected) == 0 {
		t.Fatal("R6Q broad router no longer exposes rejected over-selection")
	}

	exactAssembly := actorContextR6QBP06HumanAssembly(t, root, exact.Qualified)
	broadAssembly := actorContextR6QBP06HumanAssembly(t, root, broad.Qualified)

	if !reflect.DeepEqual(exactAssembly, broadAssembly) {
		t.Fatalf(
			"R6Q router substitution changed bounded input assembly after convergent qualification: exact=%+v broad=%+v",
			exactAssembly,
			broadAssembly,
		)
	}

	got, err := actorContextR6PDecomposeSameScopeObservation(
		exactAssembly,
		broadAssembly,
		actorContextR6QBP06HumanExpectations(),
	)
	if err != nil {
		t.Fatal(err)
	}
	actorContextR6PRequireZeroCardinality(t, got.CardinalityDeltas)
	actorContextR6PRequireZeroMembership(t, got.MembershipDeltas)
	if len(got.RetainedKnowledgeProvenance) != 0 {
		t.Fatalf(
			"R6Q router substitution fabricated retained provenance change: %+v",
			got.RetainedKnowledgeProvenance,
		)
	}
}

func TestActorContextR6QSameSourceRevisionProducesSameDecompositionAcrossRouters(t *testing.T) {
	root := repoPath()

	exact := actorContextR6EBP06RoutingProbe(
		t,
		root,
		"exact-task-router",
		actorContextR6BSelectTaskHints,
	)
	broad := actorContextR6EBP06RoutingProbe(
		t,
		root,
		"broad-heterogeneous-router",
		actorContextR6BSelectAll,
	)

	exactBefore := actorContextR6QBP06HumanAssembly(t, root, exact.Qualified)
	broadBefore := actorContextR6QBP06HumanAssembly(t, root, broad.Qualified)

	exactRevised := actorContextR6HCloneQualified(exact.Qualified)
	broadRevised := actorContextR6HCloneQualified(broad.Qualified)
	actorContextR6HReplaceFirstFingerprint(
		t,
		exactRevised,
		"business-procedure-reference|order-entry.order-handling",
		"test-only-r6q-procedure-revision",
	)
	actorContextR6HReplaceFirstFingerprint(
		t,
		broadRevised,
		"business-procedure-reference|order-entry.order-handling",
		"test-only-r6q-procedure-revision",
	)

	exactAfter := actorContextR6QBP06HumanAssembly(t, root, exactRevised)
	broadAfter := actorContextR6QBP06HumanAssembly(t, root, broadRevised)

	expectations := actorContextR6QBP06HumanExpectations()
	exactDelta, err := actorContextR6PDecomposeSameScopeObservation(
		exactBefore,
		exactAfter,
		expectations,
	)
	if err != nil {
		t.Fatal(err)
	}
	broadDelta, err := actorContextR6PDecomposeSameScopeObservation(
		broadBefore,
		broadAfter,
		expectations,
	)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(exactDelta, broadDelta) {
		t.Fatalf(
			"R6Q routing mechanism changed same source revision decomposition: exact=%+v broad=%+v",
			exactDelta,
			broadDelta,
		)
	}
	actorContextR6PRequireZeroCardinality(t, exactDelta.CardinalityDeltas)
	actorContextR6PRequireZeroMembership(t, exactDelta.MembershipDeltas)
	if len(exactDelta.RetainedKnowledgeProvenance) != 1 ||
		exactDelta.RetainedKnowledgeProvenance[0].KnowledgeKey !=
			"business-procedure-reference|order-entry.order-handling" {
		t.Fatalf(
			"R6Q source revision did not remain one retained provenance delta: %+v",
			exactDelta,
		)
	}
}

func TestActorContextR6QConsumerRuntimeObservationRemainsRouterInvariantDownstream(t *testing.T) {
	root := repoPath()

	exact := actorContextR6EBP06RoutingProbe(
		t,
		root,
		"exact-task-router",
		actorContextR6BSelectTaskHints,
	)
	broad := actorContextR6EBP06RoutingProbe(
		t,
		root,
		"broad-heterogeneous-router",
		actorContextR6BSelectAll,
	)

	exactAssembly, err := actorContextR6JAssembleBoundedInput(
		exact.Qualified,
		actorContextR6EDeniedRuntimeOriginSet(t, exact.Qualified),
	)
	if err != nil {
		t.Fatal(err)
	}
	broadAssembly, err := actorContextR6JAssembleBoundedInput(
		broad.Qualified,
		actorContextR6EDeniedRuntimeOriginSet(t, broad.Qualified),
	)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(exactAssembly, broadAssembly) {
		t.Fatalf(
			"R6Q router substitution changed runtime-only consumer-local assembly: exact=%+v broad=%+v",
			exactAssembly,
			broadAssembly,
		)
	}

	got, err := actorContextR6PDecomposeSameScopeObservation(
		exactAssembly,
		broadAssembly,
		actorContextR6QRuntimeOnlyExpectations(),
	)
	if err != nil {
		t.Fatal(err)
	}
	actorContextR6PRequireZeroCardinality(t, got.CardinalityDeltas)
	actorContextR6PRequireZeroMembership(t, got.MembershipDeltas)
	if len(got.RetainedKnowledgeProvenance) != 0 {
		t.Fatalf(
			"R6Q consumer-local runtime observation fabricated routed provenance delta: %+v",
			got.RetainedKnowledgeProvenance,
		)
	}
}

func TestActorContextR6QObservationDecompositionDoesNotExposeRouterPreference(t *testing.T) {
	typ := reflect.TypeOf(actorContextR6PObservationDecompositionProbe{})
	for _, forbidden := range []string{
		"RoutingMechanism",
		"Router",
		"PreferredRouter",
		"Winner",
		"Score",
		"Confidence",
		"OverSelection",
		"RejectedCandidates",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf(
				"R6Q downstream observation decomposition leaked router diagnostic field %q",
				forbidden,
			)
		}
	}
}

func actorContextR6QBP06HumanAssembly(
	t *testing.T,
	root string,
	qualified []actorContextR6BHeterogeneousQualified,
) actorContextR6JBoundedInputAssemblyProbe {
	t.Helper()

	origins, _, _ := actorContextR6JBP06HumanOriginSet(t, root, qualified)
	assembly, err := actorContextR6JAssembleBoundedInput(qualified, origins)
	if err != nil {
		t.Fatal(err)
	}
	return assembly
}

func actorContextR6QBP06HumanExpectations() []actorContextR6KPlaneExpectationProbe {
	return []actorContextR6KPlaneExpectationProbe{
		{
			PlaneClass:       actorContextR6KPlaneRoutedKnowledge,
			ScopeDisposition: actorContextR6KScopeRequired,
		},
		{
			PlaneClass:       actorContextR6KPlaneRoutedUse,
			ScopeDisposition: actorContextR6KScopeRequired,
		},
		{
			PlaneClass:       actorContextR6KPlaneConsumerReference,
			ScopeDisposition: actorContextR6KScopeRequired,
		},
		{
			PlaneClass:       actorContextR6KPlaneConsumerRuntime,
			ScopeDisposition: actorContextR6KScopeOutOfScope,
		},
		{
			PlaneClass:       actorContextR6KPlaneConsumerSemantic,
			ScopeDisposition: actorContextR6KScopeOutOfScope,
		},
	}
}

func actorContextR6QRuntimeOnlyExpectations() []actorContextR6KPlaneExpectationProbe {
	return []actorContextR6KPlaneExpectationProbe{
		{
			PlaneClass:       actorContextR6KPlaneRoutedKnowledge,
			ScopeDisposition: actorContextR6KScopeOutOfScope,
		},
		{
			PlaneClass:       actorContextR6KPlaneRoutedUse,
			ScopeDisposition: actorContextR6KScopeOutOfScope,
		},
		{
			PlaneClass:       actorContextR6KPlaneConsumerReference,
			ScopeDisposition: actorContextR6KScopeOutOfScope,
		},
		{
			PlaneClass:       actorContextR6KPlaneConsumerRuntime,
			ScopeDisposition: actorContextR6KScopeRequired,
		},
		{
			PlaneClass:       actorContextR6KPlaneConsumerSemantic,
			ScopeDisposition: actorContextR6KScopeOutOfScope,
		},
	}
}

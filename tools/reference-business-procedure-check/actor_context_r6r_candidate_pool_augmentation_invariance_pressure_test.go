package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/workspace"
)

const actorContextR6RUnrelatedCandidateID = "test-only.unrelated_pattern"

func TestActorContextR6RUnrelatedCandidateDoesNotChangeExactRoutingOutcome(t *testing.T) {
	root := repoPath()

	baseline := actorContextR6EBP06RoutingProbe(
		t,
		root,
		"exact-task-router",
		actorContextR6BSelectTaskHints,
	)
	augmented := actorContextR6RBP06ProbeWithUnrelatedCandidate(
		t,
		root,
		"exact-task-router",
		actorContextR6BSelectTaskHints,
	)

	if len(augmented.Available) != len(baseline.Available)+1 {
		t.Fatalf(
			"R6R unrelated candidate was not added to repository candidate pool: baseline=%d augmented=%d",
			len(baseline.Available),
			len(augmented.Available),
		)
	}
	if !reflect.DeepEqual(baseline.Selected, augmented.Selected) {
		t.Fatalf(
			"R6R exact router selected unrelated repository candidate: baseline=%+v augmented=%+v",
			baseline.Selected,
			augmented.Selected,
		)
	}
	if !reflect.DeepEqual(baseline.Qualified, augmented.Qualified) {
		t.Fatalf(
			"R6R exact routing qualification changed after unrelated pool growth: baseline=%+v augmented=%+v",
			baseline.Qualified,
			augmented.Qualified,
		)
	}
	if !reflect.DeepEqual(baseline.Disclosed, augmented.Disclosed) {
		t.Fatalf(
			"R6R exact routing disclosure changed after unrelated pool growth: baseline=%+v augmented=%+v",
			baseline.Disclosed,
			augmented.Disclosed,
		)
	}

	baselineAssembly := actorContextR6QBP06HumanAssembly(t, root, baseline.Qualified)
	augmentedAssembly := actorContextR6QBP06HumanAssembly(t, root, augmented.Qualified)
	if !reflect.DeepEqual(baselineAssembly, augmentedAssembly) {
		t.Fatalf(
			"R6R exact-router pool growth changed bounded input assembly: baseline=%+v augmented=%+v",
			baselineAssembly,
			augmentedAssembly,
		)
	}

	got, err := actorContextR6PDecomposeSameScopeObservation(
		baselineAssembly,
		augmentedAssembly,
		actorContextR6QBP06HumanExpectations(),
	)
	if err != nil {
		t.Fatal(err)
	}
	actorContextR6PRequireZeroCardinality(t, got.CardinalityDeltas)
	actorContextR6PRequireZeroMembership(t, got.MembershipDeltas)
	if len(got.RetainedKnowledgeProvenance) != 0 {
		t.Fatalf(
			"R6R exact-router unrelated pool growth fabricated provenance delta: %+v",
			got.RetainedKnowledgeProvenance,
		)
	}
}

func TestActorContextR6RBroadRouterRejectsUnrelatedCandidateWithoutDownstreamChange(t *testing.T) {
	root := repoPath()

	baseline := actorContextR6EBP06RoutingProbe(
		t,
		root,
		"broad-heterogeneous-router",
		actorContextR6BSelectAll,
	)
	augmented := actorContextR6RBP06ProbeWithUnrelatedCandidate(
		t,
		root,
		"broad-heterogeneous-router",
		actorContextR6BSelectAll,
	)

	if len(augmented.Selected) != len(baseline.Selected)+1 {
		t.Fatalf(
			"R6R broad router did not observe added repository candidate: baseline=%d augmented=%d",
			len(baseline.Selected),
			len(augmented.Selected),
		)
	}
	if !actorContextR6RHasRejection(
		augmented.Rejected,
		"application-pattern",
		actorContextR6RUnrelatedCandidateID,
		"not-in-deterministic-task-scope",
	) {
		t.Fatalf(
			"R6R broad router failed to deterministically reject unrelated candidate: %+v",
			augmented.Rejected,
		)
	}
	if !reflect.DeepEqual(baseline.Qualified, augmented.Qualified) {
		t.Fatalf(
			"R6R broad-router qualification changed after unrelated pool growth: baseline=%+v augmented=%+v",
			baseline.Qualified,
			augmented.Qualified,
		)
	}
	if !reflect.DeepEqual(baseline.Disclosed, augmented.Disclosed) {
		t.Fatalf(
			"R6R broad-router disclosure changed after unrelated pool growth: baseline=%+v augmented=%+v",
			baseline.Disclosed,
			augmented.Disclosed,
		)
	}

	baselineAssembly := actorContextR6QBP06HumanAssembly(t, root, baseline.Qualified)
	augmentedAssembly := actorContextR6QBP06HumanAssembly(t, root, augmented.Qualified)
	if !reflect.DeepEqual(baselineAssembly, augmentedAssembly) {
		t.Fatalf(
			"R6R broad-router pool growth changed bounded input assembly: baseline=%+v augmented=%+v",
			baselineAssembly,
			augmentedAssembly,
		)
	}

	got, err := actorContextR6PDecomposeSameScopeObservation(
		baselineAssembly,
		augmentedAssembly,
		actorContextR6QBP06HumanExpectations(),
	)
	if err != nil {
		t.Fatal(err)
	}
	actorContextR6PRequireZeroCardinality(t, got.CardinalityDeltas)
	actorContextR6PRequireZeroMembership(t, got.MembershipDeltas)
	if len(got.RetainedKnowledgeProvenance) != 0 {
		t.Fatalf(
			"R6R broad-router unrelated pool growth fabricated provenance delta: %+v",
			got.RetainedKnowledgeProvenance,
		)
	}
}

func TestActorContextR6RCandidatePoolGrowthDoesNotChangeScopeOrQualifiedOmission(t *testing.T) {
	root := repoPath()

	baseline := actorContextR6EBP06RoutingProbe(
		t,
		root,
		"broad-heterogeneous-router",
		actorContextR6BSelectAll,
	)
	augmented := actorContextR6RBP06ProbeWithUnrelatedCandidate(
		t,
		root,
		"broad-heterogeneous-router",
		actorContextR6BSelectAll,
	)

	baselineAssembly := actorContextR6QBP06HumanAssembly(t, root, baseline.Qualified)
	augmentedAssembly := actorContextR6QBP06HumanAssembly(t, root, augmented.Qualified)

	baselineScope, err := actorContextR6LBindScope(
		baselineAssembly,
		actorContextR6QBP06HumanExpectations(),
	)
	if err != nil {
		t.Fatal(err)
	}
	augmentedScope, err := actorContextR6LBindScope(
		augmentedAssembly,
		actorContextR6QBP06HumanExpectations(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if baselineScope.ScopeIdentity != augmentedScope.ScopeIdentity {
		t.Fatalf(
			"R6R unrelated candidate pool growth changed semantic input scope: baseline=%q augmented=%q",
			baselineScope.ScopeIdentity,
			augmentedScope.ScopeIdentity,
		)
	}
	if !reflect.DeepEqual(
		baselineAssembly.QualifiedNotMaterialized,
		augmentedAssembly.QualifiedNotMaterialized,
	) {
		t.Fatalf(
			"R6R unrelated candidate changed qualified-not-materialized audit identities: baseline=%v augmented=%v",
			baselineAssembly.QualifiedNotMaterialized,
			augmentedAssembly.QualifiedNotMaterialized,
		)
	}
	if actorContextR6RContainsQualified(
		augmented.Qualified,
		"application-pattern",
		actorContextR6RUnrelatedCandidateID,
	) {
		t.Fatalf(
			"R6R unrelated repository candidate leaked into qualified knowledge: %+v",
			augmented.Qualified,
		)
	}
}

func TestActorContextR6RDownstreamShapesDoNotExposeRepositoryPoolSize(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR6JBoundedInputAssemblyProbe{}),
		reflect.TypeOf(actorContextR6LScopeBindingProbe{}),
		reflect.TypeOf(actorContextR6PObservationDecompositionProbe{}),
	} {
		for _, forbidden := range []string{
			"AvailableCandidateCount",
			"CandidatePoolSize",
			"RepositoryKnowledgeCount",
			"RejectedCandidateCount",
			"OverSelectionCount",
			"PoolRevision",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf(
					"R6R downstream bounded shape leaked repository-pool diagnostic field %q",
					forbidden,
				)
			}
		}
	}
}

func actorContextR6RBP06ProbeWithUnrelatedCandidate(
	t *testing.T,
	root,
	mechanism string,
	selector func(
		[]actorContextR6BHeterogeneousCandidate,
		string,
	) []actorContextR6BHeterogeneousCandidate,
) actorContextR6BHeterogeneousProbe {
	t.Helper()

	workspacePath := filepath.Join(root, "reference", "workspace")
	procedurePath := filepath.Join(
		root,
		"reference",
		"business-system-design",
		"order-entry-business-procedure.json",
	)
	reviewEvidencePath := filepath.Join(
		root,
		"reference",
		"business-system-design",
		"order-entry-customer-edit-conflict.json",
	)

	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(procedurePath)
	if err != nil {
		t.Fatal(err)
	}
	sourceIdx, err := workspace.BuildSourceIndex(workspacePath, loaded)
	if err != nil {
		t.Fatal(err)
	}
	reviewEvidence, err := actorContextR6BReadReviewEvidence(reviewEvidencePath)
	if err != nil {
		t.Fatal(err)
	}
	available, err := actorContextR6BBuildAvailableCandidates(f, reviewEvidence)
	if err != nil {
		t.Fatal(err)
	}

	available = append(
		available,
		actorContextR6BHeterogeneousCandidate{
			Identity:      actorContextR6RUnrelatedCandidateID,
			KnowledgeKind: "application-pattern",
			TaskHints:     []string{"BP-02"},
		},
	)

	probe, err := deriveActorContextR6BHeterogeneousProbe(
		sourceIdx,
		f,
		reviewEvidence,
		procedurePath,
		reviewEvidencePath,
		"BP-06",
		mechanism,
		available,
		selector,
	)
	if err != nil {
		t.Fatal(err)
	}
	return probe
}

func actorContextR6RHasRejection(
	values []actorContextR6BHeterogeneousRejection,
	kind,
	identity,
	reason string,
) bool {
	for _, value := range values {
		if value.KnowledgeKind == kind &&
			value.Identity == identity &&
			value.Reason == reason {
			return true
		}
	}
	return false
}

func actorContextR6RContainsQualified(
	values []actorContextR6BHeterogeneousQualified,
	kind,
	identity string,
) bool {
	for _, value := range values {
		if value.KnowledgeKind == kind &&
			value.Identity == identity {
			return true
		}
	}
	return false
}

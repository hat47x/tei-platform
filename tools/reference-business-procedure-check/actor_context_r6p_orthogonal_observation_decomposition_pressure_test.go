package main

import (
	"reflect"
	"testing"
)

type actorContextR6PObservationDecompositionProbe struct {
	ScopeIdentity               string                                      `json:"scopeIdentity"`
	CardinalityDeltas           []actorContextR6MPlaneCardinalityDeltaProbe `json:"cardinalityDeltas"`
	MembershipDeltas            []actorContextR6NPlaneMembershipDeltaProbe  `json:"membershipDeltas"`
	RetainedKnowledgeProvenance []actorContextR6OKnowledgeProvenanceDeltaProbe `json:"retainedKnowledgeProvenance"`
	AuthorityLimits             []string                                    `json:"authorityLimits"`
}

func TestActorContextR6PFingerprintOnlyChangeStaysProvenanceOnly(t *testing.T) {
	expectations := actorContextR6LRoutedOnlyExpectations()
	before := actorContextR6OProcedureOnlyAssembly(
		"revision-a",
		"test-only:r6p/procedure",
	)
	after := actorContextR6OProcedureOnlyAssembly(
		"revision-b",
		"test-only:r6p/procedure",
	)

	got, err := actorContextR6PDecomposeSameScopeObservation(
		before,
		after,
		expectations,
	)
	if err != nil {
		t.Fatal(err)
	}

	actorContextR6PRequireZeroCardinality(t, got.CardinalityDeltas)
	actorContextR6PRequireZeroMembership(t, got.MembershipDeltas)
	if len(got.RetainedKnowledgeProvenance) != 1 ||
		got.RetainedKnowledgeProvenance[0].KnowledgeKey !=
			"business-procedure-reference|order-entry.order-handling" {
		t.Fatalf(
			"R6P fingerprint-only change did not remain provenance-only: %+v",
			got,
		)
	}
}

func TestActorContextR6PSameCountSubstitutionStaysMembershipOnly(t *testing.T) {
	expectations := actorContextR6LRoutedOnlyExpectations()
	before := actorContextR6OProcedureOnlyAssembly(
		"same-revision",
		"test-only:r6p/procedure",
	)
	after := before
	after.RoutedUseEdges = []actorContextR6IUseEdgeProbe{{
		KnowledgeKey:       "business-procedure-reference|order-entry.order-handling",
		ProjectionCategory: actorContextR3PCategoryKnownAuthorityFacts,
		DependencyKey:      "reference:test-only:r6p/procedure",
		DependencyReason:   actorContextR3SReasonAuthorizationRequirementSource,
	}}

	got, err := actorContextR6PDecomposeSameScopeObservation(
		before,
		after,
		expectations,
	)
	if err != nil {
		t.Fatal(err)
	}

	actorContextR6PRequireZeroCardinality(t, got.CardinalityDeltas)
	useDelta := actorContextR6NPlaneDelta(
		t,
		got.MembershipDeltas,
		actorContextR6KPlaneRoutedUse,
	)
	if len(useDelta.Added) != 1 || len(useDelta.Removed) != 1 {
		t.Fatalf(
			"R6P same-count substitution lost membership decomposition: %+v",
			useDelta,
		)
	}
	if len(got.RetainedKnowledgeProvenance) != 0 {
		t.Fatalf(
			"R6P same-count use substitution fabricated knowledge provenance delta: %+v",
			got.RetainedKnowledgeProvenance,
		)
	}
}

func TestActorContextR6PUseExpansionAlignsCardinalityAndMembership(t *testing.T) {
	root := repoPath()
	qualified := actorContextR6CCurrentBP06QualifiedKnowledge(t, root)
	humanHandoff, agentHandoff, _ := actorContextR6IBP06AudienceHandoffs(t, root, qualified)
	humanMaterialized, err := actorContextR6IBuildBoundedMaterialization(
		qualified,
		humanHandoff,
	)
	if err != nil {
		t.Fatal(err)
	}
	agentMaterialized, err := actorContextR6IBuildBoundedMaterialization(
		qualified,
		agentHandoff,
	)
	if err != nil {
		t.Fatal(err)
	}

	got, err := actorContextR6PDecomposeSameScopeObservation(
		actorContextR6LAssemblyFromR6I(humanMaterialized),
		actorContextR6LAssemblyFromR6I(agentMaterialized),
		actorContextR6LRoutedOnlyExpectations(),
	)
	if err != nil {
		t.Fatal(err)
	}

	useCardinality := actorContextR6PCardinalityDelta(
		t,
		got.CardinalityDeltas,
		actorContextR6KPlaneRoutedUse,
	)
	useMembership := actorContextR6NPlaneDelta(
		t,
		got.MembershipDeltas,
		actorContextR6KPlaneRoutedUse,
	)
	if useCardinality.CountDelta != 1 ||
		len(useMembership.Added) != 1 ||
		len(useMembership.Removed) != 0 {
		t.Fatalf(
			"R6P use expansion lost cardinality/membership consistency: cardinality=%+v membership=%+v",
			useCardinality,
			useMembership,
		)
	}

	knowledgeCardinality := actorContextR6PCardinalityDelta(
		t,
		got.CardinalityDeltas,
		actorContextR6KPlaneRoutedKnowledge,
	)
	knowledgeMembership := actorContextR6NPlaneDelta(
		t,
		got.MembershipDeltas,
		actorContextR6KPlaneRoutedKnowledge,
	)
	if knowledgeCardinality.CountDelta != 0 ||
		len(knowledgeMembership.Added) != 0 ||
		len(knowledgeMembership.Removed) != 0 {
		t.Fatalf(
			"R6P consumer use expansion changed knowledge membership: cardinality=%+v membership=%+v",
			knowledgeCardinality,
			knowledgeMembership,
		)
	}
	if len(got.RetainedKnowledgeProvenance) != 0 {
		t.Fatalf(
			"R6P consumer use expansion fabricated provenance change: %+v",
			got.RetainedKnowledgeProvenance,
		)
	}
}

func TestActorContextR6PDuplicateObservationFailsCardinalityMembershipConsistency(t *testing.T) {
	expectations := actorContextR6LRoutedOnlyExpectations()
	before := actorContextR6OProcedureOnlyAssembly(
		"same-revision",
		"test-only:r6p/procedure",
	)
	after := before
	after.RoutedUseEdges = append(
		append([]actorContextR6IUseEdgeProbe(nil), before.RoutedUseEdges...),
		before.RoutedUseEdges[0],
	)

	if _, err := actorContextR6PDecomposeSameScopeObservation(
		before,
		after,
		expectations,
	); err == nil {
		t.Fatal(
			"R6P accepted duplicate use observation whose cardinality changed without membership change",
		)
	}
}

func TestActorContextR6PAddedKnowledgeIsMembershipNotRetainedProvenance(t *testing.T) {
	expectations := actorContextR6LRoutedOnlyExpectations()
	before := actorContextR6OProcedureOnlyAssembly(
		"same-revision",
		"test-only:r6p/procedure",
	)
	after := before
	after.RoutedKnowledge = append(
		append([]actorContextR6ISharedKnowledgeProbe(nil), before.RoutedKnowledge...),
		actorContextR6ISharedKnowledgeProbe{
			KnowledgeKey:  "application-pattern|order_entry.customer_reference",
			KnowledgeKind: "application-pattern",
			RequiredInputFingerprints: []actorContextR0InputFingerprint{{
				Scope:       "application-pattern",
				Artifact:    "test-only:r6p/customer-reference",
				Fingerprint: "pattern-revision-a",
			}},
			SourceRefs: []string{"test-only:r6p/customer-reference"},
		},
	)
	after.RoutedUseEdges = append(
		append([]actorContextR6IUseEdgeProbe(nil), before.RoutedUseEdges...),
		actorContextR6IUseEdgeProbe{
			KnowledgeKey:       "application-pattern|order_entry.customer_reference",
			ProjectionCategory: actorContextR3PCategorySourceRefs,
			DependencyKey:      "reference:test-only:r6p/customer-reference",
			DependencyReason:   actorContextR3SReasonDrillDownLocator,
		},
	)

	got, err := actorContextR6PDecomposeSameScopeObservation(
		before,
		after,
		expectations,
	)
	if err != nil {
		t.Fatal(err)
	}

	knowledgeCardinality := actorContextR6PCardinalityDelta(
		t,
		got.CardinalityDeltas,
		actorContextR6KPlaneRoutedKnowledge,
	)
	knowledgeMembership := actorContextR6NPlaneDelta(
		t,
		got.MembershipDeltas,
		actorContextR6KPlaneRoutedKnowledge,
	)
	if knowledgeCardinality.CountDelta != 1 ||
		!reflect.DeepEqual(
			knowledgeMembership.Added,
			[]string{"application-pattern|order_entry.customer_reference"},
		) ||
		len(knowledgeMembership.Removed) != 0 {
		t.Fatalf(
			"R6P added knowledge did not stay in membership/cardinality axes: cardinality=%+v membership=%+v",
			knowledgeCardinality,
			knowledgeMembership,
		)
	}
	if len(got.RetainedKnowledgeProvenance) != 0 {
		t.Fatalf(
			"R6P added member was incorrectly reported as retained provenance delta: %+v",
			got.RetainedKnowledgeProvenance,
		)
	}
}

func TestActorContextR6PDecompositionHasNoSemanticFreshnessOrAuthorityShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR6PObservationDecompositionProbe{})
	for _, forbidden := range []string{
		"Changed",
		"SemanticChanged",
		"ScopeChanged",
		"Fresh",
		"Stale",
		"Current",
		"ShouldReevaluate",
		"Rebuild",
		"Complete",
		"Ready",
		"Authorized",
		"CanExecute",
		"NextAction",
		"Priority",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf(
				"R6P observation decomposition introduced forbidden shortcut field %q",
				forbidden,
			)
		}
	}
}

func actorContextR6PDecomposeSameScopeObservation(
	before,
	after actorContextR6JBoundedInputAssemblyProbe,
	expectations []actorContextR6KPlaneExpectationProbe,
) (actorContextR6PObservationDecompositionProbe, error) {
	beforeBinding, err := actorContextR6LBindScope(before, expectations)
	if err != nil {
		return actorContextR6PObservationDecompositionProbe{}, err
	}
	afterBinding, err := actorContextR6LBindScope(after, expectations)
	if err != nil {
		return actorContextR6PObservationDecompositionProbe{}, err
	}

	cardinality, err := actorContextR6MCompareSameScopeObservations(
		beforeBinding,
		afterBinding,
	)
	if err != nil {
		return actorContextR6PObservationDecompositionProbe{}, err
	}

	beforeMembership, err := actorContextR6NBuildScopedMembership(
		before,
		expectations,
	)
	if err != nil {
		return actorContextR6PObservationDecompositionProbe{}, err
	}
	afterMembership, err := actorContextR6NBuildScopedMembership(
		after,
		expectations,
	)
	if err != nil {
		return actorContextR6PObservationDecompositionProbe{}, err
	}
	membership, err := actorContextR6NCompareSameScopeMembership(
		beforeMembership,
		afterMembership,
	)
	if err != nil {
		return actorContextR6PObservationDecompositionProbe{}, err
	}

	provenance, err := actorContextR6OCompareRetainedKnowledgeProvenance(
		before,
		after,
		expectations,
	)
	if err != nil {
		return actorContextR6PObservationDecompositionProbe{}, err
	}

	if cardinality.ScopeIdentity != membership.ScopeIdentity ||
		cardinality.ScopeIdentity != provenance.ScopeIdentity {
		return actorContextR6PObservationDecompositionProbe{}, errProcedureMeaning(
			"R6P component observations do not share one ScopeIdentity",
		)
	}

	for _, cardinalityDelta := range cardinality.PlaneDeltas {
		membershipDelta, ok := actorContextR6PMembershipDeltaForPlane(
			membership.PlaneDeltas,
			cardinalityDelta.PlaneClass,
		)
		if !ok {
			return actorContextR6PObservationDecompositionProbe{}, errProcedureMeaning(
				"R6P membership observation lost plane " + cardinalityDelta.PlaneClass,
			)
		}
		expectedDelta := len(membershipDelta.Added) - len(membershipDelta.Removed)
		if cardinalityDelta.CountDelta != expectedDelta {
			return actorContextR6PObservationDecompositionProbe{}, errProcedureMeaning(
				"R6P cardinality/member delta mismatch for plane " + cardinalityDelta.PlaneClass,
			)
		}
	}

	knowledgeMembership, ok := actorContextR6PMembershipDeltaForPlane(
		membership.PlaneDeltas,
		actorContextR6KPlaneRoutedKnowledge,
	)
	if !ok {
		return actorContextR6PObservationDecompositionProbe{}, errProcedureMeaning(
			"R6P membership observation lost routed knowledge plane",
		)
	}
	nonRetained := map[string]bool{}
	for _, key := range knowledgeMembership.Added {
		nonRetained[key] = true
	}
	for _, key := range knowledgeMembership.Removed {
		nonRetained[key] = true
	}
	for _, delta := range provenance.KnowledgeDeltas {
		if nonRetained[delta.KnowledgeKey] {
			return actorContextR6PObservationDecompositionProbe{}, errProcedureMeaning(
				"R6P provenance delta references added/removed knowledge member " + delta.KnowledgeKey,
			)
		}
	}

	return actorContextR6PObservationDecompositionProbe{
		ScopeIdentity:               cardinality.ScopeIdentity,
		CardinalityDeltas:           cardinality.PlaneDeltas,
		MembershipDeltas:            membership.PlaneDeltas,
		RetainedKnowledgeProvenance: provenance.KnowledgeDeltas,
		AuthorityLimits: []string{
			"cardinality-membership-and-provenance-are-separate-observation-axes",
			"cardinality-delta-must-equal-added-minus-removed-membership",
			"retained-provenance-delta-cannot-reference-added-or-removed-member",
			"observation-decomposition-does-not-classify-semantic-or-freshness-change",
			"observation-decomposition-does-not-create-authority-or-response-action",
		},
	}, nil
}

func actorContextR6PMembershipDeltaForPlane(
	values []actorContextR6NPlaneMembershipDeltaProbe,
	plane string,
) (actorContextR6NPlaneMembershipDeltaProbe, bool) {
	for _, value := range values {
		if value.PlaneClass == plane {
			return value, true
		}
	}
	return actorContextR6NPlaneMembershipDeltaProbe{}, false
}

func actorContextR6PCardinalityDelta(
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
	t.Fatalf("R6P cardinality delta not found for %q: %+v", plane, values)
	return actorContextR6MPlaneCardinalityDeltaProbe{}
}

func actorContextR6PRequireZeroCardinality(
	t *testing.T,
	values []actorContextR6MPlaneCardinalityDeltaProbe,
) {
	t.Helper()
	for _, value := range values {
		if value.CountDelta != 0 {
			t.Fatalf("R6P expected zero cardinality delta: %+v", values)
		}
	}
}

func actorContextR6PRequireZeroMembership(
	t *testing.T,
	values []actorContextR6NPlaneMembershipDeltaProbe,
) {
	t.Helper()
	for _, value := range values {
		if len(value.Added) != 0 || len(value.Removed) != 0 {
			t.Fatalf("R6P expected zero membership delta: %+v", values)
		}
	}
}

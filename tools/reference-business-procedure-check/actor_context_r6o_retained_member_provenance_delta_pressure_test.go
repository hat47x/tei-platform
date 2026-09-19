package main

import (
	"reflect"
	"sort"
	"testing"
)

type actorContextR6OKnowledgeProvenanceSnapshotProbe struct {
	KnowledgeKey              string                            `json:"knowledgeKey"`
	KnowledgeKind             string                            `json:"knowledgeKind"`
	RequiredInputFingerprints []actorContextR0InputFingerprint `json:"requiredInputFingerprints"`
	SourceRefs                []string                          `json:"sourceRefs"`
}

type actorContextR6OKnowledgeProvenanceDeltaProbe struct {
	KnowledgeKey string                                      `json:"knowledgeKey"`
	Before       actorContextR6OKnowledgeProvenanceSnapshotProbe `json:"before"`
	After        actorContextR6OKnowledgeProvenanceSnapshotProbe `json:"after"`
}

type actorContextR6ORetainedMemberProvenanceDeltaProbe struct {
	ScopeIdentity   string                                      `json:"scopeIdentity"`
	BeforeAudience  string                                      `json:"beforeAudience"`
	BeforePurpose   string                                      `json:"beforePurpose"`
	AfterAudience   string                                      `json:"afterAudience"`
	AfterPurpose    string                                      `json:"afterPurpose"`
	KnowledgeDeltas []actorContextR6OKnowledgeProvenanceDeltaProbe `json:"knowledgeDeltas"`
	AuthorityLimits []string                                    `json:"authorityLimits"`
}

func TestActorContextR6ORetainedKnowledgeRevisionIsSeparateFromMembership(t *testing.T) {
	expectations := actorContextR6LRoutedOnlyExpectations()
	beforeAssembly := actorContextR6OProcedureOnlyAssembly("revision-a", "test-only:r6o/procedure")
	afterAssembly := actorContextR6OProcedureOnlyAssembly("revision-b", "test-only:r6o/procedure")

	beforeMembership, err := actorContextR6NBuildScopedMembership(beforeAssembly, expectations)
	if err != nil {
		t.Fatal(err)
	}
	afterMembership, err := actorContextR6NBuildScopedMembership(afterAssembly, expectations)
	if err != nil {
		t.Fatal(err)
	}
	membershipDelta, err := actorContextR6NCompareSameScopeMembership(
		beforeMembership,
		afterMembership,
	)
	if err != nil {
		t.Fatal(err)
	}

	knowledgeMembership := actorContextR6NPlaneDelta(
		t,
		membershipDelta.PlaneDeltas,
		actorContextR6KPlaneRoutedKnowledge,
	)
	if len(knowledgeMembership.Added) != 0 || len(knowledgeMembership.Removed) != 0 {
		t.Fatalf(
			"R6O setup converted source revision into membership change: %+v",
			knowledgeMembership,
		)
	}

	got, err := actorContextR6OCompareRetainedKnowledgeProvenance(
		beforeAssembly,
		afterAssembly,
		expectations,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.KnowledgeDeltas) != 1 {
		t.Fatalf(
			"R6O retained source revision was not visible as provenance delta: %+v",
			got,
		)
	}
	delta := got.KnowledgeDeltas[0]
	if delta.KnowledgeKey != "business-procedure-reference|order-entry.order-handling" {
		t.Fatalf("R6O provenance delta changed knowledge identity: %+v", delta)
	}
	if reflect.DeepEqual(
		delta.Before.RequiredInputFingerprints,
		delta.After.RequiredInputFingerprints,
	) {
		t.Fatalf("R6O provenance delta lost before/after fingerprint difference: %+v", delta)
	}
	if !reflect.DeepEqual(delta.Before.SourceRefs, delta.After.SourceRefs) {
		t.Fatalf("R6O fingerprint-only revision changed locator provenance unexpectedly: %+v", delta)
	}
}

func TestActorContextR6OSourceRefDifferenceIsObservedWithoutSemanticClassification(t *testing.T) {
	expectations := actorContextR6LRoutedOnlyExpectations()
	beforeAssembly := actorContextR6OProcedureOnlyAssembly(
		"same-fingerprint",
		"test-only:r6o/procedure-a",
	)
	afterAssembly := actorContextR6OProcedureOnlyAssembly(
		"same-fingerprint",
		"test-only:r6o/procedure-b",
	)

	got, err := actorContextR6OCompareRetainedKnowledgeProvenance(
		beforeAssembly,
		afterAssembly,
		expectations,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.KnowledgeDeltas) != 1 {
		t.Fatalf("R6O locator/provenance difference was not retained: %+v", got)
	}
	delta := got.KnowledgeDeltas[0]
	if !reflect.DeepEqual(
		delta.Before.RequiredInputFingerprints,
		delta.After.RequiredInputFingerprints,
	) {
		t.Fatalf("R6O locator-only setup changed fingerprints: %+v", delta)
	}
	if reflect.DeepEqual(delta.Before.SourceRefs, delta.After.SourceRefs) {
		t.Fatalf("R6O locator-only provenance difference disappeared: %+v", delta)
	}
}

func TestActorContextR6OUseMembershipChangeDoesNotFabricateKnowledgeProvenanceDelta(t *testing.T) {
	expectations := actorContextR6LRoutedOnlyExpectations()
	beforeAssembly := actorContextR6OProcedureOnlyAssembly(
		"same-fingerprint",
		"test-only:r6o/procedure",
	)
	afterAssembly := beforeAssembly
	afterAssembly.RoutedUseEdges = append(
		append([]actorContextR6IUseEdgeProbe(nil), beforeAssembly.RoutedUseEdges...),
		actorContextR6IUseEdgeProbe{
			KnowledgeKey:       "business-procedure-reference|order-entry.order-handling",
			ProjectionCategory: actorContextR3PCategoryKnownAuthorityFacts,
			DependencyKey:      "reference:test-only:r6o/procedure",
			DependencyReason:   actorContextR3SReasonAuthorizationRequirementSource,
		},
	)

	beforeMembership, err := actorContextR6NBuildScopedMembership(beforeAssembly, expectations)
	if err != nil {
		t.Fatal(err)
	}
	afterMembership, err := actorContextR6NBuildScopedMembership(afterAssembly, expectations)
	if err != nil {
		t.Fatal(err)
	}
	membershipDelta, err := actorContextR6NCompareSameScopeMembership(
		beforeMembership,
		afterMembership,
	)
	if err != nil {
		t.Fatal(err)
	}
	useDelta := actorContextR6NPlaneDelta(
		t,
		membershipDelta.PlaneDeltas,
		actorContextR6KPlaneRoutedUse,
	)
	if len(useDelta.Added) != 1 || len(useDelta.Removed) != 0 {
		t.Fatalf("R6O setup lost use membership expansion: %+v", useDelta)
	}

	got, err := actorContextR6OCompareRetainedKnowledgeProvenance(
		beforeAssembly,
		afterAssembly,
		expectations,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.KnowledgeDeltas) != 0 {
		t.Fatalf(
			"R6O use membership expansion fabricated source provenance change: %+v",
			got.KnowledgeDeltas,
		)
	}
}

func TestActorContextR6OAddedKnowledgeMemberIsNotMisreportedAsRevision(t *testing.T) {
	expectations := actorContextR6LRoutedOnlyExpectations()
	beforeAssembly := actorContextR6OProcedureOnlyAssembly(
		"same-fingerprint",
		"test-only:r6o/procedure",
	)
	afterAssembly := beforeAssembly
	afterAssembly.RoutedKnowledge = append(
		append([]actorContextR6ISharedKnowledgeProbe(nil), beforeAssembly.RoutedKnowledge...),
		actorContextR6ISharedKnowledgeProbe{
			KnowledgeKey:  "application-pattern|order_entry.customer_reference",
			KnowledgeKind: "application-pattern",
			RequiredInputFingerprints: []actorContextR0InputFingerprint{{
				Scope:       "application-pattern",
				Artifact:    "test-only:r6o/customer-reference",
				Fingerprint: "pattern-revision-a",
			}},
			SourceRefs: []string{"test-only:r6o/customer-reference"},
		},
	)
	afterAssembly.RoutedUseEdges = append(
		append([]actorContextR6IUseEdgeProbe(nil), beforeAssembly.RoutedUseEdges...),
		actorContextR6IUseEdgeProbe{
			KnowledgeKey:       "application-pattern|order_entry.customer_reference",
			ProjectionCategory: actorContextR3PCategorySourceRefs,
			DependencyKey:      "reference:test-only:r6o/customer-reference",
			DependencyReason:   actorContextR3SReasonDrillDownLocator,
		},
	)

	beforeMembership, err := actorContextR6NBuildScopedMembership(beforeAssembly, expectations)
	if err != nil {
		t.Fatal(err)
	}
	afterMembership, err := actorContextR6NBuildScopedMembership(afterAssembly, expectations)
	if err != nil {
		t.Fatal(err)
	}
	membershipDelta, err := actorContextR6NCompareSameScopeMembership(
		beforeMembership,
		afterMembership,
	)
	if err != nil {
		t.Fatal(err)
	}
	knowledgeMembership := actorContextR6NPlaneDelta(
		t,
		membershipDelta.PlaneDeltas,
		actorContextR6KPlaneRoutedKnowledge,
	)
	if !reflect.DeepEqual(
		knowledgeMembership.Added,
		[]string{"application-pattern|order_entry.customer_reference"},
	) || len(knowledgeMembership.Removed) != 0 {
		t.Fatalf("R6O setup lost added knowledge membership: %+v", knowledgeMembership)
	}

	got, err := actorContextR6OCompareRetainedKnowledgeProvenance(
		beforeAssembly,
		afterAssembly,
		expectations,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.KnowledgeDeltas) != 0 {
		t.Fatalf(
			"R6O added knowledge member was misreported as retained-member revision: %+v",
			got.KnowledgeDeltas,
		)
	}
}

func TestActorContextR6ORejectsProvenanceComparisonAcrossDifferentScope(t *testing.T) {
	before := actorContextR6OProcedureOnlyAssembly(
		"revision-a",
		"test-only:r6o/procedure",
	)
	after := actorContextR6OProcedureOnlyAssembly(
		"revision-b",
		"test-only:r6o/procedure",
	)

	beforeExpectations := actorContextR6LRoutedOnlyExpectations()
	afterExpectations := actorContextR6LRoutedOnlyExpectations()
	for i := range afterExpectations {
		if afterExpectations[i].PlaneClass == actorContextR6KPlaneConsumerRuntime {
			afterExpectations[i].ScopeDisposition = actorContextR6KScopeRequired
		}
	}

	if _, err := actorContextR6OCompareRetainedKnowledgeProvenanceWithScopes(
		before,
		after,
		beforeExpectations,
		afterExpectations,
	); err == nil {
		t.Fatal("R6O compared retained-member provenance across different semantic input scopes")
	}
}

func TestActorContextR6OProvenanceDeltaHasNoFreshnessOrAuthorityShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR6ORetainedMemberProvenanceDeltaProbe{})
	for _, forbidden := range []string{
		"Changed",
		"ChangeClass",
		"SemanticChanged",
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
				"R6O retained-member provenance delta introduced forbidden shortcut %q",
				forbidden,
			)
		}
	}
}

func actorContextR6OCompareRetainedKnowledgeProvenance(
	before,
	after actorContextR6JBoundedInputAssemblyProbe,
	expectations []actorContextR6KPlaneExpectationProbe,
) (actorContextR6ORetainedMemberProvenanceDeltaProbe, error) {
	return actorContextR6OCompareRetainedKnowledgeProvenanceWithScopes(
		before,
		after,
		expectations,
		expectations,
	)
}

func actorContextR6OCompareRetainedKnowledgeProvenanceWithScopes(
	before,
	after actorContextR6JBoundedInputAssemblyProbe,
	beforeExpectations,
	afterExpectations []actorContextR6KPlaneExpectationProbe,
) (actorContextR6ORetainedMemberProvenanceDeltaProbe, error) {
	beforeBinding, err := actorContextR6LBindScope(before, beforeExpectations)
	if err != nil {
		return actorContextR6ORetainedMemberProvenanceDeltaProbe{}, err
	}
	afterBinding, err := actorContextR6LBindScope(after, afterExpectations)
	if err != nil {
		return actorContextR6ORetainedMemberProvenanceDeltaProbe{}, err
	}
	if beforeBinding.ScopeIdentity != afterBinding.ScopeIdentity {
		return actorContextR6ORetainedMemberProvenanceDeltaProbe{}, errProcedureMeaning(
			"R6O refuses retained-member provenance comparison across different semantic input scopes",
		)
	}

	beforeKnowledge, err := actorContextR6OKnowledgeIndex(before.RoutedKnowledge)
	if err != nil {
		return actorContextR6ORetainedMemberProvenanceDeltaProbe{}, err
	}
	afterKnowledge, err := actorContextR6OKnowledgeIndex(after.RoutedKnowledge)
	if err != nil {
		return actorContextR6ORetainedMemberProvenanceDeltaProbe{}, err
	}

	deltas := []actorContextR6OKnowledgeProvenanceDeltaProbe{}
	for key, beforeValue := range beforeKnowledge {
		afterValue, retained := afterKnowledge[key]
		if !retained {
			continue
		}
		beforeSnapshot := actorContextR6ONormalizeKnowledgeSnapshot(beforeValue)
		afterSnapshot := actorContextR6ONormalizeKnowledgeSnapshot(afterValue)
		if reflect.DeepEqual(beforeSnapshot, afterSnapshot) {
			continue
		}
		if beforeSnapshot.KnowledgeKind != afterSnapshot.KnowledgeKind {
			return actorContextR6ORetainedMemberProvenanceDeltaProbe{}, errProcedureMeaning(
				"R6O retained knowledge key changed knowledge kind " + key,
			)
		}
		deltas = append(deltas, actorContextR6OKnowledgeProvenanceDeltaProbe{
			KnowledgeKey: key,
			Before:       beforeSnapshot,
			After:        afterSnapshot,
		})
	}
	sort.Slice(deltas, func(i, j int) bool {
		return deltas[i].KnowledgeKey < deltas[j].KnowledgeKey
	})

	return actorContextR6ORetainedMemberProvenanceDeltaProbe{
		ScopeIdentity:   beforeBinding.ScopeIdentity,
		BeforeAudience:  before.Audience,
		BeforePurpose:   before.Purpose,
		AfterAudience:   after.Audience,
		AfterPurpose:    after.Purpose,
		KnowledgeDeltas: deltas,
		AuthorityLimits: []string{
			"retained-member-provenance-delta-is-separate-from-membership-delta",
			"added-or-removed-knowledge-members-are-not-reported-as-revisions",
			"fingerprint-or-source-ref-difference-does-not-decide-semantic-change",
			"provenance-delta-does-not-decide-freshness-reevaluation-or-authority",
			"comparison-is-only-valid-within-one-scope-identity",
		},
	}, nil
}

func actorContextR6OKnowledgeIndex(
	values []actorContextR6ISharedKnowledgeProbe,
) (map[string]actorContextR6ISharedKnowledgeProbe, error) {
	out := map[string]actorContextR6ISharedKnowledgeProbe{}
	for _, value := range values {
		if value.KnowledgeKey == "" {
			return nil, errProcedureMeaning("R6O routed knowledge member has empty KnowledgeKey")
		}
		if _, exists := out[value.KnowledgeKey]; exists {
			return nil, errProcedureMeaning(
				"R6O duplicate routed knowledge member " + value.KnowledgeKey,
			)
		}
		out[value.KnowledgeKey] = value
	}
	return out, nil
}

func actorContextR6ONormalizeKnowledgeSnapshot(
	value actorContextR6ISharedKnowledgeProbe,
) actorContextR6OKnowledgeProvenanceSnapshotProbe {
	fingerprints := append(
		[]actorContextR0InputFingerprint(nil),
		value.RequiredInputFingerprints...,
	)
	sort.Slice(fingerprints, func(i, j int) bool {
		left := fingerprints[i].Scope + "|" +
			fingerprints[i].Artifact + "|" +
			fingerprints[i].Fingerprint
		right := fingerprints[j].Scope + "|" +
			fingerprints[j].Artifact + "|" +
			fingerprints[j].Fingerprint
		return left < right
	})
	return actorContextR6OKnowledgeProvenanceSnapshotProbe{
		KnowledgeKey:              value.KnowledgeKey,
		KnowledgeKind:             value.KnowledgeKind,
		RequiredInputFingerprints: fingerprints,
		SourceRefs: actorContextR0SortedUnique(
			append([]string(nil), value.SourceRefs...),
		),
	}
}

func actorContextR6OProcedureOnlyAssembly(
	fingerprint,
	sourceRef string,
) actorContextR6JBoundedInputAssemblyProbe {
	return actorContextR6JBoundedInputAssemblyProbe{
		Audience: actorContextR3PAudienceHuman,
		Purpose:  actorContextR3PPurposeOrientation,
		RoutedKnowledge: []actorContextR6ISharedKnowledgeProbe{{
			KnowledgeKey:  "business-procedure-reference|order-entry.order-handling",
			KnowledgeKind: "business-procedure-reference",
			RequiredInputFingerprints: []actorContextR0InputFingerprint{{
				Scope:       "business-procedure-reference",
				Artifact:    sourceRef,
				Fingerprint: fingerprint,
			}},
			SourceRefs: []string{sourceRef},
		}},
		RoutedUseEdges: []actorContextR6IUseEdgeProbe{{
			KnowledgeKey:       "business-procedure-reference|order-entry.order-handling",
			ProjectionCategory: actorContextR3PCategoryMachineReasons,
			DependencyKey:      "reference:" + sourceRef,
			DependencyReason:   actorContextR3SReasonAuthorizationRequirementSource,
		}},
	}
}

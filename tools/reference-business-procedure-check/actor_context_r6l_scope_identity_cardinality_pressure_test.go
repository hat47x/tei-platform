package main

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

type actorContextR6LScopeBindingProbe struct {
	Audience        string                                  `json:"audience"`
	Purpose         string                                  `json:"purpose"`
	ScopeIdentity   string                                  `json:"scopeIdentity"`
	Planes          []actorContextR6KPlaneObservationProbe `json:"planes"`
	AuthorityLimits []string                                `json:"authorityLimits"`
}

func TestActorContextR6LHumanAndAgentCanShareScopeIdentityWithDifferentUseCardinality(t *testing.T) {
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

	human, err := actorContextR6LBindScope(
		actorContextR6LAssemblyFromR6I(humanMaterialized),
		actorContextR6LRoutedOnlyExpectations(),
	)
	if err != nil {
		t.Fatal(err)
	}
	agent, err := actorContextR6LBindScope(
		actorContextR6LAssemblyFromR6I(agentMaterialized),
		actorContextR6LRoutedOnlyExpectations(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if human.ScopeIdentity != agent.ScopeIdentity {
		t.Fatalf(
			"R6L audience/purpose presentation changed semantic input scope identity: human=%q agent=%q",
			human.ScopeIdentity,
			agent.ScopeIdentity,
		)
	}

	humanUseCount := actorContextR6KObservedCount(
		t,
		human.Planes,
		actorContextR6KPlaneRoutedUse,
	)
	agentUseCount := actorContextR6KObservedCount(
		t,
		agent.Planes,
		actorContextR6KPlaneRoutedUse,
	)
	if humanUseCount != 1 || agentUseCount != 2 {
		t.Fatalf(
			"R6L setup lost bounded Human/Agent use cardinality difference: human=%d agent=%d",
			humanUseCount,
			agentUseCount,
		)
	}

	if actorContextR6KObservedCount(
		t,
		human.Planes,
		actorContextR6KPlaneRoutedKnowledge,
	) != 1 ||
		actorContextR6KObservedCount(
			t,
			agent.Planes,
			actorContextR6KPlaneRoutedKnowledge,
		) != 1 {
		t.Fatalf(
			"R6L shared scope duplicated knowledge bodies with use cardinality: human=%+v agent=%+v",
			human.Planes,
			agent.Planes,
		)
	}
}

func TestActorContextR6LObservedCountChangeDoesNotRewriteScopeIdentity(t *testing.T) {
	expectations := actorContextR6LRoutedOnlyExpectations()

	base := actorContextR6JBoundedInputAssemblyProbe{
		Audience: actorContextR3PAudienceAgent,
		Purpose:  actorContextR3PPurposeResolutionInspection,
		RoutedKnowledge: []actorContextR6ISharedKnowledgeProbe{{
			KnowledgeKey:  "business-procedure-reference|order-entry.order-handling",
			KnowledgeKind: "business-procedure-reference",
			SourceRefs:    []string{"test-only:r6l/procedure"},
		}},
		RoutedUseEdges: []actorContextR6IUseEdgeProbe{{
			KnowledgeKey:       "business-procedure-reference|order-entry.order-handling",
			ProjectionCategory: actorContextR3PCategoryMachineReasons,
			DependencyKey:      "reference:test-only:r6l/procedure",
			DependencyReason:   actorContextR3SReasonAuthorizationRequirementSource,
		}},
	}
	moreUses := base
	moreUses.RoutedUseEdges = append(
		append([]actorContextR6IUseEdgeProbe(nil), base.RoutedUseEdges...),
		actorContextR6IUseEdgeProbe{
			KnowledgeKey:       "business-procedure-reference|order-entry.order-handling",
			ProjectionCategory: actorContextR3PCategoryKnownAuthorityFacts,
			DependencyKey:      "reference:test-only:r6l/procedure",
			DependencyReason:   actorContextR3SReasonAuthorizationRequirementSource,
		},
	)

	first, err := actorContextR6LBindScope(base, expectations)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR6LBindScope(moreUses, expectations)
	if err != nil {
		t.Fatal(err)
	}

	if first.ScopeIdentity != second.ScopeIdentity {
		t.Fatalf(
			"R6L observed use count changed scope identity: first=%q second=%q",
			first.ScopeIdentity,
			second.ScopeIdentity,
		)
	}
	if actorContextR6KObservedCount(
		t, first.Planes, actorContextR6KPlaneRoutedUse,
	) == actorContextR6KObservedCount(
		t, second.Planes, actorContextR6KPlaneRoutedUse,
	) {
		t.Fatalf(
			"R6L setup did not change observed cardinality: first=%+v second=%+v",
			first.Planes,
			second.Planes,
		)
	}
}

func TestActorContextR6LScopeDeclarationChangeChangesScopeIdentity(t *testing.T) {
	routedOnly := actorContextR6LRoutedOnlyExpectations()
	withRuntime := append(
		[]actorContextR6KPlaneExpectationProbe(nil),
		routedOnly...,
	)
	for i := range withRuntime {
		if withRuntime[i].PlaneClass == actorContextR6KPlaneConsumerRuntime {
			withRuntime[i].ScopeDisposition = actorContextR6KScopeRequired
		}
	}

	first, err := actorContextR6LScopeIdentityFromExpectations(routedOnly)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR6LScopeIdentityFromExpectations(withRuntime)
	if err != nil {
		t.Fatal(err)
	}

	if first == second {
		t.Fatalf(
			"R6L semantic scope identity ignored declaration change: first=%q second=%q",
			first,
			second,
		)
	}
}

func TestActorContextR6LScopeIdentityIsOrderStable(t *testing.T) {
	first := actorContextR6LRoutedOnlyExpectations()
	second := append([]actorContextR6KPlaneExpectationProbe(nil), first...)
	for i, j := 0, len(second)-1; i < j; i, j = i+1, j-1 {
		second[i], second[j] = second[j], second[i]
	}

	firstID, err := actorContextR6LScopeIdentityFromExpectations(first)
	if err != nil {
		t.Fatal(err)
	}
	secondID, err := actorContextR6LScopeIdentityFromExpectations(second)
	if err != nil {
		t.Fatal(err)
	}

	if firstID != secondID {
		t.Fatalf(
			"R6L input declaration order changed scope identity: first=%q second=%q",
			firstID,
			secondID,
		)
	}
}

func TestActorContextR6LScopeBindingHasNoAuthorityOrCardinalityShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR6LScopeBindingProbe{})
	for _, forbidden := range []string{
		"UseCountIdentity",
		"KnowledgeCountIdentity",
		"Complete",
		"Ready",
		"Valid",
		"Fresh",
		"Stale",
		"Authorized",
		"CanExecute",
		"PreferredAudience",
		"Priority",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R6L scope binding introduced forbidden shortcut field %q", forbidden)
		}
	}
}

func actorContextR6LBindScope(
	assembly actorContextR6JBoundedInputAssemblyProbe,
	expectations []actorContextR6KPlaneExpectationProbe,
) (actorContextR6LScopeBindingProbe, error) {
	scopeIdentity, err := actorContextR6LScopeIdentityFromExpectations(expectations)
	if err != nil {
		return actorContextR6LScopeBindingProbe{}, err
	}
	witness, err := actorContextR6KObserveScope(assembly, expectations)
	if err != nil {
		return actorContextR6LScopeBindingProbe{}, err
	}

	return actorContextR6LScopeBindingProbe{
		Audience:      assembly.Audience,
		Purpose:       assembly.Purpose,
		ScopeIdentity: scopeIdentity,
		Planes:        witness.Planes,
		AuthorityLimits: []string{
			"scope-identity-is-declaration-based-not-observed-cardinality-based",
			"audience-purpose-binding-does-not-enter-shared-scope-identity",
			"more-use-edges-do-not-create-a-different-input-scope",
			"scope-identity-does-not-create-completeness-freshness-or-authority",
		},
	}, nil
}

func actorContextR6LScopeIdentityFromExpectations(
	expectations []actorContextR6KPlaneExpectationProbe,
) (string, error) {
	known := map[string]bool{}
	for _, plane := range actorContextR6KAllPlanes {
		known[plane] = true
	}

	seen := map[string]string{}
	for _, expectation := range expectations {
		if !known[expectation.PlaneClass] {
			return "", errProcedureMeaning(
				"R6L unknown input plane " + expectation.PlaneClass,
			)
		}
		if _, exists := seen[expectation.PlaneClass]; exists {
			return "", errProcedureMeaning(
				"R6L duplicate input plane expectation " + expectation.PlaneClass,
			)
		}
		switch expectation.ScopeDisposition {
		case actorContextR6KScopeRequired, actorContextR6KScopeOutOfScope:
		default:
			return "", errProcedureMeaning(
				"R6L unsupported scope disposition " + expectation.ScopeDisposition,
			)
		}
		seen[expectation.PlaneClass] = expectation.ScopeDisposition
	}
	if len(seen) != len(actorContextR6KAllPlanes) {
		return "", errProcedureMeaning(
			"R6L semantic scope identity requires explicit disposition for every known plane",
		)
	}

	parts := make([]string, 0, len(seen))
	for plane, disposition := range seen {
		parts = append(parts, plane+"="+disposition)
	}
	sort.Strings(parts)
	return strings.Join(parts, "|"), nil
}

func actorContextR6LRoutedOnlyExpectations() []actorContextR6KPlaneExpectationProbe {
	return []actorContextR6KPlaneExpectationProbe{
		{PlaneClass: actorContextR6KPlaneRoutedKnowledge, ScopeDisposition: actorContextR6KScopeRequired},
		{PlaneClass: actorContextR6KPlaneRoutedUse, ScopeDisposition: actorContextR6KScopeRequired},
		{PlaneClass: actorContextR6KPlaneConsumerReference, ScopeDisposition: actorContextR6KScopeOutOfScope},
		{PlaneClass: actorContextR6KPlaneConsumerRuntime, ScopeDisposition: actorContextR6KScopeOutOfScope},
		{PlaneClass: actorContextR6KPlaneConsumerSemantic, ScopeDisposition: actorContextR6KScopeOutOfScope},
	}
}

func actorContextR6LAssemblyFromR6I(
	value actorContextR6IBoundedMaterializationProbe,
) actorContextR6JBoundedInputAssemblyProbe {
	return actorContextR6JBoundedInputAssemblyProbe{
		Audience:                 value.Audience,
		Purpose:                  value.Purpose,
		RoutedKnowledge:          append([]actorContextR6ISharedKnowledgeProbe(nil), value.MaterializedKnowledge...),
		RoutedUseEdges:           append([]actorContextR6IUseEdgeProbe(nil), value.UseEdges...),
		QualifiedNotMaterialized: append([]string(nil), value.QualifiedNotMaterialized...),
	}
}

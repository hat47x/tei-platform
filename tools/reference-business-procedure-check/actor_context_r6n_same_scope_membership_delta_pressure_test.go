package main

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

type actorContextR6NPlaneMembershipProbe struct {
	PlaneClass string   `json:"planeClass"`
	Members    []string `json:"members"`
}

type actorContextR6NScopedMembershipProbe struct {
	Audience        string                                `json:"audience"`
	Purpose         string                                `json:"purpose"`
	ScopeIdentity   string                                `json:"scopeIdentity"`
	PlaneMembers    []actorContextR6NPlaneMembershipProbe `json:"planeMembers"`
	AuthorityLimits []string                              `json:"authorityLimits"`
}

type actorContextR6NPlaneMembershipDeltaProbe struct {
	PlaneClass string   `json:"planeClass"`
	Added      []string `json:"added,omitempty"`
	Removed    []string `json:"removed,omitempty"`
}

type actorContextR6NSameScopeMembershipDeltaProbe struct {
	ScopeIdentity   string                                     `json:"scopeIdentity"`
	BeforeAudience  string                                     `json:"beforeAudience"`
	BeforePurpose   string                                     `json:"beforePurpose"`
	AfterAudience   string                                     `json:"afterAudience"`
	AfterPurpose    string                                     `json:"afterPurpose"`
	PlaneDeltas     []actorContextR6NPlaneMembershipDeltaProbe `json:"planeDeltas"`
	AuthorityLimits []string                                   `json:"authorityLimits"`
}

func TestActorContextR6NHumanAgentUseExpansionPreservesKnowledgeMembership(t *testing.T) {
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
	human, err := actorContextR6NBuildScopedMembership(
		actorContextR6LAssemblyFromR6I(humanMaterialized),
		expectations,
	)
	if err != nil {
		t.Fatal(err)
	}
	agent, err := actorContextR6NBuildScopedMembership(
		actorContextR6LAssemblyFromR6I(agentMaterialized),
		expectations,
	)
	if err != nil {
		t.Fatal(err)
	}

	got, err := actorContextR6NCompareSameScopeMembership(human, agent)
	if err != nil {
		t.Fatal(err)
	}

	knowledgeDelta := actorContextR6NPlaneDelta(
		t,
		got.PlaneDeltas,
		actorContextR6KPlaneRoutedKnowledge,
	)
	if len(knowledgeDelta.Added) != 0 || len(knowledgeDelta.Removed) != 0 {
		t.Fatalf(
			"R6N Human/Agent use expansion changed shared routed knowledge membership: %+v",
			knowledgeDelta,
		)
	}

	useDelta := actorContextR6NPlaneDelta(
		t,
		got.PlaneDeltas,
		actorContextR6KPlaneRoutedUse,
	)
	if len(useDelta.Added) != 1 || len(useDelta.Removed) != 0 {
		t.Fatalf(
			"R6N Human/Agent use expansion did not expose exactly one added use member: %+v",
			useDelta,
		)
	}
	if !strings.Contains(useDelta.Added[0], actorContextR3PCategoryKnownAuthorityFacts) {
		t.Fatalf(
			"R6N added use member did not preserve Agent-only category identity: %+v",
			useDelta,
		)
	}
}

func TestActorContextR6NSameCountSubstitutionIsVisibleAsMembershipDelta(t *testing.T) {
	expectations := actorContextR6LRoutedOnlyExpectations()

	beforeAssembly := actorContextR6JBoundedInputAssemblyProbe{
		Audience: actorContextR3PAudienceAgent,
		Purpose:  actorContextR3PPurposeResolutionInspection,
		RoutedKnowledge: []actorContextR6ISharedKnowledgeProbe{{
			KnowledgeKey:  "business-procedure-reference|order-entry.order-handling",
			KnowledgeKind: "business-procedure-reference",
			SourceRefs:    []string{"test-only:r6n/procedure"},
		}},
		RoutedUseEdges: []actorContextR6IUseEdgeProbe{{
			KnowledgeKey:       "business-procedure-reference|order-entry.order-handling",
			ProjectionCategory: actorContextR3PCategoryMachineReasons,
			DependencyKey:      "reference:test-only:r6n/procedure",
			DependencyReason:   actorContextR3SReasonAuthorizationRequirementSource,
		}},
	}
	afterAssembly := beforeAssembly
	afterAssembly.RoutedUseEdges = []actorContextR6IUseEdgeProbe{{
		KnowledgeKey:       "business-procedure-reference|order-entry.order-handling",
		ProjectionCategory: actorContextR3PCategoryKnownAuthorityFacts,
		DependencyKey:      "reference:test-only:r6n/procedure",
		DependencyReason:   actorContextR3SReasonAuthorizationRequirementSource,
	}}

	before, err := actorContextR6NBuildScopedMembership(beforeAssembly, expectations)
	if err != nil {
		t.Fatal(err)
	}
	after, err := actorContextR6NBuildScopedMembership(afterAssembly, expectations)
	if err != nil {
		t.Fatal(err)
	}

	got, err := actorContextR6NCompareSameScopeMembership(before, after)
	if err != nil {
		t.Fatal(err)
	}

	useDelta := actorContextR6NPlaneDelta(
		t,
		got.PlaneDeltas,
		actorContextR6KPlaneRoutedUse,
	)
	if len(useDelta.Added) != 1 || len(useDelta.Removed) != 1 {
		t.Fatalf(
			"R6N same-count substitution was invisible at membership level: %+v",
			useDelta,
		)
	}
	if actorContextR6NPlaneMembers(before, actorContextR6KPlaneRoutedUse) == nil ||
		len(actorContextR6NPlaneMembers(before, actorContextR6KPlaneRoutedUse)) !=
			len(actorContextR6NPlaneMembers(after, actorContextR6KPlaneRoutedUse)) {
		t.Fatalf("R6N substitution setup did not preserve routed-use cardinality")
	}
}

func TestActorContextR6NSourceRevisionDoesNotChangeKnowledgeMembership(t *testing.T) {
	expectations := actorContextR6LRoutedOnlyExpectations()
	beforeAssembly := actorContextR6JBoundedInputAssemblyProbe{
		Audience: actorContextR3PAudienceHuman,
		Purpose:  actorContextR3PPurposeOrientation,
		RoutedKnowledge: []actorContextR6ISharedKnowledgeProbe{{
			KnowledgeKey:  "business-procedure-reference|order-entry.order-handling",
			KnowledgeKind: "business-procedure-reference",
			RequiredInputFingerprints: []actorContextR0InputFingerprint{{
				Scope:       "business-procedure-reference",
				Artifact:    "test-only:r6n/procedure",
				Fingerprint: "revision-a",
			}},
			SourceRefs: []string{"test-only:r6n/procedure"},
		}},
		RoutedUseEdges: []actorContextR6IUseEdgeProbe{{
			KnowledgeKey:       "business-procedure-reference|order-entry.order-handling",
			ProjectionCategory: actorContextR3PCategoryMachineReasons,
			DependencyKey:      "reference:test-only:r6n/procedure",
			DependencyReason:   actorContextR3SReasonAuthorizationRequirementSource,
		}},
	}
	afterAssembly := beforeAssembly
	afterAssembly.RoutedKnowledge = append(
		[]actorContextR6ISharedKnowledgeProbe(nil),
		beforeAssembly.RoutedKnowledge...,
	)
	afterAssembly.RoutedKnowledge[0].RequiredInputFingerprints = []actorContextR0InputFingerprint{{
		Scope:       "business-procedure-reference",
		Artifact:    "test-only:r6n/procedure",
		Fingerprint: "revision-b",
	}}

	before, err := actorContextR6NBuildScopedMembership(beforeAssembly, expectations)
	if err != nil {
		t.Fatal(err)
	}
	after, err := actorContextR6NBuildScopedMembership(afterAssembly, expectations)
	if err != nil {
		t.Fatal(err)
	}
	got, err := actorContextR6NCompareSameScopeMembership(before, after)
	if err != nil {
		t.Fatal(err)
	}

	knowledgeDelta := actorContextR6NPlaneDelta(
		t,
		got.PlaneDeltas,
		actorContextR6KPlaneRoutedKnowledge,
	)
	if len(knowledgeDelta.Added) != 0 || len(knowledgeDelta.Removed) != 0 {
		t.Fatalf(
			"R6N source revision was incorrectly converted into knowledge membership change: %+v",
			knowledgeDelta,
		)
	}
}

func TestActorContextR6NRejectsMembershipComparisonAcrossScopeIdentity(t *testing.T) {
	first := actorContextR6NScopedMembershipProbe{
		ScopeIdentity: "scope:test-only:first",
	}
	second := actorContextR6NScopedMembershipProbe{
		ScopeIdentity: "scope:test-only:second",
	}

	if _, err := actorContextR6NCompareSameScopeMembership(first, second); err == nil {
		t.Fatal("R6N compared membership across different semantic input scopes")
	}
}

func TestActorContextR6NMembershipDeltaHasNoFreshnessOrAuthorityShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR6NSameScopeMembershipDeltaProbe{})
	for _, forbidden := range []string{
		"SemanticChanged",
		"ContentChanged",
		"Fresh",
		"Stale",
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
				"R6N membership delta introduced forbidden shortcut field %q",
				forbidden,
			)
		}
	}
}

func actorContextR6NBuildScopedMembership(
	assembly actorContextR6JBoundedInputAssemblyProbe,
	expectations []actorContextR6KPlaneExpectationProbe,
) (actorContextR6NScopedMembershipProbe, error) {
	binding, err := actorContextR6LBindScope(assembly, expectations)
	if err != nil {
		return actorContextR6NScopedMembershipProbe{}, err
	}

	byPlane := map[string][]string{}
	for _, item := range assembly.RoutedKnowledge {
		byPlane[actorContextR6KPlaneRoutedKnowledge] = append(
			byPlane[actorContextR6KPlaneRoutedKnowledge],
			item.KnowledgeKey,
		)
	}
	for _, edge := range assembly.RoutedUseEdges {
		byPlane[actorContextR6KPlaneRoutedUse] = append(
			byPlane[actorContextR6KPlaneRoutedUse],
			actorContextR6NRoutedUseIdentity(edge),
		)
	}
	for _, edge := range assembly.ConsumerLocalUseEdges {
		var plane string
		switch edge.OriginClass {
		case actorContextR6DOriginConsumerReference:
			plane = actorContextR6KPlaneConsumerReference
		case actorContextR6DOriginConsumerRuntime:
			plane = actorContextR6KPlaneConsumerRuntime
		case actorContextR6DOriginConsumerSemantic:
			plane = actorContextR6KPlaneConsumerSemantic
		default:
			return actorContextR6NScopedMembershipProbe{}, errProcedureMeaning(
				"R6N unsupported consumer-local origin " + edge.OriginClass,
			)
		}
		byPlane[plane] = append(
			byPlane[plane],
			actorContextR6NConsumerLocalUseIdentity(edge),
		)
	}

	planes := make([]actorContextR6NPlaneMembershipProbe, 0, len(actorContextR6KAllPlanes))
	for _, plane := range actorContextR6KAllPlanes {
		members := actorContextR0SortedUnique(append([]string(nil), byPlane[plane]...))
		planes = append(planes, actorContextR6NPlaneMembershipProbe{
			PlaneClass: plane,
			Members:    members,
		})
	}
	sort.Slice(planes, func(i, j int) bool {
		return planes[i].PlaneClass < planes[j].PlaneClass
	})

	return actorContextR6NScopedMembershipProbe{
		Audience:      binding.Audience,
		Purpose:       binding.Purpose,
		ScopeIdentity: binding.ScopeIdentity,
		PlaneMembers:  planes,
		AuthorityLimits: []string{
			"membership-identity-is-not-source-revision-identity",
			"same-count-member-substitution-remains-visible",
			"consumer-use-membership-does-not-strengthen-authority",
			"membership-observation-does-not-decide-freshness-or-semantic-change",
		},
	}, nil
}

func actorContextR6NCompareSameScopeMembership(
	before,
	after actorContextR6NScopedMembershipProbe,
) (actorContextR6NSameScopeMembershipDeltaProbe, error) {
	if before.ScopeIdentity == "" || after.ScopeIdentity == "" {
		return actorContextR6NSameScopeMembershipDeltaProbe{}, errProcedureMeaning(
			"R6N membership comparison requires scope identity",
		)
	}
	if before.ScopeIdentity != after.ScopeIdentity {
		return actorContextR6NSameScopeMembershipDeltaProbe{}, errProcedureMeaning(
			"R6N refuses membership comparison across different semantic input scopes",
		)
	}

	beforeByPlane, err := actorContextR6NMembershipIndex(before.PlaneMembers)
	if err != nil {
		return actorContextR6NSameScopeMembershipDeltaProbe{}, err
	}
	afterByPlane, err := actorContextR6NMembershipIndex(after.PlaneMembers)
	if err != nil {
		return actorContextR6NSameScopeMembershipDeltaProbe{}, err
	}
	if len(beforeByPlane) != len(afterByPlane) {
		return actorContextR6NSameScopeMembershipDeltaProbe{}, errProcedureMeaning(
			"R6N same-scope membership observations cover different plane counts",
		)
	}

	deltas := make([]actorContextR6NPlaneMembershipDeltaProbe, 0, len(beforeByPlane))
	for plane, beforeMembers := range beforeByPlane {
		afterMembers, ok := afterByPlane[plane]
		if !ok {
			return actorContextR6NSameScopeMembershipDeltaProbe{}, errProcedureMeaning(
				"R6N same-scope after membership lost plane " + plane,
			)
		}
		added, removed := actorContextR6NSetDelta(beforeMembers, afterMembers)
		deltas = append(deltas, actorContextR6NPlaneMembershipDeltaProbe{
			PlaneClass: plane,
			Added:      added,
			Removed:    removed,
		})
	}
	sort.Slice(deltas, func(i, j int) bool {
		return deltas[i].PlaneClass < deltas[j].PlaneClass
	})

	return actorContextR6NSameScopeMembershipDeltaProbe{
		ScopeIdentity:  before.ScopeIdentity,
		BeforeAudience: before.Audience,
		BeforePurpose:  before.Purpose,
		AfterAudience:  after.Audience,
		AfterPurpose:   after.Purpose,
		PlaneDeltas:    deltas,
		AuthorityLimits: []string{
			"membership-delta-is-only-valid-within-one-scope-identity",
			"same-count-substitution-is-not-treated-as-no-observation-change",
			"source-revision-change-does-not-imply-membership-change",
			"membership-delta-does-not-decide-semantic-freshness-or-authority",
		},
	}, nil
}

func actorContextR6NMembershipIndex(
	values []actorContextR6NPlaneMembershipProbe,
) (map[string][]string, error) {
	out := map[string][]string{}
	for _, value := range values {
		if _, exists := out[value.PlaneClass]; exists {
			return nil, errProcedureMeaning(
				"R6N duplicate plane membership " + value.PlaneClass,
			)
		}
		out[value.PlaneClass] = actorContextR0SortedUnique(
			append([]string(nil), value.Members...),
		)
	}
	return out, nil
}

func actorContextR6NSetDelta(
	before,
	after []string,
) ([]string, []string) {
	beforeSet := map[string]bool{}
	afterSet := map[string]bool{}
	for _, value := range before {
		beforeSet[value] = true
	}
	for _, value := range after {
		afterSet[value] = true
	}

	added := []string{}
	removed := []string{}
	for value := range afterSet {
		if !beforeSet[value] {
			added = append(added, value)
		}
	}
	for value := range beforeSet {
		if !afterSet[value] {
			removed = append(removed, value)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

func actorContextR6NRoutedUseIdentity(
	value actorContextR6IUseEdgeProbe,
) string {
	return strings.Join([]string{
		value.KnowledgeKey,
		value.ProjectionCategory,
		value.DependencyKey,
		value.DependencyReason,
	}, "|")
}

func actorContextR6NConsumerLocalUseIdentity(
	value actorContextR6JConsumerLocalUseProbe,
) string {
	return strings.Join([]string{
		value.OriginClass,
		value.ProjectionCategory,
		value.DependencyKey,
		value.DependencyReason,
	}, "|")
}

func actorContextR6NPlaneDelta(
	t *testing.T,
	values []actorContextR6NPlaneMembershipDeltaProbe,
	plane string,
) actorContextR6NPlaneMembershipDeltaProbe {
	t.Helper()
	for _, value := range values {
		if value.PlaneClass == plane {
			return value
		}
	}
	t.Fatalf("R6N plane membership delta not found for %q: %+v", plane, values)
	return actorContextR6NPlaneMembershipDeltaProbe{}
}

func actorContextR6NPlaneMembers(
	value actorContextR6NScopedMembershipProbe,
	plane string,
) []string {
	for _, item := range value.PlaneMembers {
		if item.PlaneClass == plane {
			return item.Members
		}
	}
	return nil
}

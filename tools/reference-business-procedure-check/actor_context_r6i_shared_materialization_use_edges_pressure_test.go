package main

import (
	"reflect"
	"sort"
	"testing"
)

type actorContextR6ISharedKnowledgeProbe struct {
	KnowledgeKey              string                           `json:"knowledgeKey"`
	KnowledgeKind             string                           `json:"knowledgeKind"`
	RequiredInputFingerprints []actorContextR0InputFingerprint `json:"requiredInputFingerprints"`
	SourceRefs                []string                         `json:"sourceRefs"`
}

type actorContextR6IUseEdgeProbe struct {
	KnowledgeKey       string `json:"knowledgeKey"`
	ProjectionCategory string `json:"projectionCategory"`
	DependencyKey      string `json:"dependencyKey"`
	DependencyReason   string `json:"dependencyReason"`
}

type actorContextR6IBoundedMaterializationProbe struct {
	Audience                 string                                 `json:"audience"`
	Purpose                  string                                 `json:"purpose"`
	MaterializedKnowledge    []actorContextR6ISharedKnowledgeProbe `json:"materializedKnowledge"`
	UseEdges                 []actorContextR6IUseEdgeProbe          `json:"useEdges"`
	QualifiedNotMaterialized []string                               `json:"qualifiedNotMaterialized"`
	AuthorityLimits          []string                               `json:"authorityLimits"`
}

func TestActorContextR6IHumanAndAgentShareKnowledgeMaterializationButNotUseEdges(t *testing.T) {
	root := repoPath()
	qualified := actorContextR6CCurrentBP06QualifiedKnowledge(t, root)
	humanHandoff, agentHandoff, procedureKey := actorContextR6IBP06AudienceHandoffs(t, root, qualified)

	human, err := actorContextR6IBuildBoundedMaterialization(qualified, humanHandoff)
	if err != nil {
		t.Fatal(err)
	}
	agent, err := actorContextR6IBuildBoundedMaterialization(qualified, agentHandoff)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(human.MaterializedKnowledge, agent.MaterializedKnowledge) {
		t.Fatalf(
			"R6I audience/purpose changed shared materialized knowledge identity or revision provenance: human=%+v agent=%+v",
			human.MaterializedKnowledge,
			agent.MaterializedKnowledge,
		)
	}
	if len(human.MaterializedKnowledge) != 1 {
		t.Fatalf(
			"R6I BP-06 expected one shared routed knowledge body, got=%+v",
			human.MaterializedKnowledge,
		)
	}
	procedure := human.MaterializedKnowledge[0]
	if procedure.KnowledgeKey != "business-procedure-reference|order-entry.order-handling" ||
		procedure.KnowledgeKind != "business-procedure-reference" ||
		len(procedure.RequiredInputFingerprints) == 0 ||
		len(procedure.SourceRefs) == 0 {
		t.Fatalf("R6I shared Procedure materialization lost provenance: %+v", procedure)
	}

	if len(human.UseEdges) != 1 {
		t.Fatalf("R6I Human use-edge count=%d want=1: %+v", len(human.UseEdges), human.UseEdges)
	}
	if len(agent.UseEdges) != 2 {
		t.Fatalf("R6I Agent use-edge count=%d want=2: %+v", len(agent.UseEdges), agent.UseEdges)
	}
	if actorContextR6IHasUseCategory(human.UseEdges, actorContextR3PCategoryKnownAuthorityFacts) {
		t.Fatalf("R6I forced Agent-only use into Human materialization: %+v", human.UseEdges)
	}
	if !actorContextR6IHasUseCategory(agent.UseEdges, actorContextR3PCategoryKnownAuthorityFacts) {
		t.Fatalf("R6I Agent lost known-authority-facts use edge: %+v", agent.UseEdges)
	}

	for _, edge := range append(
		append([]actorContextR6IUseEdgeProbe(nil), human.UseEdges...),
		agent.UseEdges...,
	) {
		if edge.KnowledgeKey != procedure.KnowledgeKey ||
			edge.DependencyKey != procedureKey ||
			edge.DependencyReason != actorContextR3SReasonAuthorizationRequirementSource {
			t.Fatalf("R6I audience-specific use changed shared dependency provenance: %+v", edge)
		}
	}

	if !reflect.DeepEqual(
		human.QualifiedNotMaterialized,
		[]string{"semantic-capability|security.authorize"},
	) || !reflect.DeepEqual(
		agent.QualifiedNotMaterialized,
		[]string{"semantic-capability|security.authorize"},
	) {
		t.Fatalf(
			"R6I audience expansion changed qualified-not-materialized knowledge: human=%v agent=%v",
			human.QualifiedNotMaterialized,
			agent.QualifiedNotMaterialized,
		)
	}
}

func TestActorContextR6IAgentAdditionalUseDoesNotDuplicateMaterializedKnowledge(t *testing.T) {
	root := repoPath()
	qualified := actorContextR6CCurrentBP06QualifiedKnowledge(t, root)
	_, agentHandoff, _ := actorContextR6IBP06AudienceHandoffs(t, root, qualified)

	got, err := actorContextR6IBuildBoundedMaterialization(qualified, agentHandoff)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.UseEdges) <= len(got.MaterializedKnowledge) {
		t.Fatalf(
			"R6I setup no longer demonstrates one knowledge body feeding multiple consumer uses: materialized=%d uses=%d",
			len(got.MaterializedKnowledge),
			len(got.UseEdges),
		)
	}

	keys := map[string]bool{}
	for _, item := range got.MaterializedKnowledge {
		if keys[item.KnowledgeKey] {
			t.Fatalf("R6I duplicated materialized knowledge body for audience-specific use: %+v", got.MaterializedKnowledge)
		}
		keys[item.KnowledgeKey] = true
	}
	for _, edge := range got.UseEdges {
		if !keys[edge.KnowledgeKey] {
			t.Fatalf("R6I use edge references knowledge not materialized in bounded view: %+v", edge)
		}
	}
}

func TestActorContextR6ISharedKnowledgeIdentityExcludesAudiencePurposeAndProjectionUse(t *testing.T) {
	typ := reflect.TypeOf(actorContextR6ISharedKnowledgeProbe{})
	for _, forbidden := range []string{
		"Audience",
		"Purpose",
		"ProjectionCategory",
		"DependencyKey",
		"DependencyReason",
		"UseCount",
		"Inline",
		"DrillDown",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf(
				"R6I shared materialized knowledge identity coupled to bounded consumer use through field %q",
				forbidden,
			)
		}
	}
}

func TestActorContextR6IBoundedUseHasNoAuthorityOrActionShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR6IBoundedMaterializationProbe{})
	for _, forbidden := range []string{
		"Fresh",
		"Stale",
		"Authorized",
		"Allowed",
		"CanExecute",
		"ToolCall",
		"InvocationPlan",
		"NextAction",
		"Priority",
		"Confidence",
		"PreferredAudience",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R6I bounded materialization introduced forbidden authority/action shortcut %q", forbidden)
		}
	}
}

func actorContextR6IBP06AudienceHandoffs(
	t *testing.T,
	root string,
	qualified []actorContextR6BHeterogeneousQualified,
) (
	actorContextR6CHandoffSetProbe,
	actorContextR6CHandoffSetProbe,
	string,
) {
	t.Helper()

	core := actorContextR3QCurrentBP06Core(t)
	deps := actorContextR3SCurrentBP06ReferenceEnvelope(t, root)
	procedureDep := actorContextR6CFindReferenceDependency(
		t,
		deps,
		"business-procedure-reference",
	)
	procedureKey := actorContextR3SReferenceKey(procedureDep)
	bindings := map[string][]string{
		"business-procedure-reference|order-entry.order-handling": {procedureKey},
	}

	humanView, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	humanReasons, err := actorContextR3SBuildReasonEnvelope(
		core,
		humanView,
		deps,
		[]actorContextR3SDependencyReasonEdgeProbe{{
			Category:      actorContextR3PCategoryMachineReasons,
			DependencyKey: procedureKey,
			ReasonCode:    actorContextR3SReasonAuthorizationRequirementSource,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	human, err := actorContextR6CBuildHandoff(qualified, humanReasons, bindings)
	if err != nil {
		t.Fatal(err)
	}

	agentView, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceAgent,
		actorContextR3PPurposeResolutionInspection,
	)
	if err != nil {
		t.Fatal(err)
	}
	agentReasons, err := actorContextR3SBuildReasonEnvelope(
		core,
		agentView,
		deps,
		[]actorContextR3SDependencyReasonEdgeProbe{
			{
				Category:      actorContextR3PCategoryMachineReasons,
				DependencyKey: procedureKey,
				ReasonCode:    actorContextR3SReasonAuthorizationRequirementSource,
			},
			{
				Category:      actorContextR3PCategoryKnownAuthorityFacts,
				DependencyKey: procedureKey,
				ReasonCode:    actorContextR3SReasonAuthorizationRequirementSource,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	agent, err := actorContextR6CBuildHandoff(qualified, agentReasons, bindings)
	if err != nil {
		t.Fatal(err)
	}

	return human, agent, procedureKey
}

func actorContextR6IBuildBoundedMaterialization(
	qualified []actorContextR6BHeterogeneousQualified,
	handoff actorContextR6CHandoffSetProbe,
) (actorContextR6IBoundedMaterializationProbe, error) {
	qualifiedByKey := map[string]actorContextR6BHeterogeneousQualified{}
	for _, item := range qualified {
		qualifiedByKey[actorContextR6BQualifiedKey(item)] = item
	}

	materializedKeys := map[string]bool{}
	useEdges := []actorContextR6IUseEdgeProbe{}
	for _, record := range handoff.Records {
		item, ok := qualifiedByKey[record.KnowledgeKey]
		if !ok {
			return actorContextR6IBoundedMaterializationProbe{}, errProcedureMeaning(
				"R6I handoff references non-qualified knowledge " + record.KnowledgeKey,
			)
		}
		_ = item

		switch record.Disposition {
		case actorContextR6CDispositionConsumedByProjection:
			if record.DependencyKey == "" ||
				record.DependencyReason == "" ||
				record.ProjectionCategory == "" {
				return actorContextR6IBoundedMaterializationProbe{}, errProcedureMeaning(
					"R6I consumed handoff record lost exact projection use provenance for " + record.KnowledgeKey,
				)
			}
			materializedKeys[record.KnowledgeKey] = true
			useEdges = append(useEdges, actorContextR6IUseEdgeProbe{
				KnowledgeKey:       record.KnowledgeKey,
				ProjectionCategory: record.ProjectionCategory,
				DependencyKey:      record.DependencyKey,
				DependencyReason:   record.DependencyReason,
			})
		case actorContextR6CDispositionQualifiedNotSelected:
			if record.DependencyKey != "" ||
				record.DependencyReason != "" ||
				record.ProjectionCategory != "" {
				return actorContextR6IBoundedMaterializationProbe{}, errProcedureMeaning(
					"R6I unselected handoff record acquired invented projection use for " + record.KnowledgeKey,
				)
			}
		default:
			return actorContextR6IBoundedMaterializationProbe{}, errProcedureMeaning(
				"R6I unsupported R6C disposition " + record.Disposition,
			)
		}
	}

	materialized := make([]actorContextR6ISharedKnowledgeProbe, 0, len(materializedKeys))
	qualifiedNotMaterialized := []string{}
	for key, item := range qualifiedByKey {
		if materializedKeys[key] {
			materialized = append(materialized, actorContextR6ISharedKnowledgeProbe{
				KnowledgeKey:              key,
				KnowledgeKind:             item.KnowledgeKind,
				RequiredInputFingerprints: actorContextR6HCloneFingerprints(item.RequiredInputFingerprints),
				SourceRefs:                append([]string(nil), item.SourceRefs...),
			})
			continue
		}
		qualifiedNotMaterialized = append(qualifiedNotMaterialized, key)
	}

	sort.Slice(materialized, func(i, j int) bool {
		return materialized[i].KnowledgeKey < materialized[j].KnowledgeKey
	})
	sort.Slice(useEdges, func(i, j int) bool {
		left := useEdges[i].KnowledgeKey + "|" +
			useEdges[i].ProjectionCategory + "|" +
			useEdges[i].DependencyKey + "|" +
			useEdges[i].DependencyReason
		right := useEdges[j].KnowledgeKey + "|" +
			useEdges[j].ProjectionCategory + "|" +
			useEdges[j].DependencyKey + "|" +
			useEdges[j].DependencyReason
		return left < right
	})
	sort.Strings(qualifiedNotMaterialized)

	return actorContextR6IBoundedMaterializationProbe{
		Audience:                 handoff.Audience,
		Purpose:                  handoff.Purpose,
		MaterializedKnowledge:    materialized,
		UseEdges:                 useEdges,
		QualifiedNotMaterialized: qualifiedNotMaterialized,
		AuthorityLimits: []string{
			"shared-knowledge-body-is-materialized-once-per-bounded-view",
			"audience-specific-use-is-an-edge-not-a-copy-of-knowledge",
			"more-use-edges-do-not-strengthen-knowledge-authority",
			"consumer-use-does-not-change-source-revision-provenance",
			"qualified-not-materialized-does-not-become-missing-source",
			"bounded-materialization-does-not-authorize-execution",
		},
	}, nil
}

func actorContextR6IHasUseCategory(
	values []actorContextR6IUseEdgeProbe,
	category string,
) bool {
	for _, value := range values {
		if value.ProjectionCategory == category {
			return true
		}
	}
	return false
}

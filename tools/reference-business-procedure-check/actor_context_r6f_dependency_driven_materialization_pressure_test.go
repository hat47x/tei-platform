package main

import (
	"reflect"
	"sort"
	"testing"
)

type actorContextR6FMaterializedKnowledgeProbe struct {
	KnowledgeKey  string   `json:"knowledgeKey"`
	KnowledgeKind string   `json:"knowledgeKind"`
	SourceRefs    []string `json:"sourceRefs"`
}

type actorContextR6FMaterializationManifestProbe struct {
	Audience                  string                                      `json:"audience"`
	Purpose                   string                                      `json:"purpose"`
	MaterializedKnowledge     []actorContextR6FMaterializedKnowledgeProbe `json:"materializedKnowledge"`
	QualifiedNotMaterialized  []string                                    `json:"qualifiedNotMaterialized"`
	ConsumerLocalDependencies []string                                    `json:"consumerLocalDependencies"`
	AuthorityLimits           []string                                    `json:"authorityLimits"`
}

func TestActorContextR6FMaterializesOnlyRoutedKnowledgeConsumedByProjection(t *testing.T) {
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

	exactOrigins := actorContextR6EBP06HumanOriginSnapshot(t, root, exact)
	broadOrigins := actorContextR6EBP06HumanOriginSnapshot(t, root, broad)

	exactManifest, err := actorContextR6FBuildMaterializationManifest(
		exact.Qualified,
		exactOrigins.Dependencies,
		exactOrigins.Dependencies[0].Category,
		exactOrigins.Dependencies[0].Category,
	)
	if err != nil {
		t.Fatal(err)
	}
	// Audience/Purpose are not carried by R6D records; use the actual bounded
	// projection identity rather than deriving them from dependency content.
	exactManifest.Audience = actorContextR3PAudienceHuman
	exactManifest.Purpose = actorContextR3PPurposeOrientation

	broadManifest, err := actorContextR6FBuildMaterializationManifest(
		broad.Qualified,
		broadOrigins.Dependencies,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(exactManifest, broadManifest) {
		t.Fatalf(
			"R6F router substitution changed materialized projection context: exact=%+v broad=%+v",
			exactManifest,
			broadManifest,
		)
	}

	if len(exactManifest.MaterializedKnowledge) != 1 {
		t.Fatalf(
			"R6F materialized unexpected number of routed knowledge items: %+v",
			exactManifest.MaterializedKnowledge,
		)
	}
	materialized := exactManifest.MaterializedKnowledge[0]
	if materialized.KnowledgeKey != "business-procedure-reference|order-entry.order-handling" ||
		materialized.KnowledgeKind != "business-procedure-reference" ||
		len(materialized.SourceRefs) == 0 {
		t.Fatalf("R6F lost routed Procedure provenance: %+v", materialized)
	}

	if !reflect.DeepEqual(
		exactManifest.QualifiedNotMaterialized,
		[]string{"semantic-capability|security.authorize"},
	) {
		t.Fatalf(
			"R6F did not keep qualified-but-unused Capability out of materialized context: %+v",
			exactManifest.QualifiedNotMaterialized,
		)
	}

	if len(exactManifest.ConsumerLocalDependencies) != 1 ||
		exactManifest.ConsumerLocalDependencies[0] == "" {
		t.Fatalf(
			"R6F lost consumer-local protected-target dependency: %+v",
			exactManifest.ConsumerLocalDependencies,
		)
	}

	if len(exact.Available) <= len(exactManifest.MaterializedKnowledge) {
		t.Fatalf(
			"R6F pressure no longer demonstrates context minimization: available=%d materialized=%d",
			len(exact.Available),
			len(exactManifest.MaterializedKnowledge),
		)
	}
}

func TestActorContextR6FConsumerRuntimeDependencyDoesNotMaterializeQualifiedKnowledge(t *testing.T) {
	root := repoPath()
	exact := actorContextR6EBP06RoutingProbe(
		t,
		root,
		"exact-task-router",
		actorContextR6BSelectTaskHints,
	)
	runtimeOrigins := actorContextR6EDeniedRuntimeOriginSet(t, exact.Qualified)

	got, err := actorContextR6FBuildMaterializationManifest(
		exact.Qualified,
		runtimeOrigins.Dependencies,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.MaterializedKnowledge) != 0 {
		t.Fatalf(
			"R6F consumer-local runtime Observation caused unrelated routed knowledge materialization: %+v",
			got.MaterializedKnowledge,
		)
	}
	if !reflect.DeepEqual(
		got.QualifiedNotMaterialized,
		[]string{
			"business-procedure-reference|order-entry.order-handling",
			"semantic-capability|security.authorize",
		},
	) {
		t.Fatalf(
			"R6F lost qualified-but-unconsumed knowledge audit set: %+v",
			got.QualifiedNotMaterialized,
		)
	}
	if len(got.ConsumerLocalDependencies) != 1 ||
		got.ConsumerLocalDependencies[0] == "" {
		t.Fatalf(
			"R6F lost consumer-local runtime dependency: %+v",
			got.ConsumerLocalDependencies,
		)
	}
}

func TestActorContextR6FMaterializationManifestIsNotUniversalContextPacket(t *testing.T) {
	typ := reflect.TypeOf(actorContextR6FMaterializationManifestProbe{})
	for _, forbidden := range []string{
		"ContextPacket",
		"Payload",
		"Prompt",
		"AllKnowledge",
		"SelectedCandidates",
		"RejectedCandidates",
		"RoutingMechanism",
		"Score",
		"Confidence",
		"Authorized",
		"CanExecute",
		"Fresh",
		"Stale",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R6F materialization manifest introduced forbidden universal/authority field %q", forbidden)
		}
	}
}

func actorContextR6FBuildMaterializationManifest(
	qualified []actorContextR6BHeterogeneousQualified,
	dependencies []actorContextR6DDependencyOriginProbe,
	audience,
	purpose string,
) (actorContextR6FMaterializationManifestProbe, error) {
	qualifiedByKey := map[string]actorContextR6BHeterogeneousQualified{}
	for _, item := range qualified {
		qualifiedByKey[actorContextR6BQualifiedKey(item)] = item
	}

	materializedKeys := map[string]bool{}
	consumerLocalKeys := map[string]bool{}
	for _, dependency := range dependencies {
		switch dependency.OriginClass {
		case actorContextR6DOriginRoutedKnowledge:
			if dependency.KnowledgeKey == "" {
				return actorContextR6FMaterializationManifestProbe{}, errProcedureMeaning(
					"R6F routed dependency has no knowledge key",
				)
			}
			if _, ok := qualifiedByKey[dependency.KnowledgeKey]; !ok {
				return actorContextR6FMaterializationManifestProbe{}, errProcedureMeaning(
					"R6F routed dependency references non-qualified knowledge " + dependency.KnowledgeKey,
				)
			}
			materializedKeys[dependency.KnowledgeKey] = true
		case actorContextR6DOriginConsumerReference,
			actorContextR6DOriginConsumerRuntime,
			actorContextR6DOriginConsumerSemantic:
			if dependency.KnowledgeKey != "" {
				return actorContextR6FMaterializationManifestProbe{}, errProcedureMeaning(
					"R6F consumer-local dependency unexpectedly carries routed knowledge " + dependency.KnowledgeKey,
				)
			}
			consumerLocalKeys[dependency.DependencyKey] = true
		default:
			return actorContextR6FMaterializationManifestProbe{}, errProcedureMeaning(
				"R6F unsupported dependency origin class " + dependency.OriginClass,
			)
		}
	}

	materialized := make([]actorContextR6FMaterializedKnowledgeProbe, 0, len(materializedKeys))
	qualifiedNotMaterialized := []string{}
	for key, item := range qualifiedByKey {
		if materializedKeys[key] {
			materialized = append(materialized, actorContextR6FMaterializedKnowledgeProbe{
				KnowledgeKey:  key,
				KnowledgeKind: item.KnowledgeKind,
				SourceRefs:    append([]string(nil), item.SourceRefs...),
			})
			continue
		}
		qualifiedNotMaterialized = append(qualifiedNotMaterialized, key)
	}
	sort.Slice(materialized, func(i, j int) bool {
		return materialized[i].KnowledgeKey < materialized[j].KnowledgeKey
	})
	sort.Strings(qualifiedNotMaterialized)

	consumerLocal := make([]string, 0, len(consumerLocalKeys))
	for key := range consumerLocalKeys {
		consumerLocal = append(consumerLocal, key)
	}
	sort.Strings(consumerLocal)

	return actorContextR6FMaterializationManifestProbe{
		Audience:                  audience,
		Purpose:                   purpose,
		MaterializedKnowledge:     materialized,
		QualifiedNotMaterialized:  qualifiedNotMaterialized,
		ConsumerLocalDependencies: consumerLocal,
		AuthorityLimits: []string{
			"qualification-does-not-force-materialization",
			"materialization-follows-actual-routed-dependency-consumption",
			"consumer-local-dependencies-do-not-pull-router-pool-into-context",
			"manifest-is-derived-and-rebuildable-not-a-universal-context-packet",
			"materialized-knowledge-does-not-grant-execution-authority",
		},
	}, nil
}

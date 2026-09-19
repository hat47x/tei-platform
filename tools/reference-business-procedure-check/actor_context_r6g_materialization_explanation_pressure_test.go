package main

import (
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR6GDispositionMaterialized    = "materialized"
	actorContextR6GDispositionNotMaterialized = "qualified-not-materialized"

	actorContextR6GReasonConsumedRoutedDependency = "consumed-routed-projection-dependency"
	actorContextR6GReasonNoRoutedDependency       = "no-consumed-routed-projection-dependency"
)

type actorContextR6GMaterializationWitnessProbe struct {
	KnowledgeKey   string   `json:"knowledgeKey"`
	KnowledgeKind  string   `json:"knowledgeKind"`
	Disposition    string   `json:"disposition"`
	ReasonCode     string   `json:"reasonCode"`
	DependencyKeys []string `json:"dependencyKeys,omitempty"`
	SourceRefs     []string `json:"sourceRefs"`
}

type actorContextR6GMaterializationExplanationProbe struct {
	Audience                  string                                      `json:"audience"`
	Purpose                   string                                      `json:"purpose"`
	Witnesses                 []actorContextR6GMaterializationWitnessProbe `json:"witnesses"`
	ConsumerLocalDependencies []string                                    `json:"consumerLocalDependencies"`
	AuthorityLimits           []string                                    `json:"authorityLimits"`
}

func TestActorContextR6GExplainsMaterializedAndOmittedQualifiedKnowledge(t *testing.T) {
	root := repoPath()
	routing := actorContextR6EBP06RoutingProbe(
		t,
		root,
		"exact-task-router",
		actorContextR6BSelectTaskHints,
	)
	origins := actorContextR6EBP06HumanOriginSnapshot(t, root, routing)

	got, err := actorContextR6GBuildExplanation(
		routing.Qualified,
		origins.Dependencies,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}

	procedure := actorContextR6GWitnessByKey(
		t,
		got.Witnesses,
		"business-procedure-reference|order-entry.order-handling",
	)
	if procedure.Disposition != actorContextR6GDispositionMaterialized ||
		procedure.ReasonCode != actorContextR6GReasonConsumedRoutedDependency ||
		len(procedure.DependencyKeys) == 0 ||
		len(procedure.SourceRefs) == 0 {
		t.Fatalf("R6G lost materialization provenance for Procedure knowledge: %+v", procedure)
	}

	capability := actorContextR6GWitnessByKey(
		t,
		got.Witnesses,
		"semantic-capability|security.authorize",
	)
	if capability.Disposition != actorContextR6GDispositionNotMaterialized ||
		capability.ReasonCode != actorContextR6GReasonNoRoutedDependency ||
		len(capability.DependencyKeys) != 0 ||
		len(capability.SourceRefs) == 0 {
		t.Fatalf("R6G did not explain qualified-but-not-materialized Capability: %+v", capability)
	}

	if len(got.ConsumerLocalDependencies) != 1 || got.ConsumerLocalDependencies[0] == "" {
		t.Fatalf("R6G lost consumer-local dependency audit identity: %+v", got.ConsumerLocalDependencies)
	}
}

func TestActorContextR6GExplanationMatchesR6FMaterializationWithoutRestoringOmittedPayload(t *testing.T) {
	root := repoPath()
	routing := actorContextR6EBP06RoutingProbe(
		t,
		root,
		"exact-task-router",
		actorContextR6BSelectTaskHints,
	)
	origins := actorContextR6EBP06HumanOriginSnapshot(t, root, routing)

	manifest, err := actorContextR6FBuildMaterializationManifest(
		routing.Qualified,
		origins.Dependencies,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	explanation, err := actorContextR6GBuildExplanation(
		routing.Qualified,
		origins.Dependencies,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}

	gotMaterialized := actorContextR6GWitnessKeysByDisposition(
		explanation.Witnesses,
		actorContextR6GDispositionMaterialized,
	)
	wantMaterialized := make([]string, 0, len(manifest.MaterializedKnowledge))
	for _, item := range manifest.MaterializedKnowledge {
		wantMaterialized = append(wantMaterialized, item.KnowledgeKey)
	}
	sort.Strings(wantMaterialized)
	if !reflect.DeepEqual(gotMaterialized, wantMaterialized) {
		t.Fatalf(
			"R6G explanation disagrees with R6F materialization: got=%v want=%v",
			gotMaterialized,
			wantMaterialized,
		)
	}

	gotOmitted := actorContextR6GWitnessKeysByDisposition(
		explanation.Witnesses,
		actorContextR6GDispositionNotMaterialized,
	)
	if !reflect.DeepEqual(gotOmitted, manifest.QualifiedNotMaterialized) {
		t.Fatalf(
			"R6G explanation disagrees with R6F qualified-not-materialized set: got=%v want=%v",
			gotOmitted,
			manifest.QualifiedNotMaterialized,
		)
	}

	typ := reflect.TypeOf(actorContextR6GMaterializationWitnessProbe{})
	for _, forbidden := range []string{
		"Content",
		"Body",
		"Text",
		"Payload",
		"Prompt",
		"SerializedKnowledge",
		"AllFields",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf(
				"R6G omission witness restored non-materialized knowledge content through field %q",
				forbidden,
			)
		}
	}
}

func TestActorContextR6GRouterSubstitutionKeepsMaterializationExplanationStable(t *testing.T) {
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

	exactExplanation, err := actorContextR6GBuildExplanation(
		exact.Qualified,
		exactOrigins.Dependencies,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	broadExplanation, err := actorContextR6GBuildExplanation(
		broad.Qualified,
		broadOrigins.Dependencies,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(exactExplanation, broadExplanation) {
		t.Fatalf(
			"R6G router substitution changed materialization explanation: exact=%+v broad=%+v",
			exactExplanation,
			broadExplanation,
		)
	}
}

func TestActorContextR6GConsumerRuntimeOnlyExplainsAllQualifiedKnowledgeAsNotMaterialized(t *testing.T) {
	root := repoPath()
	routing := actorContextR6EBP06RoutingProbe(
		t,
		root,
		"exact-task-router",
		actorContextR6BSelectTaskHints,
	)
	runtimeOrigins := actorContextR6EDeniedRuntimeOriginSet(t, routing.Qualified)

	got, err := actorContextR6GBuildExplanation(
		routing.Qualified,
		runtimeOrigins.Dependencies,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}

	if materialized := actorContextR6GWitnessKeysByDisposition(
		got.Witnesses,
		actorContextR6GDispositionMaterialized,
	); len(materialized) != 0 {
		t.Fatalf("R6G runtime-only projection unexpectedly materialized routed knowledge: %v", materialized)
	}

	omitted := actorContextR6GWitnessKeysByDisposition(
		got.Witnesses,
		actorContextR6GDispositionNotMaterialized,
	)
	if !reflect.DeepEqual(omitted, []string{
		"business-procedure-reference|order-entry.order-handling",
		"semantic-capability|security.authorize",
	}) {
		t.Fatalf("R6G runtime-only omission explanation changed qualified audit set: %v", omitted)
	}
	if len(got.ConsumerLocalDependencies) != 1 ||
		got.ConsumerLocalDependencies[0] == "" {
		t.Fatalf("R6G runtime-only explanation lost consumer-local Observation: %+v", got)
	}
}

func TestActorContextR6GExplanationHasNoRankingFreshnessOrAuthorityShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR6GMaterializationExplanationProbe{})
	for _, forbidden := range []string{
		"RoutingMechanism",
		"PreferredRouter",
		"Priority",
		"Winner",
		"Score",
		"Confidence",
		"Fresh",
		"Stale",
		"Authorized",
		"CanExecute",
		"NextAction",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R6G explanation introduced forbidden shortcut field %q", forbidden)
		}
	}
}

func actorContextR6GBuildExplanation(
	qualified []actorContextR6BHeterogeneousQualified,
	dependencies []actorContextR6DDependencyOriginProbe,
	audience,
	purpose string,
) (actorContextR6GMaterializationExplanationProbe, error) {
	qualifiedByKey := map[string]actorContextR6BHeterogeneousQualified{}
	for _, item := range qualified {
		qualifiedByKey[actorContextR6BQualifiedKey(item)] = item
	}

	routedDependencies := map[string]map[string]bool{}
	consumerLocal := map[string]bool{}
	for _, dependency := range dependencies {
		switch dependency.OriginClass {
		case actorContextR6DOriginRoutedKnowledge:
			if dependency.KnowledgeKey == "" {
				return actorContextR6GMaterializationExplanationProbe{}, errProcedureMeaning(
					"R6G routed dependency has no knowledge key",
				)
			}
			if _, ok := qualifiedByKey[dependency.KnowledgeKey]; !ok {
				return actorContextR6GMaterializationExplanationProbe{}, errProcedureMeaning(
					"R6G routed dependency references non-qualified knowledge " + dependency.KnowledgeKey,
				)
			}
			if routedDependencies[dependency.KnowledgeKey] == nil {
				routedDependencies[dependency.KnowledgeKey] = map[string]bool{}
			}
			routedDependencies[dependency.KnowledgeKey][dependency.DependencyKey] = true
		case actorContextR6DOriginConsumerReference,
			actorContextR6DOriginConsumerRuntime,
			actorContextR6DOriginConsumerSemantic:
			if dependency.KnowledgeKey != "" {
				return actorContextR6GMaterializationExplanationProbe{}, errProcedureMeaning(
					"R6G consumer-local dependency carries routed knowledge key " + dependency.KnowledgeKey,
				)
			}
			consumerLocal[dependency.DependencyKey] = true
		default:
			return actorContextR6GMaterializationExplanationProbe{}, errProcedureMeaning(
				"R6G unsupported dependency origin class " + dependency.OriginClass,
			)
		}
	}

	witnesses := make([]actorContextR6GMaterializationWitnessProbe, 0, len(qualifiedByKey))
	for key, item := range qualifiedByKey {
		dependencySet := routedDependencies[key]
		dependencyKeys := make([]string, 0, len(dependencySet))
		for dependencyKey := range dependencySet {
			dependencyKeys = append(dependencyKeys, dependencyKey)
		}
		sort.Strings(dependencyKeys)

		witness := actorContextR6GMaterializationWitnessProbe{
			KnowledgeKey:  key,
			KnowledgeKind: item.KnowledgeKind,
			SourceRefs:    append([]string(nil), item.SourceRefs...),
		}
		if len(dependencyKeys) > 0 {
			witness.Disposition = actorContextR6GDispositionMaterialized
			witness.ReasonCode = actorContextR6GReasonConsumedRoutedDependency
			witness.DependencyKeys = dependencyKeys
		} else {
			witness.Disposition = actorContextR6GDispositionNotMaterialized
			witness.ReasonCode = actorContextR6GReasonNoRoutedDependency
		}
		witnesses = append(witnesses, witness)
	}
	sort.Slice(witnesses, func(i, j int) bool {
		return witnesses[i].KnowledgeKey < witnesses[j].KnowledgeKey
	})

	consumerLocalDependencies := make([]string, 0, len(consumerLocal))
	for dependencyKey := range consumerLocal {
		consumerLocalDependencies = append(consumerLocalDependencies, dependencyKey)
	}
	sort.Strings(consumerLocalDependencies)

	return actorContextR6GMaterializationExplanationProbe{
		Audience:                  audience,
		Purpose:                   purpose,
		Witnesses:                 witnesses,
		ConsumerLocalDependencies: consumerLocalDependencies,
		AuthorityLimits: []string{
			"not-materialized-is-an-explainable-qualified-state-not-a-missing-source",
			"omission-witness-keeps-identity-and-provenance-without-restoring-content",
			"router-substitution-does-not-change-materialization-explanation",
			"consumer-local-dependency-does-not-create-routed-materialization",
			"materialization-explanation-does-not-grant-authority-or-freshness",
		},
	}, nil
}

func actorContextR6GWitnessByKey(
	t *testing.T,
	values []actorContextR6GMaterializationWitnessProbe,
	knowledgeKey string,
) actorContextR6GMaterializationWitnessProbe {
	t.Helper()
	for _, value := range values {
		if value.KnowledgeKey == knowledgeKey {
			return value
		}
	}
	t.Fatalf("R6G witness not found for %q: %+v", knowledgeKey, values)
	return actorContextR6GMaterializationWitnessProbe{}
}

func actorContextR6GWitnessKeysByDisposition(
	values []actorContextR6GMaterializationWitnessProbe,
	disposition string,
) []string {
	out := []string{}
	for _, value := range values {
		if value.Disposition == disposition {
			out = append(out, value.KnowledgeKey)
		}
	}
	sort.Strings(out)
	return out
}

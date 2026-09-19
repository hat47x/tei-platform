package main

import (
	"reflect"
	"sort"
	"testing"
)

type actorContextR6JConsumerLocalUseProbe struct {
	OriginClass        string `json:"originClass"`
	ProjectionCategory string `json:"projectionCategory"`
	DependencyKey      string `json:"dependencyKey"`
	DependencyReason   string `json:"dependencyReason"`
}

type actorContextR6JBoundedInputAssemblyProbe struct {
	Audience                 string                                 `json:"audience"`
	Purpose                  string                                 `json:"purpose"`
	RoutedKnowledge          []actorContextR6ISharedKnowledgeProbe  `json:"routedKnowledge"`
	RoutedUseEdges           []actorContextR6IUseEdgeProbe          `json:"routedUseEdges"`
	ConsumerLocalUseEdges    []actorContextR6JConsumerLocalUseProbe `json:"consumerLocalUseEdges"`
	QualifiedNotMaterialized []string                               `json:"qualifiedNotMaterialized"`
	AuthorityLimits          []string                               `json:"authorityLimits"`
}

func TestActorContextR6JAssemblesRoutedAndConsumerLocalPlanesWithoutFlattening(t *testing.T) {
	root := repoPath()
	qualified := actorContextR6CCurrentBP06QualifiedKnowledge(t, root)
	origins, procedureKey, overlayKey := actorContextR6JBP06HumanOriginSet(t, root, qualified)

	got, err := actorContextR6JAssembleBoundedInput(qualified, origins)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.RoutedKnowledge) != 1 {
		t.Fatalf("R6J expected one deduplicated routed knowledge body: %+v", got.RoutedKnowledge)
	}
	procedure := got.RoutedKnowledge[0]
	if procedure.KnowledgeKey != "business-procedure-reference|order-entry.order-handling" ||
		procedure.KnowledgeKind != "business-procedure-reference" ||
		len(procedure.RequiredInputFingerprints) == 0 ||
		len(procedure.SourceRefs) == 0 {
		t.Fatalf("R6J routed Procedure body lost identity/revision provenance: %+v", procedure)
	}

	if len(got.RoutedUseEdges) != 3 {
		t.Fatalf("R6J routed use-edge count=%d want=3: %+v", len(got.RoutedUseEdges), got.RoutedUseEdges)
	}
	for _, edge := range got.RoutedUseEdges {
		if edge.KnowledgeKey != procedure.KnowledgeKey ||
			edge.DependencyKey != procedureKey {
			t.Fatalf("R6J routed use changed Procedure dependency identity: %+v", edge)
		}
	}

	if len(got.ConsumerLocalUseEdges) != 3 {
		t.Fatalf(
			"R6J consumer-local Reference use-edge count=%d want=3: %+v",
			len(got.ConsumerLocalUseEdges),
			got.ConsumerLocalUseEdges,
		)
	}
	for _, edge := range got.ConsumerLocalUseEdges {
		if edge.OriginClass != actorContextR6DOriginConsumerReference ||
			edge.DependencyKey != overlayKey {
			t.Fatalf("R6J consumer-local overlay was flattened into another input plane: %+v", edge)
		}
	}

	if !reflect.DeepEqual(
		got.QualifiedNotMaterialized,
		[]string{"semantic-capability|security.authorize"},
	) {
		t.Fatalf(
			"R6J assembly changed qualified-not-materialized Capability: %+v",
			got.QualifiedNotMaterialized,
		)
	}

	if actorContextR6JHasRoutedKnowledgeKey(
		got.RoutedKnowledge,
		"semantic-capability|security.authorize",
	) {
		t.Fatalf("R6J assembly silently materialized unused Capability: %+v", got.RoutedKnowledge)
	}
}

func TestActorContextR6JRuntimeOnlyAssemblyKeepsRoutedKnowledgeEmpty(t *testing.T) {
	root := repoPath()
	qualified := actorContextR6CCurrentBP06QualifiedKnowledge(t, root)
	origins := actorContextR6EDeniedRuntimeOriginSet(t, qualified)

	got, err := actorContextR6JAssembleBoundedInput(qualified, origins)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.RoutedKnowledge) != 0 || len(got.RoutedUseEdges) != 0 {
		t.Fatalf(
			"R6J consumer-local runtime Observation pulled routed knowledge into bounded input: knowledge=%+v uses=%+v",
			got.RoutedKnowledge,
			got.RoutedUseEdges,
		)
	}
	if len(got.ConsumerLocalUseEdges) != 1 ||
		got.ConsumerLocalUseEdges[0].OriginClass != actorContextR6DOriginConsumerRuntime ||
		got.ConsumerLocalUseEdges[0].DependencyReason != actorContextR3SReasonExactRuntimeObservation {
		t.Fatalf("R6J lost runtime Observation consumer-local plane: %+v", got.ConsumerLocalUseEdges)
	}
	if !reflect.DeepEqual(got.QualifiedNotMaterialized, []string{
		"business-procedure-reference|order-entry.order-handling",
		"semantic-capability|security.authorize",
	}) {
		t.Fatalf(
			"R6J runtime-only assembly lost qualified-but-unconsumed audit identities: %+v",
			got.QualifiedNotMaterialized,
		)
	}
}

func TestActorContextR6JConsumerLocalPlaneCannotCarryRoutedKnowledgeIdentity(t *testing.T) {
	typ := reflect.TypeOf(actorContextR6JConsumerLocalUseProbe{})
	for _, forbidden := range []string{
		"KnowledgeKey",
		"KnowledgeKind",
		"RequiredInputFingerprints",
		"SourceRefs",
		"Content",
		"Body",
		"Payload",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf(
				"R6J consumer-local plane acquired routed knowledge field %q",
				forbidden,
			)
		}
	}
}

func TestActorContextR6JAssemblyIsNotUniversalContextPacketOrAuthorityDecision(t *testing.T) {
	typ := reflect.TypeOf(actorContextR6JBoundedInputAssemblyProbe{})
	for _, forbidden := range []string{
		"ContextPacket",
		"Prompt",
		"SerializedPayload",
		"AllKnowledge",
		"RoutingMechanism",
		"Fresh",
		"Stale",
		"Authorized",
		"Allowed",
		"CanExecute",
		"InvocationPlan",
		"NextAction",
		"Priority",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R6J bounded input assembly introduced forbidden packet/authority shortcut %q", forbidden)
		}
	}
}

func actorContextR6JBP06HumanOriginSet(
	t *testing.T,
	root string,
	qualified []actorContextR6BHeterogeneousQualified,
) (
	actorContextR6DOriginSetProbe,
	string,
	string,
) {
	t.Helper()

	core := actorContextR3QCurrentBP06Core(t)
	view, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	deps := actorContextR3SCurrentBP06ReferenceEnvelope(t, root)
	procedure := actorContextR6CFindReferenceDependency(t, deps, "business-procedure-reference")
	overlay := actorContextR6CFindReferenceDependency(t, deps, "protected-target-overlay")
	procedureKey := actorContextR3SReferenceKey(procedure)
	overlayKey := actorContextR3SReferenceKey(overlay)

	reasons, err := actorContextR3SBuildReasonEnvelope(
		core,
		view,
		deps,
		[]actorContextR3SDependencyReasonEdgeProbe{
			{
				Category:      actorContextR3PCategoryStopIdentity,
				DependencyKey: procedureKey,
				ReasonCode:    actorContextR3SReasonBusinessAddressSource,
			},
			{
				Category:      actorContextR3PCategoryMachineReasons,
				DependencyKey: procedureKey,
				ReasonCode:    actorContextR3SReasonAuthorizationRequirementSource,
			},
			{
				Category:      actorContextR3PCategoryMachineReasons,
				DependencyKey: overlayKey,
				ReasonCode:    actorContextR3SReasonProtectedResourceSource,
			},
			{
				Category:      actorContextR3PCategoryResolutionNeeds,
				DependencyKey: overlayKey,
				ReasonCode:    actorContextR3SReasonUnresolvedClassification,
			},
			{
				Category:      actorContextR3PCategorySourceRefs,
				DependencyKey: procedureKey,
				ReasonCode:    actorContextR3SReasonDrillDownLocator,
			},
			{
				Category:      actorContextR3PCategorySourceRefs,
				DependencyKey: overlayKey,
				ReasonCode:    actorContextR3SReasonDrillDownLocator,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	origins, err := actorContextR6DBuildOriginSet(
		qualified,
		reasons,
		map[string][]string{
			"business-procedure-reference|order-entry.order-handling": {procedureKey},
		},
		map[string]string{
			overlayKey: actorContextR6DOriginConsumerReference,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	return origins, procedureKey, overlayKey
}

func actorContextR6JAssembleBoundedInput(
	qualified []actorContextR6BHeterogeneousQualified,
	origins actorContextR6DOriginSetProbe,
) (actorContextR6JBoundedInputAssemblyProbe, error) {
	qualifiedByKey := map[string]actorContextR6BHeterogeneousQualified{}
	for _, item := range qualified {
		qualifiedByKey[actorContextR6BQualifiedKey(item)] = item
	}

	routedKnowledgeKeys := map[string]bool{}
	routedUses := []actorContextR6IUseEdgeProbe{}
	consumerLocalUses := []actorContextR6JConsumerLocalUseProbe{}

	for _, dependency := range origins.Dependencies {
		switch dependency.OriginClass {
		case actorContextR6DOriginRoutedKnowledge:
			item, ok := qualifiedByKey[dependency.KnowledgeKey]
			if !ok || dependency.KnowledgeKey == "" {
				return actorContextR6JBoundedInputAssemblyProbe{}, errProcedureMeaning(
					"R6J routed use references non-qualified knowledge " + dependency.KnowledgeKey,
				)
			}
			_ = item
			if dependency.DependencyKey == "" ||
				dependency.DependencyReason == "" ||
				dependency.Category == "" {
				return actorContextR6JBoundedInputAssemblyProbe{}, errProcedureMeaning(
					"R6J routed use lost projection provenance for " + dependency.KnowledgeKey,
				)
			}
			routedKnowledgeKeys[dependency.KnowledgeKey] = true
			routedUses = append(routedUses, actorContextR6IUseEdgeProbe{
				KnowledgeKey:       dependency.KnowledgeKey,
				ProjectionCategory: dependency.Category,
				DependencyKey:      dependency.DependencyKey,
				DependencyReason:   dependency.DependencyReason,
			})

		case actorContextR6DOriginConsumerReference,
			actorContextR6DOriginConsumerRuntime,
			actorContextR6DOriginConsumerSemantic:
			if dependency.KnowledgeKey != "" {
				return actorContextR6JBoundedInputAssemblyProbe{}, errProcedureMeaning(
					"R6J consumer-local use unexpectedly carries routed knowledge " + dependency.KnowledgeKey,
				)
			}
			if dependency.DependencyKey == "" ||
				dependency.DependencyReason == "" ||
				dependency.Category == "" {
				return actorContextR6JBoundedInputAssemblyProbe{}, errProcedureMeaning(
					"R6J consumer-local use lost projection provenance",
				)
			}
			consumerLocalUses = append(consumerLocalUses, actorContextR6JConsumerLocalUseProbe{
				OriginClass:        dependency.OriginClass,
				ProjectionCategory: dependency.Category,
				DependencyKey:      dependency.DependencyKey,
				DependencyReason:   dependency.DependencyReason,
			})

		default:
			return actorContextR6JBoundedInputAssemblyProbe{}, errProcedureMeaning(
				"R6J unsupported origin class " + dependency.OriginClass,
			)
		}
	}

	routedKnowledge := make([]actorContextR6ISharedKnowledgeProbe, 0, len(routedKnowledgeKeys))
	qualifiedNotMaterialized := []string{}
	for key, item := range qualifiedByKey {
		if routedKnowledgeKeys[key] {
			routedKnowledge = append(routedKnowledge, actorContextR6ISharedKnowledgeProbe{
				KnowledgeKey:              key,
				KnowledgeKind:             item.KnowledgeKind,
				RequiredInputFingerprints: actorContextR6HCloneFingerprints(item.RequiredInputFingerprints),
				SourceRefs:                append([]string(nil), item.SourceRefs...),
			})
			continue
		}
		qualifiedNotMaterialized = append(qualifiedNotMaterialized, key)
	}

	sort.Slice(routedKnowledge, func(i, j int) bool {
		return routedKnowledge[i].KnowledgeKey < routedKnowledge[j].KnowledgeKey
	})
	sort.Slice(routedUses, func(i, j int) bool {
		left := routedUses[i].KnowledgeKey + "|" +
			routedUses[i].ProjectionCategory + "|" +
			routedUses[i].DependencyKey + "|" +
			routedUses[i].DependencyReason
		right := routedUses[j].KnowledgeKey + "|" +
			routedUses[j].ProjectionCategory + "|" +
			routedUses[j].DependencyKey + "|" +
			routedUses[j].DependencyReason
		return left < right
	})
	sort.Slice(consumerLocalUses, func(i, j int) bool {
		left := consumerLocalUses[i].OriginClass + "|" +
			consumerLocalUses[i].ProjectionCategory + "|" +
			consumerLocalUses[i].DependencyKey + "|" +
			consumerLocalUses[i].DependencyReason
		right := consumerLocalUses[j].OriginClass + "|" +
			consumerLocalUses[j].ProjectionCategory + "|" +
			consumerLocalUses[j].DependencyKey + "|" +
			consumerLocalUses[j].DependencyReason
		return left < right
	})
	sort.Strings(qualifiedNotMaterialized)

	return actorContextR6JBoundedInputAssemblyProbe{
		Audience:                 origins.Audience,
		Purpose:                  origins.Purpose,
		RoutedKnowledge:          routedKnowledge,
		RoutedUseEdges:           routedUses,
		ConsumerLocalUseEdges:    consumerLocalUses,
		QualifiedNotMaterialized: qualifiedNotMaterialized,
		AuthorityLimits: []string{
			"routed-knowledge-and-consumer-local-input-remain-separate-planes",
			"consumer-local-input-does-not-become-router-knowledge",
			"routed-use-edge-does-not-duplicate-knowledge-body",
			"qualified-not-materialized-does-not-become-missing-source",
			"bounded-input-assembly-is-derived-not-a-universal-context-packet",
			"input-plane-assembly-does-not-decide-freshness-or-execution-authority",
		},
	}, nil
}

func actorContextR6JHasRoutedKnowledgeKey(
	values []actorContextR6ISharedKnowledgeProbe,
	knowledgeKey string,
) bool {
	for _, value := range values {
		if value.KnowledgeKey == knowledgeKey {
			return true
		}
	}
	return false
}

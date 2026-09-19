package main

import (
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR6DOriginRoutedKnowledge   = "routed-qualified-knowledge"
	actorContextR6DOriginConsumerReference = "consumer-reference-input"
	actorContextR6DOriginConsumerRuntime   = "consumer-runtime-observation"
	actorContextR6DOriginConsumerSemantic  = "consumer-semantic-subject"
)

type actorContextR6DDependencyOriginProbe struct {
	Category         string `json:"category"`
	DependencyKey    string `json:"dependencyKey"`
	DependencyReason string `json:"dependencyReason"`
	OriginClass      string `json:"originClass"`
	KnowledgeKey     string `json:"knowledgeKey,omitempty"`
}

type actorContextR6DOriginSetProbe struct {
	Audience        string                                 `json:"audience"`
	Purpose         string                                 `json:"purpose"`
	Dependencies    []actorContextR6DDependencyOriginProbe `json:"dependencies"`
	AuthorityLimits []string                               `json:"authorityLimits"`
}

func TestActorContextR6DFullBP06HumanReasonsSeparateRoutedAndConsumerLocalInputs(t *testing.T) {
	root := repoPath()
	qualified := actorContextR6CCurrentBP06QualifiedKnowledge(t, root)
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

	got, err := actorContextR6DBuildOriginSet(
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

	if gotRouted := actorContextR6DCountOrigin(got.Dependencies, actorContextR6DOriginRoutedKnowledge); gotRouted != 3 {
		t.Fatalf("R6D routed dependency count=%d want=3: %+v", gotRouted, got.Dependencies)
	}
	if gotLocal := actorContextR6DCountOrigin(got.Dependencies, actorContextR6DOriginConsumerReference); gotLocal != 3 {
		t.Fatalf("R6D consumer-local Reference dependency count=%d want=3: %+v", gotLocal, got.Dependencies)
	}

	for _, dep := range got.Dependencies {
		switch dep.OriginClass {
		case actorContextR6DOriginRoutedKnowledge:
			if dep.KnowledgeKey != "business-procedure-reference|order-entry.order-handling" ||
				dep.DependencyKey != procedureKey {
				t.Fatalf("R6D routed provenance drifted: %+v", dep)
			}
		case actorContextR6DOriginConsumerReference:
			if dep.KnowledgeKey != "" || dep.DependencyKey != overlayKey {
				t.Fatalf("R6D consumer-local Reference was forced through routing: %+v", dep)
			}
		default:
			t.Fatalf("R6D unexpected origin class: %+v", dep)
		}
	}

	if actorContextR6DDependencyUsesKnowledge(
		got.Dependencies,
		"semantic-capability|security.authorize",
	) {
		t.Fatalf("qualified Capability was silently claimed as a current projection dependency: %+v", got.Dependencies)
	}
}

func TestActorContextR6DRuntimeObservationRemainsConsumerLocal(t *testing.T) {
	root := repoPath()
	qualified := actorContextR6CCurrentBP06QualifiedKnowledge(t, root)
	core := actorContextR3RDeniedCoreForReason(t)
	view, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}

	observation := actorContextR3RRuntimeObservationDependencyProbe{
		BusinessAddress: "order-entry.order-handling/BP-06",
		SubjectID:       "subject-order-viewer",
		Action:          "execute",
		Resource:        "order_entry.submit_order",
		Outcome:         "denied",
	}
	runtimeKey := actorContextR3SRuntimeKey(observation)
	deps := actorContextR3RDependencyEnvelopeProbe{
		RuntimeObservations: []actorContextR3RRuntimeObservationDependencyProbe{observation},
	}
	reasons, err := actorContextR3SBuildReasonEnvelope(
		core,
		view,
		deps,
		[]actorContextR3SDependencyReasonEdgeProbe{{
			Category:      actorContextR3PCategoryRuntimeOutcome,
			DependencyKey: runtimeKey,
			ReasonCode:    actorContextR3SReasonExactRuntimeObservation,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}

	got, err := actorContextR6DBuildOriginSet(
		qualified,
		reasons,
		nil,
		map[string]string{
			runtimeKey: actorContextR6DOriginConsumerRuntime,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Dependencies) != 1 ||
		got.Dependencies[0].OriginClass != actorContextR6DOriginConsumerRuntime ||
		got.Dependencies[0].KnowledgeKey != "" {
		t.Fatalf("runtime Observation was routed through knowledge selection: %+v", got)
	}
}

func TestActorContextR6DRejectsUnclassifiedAndAmbiguousDependencyOrigins(t *testing.T) {
	root := repoPath()
	qualified := actorContextR6CCurrentBP06QualifiedKnowledge(t, root)
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
	overlay := actorContextR6CFindReferenceDependency(t, deps, "protected-target-overlay")
	overlayKey := actorContextR3SReferenceKey(overlay)
	reasons, err := actorContextR3SBuildReasonEnvelope(
		core,
		view,
		deps,
		[]actorContextR3SDependencyReasonEdgeProbe{{
			Category:      actorContextR3PCategoryMachineReasons,
			DependencyKey: overlayKey,
			ReasonCode:    actorContextR3SReasonProtectedResourceSource,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := actorContextR6DBuildOriginSet(qualified, reasons, nil, nil); err == nil {
		t.Fatal("R6D accepted projection dependency with no routed or consumer-local origin")
	}

	if _, err := actorContextR6DBuildOriginSet(
		qualified,
		reasons,
		map[string][]string{
			"business-procedure-reference|order-entry.order-handling": {overlayKey},
		},
		map[string]string{
			overlayKey: actorContextR6DOriginConsumerReference,
		},
	); err == nil {
		t.Fatal("R6D accepted dependency claimed by both routing and consumer-local input")
	}
}

func TestActorContextR6DOriginClassificationHasNoAuthorityOrFreshnessShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR6DDependencyOriginProbe{})
	for _, forbidden := range []string{
		"Owner",
		"Authority",
		"Authorized",
		"CanExecute",
		"Fresh",
		"Stale",
		"Current",
		"Changed",
		"ChangeClass",
		"Priority",
		"Winner",
		"Rebuild",
		"Invalidate",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R6D origin classification introduced forbidden shortcut field %q", forbidden)
		}
	}
}

func actorContextR6DBuildOriginSet(
	qualified []actorContextR6BHeterogeneousQualified,
	reasons actorContextR3SReasonEnvelopeProbe,
	routedBindings map[string][]string,
	consumerLocal map[string]string,
) (actorContextR6DOriginSetProbe, error) {
	qualifiedByKey := map[string]bool{}
	for _, item := range qualified {
		qualifiedByKey[actorContextR6BQualifiedKey(item)] = true
	}

	routedByDependency := map[string][]string{}
	for knowledgeKey, dependencyKeys := range routedBindings {
		if !qualifiedByKey[knowledgeKey] {
			return actorContextR6DOriginSetProbe{}, errProcedureMeaning(
				"R6D routed dependency references non-qualified knowledge " + knowledgeKey,
			)
		}
		for _, dependencyKey := range dependencyKeys {
			routedByDependency[dependencyKey] = append(
				routedByDependency[dependencyKey],
				knowledgeKey,
			)
		}
	}

	records := make([]actorContextR6DDependencyOriginProbe, 0, len(reasons.Edges))
	for _, edge := range reasons.Edges {
		routed := routedByDependency[edge.DependencyKey]
		local, hasLocal := consumerLocal[edge.DependencyKey]
		if len(routed) > 0 && hasLocal {
			return actorContextR6DOriginSetProbe{}, errProcedureMeaning(
				"R6D dependency origin is ambiguous between routing and consumer-local input: " + edge.DependencyKey,
			)
		}
		if len(routed) == 0 && !hasLocal {
			return actorContextR6DOriginSetProbe{}, errProcedureMeaning(
				"R6D dependency origin is unclassified: " + edge.DependencyKey,
			)
		}

		if len(routed) > 0 {
			for _, knowledgeKey := range routed {
				records = append(records, actorContextR6DDependencyOriginProbe{
					Category:         edge.Category,
					DependencyKey:    edge.DependencyKey,
					DependencyReason: edge.ReasonCode,
					OriginClass:      actorContextR6DOriginRoutedKnowledge,
					KnowledgeKey:     knowledgeKey,
				})
			}
			continue
		}

		switch local {
		case actorContextR6DOriginConsumerReference,
			actorContextR6DOriginConsumerRuntime,
			actorContextR6DOriginConsumerSemantic:
		default:
			return actorContextR6DOriginSetProbe{}, errProcedureMeaning(
				"R6D unsupported consumer-local origin class " + local,
			)
		}
		records = append(records, actorContextR6DDependencyOriginProbe{
			Category:         edge.Category,
			DependencyKey:    edge.DependencyKey,
			DependencyReason: edge.ReasonCode,
			OriginClass:      local,
		})
	}

	sort.Slice(records, func(i, j int) bool {
		left := records[i].Category + "|" + records[i].DependencyKey + "|" +
			records[i].DependencyReason + "|" + records[i].OriginClass + "|" + records[i].KnowledgeKey
		right := records[j].Category + "|" + records[j].DependencyKey + "|" +
			records[j].DependencyReason + "|" + records[j].OriginClass + "|" + records[j].KnowledgeKey
		return left < right
	})

	return actorContextR6DOriginSetProbe{
		Audience:     reasons.Audience,
		Purpose:      reasons.Purpose,
		Dependencies: records,
		AuthorityLimits: []string{
			"routing-does-not-own-all-projection-inputs",
			"consumer-local-input-does-not-become-routed-knowledge",
			"origin-class-does-not-create-authority",
			"origin-class-does-not-evaluate-freshness",
			"unclassified-or-ambiguous-origin-fails-closed",
		},
	}, nil
}

func actorContextR6DCountOrigin(
	values []actorContextR6DDependencyOriginProbe,
	origin string,
) int {
	count := 0
	for _, value := range values {
		if value.OriginClass == origin {
			count++
		}
	}
	return count
}

func actorContextR6DDependencyUsesKnowledge(
	values []actorContextR6DDependencyOriginProbe,
	knowledgeKey string,
) bool {
	for _, value := range values {
		if value.KnowledgeKey == knowledgeKey {
			return true
		}
	}
	return false
}

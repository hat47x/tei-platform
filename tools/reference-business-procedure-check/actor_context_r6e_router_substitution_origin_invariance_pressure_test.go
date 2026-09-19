package main

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"tei/internal/workspace"
)

type actorContextR6ERouterOriginSnapshotProbe struct {
	RoutingMechanism string                                  `json:"routingMechanism"`
	QualifiedKeys    []string                                `json:"qualifiedKeys"`
	Dependencies     []actorContextR6DDependencyOriginProbe  `json:"dependencies"`
	AuthorityLimits  []string                                `json:"authorityLimits"`
}

func TestActorContextR6ERouterSubstitutionPreservesProjectionDependencyOrigins(t *testing.T) {
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

	if len(broad.Selected) <= len(exact.Selected) {
		t.Fatalf(
			"R6E broad router did not over-select before deterministic qualification: exact=%d broad=%d",
			len(exact.Selected),
			len(broad.Selected),
		)
	}
	if len(broad.Rejected) == 0 {
		t.Fatal("R6E broad router did not expose rejected over-selection")
	}
	if !reflect.DeepEqual(
		actorContextR6BQualifiedKeys(exact.Qualified),
		actorContextR6BQualifiedKeys(broad.Qualified),
	) {
		t.Fatalf(
			"R6E router substitution changed qualified BP-06 knowledge: exact=%v broad=%v",
			actorContextR6BQualifiedKeys(exact.Qualified),
			actorContextR6BQualifiedKeys(broad.Qualified),
		)
	}

	exactSnapshot := actorContextR6EBP06HumanOriginSnapshot(t, root, exact)
	broadSnapshot := actorContextR6EBP06HumanOriginSnapshot(t, root, broad)

	if !reflect.DeepEqual(exactSnapshot.QualifiedKeys, broadSnapshot.QualifiedKeys) {
		t.Fatalf(
			"R6E downstream snapshot changed qualified identity by router: exact=%v broad=%v",
			exactSnapshot.QualifiedKeys,
			broadSnapshot.QualifiedKeys,
		)
	}
	if !reflect.DeepEqual(exactSnapshot.Dependencies, broadSnapshot.Dependencies) {
		t.Fatalf(
			"R6E router substitution changed R3 dependency origin/provenance: exact=%+v broad=%+v",
			exactSnapshot.Dependencies,
			broadSnapshot.Dependencies,
		)
	}

	for _, snapshot := range []actorContextR6ERouterOriginSnapshotProbe{
		exactSnapshot,
		broadSnapshot,
	} {
		if got := actorContextR6DCountOrigin(
			snapshot.Dependencies,
			actorContextR6DOriginRoutedKnowledge,
		); got != 3 {
			t.Fatalf(
				"R6E routed dependency count changed under %s: got=%d dependencies=%+v",
				snapshot.RoutingMechanism,
				got,
				snapshot.Dependencies,
			)
		}
		if got := actorContextR6DCountOrigin(
			snapshot.Dependencies,
			actorContextR6DOriginConsumerReference,
		); got != 3 {
			t.Fatalf(
				"R6E consumer-local Reference count changed under %s: got=%d dependencies=%+v",
				snapshot.RoutingMechanism,
				got,
				snapshot.Dependencies,
			)
		}
		if actorContextR6DDependencyUsesKnowledge(
			snapshot.Dependencies,
			"semantic-capability|security.authorize",
		) {
			t.Fatalf(
				"R6E router %s promoted qualified Capability into projection dependency without an R3 reason binding: %+v",
				snapshot.RoutingMechanism,
				snapshot.Dependencies,
			)
		}
	}
}

func TestActorContextR6ERouterSubstitutionCannotCaptureConsumerRuntimeObservation(t *testing.T) {
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

	exactOrigins := actorContextR6EDeniedRuntimeOriginSet(t, exact.Qualified)
	broadOrigins := actorContextR6EDeniedRuntimeOriginSet(t, broad.Qualified)

	if !reflect.DeepEqual(exactOrigins.Dependencies, broadOrigins.Dependencies) {
		t.Fatalf(
			"R6E router substitution changed consumer-local runtime provenance: exact=%+v broad=%+v",
			exactOrigins.Dependencies,
			broadOrigins.Dependencies,
		)
	}
	if len(exactOrigins.Dependencies) != 1 ||
		exactOrigins.Dependencies[0].OriginClass != actorContextR6DOriginConsumerRuntime ||
		exactOrigins.Dependencies[0].KnowledgeKey != "" {
		t.Fatalf(
			"R6E exact router captured runtime Observation as routed knowledge: %+v",
			exactOrigins.Dependencies,
		)
	}
	if len(broadOrigins.Dependencies) != 1 ||
		broadOrigins.Dependencies[0].OriginClass != actorContextR6DOriginConsumerRuntime ||
		broadOrigins.Dependencies[0].KnowledgeKey != "" {
		t.Fatalf(
			"R6E broad router captured runtime Observation as routed knowledge: %+v",
			broadOrigins.Dependencies,
		)
	}
}

func TestActorContextR6ERouterMechanismIsDiagnosticNotDependencyIdentity(t *testing.T) {
	typ := reflect.TypeOf(actorContextR6DDependencyOriginProbe{})
	if _, found := typ.FieldByName("RoutingMechanism"); found {
		t.Fatal("R6E R3 dependency origin identity was coupled to routing mechanism")
	}

	snapshotType := reflect.TypeOf(actorContextR6ERouterOriginSnapshotProbe{})
	for _, forbidden := range []string{
		"PreferredRouter",
		"Winner",
		"Score",
		"Confidence",
		"RouterAuthority",
		"Fresh",
		"Authorized",
	} {
		if _, found := snapshotType.FieldByName(forbidden); found {
			t.Fatalf("R6E snapshot introduced forbidden router/authority shortcut %q", forbidden)
		}
	}
}

func actorContextR6EBP06RoutingProbe(
	t *testing.T,
	root,
	mechanism string,
	selector func([]actorContextR6BHeterogeneousCandidate, string) []actorContextR6BHeterogeneousCandidate,
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

func actorContextR6EBP06HumanOriginSnapshot(
	t *testing.T,
	root string,
	routing actorContextR6BHeterogeneousProbe,
) actorContextR6ERouterOriginSnapshotProbe {
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
	procedure := actorContextR6CFindReferenceDependency(
		t,
		deps,
		"business-procedure-reference",
	)
	overlay := actorContextR6CFindReferenceDependency(
		t,
		deps,
		"protected-target-overlay",
	)
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
		routing.Qualified,
		reasons,
		map[string][]string{
			"business-procedure-reference|order-entry.order-handling": {
				procedureKey,
			},
		},
		map[string]string{
			overlayKey: actorContextR6DOriginConsumerReference,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	dependencies := append(
		[]actorContextR6DDependencyOriginProbe(nil),
		origins.Dependencies...,
	)
	sort.Slice(dependencies, func(i, j int) bool {
		left := dependencies[i].Category + "|" +
			dependencies[i].DependencyKey + "|" +
			dependencies[i].DependencyReason + "|" +
			dependencies[i].OriginClass + "|" +
			dependencies[i].KnowledgeKey
		right := dependencies[j].Category + "|" +
			dependencies[j].DependencyKey + "|" +
			dependencies[j].DependencyReason + "|" +
			dependencies[j].OriginClass + "|" +
			dependencies[j].KnowledgeKey
		return left < right
	})

	return actorContextR6ERouterOriginSnapshotProbe{
		RoutingMechanism: routing.RoutingMechanism,
		QualifiedKeys:    actorContextR6BQualifiedKeys(routing.Qualified),
		Dependencies:     dependencies,
		AuthorityLimits: []string{
			"router-choice-is-not-dependency-identity",
			"router-overselection-is-removed-before-projection-handoff",
			"consumer-local-inputs-remain-router-independent",
			"qualified-capability-does-not-imply-projection-dependency",
		},
	}
}

func actorContextR6EDeniedRuntimeOriginSet(
	t *testing.T,
	qualified []actorContextR6BHeterogeneousQualified,
) actorContextR6DOriginSetProbe {
	t.Helper()

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
		RuntimeObservations: []actorContextR3RRuntimeObservationDependencyProbe{
			observation,
		},
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

	origins, err := actorContextR6DBuildOriginSet(
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
	return origins
}

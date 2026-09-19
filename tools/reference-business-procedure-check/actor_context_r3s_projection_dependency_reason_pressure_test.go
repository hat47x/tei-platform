package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

const (
	actorContextR3SReasonBusinessAddressSource          = "business-address-source"
	actorContextR3SReasonAuthorizationRequirementSource = "authorization-requirement-source"
	actorContextR3SReasonProtectedResourceSource        = "protected-resource-source"
	actorContextR3SReasonUnresolvedClassification       = "unresolved-authority-classification"
	actorContextR3SReasonDrillDownLocator               = "drill-down-locator"
	actorContextR3SReasonSourceIndexDefinitionLocator   = "source-index-definition-locator"
	actorContextR3SReasonExactRuntimeObservation        = "exact-runtime-observation"
)

type actorContextR3SDependencyReasonEdgeProbe struct {
	Category      string
	DependencyKey string
	ReasonCode    string
}

type actorContextR3SReasonEnvelopeProbe struct {
	Audience string
	Purpose  string
	Edges    []actorContextR3SDependencyReasonEdgeProbe
}

func TestActorContextR3SCurrentBP06HumanViewKeepsSelectedDependencyReasons(t *testing.T) {
	root := repoPath()
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

	edges := []actorContextR3SDependencyReasonEdgeProbe{
		{
			Category:      actorContextR3PCategoryStopIdentity,
			DependencyKey: actorContextR3SReferenceKey(deps.ReferenceArtifacts[0]),
			ReasonCode:    actorContextR3SReasonBusinessAddressSource,
		},
		{
			Category:      actorContextR3PCategoryMachineReasons,
			DependencyKey: actorContextR3SReferenceKey(deps.ReferenceArtifacts[0]),
			ReasonCode:    actorContextR3SReasonAuthorizationRequirementSource,
		},
		{
			Category:      actorContextR3PCategoryMachineReasons,
			DependencyKey: actorContextR3SReferenceKey(deps.ReferenceArtifacts[1]),
			ReasonCode:    actorContextR3SReasonProtectedResourceSource,
		},
		{
			Category:      actorContextR3PCategoryResolutionNeeds,
			DependencyKey: actorContextR3SReferenceKey(deps.ReferenceArtifacts[1]),
			ReasonCode:    actorContextR3SReasonUnresolvedClassification,
		},
		{
			Category:      actorContextR3PCategorySourceRefs,
			DependencyKey: actorContextR3SReferenceKey(deps.ReferenceArtifacts[0]),
			ReasonCode:    actorContextR3SReasonDrillDownLocator,
		},
		{
			Category:      actorContextR3PCategorySourceRefs,
			DependencyKey: actorContextR3SReferenceKey(deps.ReferenceArtifacts[1]),
			ReasonCode:    actorContextR3SReasonDrillDownLocator,
		},
	}

	got, err := actorContextR3SBuildReasonEnvelope(core, view, deps, edges)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Edges) != len(edges) {
		t.Fatalf("R3S lost selected Human dependency reasons: %+v", got)
	}
	for _, edge := range got.Edges {
		if actorContextR0Contains(view.OmittedCategories, edge.Category) {
			t.Fatalf("R3S emitted dependency reason for omitted Human category: %+v", edge)
		}
	}
}

func TestActorContextR3SOmittedKnownFactReasonIsRejectedForHumanButAcceptedForAgent(t *testing.T) {
	root := repoPath()
	core := actorContextR3QCurrentBP06Core(t)
	deps := actorContextR3SCurrentBP06ReferenceEnvelope(t, root)
	edge := actorContextR3SDependencyReasonEdgeProbe{
		Category:      actorContextR3PCategoryKnownAuthorityFacts,
		DependencyKey: actorContextR3SReferenceKey(deps.ReferenceArtifacts[0]),
		ReasonCode:    actorContextR3SReasonAuthorizationRequirementSource,
	}

	human, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := actorContextR3SBuildReasonEnvelope(core, human, deps, []actorContextR3SDependencyReasonEdgeProbe{edge}); err == nil {
		t.Fatal("R3S accepted dependency reason for omitted Human category")
	}

	agent, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceAgent,
		actorContextR3PPurposeResolutionInspection,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := actorContextR3SBuildReasonEnvelope(core, agent, deps, []actorContextR3SDependencyReasonEdgeProbe{edge}); err != nil {
		t.Fatalf("R3S rejected selected Agent known-fact reason edge: %v", err)
	}
}

func TestActorContextR3SSourceIndexSubjectCanExplainSelectedDrillDownWithoutBecomingAuthority(t *testing.T) {
	root := repoPath()
	workspacePath := filepath.Join(root, "reference", "workspace")
	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := workspace.BuildSourceIndex(workspacePath, loaded)
	if err != nil {
		t.Fatal(err)
	}
	subject, err := actorContextR3RSemanticDependencyFromSourceIndex(
		idx,
		sourceindex.Subject{Kind: "application-pattern", ID: "order_entry.submit_order"},
	)
	if err != nil {
		t.Fatal(err)
	}

	core := actorContextR3QCurrentBP06Core(t)
	core.SourceRefs = actorContextR0SortedUnique(append(
		core.SourceRefs,
		"reference/workspace/"+subject.Locator,
	))
	view, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	deps := actorContextR3RDependencyEnvelopeProbe{
		ProfileFingerprint: actorContextR3QSnapshotForProfile(
			t, core, actorContextR3PAudienceHuman, actorContextR3PPurposeOrientation,
		).ProfileFingerprint,
		SemanticSubjects: []actorContextR3RSemanticSubjectDependencyProbe{subject},
	}
	edge := actorContextR3SDependencyReasonEdgeProbe{
		Category:      actorContextR3PCategorySourceRefs,
		DependencyKey: actorContextR3SSemanticKey(subject),
		ReasonCode:    actorContextR3SReasonSourceIndexDefinitionLocator,
	}
	got, err := actorContextR3SBuildReasonEnvelope(core, view, deps, []actorContextR3SDependencyReasonEdgeProbe{edge})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Edges) != 1 {
		t.Fatalf("R3S lost Source Index drill-down reason: %+v", got)
	}

	typ := reflect.TypeOf(edge)
	for _, forbidden := range []string{"Authority", "Authorized", "PreferredSource", "Confidence"} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3S reason edge introduced authority shortcut field %q", forbidden)
		}
	}
}

func TestActorContextR3SRuntimeOutcomeRequiresExactObservationReason(t *testing.T) {
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
	deps := actorContextR3RDependencyEnvelopeProbe{
		RuntimeObservations: []actorContextR3RRuntimeObservationDependencyProbe{observation},
	}
	edge := actorContextR3SDependencyReasonEdgeProbe{
		Category:      actorContextR3PCategoryRuntimeOutcome,
		DependencyKey: actorContextR3SRuntimeKey(observation),
		ReasonCode:    actorContextR3SReasonExactRuntimeObservation,
	}
	if _, err := actorContextR3SBuildReasonEnvelope(core, view, deps, []actorContextR3SDependencyReasonEdgeProbe{edge}); err != nil {
		t.Fatalf("R3S rejected exact runtime Observation reason: %v", err)
	}
	if _, err := actorContextR3SBuildReasonEnvelope(core, view, deps, nil); err == nil {
		t.Fatal("R3S accepted non-empty runtime outcome without Observation reason edge")
	}
}

func TestActorContextR3SRejectsOrphanDependencyKey(t *testing.T) {
	core := actorContextR3QCurrentBP06Core(t)
	view, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := actorContextR3SBuildReasonEnvelope(
		core,
		view,
		actorContextR3RDependencyEnvelopeProbe{},
		[]actorContextR3SDependencyReasonEdgeProbe{{
			Category:      actorContextR3PCategorySourceRefs,
			DependencyKey: "reference:missing",
			ReasonCode:    actorContextR3SReasonDrillDownLocator,
		}},
	); err == nil {
		t.Fatal("R3S accepted orphan dependency reason edge")
	}
}

func TestActorContextR3SReasonEnvelopeHasNoRoutingOrAuthorityShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3SReasonEnvelopeProbe{})
	for _, forbidden := range []string{
		"Owner",
		"HandoffOwner",
		"SuggestedAction",
		"NextAction",
		"PreferredSource",
		"Confidence",
		"Authorized",
		"CanExecute",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3S reason envelope introduced forbidden shortcut field %q", forbidden)
		}
	}
}

func actorContextR3SCurrentBP06ReferenceEnvelope(
	t *testing.T,
	root string,
) actorContextR3RDependencyEnvelopeProbe {
	t.Helper()
	inputs := []struct {
		class string
		path  string
	}{
		{"business-procedure-reference", "reference/business-system-design/order-entry-business-procedure.json"},
		{"protected-target-overlay", "reference/business-system-design/order-entry-protected-target-overlay.json"},
	}
	deps := make([]actorContextR3RReferenceArtifactDependencyProbe, 0, len(inputs))
	for _, input := range inputs {
		raw, err := os.ReadFile(filepath.Join(root, input.path))
		if err != nil {
			t.Fatal(err)
		}
		deps = append(deps, actorContextR3RReferenceArtifactDependencyProbe{
			InputClass:         input.class,
			ContentFingerprint: sourceindex.Fingerprint(raw),
			Locator:            input.path,
		})
	}
	sort.Slice(deps, func(i, j int) bool { return deps[i].InputClass < deps[j].InputClass })
	return actorContextR3RDependencyEnvelopeProbe{ReferenceArtifacts: deps}
}

func actorContextR3SBuildReasonEnvelope(
	core actorContextR3OStopProjectionCoreProbe,
	view actorContextR3PBoundedStopViewProbe,
	dependencies actorContextR3RDependencyEnvelopeProbe,
	edges []actorContextR3SDependencyReasonEdgeProbe,
) (actorContextR3SReasonEnvelopeProbe, error) {
	if err := actorContextR3PValidateCategoryPartition(view.SelectedCategories, view.OmittedCategories); err != nil {
		return actorContextR3SReasonEnvelopeProbe{}, err
	}
	selected := map[string]bool{}
	for _, category := range view.SelectedCategories {
		selected[category] = true
	}

	validKeys := map[string]bool{}
	for _, dep := range dependencies.SemanticSubjects {
		validKeys[actorContextR3SSemanticKey(dep)] = true
	}
	for _, dep := range dependencies.ReferenceArtifacts {
		validKeys[actorContextR3SReferenceKey(dep)] = true
	}
	for _, dep := range dependencies.RuntimeObservations {
		validKeys[actorContextR3SRuntimeKey(dep)] = true
	}

	out := actorContextR3SReasonEnvelopeProbe{
		Audience: view.Audience,
		Purpose:  view.Purpose,
	}
	runtimeReasonPresent := false
	for _, edge := range edges {
		if !selected[edge.Category] {
			return actorContextR3SReasonEnvelopeProbe{}, fmt.Errorf("R3S reason edge targets omitted category %q", edge.Category)
		}
		if edge.ReasonCode == "" {
			return actorContextR3SReasonEnvelopeProbe{}, fmt.Errorf("R3S reason edge requires reason code")
		}
		if !validKeys[edge.DependencyKey] {
			return actorContextR3SReasonEnvelopeProbe{}, fmt.Errorf("R3S reason edge references unknown dependency %q", edge.DependencyKey)
		}
		if edge.Category == actorContextR3PCategoryRuntimeOutcome &&
			edge.ReasonCode == actorContextR3SReasonExactRuntimeObservation {
			runtimeReasonPresent = true
		}
		out.Edges = append(out.Edges, edge)
	}

	if selected[actorContextR3PCategoryRuntimeOutcome] &&
		core.RuntimeOutcome != "" &&
		!runtimeReasonPresent {
		return actorContextR3SReasonEnvelopeProbe{}, fmt.Errorf("R3S non-empty runtime outcome requires exact Observation reason")
	}

	sort.Slice(out.Edges, func(i, j int) bool {
		if out.Edges[i].Category != out.Edges[j].Category {
			return out.Edges[i].Category < out.Edges[j].Category
		}
		if out.Edges[i].DependencyKey != out.Edges[j].DependencyKey {
			return out.Edges[i].DependencyKey < out.Edges[j].DependencyKey
		}
		return out.Edges[i].ReasonCode < out.Edges[j].ReasonCode
	})
	return out, nil
}

func actorContextR3SSemanticKey(dep actorContextR3RSemanticSubjectDependencyProbe) string {
	return "semantic:" + dep.Subject.Kind + ":" + dep.Subject.ID
}

func actorContextR3SReferenceKey(dep actorContextR3RReferenceArtifactDependencyProbe) string {
	return "reference:" + dep.InputClass + ":" + dep.Locator
}

func actorContextR3SRuntimeKey(dep actorContextR3RRuntimeObservationDependencyProbe) string {
	return "runtime:" + dep.BusinessAddress + ":" + dep.SubjectID + ":" + dep.Action + ":" + dep.Resource + ":" + dep.Outcome
}

func actorContextR3RDeniedCoreForReason(t *testing.T) actorContextR3OStopProjectionCoreProbe {
	t.Helper()
	explanation := actorContextR3MBuildStopExplanation(
		"order-entry.order-handling/BP-06",
		"authorization-evaluation",
		nil,
		[]actorContextR3MAuthorityFactProbe{
			{
				Kind:      "authorization-requirement",
				Value:     actorContextR3BAuthorizationRequired,
				SourceRef: "business-procedure-security-boundary",
			},
			{
				Kind:      "protected-resource",
				Value:     "order_entry.submit_order",
				SourceRef: "reference-pressure-overlay:BP-06",
			},
		},
		[]string{"runtime-observation:test-only-current-authorization"},
	)
	taxonomy := actorContextR3NClassifyStop(actorContextR3NStopInputProbe{
		BusinessAddress:    "order-entry.order-handling/BP-06",
		EvaluationStatus:   actorContextR3IEvaluationCurrentObserved,
		ObservationOutcome: "denied",
	})
	core, err := actorContextR3OBuildCommonStopCore(explanation, taxonomy)
	if err != nil {
		t.Fatal(err)
	}
	return core
}

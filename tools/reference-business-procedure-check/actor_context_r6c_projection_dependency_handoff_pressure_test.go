package main

import (
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"tei/internal/workspace"
)

const (
	actorContextR6CDispositionConsumedByProjection = "consumed-by-projection"
	actorContextR6CDispositionQualifiedNotSelected = "qualified-not-selected-by-consumer"
)

type actorContextR6CProjectionHandoffProbe struct {
	KnowledgeKey       string `json:"knowledgeKey"`
	KnowledgeKind      string `json:"knowledgeKind"`
	Audience           string `json:"audience"`
	Purpose            string `json:"purpose"`
	Disposition        string `json:"disposition"`
	ProjectionCategory string `json:"projectionCategory,omitempty"`
	DependencyKey      string `json:"dependencyKey,omitempty"`
	DependencyReason   string `json:"dependencyReason,omitempty"`
}

type actorContextR6CHandoffSetProbe struct {
	Audience        string                                  `json:"audience"`
	Purpose         string                                  `json:"purpose"`
	Records         []actorContextR6CProjectionHandoffProbe `json:"records"`
	AuthorityLimits []string                                `json:"authorityLimits"`
}

func TestActorContextR6CQualifiedKnowledgeDoesNotAutomaticallyBecomeProjectionDependency(t *testing.T) {
	root := repoPath()
	qualified := actorContextR6CCurrentBP06QualifiedKnowledge(t, root)
	if got := actorContextR6BQualifiedKeys(qualified); !reflect.DeepEqual(got, []string{
		"business-procedure-reference|order-entry.order-handling",
		"semantic-capability|security.authorize",
	}) {
		t.Fatalf("R6C test setup lost current BP-06 qualified knowledge: %v", got)
	}

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
	procedureDep := actorContextR6CFindReferenceDependency(t, deps, "business-procedure-reference")
	procedureKey := actorContextR3SReferenceKey(procedureDep)

	reasons, err := actorContextR3SBuildReasonEnvelope(
		core,
		view,
		deps,
		[]actorContextR3SDependencyReasonEdgeProbe{
			{
				Category:      actorContextR3PCategoryMachineReasons,
				DependencyKey: procedureKey,
				ReasonCode:    actorContextR3SReasonAuthorizationRequirementSource,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	got, err := actorContextR6CBuildHandoff(
		qualified,
		reasons,
		map[string][]string{
			"business-procedure-reference|order-entry.order-handling": {procedureKey},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	procedureRecords := actorContextR6CRecordsForKnowledge(
		got.Records,
		"business-procedure-reference|order-entry.order-handling",
	)
	if len(procedureRecords) != 1 ||
		procedureRecords[0].Disposition != actorContextR6CDispositionConsumedByProjection ||
		procedureRecords[0].ProjectionCategory != actorContextR3PCategoryMachineReasons ||
		procedureRecords[0].DependencyKey != procedureKey ||
		procedureRecords[0].DependencyReason != actorContextR3SReasonAuthorizationRequirementSource {
		t.Fatalf("R6C did not preserve the existing R3S dependency reason: %+v", procedureRecords)
	}

	capabilityRecords := actorContextR6CRecordsForKnowledge(
		got.Records,
		"semantic-capability|security.authorize",
	)
	if len(capabilityRecords) != 1 ||
		capabilityRecords[0].Disposition != actorContextR6CDispositionQualifiedNotSelected {
		t.Fatalf("qualified Capability was silently promoted to current projection dependency: %+v", capabilityRecords)
	}
	if capabilityRecords[0].ProjectionCategory != "" ||
		capabilityRecords[0].DependencyKey != "" ||
		capabilityRecords[0].DependencyReason != "" {
		t.Fatalf("unselected qualified Capability acquired invented projection provenance: %+v", capabilityRecords)
	}

	if actorContextR6CHasDependencyKeyPrefix(got.Records, "semantic:semantic-capability:security.authorize") {
		t.Fatalf("R6C invented an R3 semantic dependency just because Capability knowledge was qualified: %+v", got.Records)
	}
}

func TestActorContextR6CSameQualifiedKnowledgeCanFeedAudienceSpecificCategoriesWithoutChangingDependencyIdentity(t *testing.T) {
	root := repoPath()
	qualified := actorContextR6CCurrentBP06QualifiedKnowledge(t, root)
	core := actorContextR3QCurrentBP06Core(t)
	deps := actorContextR3SCurrentBP06ReferenceEnvelope(t, root)
	procedureDep := actorContextR6CFindReferenceDependency(t, deps, "business-procedure-reference")
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
		[]actorContextR3SDependencyReasonEdgeProbe{
			{
				Category:      actorContextR3PCategoryMachineReasons,
				DependencyKey: procedureKey,
				ReasonCode:    actorContextR3SReasonAuthorizationRequirementSource,
			},
		},
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

	humanConsumed := actorContextR6CConsumedRecordsForKnowledge(
		human.Records,
		"business-procedure-reference|order-entry.order-handling",
	)
	agentConsumed := actorContextR6CConsumedRecordsForKnowledge(
		agent.Records,
		"business-procedure-reference|order-entry.order-handling",
	)
	if len(humanConsumed) != 1 || len(agentConsumed) != 2 {
		t.Fatalf("R6C lost bounded audience-specific consumption: human=%+v agent=%+v", humanConsumed, agentConsumed)
	}

	if humanConsumed[0].DependencyKey != procedureKey ||
		humanConsumed[0].DependencyReason != actorContextR3SReasonAuthorizationRequirementSource {
		t.Fatalf("Human handoff changed R3 dependency provenance: %+v", humanConsumed[0])
	}
	for _, record := range agentConsumed {
		if record.DependencyKey != procedureKey ||
			record.DependencyReason != actorContextR3SReasonAuthorizationRequirementSource {
			t.Fatalf("Agent handoff changed shared dependency provenance: %+v", record)
		}
	}

	if actorContextR6CContainsCategory(humanConsumed, actorContextR3PCategoryKnownAuthorityFacts) {
		t.Fatalf("R6C forced Agent-only known-authority-facts into Human disclosure: %+v", humanConsumed)
	}
	if !actorContextR6CContainsCategory(agentConsumed, actorContextR3PCategoryKnownAuthorityFacts) {
		t.Fatalf("R6C lost selected Agent known-authority-facts category: %+v", agentConsumed)
	}

	humanCapability := actorContextR6CRecordsForKnowledge(
		human.Records,
		"semantic-capability|security.authorize",
	)
	agentCapability := actorContextR6CRecordsForKnowledge(
		agent.Records,
		"semantic-capability|security.authorize",
	)
	if len(humanCapability) != 1 || len(agentCapability) != 1 ||
		humanCapability[0].Disposition != actorContextR6CDispositionQualifiedNotSelected ||
		agentCapability[0].Disposition != actorContextR6CDispositionQualifiedNotSelected {
		t.Fatalf("audience expansion silently promoted qualified Capability to projection authority: human=%+v agent=%+v", humanCapability, agentCapability)
	}
}

func TestActorContextR6CHandoffRejectsReasonDependencyNotBoundToQualifiedKnowledge(t *testing.T) {
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
		[]actorContextR3SDependencyReasonEdgeProbe{
			{
				Category:      actorContextR3PCategoryMachineReasons,
				DependencyKey: overlayKey,
				ReasonCode:    actorContextR3SReasonProtectedResourceSource,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := actorContextR6CBuildHandoff(
		qualified,
		reasons,
		map[string][]string{
			"business-procedure-reference|order-entry.order-handling": {
				actorContextR3SReferenceKey(
					actorContextR6CFindReferenceDependency(t, deps, "business-procedure-reference"),
				),
			},
		},
	); err == nil {
		t.Fatal("R6C accepted a projection dependency reason that no qualified routed knowledge binding explains")
	}
}

func TestActorContextR6CHandoffHasNoFreshnessOrAuthorityShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR6CProjectionHandoffProbe{})
	for _, forbidden := range []string{
		"Fresh",
		"Stale",
		"Current",
		"Changed",
		"ChangeClass",
		"Authorized",
		"CanExecute",
		"Executable",
		"Authority",
		"Score",
		"Confidence",
		"Winner",
		"Rebuild",
		"Invalidate",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R6C handoff duplicated R3 freshness/authority semantics through field %q", forbidden)
		}
	}
}

func actorContextR6CCurrentBP06QualifiedKnowledge(
	t *testing.T,
	root string,
) []actorContextR6BHeterogeneousQualified {
	t.Helper()

	workspacePath := filepath.Join(root, "reference", "workspace")
	procedurePath := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	reviewEvidencePath := filepath.Join(root, "reference", "business-system-design", "order-entry-customer-edit-conflict.json")

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
		"exact-task-router",
		available,
		actorContextR6BSelectTaskHints,
	)
	if err != nil {
		t.Fatal(err)
	}
	return probe.Qualified
}

func actorContextR6CBuildHandoff(
	qualified []actorContextR6BHeterogeneousQualified,
	reasons actorContextR3SReasonEnvelopeProbe,
	knowledgeDependencyKeys map[string][]string,
) (actorContextR6CHandoffSetProbe, error) {
	qualifiedByKey := map[string]actorContextR6BHeterogeneousQualified{}
	for _, item := range qualified {
		key := actorContextR6BQualifiedKey(item)
		qualifiedByKey[key] = item
	}

	dependencyOwners := map[string][]string{}
	for knowledgeKey, dependencyKeys := range knowledgeDependencyKeys {
		if _, ok := qualifiedByKey[knowledgeKey]; !ok {
			return actorContextR6CHandoffSetProbe{}, errProcedureMeaning(
				"R6C dependency binding references non-qualified knowledge " + knowledgeKey,
			)
		}
		for _, dependencyKey := range dependencyKeys {
			dependencyOwners[dependencyKey] = append(dependencyOwners[dependencyKey], knowledgeKey)
		}
	}

	consumed := map[string]bool{}
	records := []actorContextR6CProjectionHandoffProbe{}
	for _, edge := range reasons.Edges {
		owners := dependencyOwners[edge.DependencyKey]
		if len(owners) == 0 {
			return actorContextR6CHandoffSetProbe{}, errProcedureMeaning(
				"R6C projection dependency has no qualified knowledge binding: " + edge.DependencyKey,
			)
		}
		for _, knowledgeKey := range owners {
			item := qualifiedByKey[knowledgeKey]
			records = append(records, actorContextR6CProjectionHandoffProbe{
				KnowledgeKey:       knowledgeKey,
				KnowledgeKind:      item.KnowledgeKind,
				Audience:           reasons.Audience,
				Purpose:            reasons.Purpose,
				Disposition:        actorContextR6CDispositionConsumedByProjection,
				ProjectionCategory: edge.Category,
				DependencyKey:      edge.DependencyKey,
				DependencyReason:   edge.ReasonCode,
			})
			consumed[knowledgeKey] = true
		}
	}

	for _, item := range qualified {
		key := actorContextR6BQualifiedKey(item)
		if consumed[key] {
			continue
		}
		records = append(records, actorContextR6CProjectionHandoffProbe{
			KnowledgeKey:  key,
			KnowledgeKind: item.KnowledgeKind,
			Audience:      reasons.Audience,
			Purpose:       reasons.Purpose,
			Disposition:   actorContextR6CDispositionQualifiedNotSelected,
		})
	}

	sort.Slice(records, func(i, j int) bool {
		left := records[i].KnowledgeKey + "|" + records[i].ProjectionCategory + "|" +
			records[i].DependencyKey + "|" + records[i].DependencyReason
		right := records[j].KnowledgeKey + "|" + records[j].ProjectionCategory + "|" +
			records[j].DependencyKey + "|" + records[j].DependencyReason
		return left < right
	})

	return actorContextR6CHandoffSetProbe{
		Audience: reasons.Audience,
		Purpose:  reasons.Purpose,
		Records:  records,
		AuthorityLimits: []string{
			"qualified-knowledge-is-not-automatically-a-projection-dependency",
			"projection-dependency-provenance-remains-owned-by-r3",
			"consumer-specific-category-does-not-change-dependency-identity",
			"unselected-qualified-knowledge-does-not-gain-invented-reason-edges",
			"handoff-does-not-evaluate-freshness-or-authorization",
		},
	}, nil
}

func actorContextR6CFindReferenceDependency(
	t *testing.T,
	deps actorContextR3RDependencyEnvelopeProbe,
	inputClass string,
) actorContextR3RReferenceArtifactDependencyProbe {
	t.Helper()
	for _, dep := range deps.ReferenceArtifacts {
		if dep.InputClass == inputClass {
			return dep
		}
	}
	t.Fatalf("R6C test setup lost Reference dependency class %q: %+v", inputClass, deps.ReferenceArtifacts)
	return actorContextR3RReferenceArtifactDependencyProbe{}
}

func actorContextR6CRecordsForKnowledge(
	values []actorContextR6CProjectionHandoffProbe,
	knowledgeKey string,
) []actorContextR6CProjectionHandoffProbe {
	out := []actorContextR6CProjectionHandoffProbe{}
	for _, value := range values {
		if value.KnowledgeKey == knowledgeKey {
			out = append(out, value)
		}
	}
	return out
}

func actorContextR6CConsumedRecordsForKnowledge(
	values []actorContextR6CProjectionHandoffProbe,
	knowledgeKey string,
) []actorContextR6CProjectionHandoffProbe {
	out := []actorContextR6CProjectionHandoffProbe{}
	for _, value := range values {
		if value.KnowledgeKey == knowledgeKey &&
			value.Disposition == actorContextR6CDispositionConsumedByProjection {
			out = append(out, value)
		}
	}
	return out
}

func actorContextR6CContainsCategory(
	values []actorContextR6CProjectionHandoffProbe,
	category string,
) bool {
	for _, value := range values {
		if value.ProjectionCategory == category {
			return true
		}
	}
	return false
}

func actorContextR6CHasDependencyKeyPrefix(
	values []actorContextR6CProjectionHandoffProbe,
	prefix string,
) bool {
	for _, value := range values {
		if strings.HasPrefix(value.DependencyKey, prefix) {
			return true
		}
	}
	return false
}

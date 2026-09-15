package main

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"tei/internal/driver"
	"tei/internal/workspace"
)

type candidateProofScopePacketProbe struct {
	CandidateID      string
	RequiredProofIDs []string
	DimensionByProof map[string]string
}

type candidateEliminationCertificatePacketProbe struct {
	CandidateID   string
	ProofIDs      []string
	EvidencePaths []string
	Reasons       []string
}

type unresolvedProofMemberPacketProbe struct {
	ProofID            string
	Question           string
	EvidencePaths      []string
	MissingAuthorities []string
}

type unresolvedDimensionPacketProbe struct {
	Dimension          string
	MemberProofs       []unresolvedProofMemberPacketProbe
	MissingAuthorities []string
}

type candidateLocalProofPacketProbe struct {
	CandidateID          string
	SupportedProofs      []crossViewProofResultProbe
	UnresolvedDimensions []unresolvedDimensionPacketProbe
	Elimination          *candidateEliminationCertificatePacketProbe
}

func TestCandidateLocalProofPacketSelectsOnlyCandidateScopedProofs(t *testing.T) {
	results := loadOrderEntryProofResultsPacketProbe(t)
	packet, err := buildCandidateLocalProofPacketProbe(results, candidateProofScopePacketProbe{
		CandidateID:      "R1-C3-authorization-target",
		RequiredProofIDs: []string{"P4-06a", "P4-06b"},
		DimensionByProof: map[string]string{
			"P4-06b": "business-protected-target-authority",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if packet.Elimination != nil {
		t.Fatalf("unrelated P3-02 contradiction eliminated Authorization candidate: %+v", packet.Elimination)
	}
	if len(packet.SupportedProofs) != 1 || packet.SupportedProofs[0].ID != "P4-06a" {
		t.Fatalf("candidate-local supported proof selection=%+v", packet.SupportedProofs)
	}
	if len(packet.UnresolvedDimensions) != 1 {
		t.Fatalf("candidate-local unresolved dimensions=%+v", packet.UnresolvedDimensions)
	}
	dimension := packet.UnresolvedDimensions[0]
	if dimension.Dimension != "business-protected-target-authority" || len(dimension.MemberProofs) != 1 || dimension.MemberProofs[0].ProofID != "P4-06b" {
		t.Fatalf("Authorization target unresolved trace was not preserved: %+v", dimension)
	}
	if candidatePacketContainsProofID(packet, "P3-02") {
		t.Fatalf("candidate packet imported unrelated mutation contradiction: %+v", packet)
	}
}

func TestCandidateLocalProofPacketUsesCompactContradictionCertificate(t *testing.T) {
	results := loadOrderEntryProofResultsPacketProbe(t)
	packet, err := buildCandidateLocalProofPacketProbe(results, candidateProofScopePacketProbe{
		CandidateID:      "R1-C1-contradictory-composite",
		RequiredProofIDs: []string{"P3-02", "P4-06b"},
		DimensionByProof: map[string]string{
			"P4-06b": "business-protected-target-authority",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if packet.Elimination == nil {
		t.Fatal("decisive contradiction did not eliminate the candidate")
	}
	if !reflect.DeepEqual(packet.Elimination.ProofIDs, []string{"P3-02"}) {
		t.Fatalf("elimination certificate retained non-decisive proof log: %+v", packet.Elimination)
	}
	if !containsStringProofProbe(packet.Elimination.EvidencePaths, "application:order_entry/datasource:master/access:read") {
		t.Fatalf("elimination certificate lost authority evidence: %+v", packet.Elimination)
	}
	if len(packet.SupportedProofs) != 0 || len(packet.UnresolvedDimensions) != 0 {
		t.Fatalf("eliminated packet retained frontier proof payload: %+v", packet)
	}
}

func TestCandidateLocalProofPacketKeepsDifferentSubmitUnresolvedDimensionsSeparate(t *testing.T) {
	results := loadOrderEntryProofResultsPacketProbe(t)
	packet, err := buildCandidateLocalProofPacketProbe(results, candidateProofScopePacketProbe{
		CandidateID:      "R1-C-submit-composite",
		RequiredProofIDs: []string{"P3-07", "P4-06b"},
		DimensionByProof: map[string]string{
			"P3-07":  "mutation-effect-authority",
			"P4-06b": "business-protected-target-authority",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if packet.Elimination != nil {
		t.Fatalf("unresolved submit candidate was eliminated: %+v", packet.Elimination)
	}
	if len(packet.UnresolvedDimensions) != 2 {
		t.Fatalf("different submit unresolved dimensions were flattened: %+v", packet.UnresolvedDimensions)
	}

	byDimension := unresolvedDimensionsByIDPacketProbe(packet.UnresolvedDimensions)
	mutation := byDimension["mutation-effect-authority"]
	if len(mutation.MemberProofs) != 1 || mutation.MemberProofs[0].ProofID != "P3-07" || !containsStringProofProbe(mutation.MissingAuthorities, "capability-to-database-write-attribution") {
		t.Fatalf("mutation-effect unresolved trace=%+v", mutation)
	}
	securityTarget := byDimension["business-protected-target-authority"]
	if len(securityTarget.MemberProofs) != 1 || securityTarget.MemberProofs[0].ProofID != "P4-06b" || !containsStringProofProbe(securityTarget.MissingAuthorities, "business-protected-target-authority") {
		t.Fatalf("Business security target unresolved trace=%+v", securityTarget)
	}
}

func TestCandidateLocalProofPacketAggregationPreservesPerProofMissingAuthority(t *testing.T) {
	results := []crossViewProofResultProbe{
		{
			ID:                 "P6-FROM",
			Family:             "responsibility-handoff",
			Question:           "handoff origin is accepted",
			Result:             proofUnresolvedProbe,
			MissingAuthorities: []string{"accepted-transfer-origin"},
			Reason:             "accepted transfer origin is missing",
		},
		{
			ID:                 "P6-CONDITION",
			Family:             "responsibility-handoff",
			Question:           "handoff continuation predicate is accepted",
			Result:             proofUnresolvedProbe,
			MissingAuthorities: []string{"accepted-continuation-predicate"},
			Reason:             "accepted continuation predicate is missing",
		},
	}
	packet, err := buildCandidateLocalProofPacketProbe(results, candidateProofScopePacketProbe{
		CandidateID:      "R2-M2-responsibility-continuation",
		RequiredProofIDs: []string{"P6-FROM", "P6-CONDITION"},
		DimensionByProof: map[string]string{
			"P6-FROM":      "responsibility-continuation",
			"P6-CONDITION": "responsibility-continuation",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(packet.UnresolvedDimensions) != 1 {
		t.Fatalf("same semantic dimension was not aggregated: %+v", packet.UnresolvedDimensions)
	}
	dimension := packet.UnresolvedDimensions[0]
	if !reflect.DeepEqual(dimension.MissingAuthorities, []string{"accepted-continuation-predicate", "accepted-transfer-origin"}) {
		t.Fatalf("combined missing authorities=%v", dimension.MissingAuthorities)
	}
	if len(dimension.MemberProofs) != 2 {
		t.Fatalf("aggregation lost member proof trace: %+v", dimension)
	}
	members := unresolvedMembersByProofIDPacketProbe(dimension.MemberProofs)
	if !reflect.DeepEqual(members["P6-FROM"].MissingAuthorities, []string{"accepted-transfer-origin"}) {
		t.Fatalf("P6-FROM missing authority was flattened: %+v", members["P6-FROM"])
	}
	if !reflect.DeepEqual(members["P6-CONDITION"].MissingAuthorities, []string{"accepted-continuation-predicate"}) {
		t.Fatalf("P6-CONDITION missing authority was flattened: %+v", members["P6-CONDITION"])
	}
}

func TestCandidateLocalProofPacketFailsClosedWhenRequiredProofIsMissing(t *testing.T) {
	results := loadOrderEntryProofResultsPacketProbe(t)
	_, err := buildCandidateLocalProofPacketProbe(results, candidateProofScopePacketProbe{
		CandidateID:      "missing-proof-candidate",
		RequiredProofIDs: []string{"P9-MISSING"},
	})
	if err == nil {
		t.Fatal("missing required proof was silently treated as safe")
	}
	if !strings.Contains(err.Error(), "required proof") {
		t.Fatalf("missing-proof failure reason is unclear: %v", err)
	}
}

func TestCandidateLocalProofPacketKeepsR2ObservedFactSeparateFromHandoffAuthority(t *testing.T) {
	results := deriveSecondCorpusProofResultsPacketProbe(t)

	commandCandidate, err := buildCandidateLocalProofPacketProbe(results, candidateProofScopePacketProbe{
		CandidateID:      "R2-M1-command-means-completion",
		RequiredProofIDs: []string{"P5-COMMAND-COMPLETION"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if commandCandidate.Elimination == nil || !reflect.DeepEqual(commandCandidate.Elimination.ProofIDs, []string{"P5-COMMAND-COMPLETION"}) {
		t.Fatalf("Command acceptance -> physical completion contradiction was not contracted: %+v", commandCandidate)
	}

	observedThenContinue, err := buildCandidateLocalProofPacketProbe(results, candidateProofScopePacketProbe{
		CandidateID:      "R2-M2-observed-then-continue",
		RequiredProofIDs: []string{"P5-PHYSICAL-OBSERVED", "P6-HANDOFF-AUTHORITY"},
		DimensionByProof: map[string]string{
			"P6-HANDOFF-AUTHORITY": "responsibility-continuation",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if observedThenContinue.Elimination != nil {
		t.Fatalf("supported physical fact plus unresolved Handoff was incorrectly eliminated: %+v", observedThenContinue)
	}
	if len(observedThenContinue.SupportedProofs) != 1 || observedThenContinue.SupportedProofs[0].ID != "P5-PHYSICAL-OBSERVED" {
		t.Fatalf("positive physical Observation support was lost: %+v", observedThenContinue.SupportedProofs)
	}
	if len(observedThenContinue.UnresolvedDimensions) != 1 || observedThenContinue.UnresolvedDimensions[0].MemberProofs[0].ProofID != "P6-HANDOFF-AUTHORITY" {
		t.Fatalf("responsibility Handoff remained insufficiently explicit: %+v", observedThenContinue.UnresolvedDimensions)
	}
	if !containsStringProofProbe(observedThenContinue.UnresolvedDimensions[0].MissingAuthorities, "accepted-business-handoff-source") {
		t.Fatalf("accepted Business Handoff source gap was lost: %+v", observedThenContinue.UnresolvedDimensions[0])
	}
}

func TestCandidateLocalProofPacketIsDerivedAndNotAcceptanceAuthority(t *testing.T) {
	packetType := reflect.TypeOf(candidateLocalProofPacketProbe{})
	want := []string{"CandidateID", "SupportedProofs", "UnresolvedDimensions", "Elimination"}
	if got := fieldNames(packetType); !reflect.DeepEqual(got, want) {
		t.Fatalf("candidate-local packet shape changed: got=%v want=%v", got, want)
	}
	for _, forbidden := range []string{
		"Accepted", "Valid", "Complete", "Confidence", "Score", "Ranking", "Canonical", "Authoritative", "WriteBack", "SourceOfTruth",
	} {
		if _, found := packetType.FieldByName(forbidden); found {
			t.Fatalf("candidate-local derived packet gained authority/ranking field %q", forbidden)
		}
	}
}

func buildCandidateLocalProofPacketProbe(results []crossViewProofResultProbe, scope candidateProofScopePacketProbe) (candidateLocalProofPacketProbe, error) {
	if strings.TrimSpace(scope.CandidateID) == "" {
		return candidateLocalProofPacketProbe{}, fmt.Errorf("candidate ID is required")
	}
	byID := make(map[string]crossViewProofResultProbe, len(results))
	for _, result := range results {
		if strings.TrimSpace(result.ID) == "" {
			return candidateLocalProofPacketProbe{}, fmt.Errorf("proof result without ID")
		}
		if _, exists := byID[result.ID]; exists {
			return candidateLocalProofPacketProbe{}, fmt.Errorf("proof result %s is ambiguous", result.ID)
		}
		byID[result.ID] = result
	}

	selected := make([]crossViewProofResultProbe, 0, len(scope.RequiredProofIDs))
	seenRequired := map[string]struct{}{}
	for _, proofID := range scope.RequiredProofIDs {
		if _, duplicate := seenRequired[proofID]; duplicate {
			return candidateLocalProofPacketProbe{}, fmt.Errorf("required proof %s is duplicated for candidate %s", proofID, scope.CandidateID)
		}
		seenRequired[proofID] = struct{}{}
		result, ok := byID[proofID]
		if !ok {
			return candidateLocalProofPacketProbe{}, fmt.Errorf("required proof %s is missing for candidate %s", proofID, scope.CandidateID)
		}
		selected = append(selected, result)
	}

	packet := candidateLocalProofPacketProbe{CandidateID: scope.CandidateID}
	contradictions := make([]crossViewProofResultProbe, 0)
	unresolvedByDimension := map[string][]crossViewProofResultProbe{}
	for _, result := range selected {
		switch result.Result {
		case proofSupportedProbe:
			packet.SupportedProofs = append(packet.SupportedProofs, result)
		case proofContradictionProbe:
			contradictions = append(contradictions, result)
		case proofUnresolvedProbe:
			dimension := strings.TrimSpace(scope.DimensionByProof[result.ID])
			if dimension == "" {
				return candidateLocalProofPacketProbe{}, fmt.Errorf("unresolved proof %s lacks semantic dimension for candidate %s", result.ID, scope.CandidateID)
			}
			unresolvedByDimension[dimension] = append(unresolvedByDimension[dimension], result)
		default:
			return candidateLocalProofPacketProbe{}, fmt.Errorf("proof %s has unsupported result %q", result.ID, result.Result)
		}
	}

	if len(contradictions) > 0 {
		packet.SupportedProofs = nil
		packet.UnresolvedDimensions = nil
		packet.Elimination = buildEliminationCertificatePacketProbe(scope.CandidateID, contradictions)
		return packet, nil
	}

	sort.Slice(packet.SupportedProofs, func(i, j int) bool { return packet.SupportedProofs[i].ID < packet.SupportedProofs[j].ID })
	packet.UnresolvedDimensions = buildUnresolvedDimensionsPacketProbe(unresolvedByDimension)
	return packet, nil
}

func buildEliminationCertificatePacketProbe(candidateID string, contradictions []crossViewProofResultProbe) *candidateEliminationCertificatePacketProbe {
	proofIDs := make([]string, 0, len(contradictions))
	evidence := make([]string, 0)
	reasons := make([]string, 0, len(contradictions))
	for _, result := range contradictions {
		proofIDs = append(proofIDs, result.ID)
		evidence = append(evidence, result.EvidencePaths...)
		reasons = append(reasons, result.Reason)
	}
	return &candidateEliminationCertificatePacketProbe{
		CandidateID:   candidateID,
		ProofIDs:      uniqueSortedStringsPacketProbe(proofIDs),
		EvidencePaths: uniqueSortedStringsPacketProbe(evidence),
		Reasons:       uniqueSortedStringsPacketProbe(reasons),
	}
}

func buildUnresolvedDimensionsPacketProbe(byDimension map[string][]crossViewProofResultProbe) []unresolvedDimensionPacketProbe {
	dimensions := make([]string, 0, len(byDimension))
	for dimension := range byDimension {
		dimensions = append(dimensions, dimension)
	}
	sort.Strings(dimensions)

	result := make([]unresolvedDimensionPacketProbe, 0, len(dimensions))
	for _, dimension := range dimensions {
		proofs := append([]crossViewProofResultProbe(nil), byDimension[dimension]...)
		sort.Slice(proofs, func(i, j int) bool { return proofs[i].ID < proofs[j].ID })
		item := unresolvedDimensionPacketProbe{Dimension: dimension}
		combinedMissing := make([]string, 0)
		for _, proof := range proofs {
			member := unresolvedProofMemberPacketProbe{
				ProofID:            proof.ID,
				Question:           proof.Question,
				EvidencePaths:      uniqueSortedStringsPacketProbe(proof.EvidencePaths),
				MissingAuthorities: uniqueSortedStringsPacketProbe(proof.MissingAuthorities),
			}
			item.MemberProofs = append(item.MemberProofs, member)
			combinedMissing = append(combinedMissing, proof.MissingAuthorities...)
		}
		item.MissingAuthorities = uniqueSortedStringsPacketProbe(combinedMissing)
		result = append(result, item)
	}
	return result
}

func loadOrderEntryProofResultsPacketProbe(t *testing.T) []crossViewProofResultProbe {
	t.Helper()
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	policy, err := readLegacyAuthorizationFixtureProbe(filepath.Join(root, "reference", "legacy-auth", "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	results, err := deriveOrderEntryCrossViewProofReportProbe(loaded.World, f, policy)
	if err != nil {
		t.Fatal(err)
	}
	return results
}

func deriveSecondCorpusProofResultsPacketProbe(t *testing.T) []crossViewProofResultProbe {
	t.Helper()

	commandPressure := physicalWorldCompletionPressure{
		BusinessFact:    "manufacturing physically completed",
		CommandAccepted: true,
	}
	if diagnostics := physicalWorldCompletionDiagnostics("SECOND-CORPUS", commandPressure); !hasDiagnosticCodePacketProbe(diagnostics, "BUSINESS_PHYSICAL_COMPLETION_REQUIRES_OBSERVATION") {
		t.Fatalf("test setup: Command acceptance no longer pressures physical completion: %+v", diagnostics)
	}

	execution, err := driver.ExecuteObservationRead(
		context.Background(),
		"second-corpus.physical-state.read",
		driver.ObservationEvidenceExternalObserved,
		func(context.Context) (any, error) { return true, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	observedPressure := physicalWorldCompletionPressure{
		BusinessFact:      "finished goods physically present",
		CommandAccepted:   true,
		ObservationResult: &execution,
	}
	if diagnostics := physicalWorldCompletionDiagnostics("SECOND-CORPUS", observedPressure); len(diagnostics) != 0 {
		t.Fatalf("test setup: positive external Observation is not supported: %+v", diagnostics)
	}

	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}
	withoutExplicitHandoff := set
	withoutExplicitHandoff.Handoffs = nil
	model, err := deriveNormalizedBusinessModelProbe(withoutExplicitHandoff, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Handoffs) != 0 {
		t.Fatalf("test setup: Observation/adjacency synthesized Handoff: %+v", model.Handoffs)
	}

	return []crossViewProofResultProbe{
		{
			ID:            "P5-COMMAND-COMPLETION",
			Family:        "system-result-actuality",
			Question:      "Command acceptance establishes manufacturing physical completion",
			Result:        proofContradictionProbe,
			EvidencePaths: []string{"driver-boundary:command-accepted/physical-completion-requires-observation"},
			Reason:        "Command acceptance alone cannot establish physical completion",
		},
		{
			ID:            "P5-PHYSICAL-OBSERVED",
			Family:        "system-result-actuality",
			Question:      "finished goods physically present has been explicitly observed",
			Result:        proofSupportedProbe,
			EvidencePaths: []string{"runtime-observation:second-corpus.physical-state.read/external-observed/result:true"},
			Reason:        "explicit external Observation supports only the observed physical fact",
		},
		{
			ID:                 "P6-HANDOFF-AUTHORITY",
			Family:             "responsibility-handoff",
			Question:           "observed physical completion establishes responsibility Handoff",
			Result:             proofUnresolvedProbe,
			EvidencePaths:      []string{"runtime-observation:second-corpus.physical-state.read/external-observed/result:true"},
			MissingAuthorities: []string{"accepted-business-handoff-source", "accepted-handoff-endpoints", "accepted-handoff-condition"},
			Reason:             "positive physical Observation does not author Business Handoff from/to/condition",
		},
	}
}

func hasDiagnosticCodePacketProbe(diagnostics []diagnostic, code string) bool {
	for _, item := range diagnostics {
		if item.Code == code {
			return true
		}
	}
	return false
}

func candidatePacketContainsProofID(packet candidateLocalProofPacketProbe, proofID string) bool {
	for _, proof := range packet.SupportedProofs {
		if proof.ID == proofID {
			return true
		}
	}
	for _, dimension := range packet.UnresolvedDimensions {
		for _, member := range dimension.MemberProofs {
			if member.ProofID == proofID {
				return true
			}
		}
	}
	if packet.Elimination != nil && containsStringProofProbe(packet.Elimination.ProofIDs, proofID) {
		return true
	}
	return false
}

func unresolvedDimensionsByIDPacketProbe(items []unresolvedDimensionPacketProbe) map[string]unresolvedDimensionPacketProbe {
	result := make(map[string]unresolvedDimensionPacketProbe, len(items))
	for _, item := range items {
		result[item.Dimension] = item
	}
	return result
}

func unresolvedMembersByProofIDPacketProbe(items []unresolvedProofMemberPacketProbe) map[string]unresolvedProofMemberPacketProbe {
	result := make(map[string]unresolvedProofMemberPacketProbe, len(items))
	for _, item := range items {
		result[item.ProofID] = item
	}
	return result
}

func uniqueSortedStringsPacketProbe(items []string) []string {
	set := map[string]struct{}{}
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			set[item] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for item := range set {
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}

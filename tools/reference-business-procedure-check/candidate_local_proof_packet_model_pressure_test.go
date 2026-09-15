package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
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

func TestCandidateLocalProofPacketUsesOnlyDecisiveContradictionForElimination(t *testing.T) {
	results := []crossViewProofResultProbe{
		{
			ID:            "P-CONTRA",
			Family:        "mutation-authority",
			Question:      "candidate mutates read-only authority",
			Result:        proofContradictionProbe,
			EvidencePaths: []string{"authority:read-only"},
			Reason:        "candidate conflicts with explicit read-only authority",
		},
		{
			ID:                 "P-UNRES",
			Family:             "business-security-boundary",
			Question:           "candidate protected target is accepted",
			Result:             proofUnresolvedProbe,
			MissingAuthorities: []string{"business-protected-target-authority"},
			Reason:             "Business target authority is missing",
		},
	}

	packet, err := buildCandidateLocalProofPacketProbe(results, candidateProofScopePacketProbe{
		CandidateID:      "contradictory-composite",
		RequiredProofIDs: []string{"P-CONTRA", "P-UNRES"},
	})
	if err != nil {
		t.Fatalf("decisive contradiction should not require frontier-only unresolved grouping: %v", err)
	}
	if packet.Elimination == nil {
		t.Fatal("decisive contradiction did not eliminate the candidate")
	}
	if !reflect.DeepEqual(packet.Elimination.ProofIDs, []string{"P-CONTRA"}) {
		t.Fatalf("elimination certificate retained non-decisive proof log: %+v", packet.Elimination)
	}
	if !reflect.DeepEqual(packet.Elimination.EvidencePaths, []string{"authority:read-only"}) {
		t.Fatalf("elimination certificate lost authority evidence: %+v", packet.Elimination)
	}
	if len(packet.SupportedProofs) != 0 || len(packet.UnresolvedDimensions) != 0 {
		t.Fatalf("eliminated packet retained frontier payload: %+v", packet)
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
		CandidateID:      "responsibility-continuation",
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
	members := unresolvedMembersByProofIDPacketProbe(dimension.MemberProofs)
	if !reflect.DeepEqual(members["P6-FROM"].MissingAuthorities, []string{"accepted-transfer-origin"}) {
		t.Fatalf("P6-FROM missing authority was flattened: %+v", members["P6-FROM"])
	}
	if !reflect.DeepEqual(members["P6-CONDITION"].MissingAuthorities, []string{"accepted-continuation-predicate"}) {
		t.Fatalf("P6-CONDITION missing authority was flattened: %+v", members["P6-CONDITION"])
	}
}

func TestCandidateLocalProofPacketFailsClosedWhenRequiredProofIsMissing(t *testing.T) {
	_, err := buildCandidateLocalProofPacketProbe(nil, candidateProofScopePacketProbe{
		CandidateID:      "missing-proof-candidate",
		RequiredProofIDs: []string{"P-MISSING"},
	})
	if err == nil {
		t.Fatal("missing required proof was silently treated as safe")
	}
	if !strings.Contains(err.Error(), "required proof") {
		t.Fatalf("missing-proof failure reason is unclear: %v", err)
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
	for _, result := range selected {
		switch result.Result {
		case proofSupportedProbe, proofUnresolvedProbe:
			// Frontier payload is built only if no decisive contradiction exists.
		case proofContradictionProbe:
			contradictions = append(contradictions, result)
		default:
			return candidateLocalProofPacketProbe{}, fmt.Errorf("proof %s has unsupported result %q", result.ID, result.Result)
		}
	}
	if len(contradictions) > 0 {
		packet.Elimination = buildEliminationCertificatePacketProbe(scope.CandidateID, contradictions)
		return packet, nil
	}

	unresolvedByDimension := map[string][]crossViewProofResultProbe{}
	for _, result := range selected {
		switch result.Result {
		case proofSupportedProbe:
			packet.SupportedProofs = append(packet.SupportedProofs, result)
		case proofUnresolvedProbe:
			dimension := strings.TrimSpace(scope.DimensionByProof[result.ID])
			if dimension == "" {
				return candidateLocalProofPacketProbe{}, fmt.Errorf("unresolved proof %s lacks semantic dimension for candidate %s", result.ID, scope.CandidateID)
			}
			unresolvedByDimension[dimension] = append(unresolvedByDimension[dimension], result)
		case proofContradictionProbe:
			return candidateLocalProofPacketProbe{}, fmt.Errorf("internal error: contradiction %s escaped elimination", result.ID)
		default:
			return candidateLocalProofPacketProbe{}, fmt.Errorf("proof %s has unsupported result %q", result.ID, result.Result)
		}
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

package main

import (
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

func TestCandidateLocalProofPacketSelectsOnlyCandidateScopedR1Proofs(t *testing.T) {
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

func TestCandidateLocalProofPacketKeepsDifferentR1SubmitUnresolvedDimensionsSeparate(t *testing.T) {
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

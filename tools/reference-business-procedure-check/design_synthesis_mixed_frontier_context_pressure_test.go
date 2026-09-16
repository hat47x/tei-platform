package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// mixedFrontierCandidateProbe is deliberately test-only. It keeps candidate
// meaning, current M1 relation, discovery traces, and proof results on separate
// axes so a bounded context cannot collapse them into one origin/status/score.
type mixedFrontierCandidateProbe struct {
	CandidateID            string
	SemanticClaim          string
	ExistingTargetIdentity string
	ProposalOnlyMeaning    string
	DiscoveryRefs          []string
	Proofs                 []mixedFrontierProofProbe
	UnresolvedDimensions   []string
}

type mixedFrontierProofProbe struct {
	ProofID              string
	Result               string
	Applicable           bool
	AcceptedAuthorityRef string
	MissingAuthorities   []string
	Reason                string
}

type mixedFrontierEliminationProbe struct {
	CandidateID            string
	ContradictedClaim      string
	DecisiveProofID        string
	AcceptedAuthorityRef   string
	Reason                  string
}

type mixedFrontierContextProbe struct {
	Frontier   []mixedFrontierCandidateProbe
	Eliminated []mixedFrontierEliminationProbe
}

const (
	mixedFrontierSupportedProbe     = "supported"
	mixedFrontierContradictionProbe = "contradiction"
	mixedFrontierUnresolvedProbe    = "unresolved"
)

func TestMixedFrontierContextPreservesSupportedExistingAndUnresolvedProposal(t *testing.T) {
	candidates := []mixedFrontierCandidateProbe{
		{
			CandidateID:            "C-A",
			SemanticClaim:          "reuse the existing read-only customer reference",
			ExistingTargetIdentity: "order_entry.customer_reference",
			DiscoveryRefs:          []string{"exact:order_entry.customer_reference", "intent:customer-confirmation"},
			Proofs: []mixedFrontierProofProbe{
				{
					ProofID:              "P2-02",
					Result:               mixedFrontierSupportedProbe,
					Applicable:           true,
					AcceptedAuthorityRef: "relation:sales.order.customer",
					Reason:                "accepted relation and read path are exact",
				},
			},
		},
		{
			CandidateID:         "C-B",
			SemanticClaim:       "hold customer confirmation data as an order-owned snapshot",
			ProposalOnlyMeaning: "order-owned customer snapshot with explicit freshness semantics",
			DiscoveryRefs:       []string{"human:synthetic-partial-intent", "pattern:historical-snapshot"},
			Proofs: []mixedFrontierProofProbe{
				{
					ProofID:            "P-SNAPSHOT-NEED",
					Result:             mixedFrontierUnresolvedProbe,
					Applicable:         true,
					MissingAuthorities: []string{"accepted snapshot requirement", "copy timing authority", "freshness semantics"},
					Reason:              "current source neither requires nor forbids an order-owned snapshot",
				},
			},
			UnresolvedDimensions: []string{"snapshot necessity", "copy timing", "freshness semantics"},
		},
		{
			CandidateID:         "C-C",
			SemanticClaim:       "update customer master from order handling",
			ProposalOnlyMeaning: "customer master mutation from order handling",
			DiscoveryRefs:       []string{"human:synthetic-partial-intent"},
			Proofs: []mixedFrontierProofProbe{
				{
					ProofID:              "P3-02",
					Result:               mixedFrontierContradictionProbe,
					Applicable:           true,
					AcceptedAuthorityRef: "application:order_entry/database:master/access:read",
					Reason:                "master datasource is accepted as read-only",
				},
			},
		},
	}

	context, err := contractMixedFrontierProbe(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if len(context.Frontier) != 2 {
		t.Fatalf("mixed frontier was forced to one candidate: %+v", context.Frontier)
	}
	if len(context.Eliminated) != 1 || context.Eliminated[0].CandidateID != "C-C" {
		t.Fatalf("exact contradiction was not separated into an elimination certificate: %+v", context.Eliminated)
	}

	existing := findMixedFrontierCandidateProbe(context.Frontier, "C-A")
	proposal := findMixedFrontierCandidateProbe(context.Frontier, "C-B")
	if existing == nil || proposal == nil {
		t.Fatalf("supported existing and unresolved proposal must both survive: %+v", context.Frontier)
	}
	if existing.ExistingTargetIdentity != "order_entry.customer_reference" {
		t.Fatalf("existing candidate lost exact M1 identity: %+v", existing)
	}
	if len(existing.DiscoveryRefs) != 2 {
		t.Fatalf("multiple discovery traces were flattened: %+v", existing.DiscoveryRefs)
	}
	if proposal.ExistingTargetIdentity != "" {
		t.Fatalf("proposal-only candidate acquired a fabricated accepted identity: %+v", proposal)
	}
	if strings.TrimSpace(proposal.ProposalOnlyMeaning) == "" || len(proposal.UnresolvedDimensions) == 0 {
		t.Fatalf("proposal-only meaning or unresolved dimensions were lost: %+v", proposal)
	}
}

func TestMixedFrontierDoesNotUseSupportCountAsWinnerSelection(t *testing.T) {
	candidates := []mixedFrontierCandidateProbe{
		{
			CandidateID:            "C-A",
			SemanticClaim:          "existing reuse",
			ExistingTargetIdentity: "order_entry.customer_reference",
			Proofs: []mixedFrontierProofProbe{
				{ProofID: "P1", Result: mixedFrontierSupportedProbe, Applicable: true},
				{ProofID: "P2", Result: mixedFrontierSupportedProbe, Applicable: true},
			},
		},
		{
			CandidateID:         "C-B",
			SemanticClaim:       "new snapshot proposal",
			ProposalOnlyMeaning: "snapshot",
			Proofs: []mixedFrontierProofProbe{
				{ProofID: "P3", Result: mixedFrontierUnresolvedProbe, Applicable: true, MissingAuthorities: []string{"snapshot requirement"}},
			},
		},
	}

	context, err := contractMixedFrontierProbe(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if len(context.Frontier) != 2 {
		t.Fatalf("support-count preference incorrectly became contraction: %+v", context.Frontier)
	}
}

func TestMixedFrontierContextShapeRejectsAuthorityAndRankingShortcutFields(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(mixedFrontierCandidateProbe{}),
		reflect.TypeOf(mixedFrontierContextProbe{}),
	} {
		for i := 0; i < typ.NumField(); i++ {
			name := strings.ToLower(typ.Field(i).Name)
			for _, forbidden := range []string{"origin", "winner", "preferred", "score", "confidence", "accepted", "valid"} {
				if strings.Contains(name, forbidden) {
					t.Fatalf("test-only mixed frontier shape reintroduced shortcut field %q in %s", typ.Field(i).Name, typ.Name())
				}
			}
		}
	}
}

func contractMixedFrontierProbe(candidates []mixedFrontierCandidateProbe) (mixedFrontierContextProbe, error) {
	result := mixedFrontierContextProbe{}
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		candidate.CandidateID = strings.TrimSpace(candidate.CandidateID)
		candidate.SemanticClaim = strings.TrimSpace(candidate.SemanticClaim)
		if candidate.CandidateID == "" || candidate.SemanticClaim == "" {
			return mixedFrontierContextProbe{}, fmt.Errorf("candidate id and semantic claim are required")
		}
		if _, exists := seen[candidate.CandidateID]; exists {
			return mixedFrontierContextProbe{}, fmt.Errorf("duplicate candidate id %s", candidate.CandidateID)
		}
		seen[candidate.CandidateID] = struct{}{}
		candidate.DiscoveryRefs = uniqueSortedMixedFrontierStringsProbe(candidate.DiscoveryRefs)
		candidate.UnresolvedDimensions = uniqueSortedMixedFrontierStringsProbe(candidate.UnresolvedDimensions)

		eliminated := false
		for _, proof := range candidate.Proofs {
			if !proof.Applicable || proof.Result != mixedFrontierContradictionProbe {
				continue
			}
			if strings.TrimSpace(proof.ProofID) == "" || strings.TrimSpace(proof.AcceptedAuthorityRef) == "" {
				return mixedFrontierContextProbe{}, fmt.Errorf("candidate %s contradiction lacks decisive proof or authority", candidate.CandidateID)
			}
			result.Eliminated = append(result.Eliminated, mixedFrontierEliminationProbe{
				CandidateID:          candidate.CandidateID,
				ContradictedClaim:    candidate.SemanticClaim,
				DecisiveProofID:      proof.ProofID,
				AcceptedAuthorityRef: proof.AcceptedAuthorityRef,
				Reason:               proof.Reason,
			})
			eliminated = true
			break
		}
		if !eliminated {
			result.Frontier = append(result.Frontier, candidate)
		}
	}
	sort.Slice(result.Frontier, func(i, j int) bool {
		return result.Frontier[i].CandidateID < result.Frontier[j].CandidateID
	})
	sort.Slice(result.Eliminated, func(i, j int) bool {
		return result.Eliminated[i].CandidateID < result.Eliminated[j].CandidateID
	})
	return result, nil
}

func findMixedFrontierCandidateProbe(candidates []mixedFrontierCandidateProbe, id string) *mixedFrontierCandidateProbe {
	for i := range candidates {
		if candidates[i].CandidateID == id {
			return &candidates[i]
		}
	}
	return nil
}

func uniqueSortedMixedFrontierStringsProbe(items []string) []string {
	set := map[string]struct{}{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			set[trimmed] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for item := range set {
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}

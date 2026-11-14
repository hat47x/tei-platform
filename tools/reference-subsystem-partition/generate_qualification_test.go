package partition

import (
	"fmt"
	"testing"
)

// A must-link component may contain multiple Functions. The heuristic path
// must not mistake component count for Function count and report infeasible.
func TestHeuristicMustLinkKeepsFeasibleCandidate(t *testing.T) {
	p := CandidateProblem{
		Functions: make([]Function, 11),
		MustLinks: []MustLink{{Left: "f00", Right: "f01"}},
	}
	for i := range p.Functions {
		p.Functions[i] = Function{
			ID: fmt.Sprintf("f%02d", i), Allowed: []string{"sales"},
		}
	}
	candidates, status := GenerateCandidates(p)
	if status != string(StatusCandidateRanked) || len(candidates) == 0 {
		t.Fatalf("feasible single-allowed setup cannot be infeasible: status=%s candidates=%d", status, len(candidates))
	}
	if !candidates[0].ConstraintsMet || len(candidates[0].Assignments) != 11 {
		t.Fatalf("must-link collapsed Function count or candidate is incomplete: %+v", candidates[0])
	}
	for _, a := range candidates[0].Assignments {
		if a.Subsystem != "sales" {
			t.Fatalf("unexpected placement in single-allowed setup: %+v", a)
		}
	}
}

// With 10 Functions and 3 allowed memberships each, the 59,049 possible
// placements exceed the fixed 4,096 enumerator cap. "exhaustive-proven"
// would claim a proof that the implementation has not completed.
func TestExhaustiveLabelRequiresFiniteCompleteEnumeration(t *testing.T) {
	p := CandidateProblem{Functions: make([]Function, 10)}
	for i := range p.Functions {
		p.Functions[i] = Function{
			ID:      fmt.Sprintf("f%02d", i),
			Allowed: []string{"sales", "billing", "inventory"},
		}
	}
	candidates, status := GenerateCandidates(p)
	if status != string(StatusCandidateRanked) || len(candidates) == 0 {
		t.Fatalf("capped enumeration cannot claim exhaustive proof: status=%s candidates=%d", status, len(candidates))
	}
	for _, c := range candidates {
		if c.Status == string(StatusExhaustiveProven) {
			t.Fatalf("partial placement exploration was labeled exhaustive: %+v", c)
		}
	}
}

// A caller's smaller candidate limit must not be ignored when classifying
// enumeration as exhaustive; small fully-enumerable cases still keep proof.
func TestExhaustiveStatusRespectsRequestedCandidateBound(t *testing.T) {
	p := CandidateProblem{Functions: make([]Function, 4)}
	for i := range p.Functions {
		p.Functions[i] = Function{
			ID:      fmt.Sprintf("f%02d", i),
			Allowed: []string{"sales", "billing"},
		}
	}
	limited, limitedStatus := GenerateCandidates(p, WithMaxCandidates(4))
	if limitedStatus != string(StatusCandidateRanked) || len(limited) == 0 {
		t.Fatalf("candidate limit lower than 16 placements must not claim exhaustive: status=%s n=%d", limitedStatus, len(limited))
	}
	full, fullStatus := GenerateCandidates(p)
	if fullStatus != string(StatusExhaustiveProven) || len(full) != 16 {
		t.Fatalf("uncapped 2^4 placement enumeration must remain proven: status=%s n=%d", fullStatus, len(full))
	}
}

package partition

import "testing"

// The evaluator must not claim a numeric result or panic when a caller sends
// candidates that contain no feasible partition. This is independent of the
// generator's own infeasibility/heuristic classification.
func TestEvaluateCandidatesNoFeasibleFrontReturnsError(t *testing.T) {
	candidates := []Candidate{{
		ID: "infeasible", Status: string(StatusInfeasible),
		ConstraintsMet: false, Evaluation: Evaluation{CrossWriteCount: 0},
	}}
	reports, err := EvaluateCandidates(candidates)
	if err == nil || len(reports) != 0 {
		t.Fatalf("no feasible candidate must not produce a score: reports=%+v err=%v", reports, err)
	}
}

func TestEvaluateCandidatesNonDominatedShareParetoFrontRank(t *testing.T) {
	candidates := []Candidate{
		{
			ID: "writes-first", Status: string(StatusCandidateRanked), ConstraintsMet: true,
			Assignments: []Assignment{{Function: "F", Subsystem: "sales"}},
			Evaluation: Evaluation{CrossWriteCount: 1, CrossWriteWeight: 2, CrossReadCount: 3,
				TotalWeight: 4, InternalWeight: 2},
		},
		{
			ID: "reads-first", Status: string(StatusCandidateRanked), ConstraintsMet: true,
			Assignments: []Assignment{{Function: "F", Subsystem: "billing"}},
			Evaluation: Evaluation{CrossWriteCount: 3, CrossWriteWeight: 4, CrossReadCount: 1,
				TotalWeight: 4, InternalWeight: 0},
		},
		{
			ID: "dominated", Status: string(StatusCandidateRanked), ConstraintsMet: true,
			Assignments: []Assignment{{Function: "F", Subsystem: "inventory"}},
			Evaluation: Evaluation{CrossWriteCount: 4, CrossWriteWeight: 5, CrossReadCount: 4,
				TotalWeight: 4, InternalWeight: 0},
		},
	}
	reports, err := EvaluateCandidates(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 3 {
		t.Fatalf("expected three candidate reports, got %d", len(reports))
	}
	seen := map[string]bool{}
	for _, report := range reports {
		if report.ID == "dominated" {
			if report.Breakdown.ParetoRank != nil {
				t.Fatalf("no secondary Pareto layers are computed; dominated rank must remain unknown: %+v", report)
			}
			continue
		}
		seen[report.ID] = true
		if report.Breakdown.ParetoRank == nil || *report.Breakdown.ParetoRank != 1 {
			t.Fatalf("both non-dominated tradeoff candidates must be on front 1: %+v", report)
		}
	}
	if !seen["writes-first"] || !seen["reads-first"] {
		t.Fatalf("missing one first-front candidate: %+v", seen)
	}
	// The two supplied CrossReadCount fields are synthetic objective values.
	// This test does NOT qualify the current generator's total-read value as
	// cross-subsystem read without a separate Data Owner mapping.
}

func TestEvaluateCandidatesInternalCohesionUsesFullWeightDenominator(t *testing.T) {
	candidates := []Candidate{{
		ID: "fully-internal", Status: string(StatusCandidateRanked), ConstraintsMet: true,
		Assignments: []Assignment{{Function: "F", Subsystem: "sales"}},
		Evaluation: Evaluation{CrossWriteCount: 0, CrossReadCount: 0,
			TotalWeight: 1, InternalWeight: 1, Cohesion: 1},
	}}
	reports, err := EvaluateCandidates(candidates)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, axis := range reports[0].Breakdown.Axes {
		if axis.Name != "internal_cohesion" {
			continue
		}
		found = true
		if axis.Unit != "basis-points" || axis.Value != 10000 {
			t.Fatalf("100%% internal weight should be 10000 basis points, got %+v", axis)
		}
	}
	if !found {
		t.Fatal("missing internal_cohesion axis")
	}
}

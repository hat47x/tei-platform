package partition

import (
	"encoding/json"
	"fmt"
	"sort"
)

// StatusKind is the lifecycle status of a candidate or report.
type StatusKind string

const (
	StatusExact            StatusKind = "exact-for-supplied-graph"
	StatusExhaustiveProven StatusKind = "exhaustive-proven"
	StatusCandidateRanked  StatusKind = "candidate-ranked"
	StatusInfeasible       StatusKind = "infeasible"
	StatusUnknown          StatusKind = "unknown"
)

// BreakdownAxis is one evaluation objective.
type BreakdownAxis struct {
	Name      string `json:"name"`
	Value     int64  `json:"value"`
	Weight    int64  `json:"weight"`
	Direction string `json:"direction"` // minimize or maximize
	Unit      string `json:"unit"`
	Notes     string `json:"notes"`
}

// Breakdown contains per-axis evaluations and a derived single-score auxiliary view.
type Breakdown struct {
	Axes        []BreakdownAxis `json:"axes"`
	SingleScore *float64        `json:"singleScore,omitempty"` // omitted when units are incompatible
	ParetoRank  *int            `json:"paretoRank,omitempty"`
}

// CandidateReport bundles all outputs for one candidate partition.
type CandidateReport struct {
	ID              string       `json:"id"`
	Status          StatusKind   `json:"status"`
	Assignments     []Assignment `json:"assignments"`
	Evaluation      Evaluation   `json:"evaluation"`
	Breakdown       Breakdown    `json:"breakdown"`
	AuthorityLimit  string       `json:"authorityLimit"`
	ProvenanceNotes []string     `json:"provenanceNotes"`
}

// EvaluateCandidates computes the multi-objective breakdown for each candidate.
func EvaluateCandidates(candidates []Candidate) ([]CandidateReport, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates to evaluate")
	}
	pareto := ParetoFront(candidates)
	if len(pareto) == 0 {
		return nil, fmt.Errorf("no feasible candidates to evaluate")
	}
	paretoRank := map[string]*int{}
	for _, c := range pareto {
		// Every non-dominated candidate belongs to the same first front.
		// Secondary Pareto layers are not computed; leave those ranks unset.
		r := 1
		paretoRank[c.ID] = &r
	}

	reports := make([]CandidateReport, 0, len(candidates))
	for _, c := range candidates {
		ev := c.Evaluation
		cohesionBasisPoints := int64(0)
		if ev.TotalWeight > 0 {
			cohesionBasisPoints = int64(float64(ev.InternalWeight) * 10000 / float64(ev.TotalWeight))
		}
		axes := []BreakdownAxis{
			{Name: "cross_boundary_write_count", Value: int64(ev.CrossWriteCount), Weight: 0, Direction: "minimize", Unit: "occurrence", Notes: "C/U/D occurrences across subsystem boundary"},
			{Name: "cross_boundary_write_weight", Value: ev.CrossWriteWeight, Weight: 0, Direction: "minimize", Unit: "weight", Notes: "caller-declared positive weight"},
			{Name: "internal_cohesion", Value: cohesionBasisPoints, Weight: 0, Direction: "maximize", Unit: "basis-points", Notes: "internal weight / total weight"},
			{Name: "cross_boundary_read_count", Value: int64(ev.CrossReadCount), Weight: 0, Direction: "minimize", Unit: "occurrence", Notes: "read occurrences across boundary"},
			{Name: "change_impact_reach", Value: 0, Weight: 0, Direction: "minimize", Unit: "function-count", Notes: "placeholder; callers may plug AnalyzeImpact reach"},
			{Name: "team_comprehensibility", Value: 0, Weight: 0, Direction: "maximize", Unit: "caller-score", Notes: "caller-declared coefficient only"},
			{Name: "migration_cost", Value: int64(ev.CrossWriteCount + len(c.Assignments) - len(pareto[0].Assignments)), Weight: 0, Direction: "minimize", Unit: "function-count", Notes: "simplified movement proxy"},
			{Name: "max_size", Value: 0, Weight: 0, Direction: "minimize", Unit: "function-count", Notes: "largest subsystem size"},
		}
		maxSize := 0
		for _, s := range ev.Sizes {
			if s > maxSize {
				maxSize = s
			}
		}
		axes[7].Value = int64(maxSize)

		var single *float64
		// single score only when comparing same-unit axes; here we omit incompatible mix.
		_ = single
		reports = append(reports, CandidateReport{
			ID: c.ID, Status: StatusKind(c.Status), Assignments: c.Assignments, Evaluation: ev,
			Breakdown:      Breakdown{Axes: axes, SingleScore: single, ParetoRank: paretoRank[c.ID]},
			AuthorityLimit: AuthorityLimit, ProvenanceNotes: c.ConstraintIssues,
		})
	}
	sort.Slice(reports, func(i, j int) bool {
		if reports[i].Status != reports[j].Status {
			return reports[i].Status < reports[j].Status
		}
		if reports[i].Evaluation.CrossWriteCount != reports[j].Evaluation.CrossWriteCount {
			return reports[i].Evaluation.CrossWriteCount < reports[j].Evaluation.CrossWriteCount
		}
		if reports[i].Evaluation.CrossWriteWeight != reports[j].Evaluation.CrossWriteWeight {
			return reports[i].Evaluation.CrossWriteWeight < reports[j].Evaluation.CrossWriteWeight
		}
		return reports[i].ID < reports[j].ID
	})
	return reports, nil
}

// MarshalCandidateReport deterministically encodes a report.
func MarshalCandidateReport(r CandidateReport) []byte {
	b, _ := json.Marshal(r)
	return b
}

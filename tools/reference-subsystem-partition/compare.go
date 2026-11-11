package partition

import (
	"sort"
	"time"
)

// ComparisonRow is one candidate in a before/after comparison table.
type ComparisonRow struct {
	ID               string  `json:"id"`
	Status           string  `json:"status"`
	CrossWriteCount  int     `json:"crossWriteCount"`
	CrossWriteWeight int64   `json:"crossWriteWeight"`
	CrossReadCount   int     `json:"crossReadCount"`
	Cohesion         float64 `json:"cohesion"`
	MaxSize          int     `json:"maxSize"`
	FunctionCount    int     `json:"functionCount"`
}

// ComparisonReport is a deterministic sorted comparison of candidates.
type ComparisonReport struct {
	SchemaVersion  string          `json:"schemaVersion"`
	GeneratedAt    string          `json:"generatedAt"`
	BaseCommit     string          `json:"baseCommit"`
	AuthorityLimit string          `json:"authorityLimit"`
	Rows           []ComparisonRow `json:"rows"`
}

// BuildComparisonReport deterministically ranks candidates by cross-boundary write/read objectives.
func BuildComparisonReport(candidates []Candidate) (ComparisonReport, error) {
	rows := make([]ComparisonRow, 0, len(candidates))
	for _, c := range candidates {
		maxSize := 0
		for _, s := range c.Evaluation.Sizes {
			if s > maxSize {
				maxSize = s
			}
		}
		rows = append(rows, ComparisonRow{
			ID: c.ID, Status: c.Status,
			CrossWriteCount: c.Evaluation.CrossWriteCount, CrossWriteWeight: c.Evaluation.CrossWriteWeight,
			CrossReadCount: c.Evaluation.CrossReadCount, Cohesion: c.Evaluation.Cohesion,
			MaxSize: maxSize, FunctionCount: len(c.Assignments),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CrossWriteCount != rows[j].CrossWriteCount {
			return rows[i].CrossWriteCount < rows[j].CrossWriteCount
		}
		if rows[i].CrossWriteWeight != rows[j].CrossWriteWeight {
			return rows[i].CrossWriteWeight < rows[j].CrossWriteWeight
		}
		if rows[i].CrossReadCount != rows[j].CrossReadCount {
			return rows[i].CrossReadCount < rows[j].CrossReadCount
		}
		if rows[i].Cohesion != rows[j].Cohesion {
			return rows[i].Cohesion > rows[j].Cohesion
		}
		if rows[i].FunctionCount != rows[j].FunctionCount {
			return rows[i].FunctionCount < rows[j].FunctionCount
		}
		return rows[i].ID < rows[j].ID
	})
	return ComparisonReport{
		SchemaVersion:  "tei.subsystem-partition.comparison/v0",
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		BaseCommit:     "",
		AuthorityLimit: AuthorityLimit,
		Rows:           rows,
	}, nil
}

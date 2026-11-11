package partition

import (
	"sort"
	"strings"
	"time"

	"tei/internal/threeelement"
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

// PartitionSnapshot is one side of a before/after partition comparison.
// It carries only placement, access edges and provenance; it is never a Source.
type PartitionSnapshot struct {
	Label       string                    `json:"label"`
	Assignments []Assignment              `json:"assignments"`
	Writes      []Write                   `json:"writes"`
	ReadEdges   []ReadEdge                `json:"readEdges,omitempty"`
	Coverage    map[string]ProvenanceKind `json:"coverage,omitempty"` // subject id -> provenance
}

// WriteReclassification counts write occurrences by operation, split into
// internal (same subsystem as the declared owner) and cross-boundary.
type WriteReclassification struct {
	Internal map[string]int `json:"internal"`
	Cross    map[string]int `json:"cross"`
}

// PartitionComparison is the deterministic before/after comparison table.
// A "moved" function is a placement change only; it is never a Source definition change.
type PartitionComparison struct {
	SchemaVersion        string                `json:"schemaVersion"`
	GeneratedAt          string                `json:"generatedAt"`
	BaseCommit           string                `json:"baseCommit"`
	AuthorityLimit       string                `json:"authorityLimit"`
	BeforeLabel          string                `json:"beforeLabel"`
	AfterLabel           string                `json:"afterLabel"`
	MovedFunctions       []string              `json:"movedFunctions"`
	MovedFunctionCount   int                   `json:"movedFunctionCount"`
	AddedFunctions       []string              `json:"addedFunctions,omitempty"`
	RemovedFunctions     []string              `json:"removedFunctions,omitempty"`
	AddedData            []string              `json:"addedData,omitempty"`
	RemovedData          []string              `json:"removedData,omitempty"`
	BeforeCrossWrites    int                   `json:"beforeCrossWrites"`
	AfterCrossWrites     int                   `json:"afterCrossWrites"`
	BoundaryDelta        int                   `json:"boundaryDelta"`
	Reclassification     WriteReclassification `json:"reclassification"`
	UnresolvedCount      int                   `json:"unresolvedCount"`
	NotExaminedCount     int                   `json:"notExaminedCount"`
	DiffModelsConsistent bool                  `json:"diffModelsConsistent"`
	Notes                []string              `json:"notes,omitempty"`
}

// ComparePartitions deterministically compares two partition snapshots.
//
// The comparison reports placement movement only. Function/Data set differences are
// cross-checked against threeelement.DiffModels so a placement change is never
// mistaken for a Source definition change.
func ComparePartitions(before, after PartitionSnapshot) (PartitionComparison, error) {
	beforeAssign := assignmentMap(before.Assignments)
	afterAssign := assignmentMap(after.Assignments)

	moved := make([]string, 0)
	for fn, sub := range afterAssign {
		if prev, ok := beforeAssign[fn]; ok && prev != sub {
			moved = append(moved, fn)
		}
	}
	sort.Strings(moved)

	addedFn := sortedDifference(afterAssign, beforeAssign)
	removedFn := sortedDifference(beforeAssign, afterAssign)

	beforeData := dataIDSet(before.Writes, before.ReadEdges)
	afterData := dataIDSet(after.Writes, after.ReadEdges)
	addedData := sortedDifferenceSet(afterData, beforeData)
	removedData := sortedDifferenceSet(beforeData, afterData)

	reclass := WriteReclassification{Internal: map[string]int{}, Cross: map[string]int{}}
	for _, w := range after.Writes {
		op := strings.TrimSpace(w.Operation)
		if op == "" {
			op = "unknown"
		}
		if sub, ok := afterAssign[w.Function]; ok && sub == strings.TrimSpace(w.Owner) {
			reclass.Internal[op]++
		} else {
			reclass.Cross[op]++
		}
	}
	beforeCross := countCrossWrites(before.Writes, beforeAssign)
	afterCross := countCrossWrites(after.Writes, afterAssign)

	unresolved, notExamined := countCoverageGaps(after.Coverage)

	consistent := diffModelsConsistent(beforeAssign, afterAssign, beforeData, afterData)

	notes := make([]string, 0)
	notes = append(notes, "moved-functions-are-placement-changes-not-source-definition-changes")
	if !consistent {
		notes = append(notes, "function-or-data-set-differs-between-snapshots")
	}

	return PartitionComparison{
		SchemaVersion:        "tei.subsystem-partition.comparison-partition/v0",
		GeneratedAt:          time.Now().UTC().Format(time.RFC3339),
		BaseCommit:           "",
		AuthorityLimit:       AuthorityLimit,
		BeforeLabel:          before.Label,
		AfterLabel:           after.Label,
		MovedFunctions:       moved,
		MovedFunctionCount:   len(moved),
		AddedFunctions:       addedFn,
		RemovedFunctions:     removedFn,
		AddedData:            addedData,
		RemovedData:          removedData,
		BeforeCrossWrites:    beforeCross,
		AfterCrossWrites:     afterCross,
		BoundaryDelta:        afterCross - beforeCross,
		Reclassification:     reclass,
		UnresolvedCount:      unresolved,
		NotExaminedCount:     notExamined,
		DiffModelsConsistent: consistent,
		Notes:                notes,
	}, nil
}

func assignmentMap(assignments []Assignment) map[string]string {
	out := make(map[string]string, len(assignments))
	for _, a := range assignments {
		out[a.Function] = a.Subsystem
	}
	return out
}

func sortedDifference(a, b map[string]string) []string {
	out := make([]string, 0)
	for k := range a {
		if _, ok := b[k]; !ok {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func sortedDifferenceSet(a, b map[string]bool) []string {
	out := make([]string, 0)
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func dataIDSet(writes []Write, reads []ReadEdge) map[string]bool {
	out := map[string]bool{}
	for _, w := range writes {
		out[w.Data] = true
	}
	for _, r := range reads {
		out[r.Data] = true
	}
	return out
}

func countCrossWrites(writes []Write, assign map[string]string) int {
	cross := 0
	for _, w := range writes {
		if sub, ok := assign[w.Function]; !ok || sub != strings.TrimSpace(w.Owner) {
			cross++
		}
	}
	return cross
}

func countCoverageGaps(coverage map[string]ProvenanceKind) (unresolved int, notExamined int) {
	for _, p := range coverage {
		switch p {
		case ProvUnresolved:
			unresolved++
		case ProvNotExamined:
			notExamined++
		}
	}
	return unresolved, notExamined
}

// diffModelsConsistent builds minimal three-element models for both snapshots and
// verifies that the Function/Data sets are unchanged, so that any reported
// movement is placement-only.
func diffModelsConsistent(beforeAssign, afterAssign map[string]string, beforeData, afterData map[string]bool) bool {
	beforeFn := make([]string, 0, len(beforeAssign))
	for fn := range beforeAssign {
		beforeFn = append(beforeFn, fn)
	}
	sort.Strings(beforeFn)
	afterFn := make([]string, 0, len(afterAssign))
	for fn := range afterAssign {
		afterFn = append(afterFn, fn)
	}
	sort.Strings(afterFn)
	beforeDataList := sortedSet(beforeData)
	afterDataList := sortedSet(afterData)

	beforeModel := minimalModel("partition-before", beforeFn, beforeDataList)
	afterModel := minimalModel("partition-after", afterFn, afterDataList)
	diff := threeelement.DiffModels(beforeModel, afterModel)
	if len(diff.Added) != 0 || len(diff.Removed) != 0 {
		return false
	}
	for _, e := range diff.Changed {
		if e.Category == "data" || e.Category == "function" {
			return false
		}
	}
	return true
}

func minimalModel(id string, functions []string, dataIDs []string) threeelement.ThreeElementModel {
	m := threeelement.NewThreeElementModel(id, id)
	for _, fn := range functions {
		m.FunctionElements = append(m.FunctionElements, threeelement.FunctionElement{ID: fn, Name: fn})
	}
	for _, d := range dataIDs {
		m.DataElements = append(m.DataElements, threeelement.DataElement{ID: d, Name: d})
	}
	return m
}

func sortedSet(in map[string]bool) []string {
	out := make([]string, 0, len(in))
	for k := range in {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

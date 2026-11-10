package query

import (
	"tei/internal/core"
	"tei/internal/semantic"
)

// RelationQuery represents a query that traverses semantic relations
// to fetch related data without requiring explicit SQL JOINs.
type RelationQuery struct {
	ID            core.CanonicalID      `json:"id"`
	Application   core.CanonicalID      `json:"application"`
	Datasource    string                `json:"datasource"`
	Root          core.CanonicalID      `json:"root"`      // root data ID
	Relations     []RelationStep        `json:"relations"` // ordered traversal
	Filters       []Filter              `json:"filters,omitempty"`
	Sorts         []Sort                `json:"sorts,omitempty"`
	PageSize      int                   `json:"pageSize,omitempty"`
	PageOffset    int                   `json:"pageOffset,omitempty"`
	Depth         int                   `json:"depth,omitempty"` // max traversal depth
	Mode          RelationQueryMode     `json:"mode,omitempty"`  // tree, graph, path
	FetchMode     RelationFetchMode     `json:"fetchMode,omitempty"`
	BatchSize     int                   `json:"batchSize,omitempty"`
	ResultColumns []ResultColumnMapping `json:"resultColumns"`
}

// Filter represents a WHERE clause expression.
type Filter struct {
	Expression string `json:"expression"`
	// Temporal specifies optional validity-time bounds for temporal relations.
	Temporal *TemporalFilter `json:"temporal,omitempty"`
}

// TemporalFilter represents a temporal condition on a relation traversal.
type TemporalFilter struct {
	// ValidAt is a point-in-time predicate (e.g., AS OF).
	ValidAt string `json:"validAt,omitempty"`
	// Between specifies a closed interval predicate.
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

// Sort represents an ORDER BY expression.
type Sort struct {
	Expression string `json:"expression"`
}

type RelationQueryMode string

const (
	RelationQueryModeTree  RelationQueryMode = "tree"
	RelationQueryModeGraph RelationQueryMode = "graph"
	RelationQueryModePath  RelationQueryMode = "path"
)

// RelationQueryModes lists all valid modes.
var RelationQueryModes = []RelationQueryMode{
	RelationQueryModeTree,
	RelationQueryModeGraph,
	RelationQueryModePath,
}

func (m RelationQueryMode) Valid() bool {
	for _, valid := range RelationQueryModes {
		if m == valid {
			return true
		}
	}
	return false
}

type RelationFetchMode string

const (
	RelationFetchModeEager RelationFetchMode = "eager"
	RelationFetchModeLazy  RelationFetchMode = "lazy"
	RelationFetchModeBatch RelationFetchMode = "batch"
)

var RelationFetchModes = []RelationFetchMode{
	RelationFetchModeEager,
	RelationFetchModeLazy,
	RelationFetchModeBatch,
}

func (m RelationFetchMode) Valid() bool {
	for _, valid := range RelationFetchModes {
		if m == valid {
			return true
		}
	}
	return false
}

// RelationStep represents one traversal step in a relation query.
type RelationStep struct {
	Relation    core.CanonicalID `json:"relation"`
	Direction   string           `json:"direction,omitempty"` // outgoing, incoming, both
	Alias       string           `json:"alias,omitempty"`
	Cardinality string           `json:"cardinality,omitempty"` // one, many
	JoinKind    string           `json:"joinKind,omitempty"`    // inner, left, right
}

// ResolvedRelationQuery resolves a RelationQuery against the semantic model.
type ResolvedRelationQuery struct {
	Query      RelationQuery
	RootData   semantic.Data
	Steps      []ResolvedRelationStep
	Columns    []ResolvedColumn
	Parameters []ResolvedParameter
	SQL        string
}

// ResolvedRelationStep resolves a single relation step.
type ResolvedRelationStep struct {
	Step      RelationStep
	Relation  semantic.Relation
	Source    semantic.Data
	Target    semantic.Data
	JoinTable string
	JoinAlias string
}

// ResolvedColumn represents a resolved output column.
type ResolvedColumn struct {
	Column      string
	Attribute   core.CanonicalID
	DataType    core.CanonicalID
	SourceTable string
	SourceAttr  string
}

// ResolvedParameter represents a query parameter.
type ResolvedParameter struct {
	Name      string
	Attribute core.CanonicalID
	DataType  core.CanonicalID
	Required  bool
}

package analyzer

import (
	"tei/internal/core"
	"tei/internal/environment"
)

type SchemaTable struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns,omitempty"`
}

type SchemaContext struct {
	Tables []SchemaTable `json:"tables,omitempty"`
}

type Request struct {
	ProtocolVersion int                          `json:"protocolVersion"`
	Operation       string                       `json:"operation"`
	Query           core.CanonicalID             `json:"query"`
	Dialect         environment.DatabaseProvider `json:"dialect"`
	Source          string                       `json:"source"`
	Schema          *SchemaContext               `json:"schema,omitempty"`
}

type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Scope is a normalized SQL query scope. It deliberately does not mirror a
// SQLGlot AST node: the Go side only needs lexical visibility and hierarchy.
type Scope struct {
	ID       string `json:"id"`
	ParentID string `json:"parentId,omitempty"`
	Kind     string `json:"kind"` // root, cte, subquery
	Name     string `json:"name,omitempty"`
}

// Operation is a small logical relational operation emitted by the analyzer.
// ScopeID keeps same-named sources in nested queries distinct.
type Operation struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	ScopeID     string   `json:"scopeId,omitempty"`
	Inputs      []string `json:"inputs,omitempty"`
	Source      string   `json:"source,omitempty"`
	SourceKind  string   `json:"sourceKind,omitempty"` // table, cte, subquery
	Binding     string   `json:"binding,omitempty"`    // cte/subquery binding target
	Alias       string   `json:"alias,omitempty"`
	JoinKind    string   `json:"joinKind,omitempty"`
	Expressions []string `json:"expressions,omitempty"`
}

// ColumnOccurrence represents a column reference in one lexical SQL scope.
// Correlation is explicit so TEI never has to infer it from flattened names.
type ColumnOccurrence struct {
	ID           string `json:"id"`
	ScopeID      string `json:"scopeId"`
	Name         string `json:"name"`
	Qualifier    string `json:"qualifier,omitempty"`
	Correlated   bool   `json:"correlated,omitempty"`
	OuterScopeID string `json:"outerScopeId,omitempty"`
}

// OutputField is an authored SELECT output. Inputs contain ColumnOccurrence
// IDs and form the first stable source-level lineage graph.
type OutputField struct {
	ID         string   `json:"id"`
	ScopeID    string   `json:"scopeId"`
	Name       string   `json:"name,omitempty"`
	Expression string   `json:"expression,omitempty"`
	Inputs     []string `json:"inputs,omitempty"`
}

type Result struct {
	ProtocolVersion int                `json:"protocolVersion"`
	Status          string             `json:"status"`
	Query           core.CanonicalID   `json:"query"`
	Tables          []string           `json:"tables,omitempty"`
	Columns         []string           `json:"columns,omitempty"`
	TableAliases    map[string]string  `json:"tableAliases,omitempty"`
	Scopes          []Scope            `json:"scopes,omitempty"`
	ColumnRefs      []ColumnOccurrence `json:"columnRefs,omitempty"`
	OutputFields    []OutputField      `json:"outputFields,omitempty"`
	Operations      []Operation        `json:"operations,omitempty"`
	Lowering        string             `json:"lowering,omitempty"`
	Diagnostics     []Diagnostic       `json:"diagnostics,omitempty"`
}

func (r Result) OK() bool { return r.Status == "ok" }

package studiohost

import (
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
	"tei/internal/query"
)

// RowSourceResult is what a Studio caller hands over: the query that produced the
// rows, the columns they were read with, and the values.
//
// It is declared here rather than taking the driver's mapped result so the Studio
// host keeps its ownership boundary: this package holds no executor and no
// connection, and the conversion stays in internal/hook where the declaration is
// read.
type RowSourceResult struct {
	Query   core.CanonicalID
	Columns []hook.ColumnName
	Rows    [][]any
}

// RunHooksForRowSource is the Studio host's entry for a row-scoped hook whose
// declaration names its source.
//
// The row set comes from the declaration, not from the caller: a page produced by
// a query the hook did not name is refused by internal/hook before any rule runs,
// so a Studio caller cannot decide which rows apply by passing whichever page it
// has. The rule is supplied by the caller, as the existing row entry already
// requires, because the reference host carries no business rule of its own.
func (h *ReadOnly) RunHooksForRowSource(world ir.World, applicationID, patternID core.CanonicalID, result RowSourceResult, rule hook.Implementation) (*HookRunReport, error) {
	rows, err := hook.RowsForDeclaredSourceStrings(world, applicationID, patternID, result.Query, result.Columns, result.Rows)
	if err != nil {
		return nil, err
	}
	return h.runHooksWithRows(world, applicationID, patternID, nil, rows, rule)
}

// NewRowSourceResult adapts an executed application query into the form this entry
// takes, naming the query that was executed so the declaration check compares it
// with what the hook wrote down. It invents no column mapping: the query's own
// declared result mapping is passed through.
func NewRowSourceResult(queryID core.CanonicalID, columns []query.ResultColumnMapping, rows [][]any) RowSourceResult {
	names := make([]hook.ColumnName, len(columns))
	for i, column := range columns {
		names[i] = hook.ColumnName{Raw: column.Column, Attribute: column.Attribute}
	}
	return RowSourceResult{Query: queryID, Columns: names, Rows: rows}
}

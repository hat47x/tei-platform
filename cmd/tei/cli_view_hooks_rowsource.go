package main

import (
	"context"

	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
	"tei/internal/query"
)

// rowSourceResult is the part of a query execution this host needs: which query
// produced the rows, the columns those rows were read with, and the values.
//
// It is declared here rather than taking the driver's mapped result so the CLI
// keeps its existing boundary: the driver owns query execution, and this entry
// holds no executor, no connection and no authority to read a database. A caller
// that does hold a mapped result passes the three fields it already has.
type rowSourceResult struct {
	Query   core.CanonicalID
	Columns []hook.ColumnName
	Rows    [][]any
}

// runDeclaredHooksForRowSource is the host entry that supplies the row set a
// row-scoped hook declared.
//
// The declaration names its source; this entry only converts the result the
// caller already executed. The conversion and the declaration check live in
// internal/hook (RowsForDeclaredSource), so the CLI and Studio cannot drift into
// two answers about which rows apply to which declaration, and the hook boundary
// keeps its direction: it still does not import the driver.
//
// A declared source that produced no rows is not an error: an empty page is a
// real browse result, and refusing it would turn "nothing matched" into a
// failure. The hook phases then run with the empty row set, which the runner
// reports as executed with no row visited — the same answer it gives for an
// empty page it did receive.
func runDeclaredHooksForRowSource(ctx context.Context, world ir.World, applicationID, patternID core.CanonicalID, result rowSourceResult, rule hook.Implementation) (*cliHookReport, error) {
	rows, err := hook.RowsForDeclaredSourceStrings(world, applicationID, patternID, result.Query, result.Columns, result.Rows)
	if err != nil {
		return nil, err
	}
	return runDeclaredHooksWithRows(ctx, world, applicationID, patternID, nil, rows, rule)
}

// queryResultRowSource adapts an executed application query into the form the
// hook entry takes, naming the query that was executed so the declaration check
// can compare it with what the hook wrote down.
func queryResultRowSource(queryID core.CanonicalID, columns []query.ResultColumnMapping, rows [][]any) rowSourceResult {
	names := make([]hook.ColumnName, len(columns))
	for i, column := range columns {
		names[i] = hook.ColumnName{Raw: column.Column, Attribute: column.Attribute}
	}
	return rowSourceResult{Query: queryID, Columns: names, Rows: rows}
}

package hook

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
)

// RowsForDeclaredSource converts the result of the query a row-scoped hook names
// as its row source into the row set Runner.Rows takes.
//
// The declaration says where the rows come from; the host says what came back.
// The two are compared here instead of being assumed to agree, so a page produced
// by a different query cannot be applied to a hook that declared another one. The
// conversion itself stays in RowsFromColumns: this entry adds the declaration
// check and the "every row can be decided" check, not a second mapping.
//
// It is placed here rather than in a host because both hosts would otherwise
// repeat the same checks — which source is declared, whether the result is that
// source, whether the declared reads are carried — and the hook boundary is the
// only place that owns the declaration. It still does not import the driver: the
// caller converts its own mapped result into ColumnRow, the same type the
// previous row conversion already uses.
func RowsForDeclaredSource(w ir.World, applicationID, patternID core.CanonicalID, queryID core.CanonicalID, rows []ColumnRow) ([]map[core.CanonicalID]string, error) {
	runner := Runner{World: w}
	pattern, declarations, err := rowScopedDeclarations(runner, applicationID, patternID)
	if err != nil {
		return nil, err
	}
	if len(declarations) == 0 {
		return nil, &Error{
			Code:    "HOOK_ROW_SOURCE_NOT_ROW_SCOPED",
			Message: fmt.Sprintf("pattern %s declares no row-scoped hook, so a row source has nothing to apply to", pattern.ID),
		}
	}
	for _, declaration := range declarations {
		if declaration.Rows == nil {
			// A row scope without a source is a declaration whose supplier nobody
			// named. Refusing keeps "no source" from reading as "whatever rows the
			// caller happened to have".
			return nil, &Error{
				Code:    "HOOK_ROW_SOURCE_UNDECLARED",
				Message: fmt.Sprintf("hook %s declares scope row but no row source, so rows cannot be attributed to it", declaration.Phase),
			}
		}
		if declaration.Rows.Query != queryID {
			return nil, &Error{
				Code:    "HOOK_ROW_SOURCE_MISMATCH",
				Message: fmt.Sprintf("hook %s declares row source query %s, but the result is from query %s", declaration.Phase, declaration.Rows.Query, queryID),
			}
		}
	}
	converted, err := RowsFromColumns(rows)
	if err != nil {
		return nil, &Error{
			Code:    "HOOK_ROW_SOURCE_CONVERSION_FAILED",
			Message: fmt.Sprintf("row source query %s returned rows that cannot be used: %v", queryID, err),
			Cause:   err,
		}
	}
	// A row whose declared read has no value cannot be decided. The row type cannot
	// tell "read and empty" from "not read", so a missing key is reported here
	// rather than left for the rule to guess at, and the row is named.
	for index, row := range converted {
		for _, declaration := range declarations {
			for _, attribute := range declaration.Reads {
				if _, ok := row[attribute]; ok {
					continue
				}
				return nil, &Error{
					Code:    "HOOK_ROW_SOURCE_READ_MISSING",
					Message: fmt.Sprintf("row %d of query %s does not carry %s that hook %s declares to read", index, queryID, attribute, declaration.Phase),
				}
			}
		}
	}
	return converted, nil
}

// RowsForDeclaredSourceStrings is RowsForDeclaredSource for a host that already
// holds column names and values separately, which is how a mapped result arrives:
// the columns are shared by every row and the values are one slice per row.
func RowsForDeclaredSourceStrings(w ir.World, applicationID, patternID core.CanonicalID, queryID core.CanonicalID, columns []ColumnName, values [][]any) ([]map[core.CanonicalID]string, error) {
	rows := make([]ColumnRow, len(values))
	for index, row := range values {
		if len(row) != len(columns) {
			return nil, &Error{
				Code:    "HOOK_ROW_SOURCE_CONVERSION_FAILED",
				Message: fmt.Sprintf("row %d of query %s has %d columns but %d values", index, queryID, len(columns), len(row)),
			}
		}
		rows[index] = ColumnRow{Columns: columns, Values: row}
	}
	return RowsForDeclaredSource(w, applicationID, patternID, queryID, rows)
}

// rowScopedDeclarations resolves the pattern and returns its row-scoped hooks.
func rowScopedDeclarations(r Runner, applicationID, patternID core.CanonicalID) (*application.Pattern, []*application.Hook, error) {
	pattern := r.findPattern(applicationID, patternID)
	if pattern == nil {
		return nil, nil, &Error{Code: "HOOK_PATTERN_NOT_FOUND", Message: fmt.Sprintf("pattern %s is not defined in application %s", patternID, applicationID)}
	}
	if pattern.Interaction == nil {
		return nil, nil, &Error{Code: "HOOK_PATTERN_NOT_INTERACTIVE", Message: fmt.Sprintf("pattern %s has no interaction to declare hooks", patternID)}
	}
	var out []*application.Hook
	for i := range pattern.Interaction.Hooks {
		if pattern.Interaction.Hooks[i].Scope != "row" {
			continue
		}
		out = append(out, &pattern.Interaction.Hooks[i])
	}
	return pattern, out, nil
}

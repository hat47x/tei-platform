package hook

import (
	"fmt"

	"tei/internal/core"
)

// ColumnRow is one row of a query result together with the columns it was read
// with, in the same order.
//
// It is declared here rather than taking the driver's result type so the hook
// boundary keeps its direction: the hook does not import the driver. A host that
// holds a mapped result converts it with RowsFromColumns and hands the rows to
// RunRows.
type ColumnRow struct {
	Columns []ColumnName
	Values  []any
}

// ColumnName is one column of a query result: the database column and the
// semantic attribute it is mapped to.
type ColumnName struct {
	Raw       string
	Attribute core.CanonicalID
}

// RowsFromColumns converts a mapped query result into the row set a row-scoped
// hook is applied to.
//
// The mapping is explicit — a column becomes an attribute only because the result
// mapping named it — so this function does not guess a name from another.
//
// A nil value becomes the empty string, because the row type cannot tell "read and
// empty" from "not read" (see plan/evidence/XTEA_LOWCODE_ROW_SCOPE_VALUE_STATE):
// the attribute is present, so it counts as read, and the hook sees an empty value
// rather than a missing one. A column with no mapped attribute is not carried: it
// has no semantic name for a declaration to read.
func RowsFromColumns(rows []ColumnRow) ([]map[core.CanonicalID]string, error) {
	out := make([]map[core.CanonicalID]string, 0, len(rows))
	for index, row := range rows {
		if len(row.Columns) != len(row.Values) {
			return nil, fmt.Errorf("row %d has %d columns but %d values", index, len(row.Columns), len(row.Values))
		}
		values := make(map[core.CanonicalID]string, len(row.Columns))
		for column := range row.Columns {
			attribute := row.Columns[column].Attribute
			if attribute == "" {
				// No semantic name: a declaration cannot read it, and inventing one
				// from the raw column name would make the mapping implicit.
				continue
			}
			values[attribute] = valueString(row.Values[column])
		}
		out = append(out, values)
	}
	return out, nil
}

// valueString renders one runtime value the way a declared read is compared.
// A nil value is the empty string; anything else keeps its printed form.
func valueString(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprintf("%v", value)
}

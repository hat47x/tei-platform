package orderdatarealization

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"

	"tei/internal/database/persistence"
	"tei/internal/ir"
)

// rowWriter builds INSERT statements from one version's storage mapping. The
// old-version writer and the current adapter use the same builder; they differ
// only in the mapping version they were built from.
type rowWriter struct {
	order, line persistence.Mapping
}

func rowWriterOf(world ir.World) (rowWriter, error) {
	var w rowWriter
	var found int
	for _, m := range world.Persistence {
		switch m.ID {
		case "sales_core.order":
			w.order, found = m, found+1
		case "sales_core.order_line":
			w.line, found = m, found+1
		}
	}
	if found != 2 {
		return w, fmt.Errorf("storage mappings sales_core.order and sales_core.order_line not found")
	}
	return w, nil
}

func (w rowWriter) orderColumns() []string {
	cols := make([]string, len(w.order.Columns))
	for i, c := range w.order.Columns {
		cols[i] = c.Column
	}
	return cols
}

// insert renders one INSERT for a mapping. values are keyed by attribute ID;
// every mapped attribute must be present (nil means NULL). support values are
// keyed by relation-storage source attribute.
func insertFor(m persistence.Mapping, values map[string]any, support map[string]any) (string, []any, error) {
	var cols, marks []string
	var args []any
	for _, c := range m.Columns {
		v, ok := values[string(c.Attribute)]
		if !ok {
			return "", nil, fmt.Errorf("%s: no value for %s", m.ID, c.Attribute)
		}
		if v == nil && c.Nullability == persistence.Required {
			return "", nil, fmt.Errorf("%s: %s is required", m.ID, c.Attribute)
		}
		cols = append(cols, quoteIdent(c.Column))
		args = append(args, v)
		marks = append(marks, fmt.Sprintf("$%d", len(args)))
	}
	for k := range values {
		if !mappingHas(m, k) {
			return "", nil, fmt.Errorf("%s: attribute %s is not mapped in this version", m.ID, k)
		}
	}
	for _, rs := range m.RelationStorage {
		for _, comp := range rs.Components {
			v, ok := support[string(comp.SourceAttribute)]
			if !ok {
				return "", nil, fmt.Errorf("%s: no relation storage value for %s", m.ID, comp.SourceAttribute)
			}
			cols = append(cols, quoteIdent(comp.Column))
			args = append(args, v)
			marks = append(marks, fmt.Sprintf("$%d", len(args)))
		}
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", qualifiedTable(m.Table), strings.Join(cols, ", "), strings.Join(marks, ", ")), args, nil
}

func mappingHas(m persistence.Mapping, attr string) bool {
	for _, c := range m.Columns {
		if string(c.Attribute) == attr {
			return true
		}
	}
	return false
}

// rowsDigest hashes the given columns of the given orders, ordered by order_id.
func rowsDigest(ctx context.Context, db *sql.DB, cols []string, ids []int64) (string, int, error) {
	if len(ids) == 0 {
		return "", 0, fmt.Errorf("no rows to digest")
	}
	exprs := make([]string, len(cols))
	for i, c := range cols {
		exprs[i] = "coalesce(" + quoteIdent(c) + "::text, '<null>')"
	}
	idList := make([]string, len(ids))
	for i, id := range ids {
		idList[i] = fmt.Sprint(id)
	}
	q := fmt.Sprintf(`SELECT %s FROM public.orders WHERE order_id IN (%s) ORDER BY order_id`, strings.Join(exprs, " || chr(31) || "), strings.Join(idList, ","))
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return "", 0, err
	}
	defer rows.Close()
	h := sha256.New()
	n := 0
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return "", 0, err
		}
		fmt.Fprintf(h, "%d:%s\n", len(s), s)
		n++
	}
	if err := rows.Err(); err != nil {
		return "", 0, err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), n, nil
}

func countRows(ctx context.Context, db *sql.DB, query string, args ...any) (int, error) {
	var n int
	err := db.QueryRowContext(ctx, query, args...).Scan(&n)
	return n, err
}

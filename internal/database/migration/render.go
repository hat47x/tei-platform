package migration

import (
	"fmt"
	"strings"

	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

func Render(provider environment.DatabaseProvider, diff schemadiff.Diff) ([]string, error) {
	switch provider {
	case environment.ProviderPostgres:
		return renderPostgres(diff)
	default:
		return nil, fmt.Errorf("migration SQL rendering is not implemented for provider %q", provider)
	}
}

func renderPostgres(diff schemadiff.Diff) ([]string, error) {
	var out []string
	created := map[string]bool{}
	for _, c := range diff.Changes {
		switch c.Kind {
		case schemadiff.CreateTable:
			created[c.Table] = true
			// Columns are emitted by following AddColumn changes to retain one common change model.
			out = append(out, fmt.Sprintf("CREATE TABLE %s ();", qname(c.Table)))
		case schemadiff.DropTable:
			out = append(out, fmt.Sprintf("DROP TABLE %s;", qname(c.Table)))
		case schemadiff.AddColumn:
			if c.Column == nil {
				return nil, fmt.Errorf("add-column %s has no column", c.Table)
			}
			stmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", qname(c.Table), qid(c.Column.Name), c.Column.PhysicalType)
			if !c.Column.Nullable {
				stmt += " NOT NULL"
			}
			if strings.TrimSpace(c.Column.Default) != "" {
				stmt += " DEFAULT " + c.Column.Default
			}
			out = append(out, stmt+";")
		case schemadiff.DropColumn:
			if c.BeforeColumn == nil {
				return nil, fmt.Errorf("drop-column %s has no before column", c.Table)
			}
			out = append(out, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", qname(c.Table), qid(c.BeforeColumn.Name)))
		case schemadiff.AlterColumn:
			if c.Column == nil {
				return nil, fmt.Errorf("alter-column %s has no target column", c.Table)
			}
			out = append(out, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;", qname(c.Table), qid(c.Column.Name), c.Column.PhysicalType))
			if c.Column.Nullable {
				out = append(out, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;", qname(c.Table), qid(c.Column.Name)))
			} else {
				out = append(out, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;", qname(c.Table), qid(c.Column.Name)))
			}
		case schemadiff.ChangePrimaryKey:
			// Constraint names are not yet canonicalized in Physical Schema IR; emit a marked manual step rather than guessing a destructive constraint name.
			cols := joinIDs(c.PrimaryKey)
			out = append(out, fmt.Sprintf("-- TEI: primary key change required on %s to (%s); constraint-name resolution required", qname(c.Table), cols))
		case schemadiff.AddIndex:
			if c.Index == nil {
				return nil, fmt.Errorf("add-index %s has no index", c.Table)
			}
			columns, err := simpleIndexColumns(c.Index)
			if err != nil {
				return nil, fmt.Errorf("add-index %s.%s: %w", c.Table, c.Index.Name, err)
			}
			unique := ""
			if c.Index.Unique {
				unique = "UNIQUE "
			}
			out = append(out, fmt.Sprintf("CREATE %sINDEX %s ON %s%s (%s);", unique, qid(c.Index.Name), qname(c.Table), indexMethod(c.Index), joinIDs(columns)))
		case schemadiff.DropIndex:
			if c.Index == nil {
				return nil, fmt.Errorf("drop-index %s has no index", c.Table)
			}
			out = append(out, fmt.Sprintf("DROP INDEX %s;", qid(c.Index.Name)))
		}
	}
	_ = created
	return out, nil
}

func qid(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }
func qname(s string) string {
	parts := strings.Split(s, ".")
	for i := range parts {
		parts[i] = qid(parts[i])
	}
	return strings.Join(parts, ".")
}
func joinIDs(xs []string) string {
	ys := make([]string, len(xs))
	for i, x := range xs {
		ys[i] = qid(x)
	}
	return strings.Join(ys, ", ")
}

func simpleIndexColumns(index *physical.Index) ([]string, error) {
	if len(index.IncludedColumns) != 0 || index.Predicate != nil {
		return nil, fmt.Errorf("advanced index rendering is not implemented")
	}
	if index.UniqueNullTreatment == physical.UniqueNullTreatmentNotDistinct {
		return nil, fmt.Errorf("NULLS NOT DISTINCT index rendering is not implemented")
	}
	columns := make([]string, 0, len(index.Keys))
	for _, key := range index.Keys {
		if key.Expression != nil || strings.TrimSpace(key.Column) == "" {
			return nil, fmt.Errorf("expression index rendering is not implemented")
		}
		if strings.TrimSpace(key.Collation) != "" || strings.TrimSpace(key.OperatorClass) != "" || key.Order != physical.IndexOrderDefault || key.NullsOrder != physical.IndexNullsOrderDefault {
			return nil, fmt.Errorf("advanced index key rendering is not implemented")
		}
		columns = append(columns, key.Column)
	}
	return columns, nil
}

func indexMethod(i *physical.Index) string {
	if strings.TrimSpace(i.Method) == "" || strings.EqualFold(i.Method, "btree") {
		return ""
	}
	return " USING " + i.Method
}

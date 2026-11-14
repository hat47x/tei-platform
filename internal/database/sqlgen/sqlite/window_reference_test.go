package sqlite

import (
	"context"
	"database/sql"
	"testing"

	sqlgen "tei/internal/database/sqlgen"

	_ "modernc.org/sqlite"
)

// CTK-01/03/10: this is a narrow SQLite adapter and database execution test.
// It does not establish TEI Query lowering, arbitrary Provider equivalence,
// an accepted business meaning, or a Studio/native Consumer result.
func TestCTKWindowSQLiteGeneratedSQL(t *testing.T) {
	db := openCTKSQLite(t)
	ctx := context.Background()
	compiled, err := NewDialect().Select(ctx, ctkWindowSelect())
	if err != nil {
		t.Fatalf("generate SQL: %v", err)
	}
	if len(compiled.Args) != 0 {
		t.Fatalf("unexpected binding arguments: %#v", compiled.Args)
	}

	rows, err := db.QueryContext(ctx, compiled.Statement, compiled.Args...)
	if err != nil {
		t.Fatalf("execute generated SQLite statement: %v\n%s", err, compiled.Statement)
	}
	defer rows.Close()

	expected := []struct {
		group, id     string
		sequence      int64
		value         int64
		valueIsPresent bool
		moving, runningRows, runningRange float64
	}{
		{"A", "A1", 1, 10, true, 10, 10, 10},
		{"A", "A2", 2, 20, true, 15, 15, 15},
		{"A", "A3", 3, 30, true, 20, 20, 20},
		{"A", "A4", 4, 40, true, 30, 25, 25},
		{"B", "B1", 1, 5, true, 5, 5, 5},
		{"B", "B2", 2, 0, false, 5, 5, 10},
		{"B", "B3", 2, 15, true, 10, 10, 10},
	}

	i := 0
	for rows.Next() {
		if i >= len(expected) {
			t.Fatalf("more than %d rows in generated SQL output", len(expected))
		}
		var group, id string
		var seq int64
		var value sql.NullInt64
		var moving, runningRows, runningRange float64
		if err := rows.Scan(&group, &id, &seq, &value, &moving, &runningRows, &runningRange); err != nil {
			t.Fatalf("scan row %d: %v", i, err)
		}
		want := expected[i]
		if group != want.group || id != want.id || seq != want.sequence {
			t.Errorf("row %d identity = (%s, %s, %d), want (%s, %s, %d)", i, group, id, seq, want.group, want.id, want.sequence)
		}
		if value.Valid != want.valueIsPresent || (value.Valid && value.Int64 != want.value) {
			t.Errorf("row %s value = %+v, want valid=%v value=%d", id, value, want.valueIsPresent, want.value)
		}
		if moving != want.moving || runningRows != want.runningRows || runningRange != want.runningRange {
			t.Errorf("row %s averages = (%v, %v, %v), want (%v, %v, %v)", id, moving, runningRows, runningRange, want.moving, want.runningRows, want.runningRange)
		}
		i++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read generated SQL rows: %v", err)
	}
	if i != len(expected) {
		t.Fatalf("got %d rows, want %d", i, len(expected))
	}
}

// A presentation LIMIT runs after the Window calculation, whereas filtering
// the source rows before Window evaluation changes the input closure.
// The same observation A3 must therefore retain 20 or become 25 respectively.
func TestCTKWindowDisplayLimitVsInputFilter(t *testing.T) {
	db := openCTKSQLite(t)
	ctx := context.Background()

	t.Run("presentation limit does not truncate window input", func(t *testing.T) {
		opts := ctkWindowSelect()
		limit := 3
		opts.Limit = &limit
		if got := ctkA3Moving(t, ctx, db, opts); got != 20 {
			t.Errorf("A3 moving average = %v, want 20", got)
		}
	})

	t.Run("pre-window input filter changes average", func(t *testing.T) {
		opts := ctkWindowSelect()
		opts.Where = []sqlgen.Condition{{Column: "obs_id", Operator: "!=", Value: "A1"}}
		if got := ctkA3Moving(t, ctx, db, opts); got != 25 {
			t.Errorf("A3 moving average = %v, want 25", got)
		}
	})
}

func ctkA3Moving(t *testing.T, ctx context.Context, db *sql.DB, opts sqlgen.SelectOptions) float64 {
	t.Helper()
	compiled, err := NewDialect().Select(ctx, opts)
	if err != nil {
		t.Fatalf("generate SQL: %v", err)
	}
	rows, err := db.QueryContext(ctx, compiled.Statement, compiled.Args...)
	if err != nil {
		t.Fatalf("execute generated SQL: %v\n%s", err, compiled.Statement)
	}
	defer rows.Close()

	found := false
	var result float64
	for rows.Next() {
		var group, id string
		var seq int64
		var value sql.NullInt64
		var moving, runningRows, runningRange float64
		if err := rows.Scan(&group, &id, &seq, &value, &moving, &runningRows, &runningRange); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if id == "A3" {
			if found {
				t.Fatal("A3 returned more than once")
			}
			found, result = true, moving
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read rows: %v", err)
	}
	if !found {
		t.Fatal("A3 missing from query result")
	}
	return result
}

func openCTKSQLite(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func ctkWindowSelect() sqlgen.SelectOptions {
	return sqlgen.SelectOptions{
		CTE: `obs(grp, seq, obs_id, value) AS (
			VALUES ('A',1,'A1',10), ('A',2,'A2',20),
			       ('A',3,'A3',30), ('A',4,'A4',40),
			       ('B',1,'B1',5), ('B',2,'B2',NULL),
			       ('B',2,'B3',15)
		)`,
		Table: "obs",
		Columns: []string{
			"grp", "obs_id", "seq", "value",
			"AVG(value) OVER moving AS avg3",
			"AVG(value) OVER cumulative_rows AS avg_rows",
			"AVG(value) OVER cumulative_range AS avg_range",
		},
		Window: `moving AS (
			PARTITION BY grp ORDER BY seq, obs_id
			ROWS BETWEEN 2 PRECEDING AND CURRENT ROW
		), cumulative_rows AS (
			PARTITION BY grp ORDER BY seq, obs_id
			ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
		), cumulative_range AS (
			PARTITION BY grp ORDER BY seq
			RANGE BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
		)`,
		OrderBy: []sqlgen.OrderClause{
			{Column: "grp"}, {Column: "seq"}, {Column: "obs_id"},
		},
	}
}

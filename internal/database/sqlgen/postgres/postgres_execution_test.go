package postgres

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	sqlgen "tei/internal/database/sqlgen"
)

// This is a REAL PostgreSQL execution test for the statements this adapter
// generates (#4415 follow-up). Until it existed, TestPostgresIntegration only
// compared generated strings: nothing showed that PostgreSQL accepts them, so a
// clause in the wrong place looked like a pass. The SQLite execution test caught
// exactly that class of bug for the SQLite adapter.
//
// It is opt-in and needs a disposable database, because it creates and drops
// public.tei_sqlgen_probe. Never point it at a live database.
//
//	CREATE DATABASE tei_sqlgen_e2e;
//	TEI_SQLGEN_PG_DSN='postgres://.../tei_sqlgen_e2e?sslmode=disable' \
//	  TEI_SQLGEN_PG_CONFIRM_ISOLATED=tei_sqlgen_e2e \
//	  go test ./internal/database/sqlgen/postgres -run TestPostgresExecution -v
func TestPostgresExecution(t *testing.T) {
	dsn := os.Getenv("TEI_SQLGEN_PG_DSN")
	if dsn == "" {
		t.Skip("real PostgreSQL evidence unavailable: TEI_SQLGEN_PG_DSN unset")
	}
	if os.Getenv("TEI_SQLGEN_PG_CONFIRM_ISOLATED") != "tei_sqlgen_e2e" {
		t.Fatal("refusing to create public.tei_sqlgen_probe without the explicit disposable database confirmation")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("failed to open the pool: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("failed to reach PostgreSQL: %v", err)
	}

	// The probe table is dropped first so a previous failed run cannot change
	// what this one measures.
	for _, stmt := range []string{
		`DROP TABLE IF EXISTS public.tei_sqlgen_probe`,
		`CREATE TABLE public.tei_sqlgen_probe (
			id bigint PRIMARY KEY,
			customer_id bigint NOT NULL,
			amount numeric(18,2) NOT NULL,
			status varchar(255) NOT NULL,
			created_at timestamptz NOT NULL
		)`,
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("failed to prepare the probe table: %v", err)
		}
	}
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), `DROP TABLE IF EXISTS public.tei_sqlgen_probe`); err != nil {
			t.Logf("failed to drop the probe table: %v", err)
		}
	})

	dialect := NewDialect()

	// INSERT: the generated $1, $2 placeholders have to bind in the order the
	// adapter wrote them.
	insertSQL, err := dialect.Insert(ctx, sqlgen.InsertOptions{
		Table:   "tei_sqlgen_probe",
		Columns: []string{"id", "customer_id", "amount", "status", "created_at"},
		Values: map[string]any{
			"id": int64(1), "customer_id": int64(7), "amount": 100.50,
			"status": "active", "created_at": "2026-11-01T00:00:00Z",
		},
	})
	if err != nil {
		t.Fatalf("Insert error: %v", err)
	}
	if _, err := db.ExecContext(ctx, insertSQL.Statement, insertSQL.Args...); err != nil {
		t.Fatalf("INSERT execution failed: %v (sql=%s)", err, insertSQL.Statement)
	}

	// INSERT with RETURNING: PostgreSQL has to hand the row back.
	returning, err := dialect.Insert(ctx, sqlgen.InsertOptions{
		Table:   "tei_sqlgen_probe",
		Columns: []string{"id", "customer_id", "amount", "status", "created_at"},
		Values: map[string]any{
			"id": int64(2), "customer_id": int64(7), "amount": 50.25,
			"status": "active", "created_at": "2026-11-02T00:00:00Z",
		},
		Returning: []string{"id", "amount"},
	})
	if err != nil {
		t.Fatalf("Insert with RETURNING error: %v", err)
	}
	var returnedID int64
	var returnedAmount string
	if err := db.QueryRowContext(ctx, returning.Statement, returning.Args...).Scan(&returnedID, &returnedAmount); err != nil {
		t.Fatalf("INSERT ... RETURNING failed: %v (sql=%s)", err, returning.Statement)
	}
	if returnedID != 2 {
		t.Errorf("RETURNING id = %d, want 2", returnedID)
	}

	// SELECT with WHERE / ORDER BY / LIMIT, read through the generated statement.
	selectSQL, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:   "tei_sqlgen_probe",
		Columns: []string{"id", "amount"},
		Where:   []sqlgen.Condition{{Column: "customer_id", Operator: "=", Value: int64(7)}},
		OrderBy: []sqlgen.OrderClause{{Column: "id"}},
	})
	if err != nil {
		t.Fatalf("Select error: %v", err)
	}
	rows, err := db.QueryContext(ctx, selectSQL.Statement, selectSQL.Args...)
	if err != nil {
		t.Fatalf("SELECT execution failed: %v (sql=%s)", err, selectSQL.Statement)
	}
	var selected []int64
	for rows.Next() {
		var id int64
		var amount string
		if err := rows.Scan(&id, &amount); err != nil {
			t.Fatalf("SELECT scan failed: %v", err)
		}
		selected = append(selected, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("SELECT rows error: %v", err)
	}
	rows.Close()
	if len(selected) != 2 || selected[0] != 1 || selected[1] != 2 {
		t.Errorf("SELECT ids = %v, want [1 2]", selected)
	}

	// CTE: the WITH clause the adapter emits has to precede the SELECT keyword
	// and the outer query names the CTE, not the base table.
	cteSQL, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:   "recent",
		Columns: []string{"id", "amount"},
		CTE:     "recent AS (SELECT * FROM tei_sqlgen_probe WHERE amount > 60)",
	})
	if err != nil {
		t.Fatalf("CTE Select error: %v", err)
	}
	rows, err = db.QueryContext(ctx, cteSQL.Statement, cteSQL.Args...)
	if err != nil {
		t.Fatalf("CTE execution failed: %v (sql=%s)", err, cteSQL.Statement)
	}
	cteCount := 0
	for rows.Next() {
		var id int64
		var amount string
		if err := rows.Scan(&id, &amount); err != nil {
			t.Fatalf("CTE scan failed: %v", err)
		}
		cteCount++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("CTE rows error: %v", err)
	}
	rows.Close()
	if cteCount != 1 {
		t.Errorf("CTE rows = %d, want 1", cteCount)
	}

	// Named window: PostgreSQL reads WINDOW after GROUP BY / HAVING and before
	// ORDER BY. A misplaced clause is a syntax error here, which is the point of
	// running this against a real server.
	windowSQL, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:   "tei_sqlgen_probe",
		Columns: []string{"id", "SUM(amount) OVER w AS running"},
		Window:  "w AS (PARTITION BY customer_id ORDER BY created_at)",
		OrderBy: []sqlgen.OrderClause{{Column: "id"}},
	})
	if err != nil {
		t.Fatalf("Window Select error: %v", err)
	}
	rows, err = db.QueryContext(ctx, windowSQL.Statement, windowSQL.Args...)
	if err != nil {
		t.Fatalf("WINDOW execution failed: %v (sql=%s)", err, windowSQL.Statement)
	}
	running := map[int64]string{}
	for rows.Next() {
		var id int64
		var sum string
		if err := rows.Scan(&id, &sum); err != nil {
			t.Fatalf("WINDOW scan failed: %v", err)
		}
		running[id] = sum
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("WINDOW rows error: %v", err)
	}
	rows.Close()
	if got := running[1]; got != "100.50" {
		t.Errorf("running total for id=1 = %q, want 100.50", got)
	}
	if got := running[2]; got != "150.75" {
		t.Errorf("running total for id=2 = %q, want 150.75", got)
	}

	// Set operation: the adapter closes its statement with the keyword, so the
	// caller appends the other side. UNION removes the overlap, UNION ALL keeps
	// it.
	base, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:   "tei_sqlgen_probe",
		Columns: []string{"id", "amount"},
	})
	if err != nil {
		t.Fatalf("base Select error: %v", err)
	}
	other, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:   "tei_sqlgen_probe",
		Columns: []string{"id", "amount"},
		Where:   []sqlgen.Condition{{Column: "customer_id", Operator: "=", Value: int64(7)}},
	})
	if err != nil {
		t.Fatalf("other Select error: %v", err)
	}
	for _, tc := range []struct {
		name string
		all  bool
		want int
	}{
		{"UNION", false, 2},
		{"UNION ALL", true, 4},
	} {
		keyword := string(sqlgen.SetUnion)
		if tc.all {
			keyword += " ALL"
		}
		combined := base.Statement + " " + keyword + " " + other.Statement
		args := append(append([]any{}, base.Args...), other.Args...)
		rows, err = db.QueryContext(ctx, combined, args...)
		if err != nil {
			t.Fatalf("%s execution failed: %v (sql=%s)", tc.name, err, combined)
		}
		count := 0
		for rows.Next() {
			var id int64
			var amount string
			if err := rows.Scan(&id, &amount); err != nil {
				t.Fatalf("%s scan failed: %v", tc.name, err)
			}
			count++
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("%s rows error: %v", tc.name, err)
		}
		rows.Close()
		if count != tc.want {
			t.Errorf("%s rows = %d, want %d", tc.name, count, tc.want)
		}
	}

	// UPDATE with RETURNING and a WHERE clause: the WHERE placeholder has to
	// continue after the assignment placeholders.
	updateSQL, err := dialect.Update(ctx, sqlgen.UpdateOptions{
		Table:       "tei_sqlgen_probe",
		Assignments: map[string]any{"status": "shipped"},
		Where:       []sqlgen.Condition{{Column: "id", Operator: "=", Value: int64(1)}},
		Returning:   []string{"id", "status"},
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}
	var updatedID int64
	var updatedStatus string
	if err := db.QueryRowContext(ctx, updateSQL.Statement, updateSQL.Args...).Scan(&updatedID, &updatedStatus); err != nil {
		t.Fatalf("UPDATE ... RETURNING failed: %v (sql=%s)", err, updateSQL.Statement)
	}
	if updatedStatus != "shipped" {
		t.Errorf("UPDATE status = %q, want shipped", updatedStatus)
	}

	// DELETE: generated placeholders bind and the row goes away.
	deleteSQL, err := dialect.Delete(ctx, sqlgen.DeleteOptions{
		Table: "tei_sqlgen_probe",
		Where: []sqlgen.Condition{{Column: "id", Operator: "=", Value: int64(2)}},
	})
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if _, err := db.ExecContext(ctx, deleteSQL.Statement, deleteSQL.Args...); err != nil {
		t.Fatalf("DELETE execution failed: %v (sql=%s)", err, deleteSQL.Statement)
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM public.tei_sqlgen_probe WHERE id = 2").Scan(&count); err != nil {
		t.Fatalf("COUNT after DELETE failed: %v", err)
	}
	if count != 0 {
		t.Errorf("COUNT after DELETE = %d, want 0", count)
	}
}

// TestPostgresConstraintFailureClassification runs a real constraint violation
// through ToDatabaseFailure, so the SQLSTATE path is exercised against a server
// rather than against a constructed error.
func TestPostgresConstraintFailureClassification(t *testing.T) {
	dsn := os.Getenv("TEI_SQLGEN_PG_DSN")
	if dsn == "" {
		t.Skip("real PostgreSQL evidence unavailable: TEI_SQLGEN_PG_DSN unset")
	}
	if os.Getenv("TEI_SQLGEN_PG_CONFIRM_ISOLATED") != "tei_sqlgen_e2e" {
		t.Fatal("refusing to create public.tei_sqlgen_unique without the explicit disposable database confirmation")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("failed to open the pool: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	for _, stmt := range []string{
		`DROP TABLE IF EXISTS public.tei_sqlgen_unique`,
		`CREATE TABLE public.tei_sqlgen_unique (code varchar(255) PRIMARY KEY)`,
		`INSERT INTO public.tei_sqlgen_unique (code) VALUES ('A')`,
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("failed to prepare the unique probe: %v", err)
		}
	}
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), `DROP TABLE IF EXISTS public.tei_sqlgen_unique`); err != nil {
			t.Logf("failed to drop the unique probe: %v", err)
		}
	})

	dialect := NewDialect()
	insertSQL, err := dialect.Insert(ctx, sqlgen.InsertOptions{
		Table:   "tei_sqlgen_unique",
		Columns: []string{"code"},
		Values:  map[string]any{"code": "A"},
	})
	if err != nil {
		t.Fatalf("Insert error: %v", err)
	}
	_, execErr := db.ExecContext(ctx, insertSQL.Statement, insertSQL.Args...)
	if execErr == nil {
		t.Fatal("expected the duplicate primary key to be refused")
	}

	failure := sqlgen.ToDatabaseFailure("postgres", execErr)
	if failure == nil {
		t.Fatal("ToDatabaseFailure returned nil for a real error")
	}
	if code := failure.FailureCode(); code != "POSTGRES_UNIQUE_VIOLATION" {
		t.Errorf("FailureCode() = %q, want POSTGRES_UNIQUE_VIOLATION", code)
	}
	if failure.HasEffectUnknown() {
		t.Error("a constraint violation refused before it changed anything must not report an unknown effect")
	}
	if strings.Contains(strings.ToLower(failure.FailureMessage()), "database error from postgres") == false {
		t.Errorf("message lost its provider prefix: %q", failure.FailureMessage())
	}
	if failure.FailureCause() != execErr {
		t.Error("the original driver error was not preserved")
	}
}

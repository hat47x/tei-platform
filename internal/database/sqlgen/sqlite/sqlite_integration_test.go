package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"tei/internal/database/sqlgen"

	_ "modernc.org/sqlite"
)

// TestSQLiteIntegration verifies the SQLite dialect against a real in-memory SQLite database.
func TestSQLiteIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE orders (id INTEGER PRIMARY KEY, amount REAL, status TEXT)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	dialect := NewDialect()
	ctx := context.Background()

	// Test INSERT
	insertSQL, err := dialect.Insert(ctx, sqlgen.InsertOptions{
		Table:   "orders",
		Columns: []string{"id", "amount", "status"},
		Values:  map[string]any{"id": 1, "amount": 100.5, "status": "active"},
	})
	if err != nil {
		t.Fatalf("Insert error: %v", err)
	}

	_, err = db.Exec(insertSQL.Statement, insertSQL.Args...)
	if err != nil {
		t.Fatalf("INSERT execution failed: %v", err)
	}

	// Test SELECT
	selectSQL, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:   "orders",
		Columns: []string{"id", "amount", "status"},
		Where:   []sqlgen.Condition{{Column: "id", Operator: "=", Value: 1}},
	})
	if err != nil {
		t.Fatalf("Select error: %v", err)
	}

	var id int
	var amount float64
	var status string
	err = db.QueryRow(selectSQL.Statement, selectSQL.Args...).Scan(&id, &amount, &status)
	if err != nil {
		t.Fatalf("SELECT execution failed: %v", err)
	}

	if id != 1 || amount != 100.5 || status != "active" {
		t.Errorf("SELECT returned unexpected values: id=%d, amount=%f, status=%s", id, amount, status)
	}

	// Test UPDATE
	updateSQL, err := dialect.Update(ctx, sqlgen.UpdateOptions{
		Table:       "orders",
		Assignments: map[string]any{"status": "shipped"},
		Where:       []sqlgen.Condition{{Column: "id", Operator: "=", Value: 1}},
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	_, err = db.Exec(updateSQL.Statement, updateSQL.Args...)
	if err != nil {
		t.Fatalf("UPDATE execution failed: %v", err)
	}

	// Verify UPDATE
	err = db.QueryRow("SELECT status FROM orders WHERE id = 1").Scan(&status)
	if err != nil {
		t.Fatalf("SELECT after UPDATE failed: %v", err)
	}
	if status != "shipped" {
		t.Errorf("UPDATE did not change status: got %s, want shipped", status)
	}

	// Test DELETE
	deleteSQL, err := dialect.Delete(ctx, sqlgen.DeleteOptions{
		Table: "orders",
		Where: []sqlgen.Condition{{Column: "id", Operator: "=", Value: 1}},
	})
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	_, err = db.Exec(deleteSQL.Statement, deleteSQL.Args...)
	if err != nil {
		t.Fatalf("DELETE execution failed: %v", err)
	}

	// Verify DELETE
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM orders WHERE id = 1").Scan(&count)
	if err != nil {
		t.Fatalf("COUNT query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("COUNT after DELETE = %d, want 0", count)
	}
}

// TestSQLiteExtendedFeaturesExecute runs the extended contract (CTE, named
// window, set operation) against a real in-memory SQLite, so the generated SQL
// is checked by the engine rather than only by string matching.
func TestSQLiteExtendedFeaturesExecute(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE orders (
		id INTEGER PRIMARY KEY,
		customer_id INTEGER NOT NULL,
		amount REAL NOT NULL,
		created_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
	for _, row := range []struct {
		id       int
		customer int
		amount   float64
		at       string
	}{
		{1, 7, 100.5, "2026-11-01"},
		{2, 7, 50.25, "2026-11-02"},
		{3, 8, 200.00, "2026-11-03"},
	} {
		if _, err := db.Exec("INSERT INTO orders (id, customer_id, amount, created_at) VALUES (?, ?, ?, ?)",
			row.id, row.customer, row.amount, row.at); err != nil {
			t.Fatalf("failed to seed row %d: %v", row.id, err)
		}
	}

	dialect := NewDialect()
	ctx := context.Background()

	// A CTE narrows the rows before the outer query reads them, so the outer
	// query names the CTE rather than the base table.
	cteSQL, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:   "recent",
		Columns: []string{"id", "amount"},
		CTE:     "recent AS (SELECT * FROM orders WHERE amount > 60)",
	})
	if err != nil {
		t.Fatalf("CTE Select error: %v", err)
	}
	rows, err := db.Query(cteSQL.Statement, cteSQL.Args...)
	if err != nil {
		t.Fatalf("CTE query failed: %v", err)
	}
	var cteIDs []int
	for rows.Next() {
		var id int
		var amount float64
		if err := rows.Scan(&id, &amount); err != nil {
			t.Fatalf("CTE scan failed: %v", err)
		}
		if amount <= 60 {
			t.Errorf("CTE returned a row the CTE should have filtered out: amount=%v", amount)
		}
		cteIDs = append(cteIDs, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("CTE rows error: %v", err)
	}
	rows.Close()
	if len(cteIDs) != 2 {
		t.Errorf("CTE returned %d rows, want 2", len(cteIDs))
	}

	// A named window carries the partition and order across select items.
	windowSQL, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:   "orders",
		Columns: []string{"id", "SUM(amount) OVER w AS running"},
		Window:  "w AS (PARTITION BY customer_id ORDER BY created_at)",
	})
	if err != nil {
		t.Fatalf("Window Select error: %v", err)
	}
	rows, err = db.Query(windowSQL.Statement, windowSQL.Args...)
	if err != nil {
		t.Fatalf("window query failed: %v", err)
	}
	runningByID := map[int]float64{}
	for rows.Next() {
		var id int
		var running float64
		if err := rows.Scan(&id, &running); err != nil {
			t.Fatalf("window scan failed: %v", err)
		}
		runningByID[id] = running
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("window rows error: %v", err)
	}
	rows.Close()
	if got := runningByID[1]; got != 100.5 {
		t.Errorf("running total for id=1 = %v, want 100.5", got)
	}
	if got := runningByID[2]; got != 150.75 {
		t.Errorf("running total for id=2 = %v, want 150.75", got)
	}
	if got := runningByID[3]; got != 200.00 {
		t.Errorf("running total for id=3 = %v, want 200", got)
	}

	// A set operation combines two selects; the adapter only emits the keyword,
	// so the caller supplies the other side through the CTE-free statement.
	base, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:   "orders",
		Columns: []string{"id", "amount"},
	})
	if err != nil {
		t.Fatalf("base Select error: %v", err)
	}
	other, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:   "orders",
		Columns: []string{"id", "amount"},
		Where:   []sqlgen.Condition{{Column: "customer_id", Operator: "=", Value: 8}},
	})
	if err != nil {
		t.Fatalf("other Select error: %v", err)
	}
	combined := base.Statement + " " + string(sqlgen.SetUnion) + " " + other.Statement
	args := append(append([]any{}, base.Args...), other.Args...)
	rows, err = db.Query(combined, args...)
	if err != nil {
		t.Fatalf("union query failed: %v (sql=%s)", err, combined)
	}
	unionCount := 0
	for rows.Next() {
		var id int
		var amount float64
		if err := rows.Scan(&id, &amount); err != nil {
			t.Fatalf("union scan failed: %v", err)
		}
		unionCount++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("union rows error: %v", err)
	}
	rows.Close()
	// Both sides read the same table, so UNION removes the overlapping row and
	// leaves one row per distinct order.
	if unionCount != 3 {
		t.Errorf("UNION returned %d rows, want 3", unionCount)
	}

	// UNION ALL keeps the duplicates, so the same pair of selects yields four.
	combinedAll := base.Statement + " " + string(sqlgen.SetUnion) + " ALL " + other.Statement
	rows, err = db.Query(combinedAll, args...)
	if err != nil {
		t.Fatalf("union all query failed: %v (sql=%s)", err, combinedAll)
	}
	unionAllCount := 0
	for rows.Next() {
		var id int
		var amount float64
		if err := rows.Scan(&id, &amount); err != nil {
			t.Fatalf("union all scan failed: %v", err)
		}
		unionAllCount++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("union all rows error: %v", err)
	}
	rows.Close()
	if unionAllCount != 4 {
		t.Errorf("UNION ALL returned %d rows, want 4", unionAllCount)
	}

	// The adapter renders the set-operation keyword it was given, so a caller
	// that appends the other side gets a statement the engine accepts.
	withKeyword, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:        "orders",
		Columns:      []string{"id", "amount"},
		SetOperation: sqlgen.SetUnion,
	})
	if err != nil {
		t.Fatalf("set operation Select error: %v", err)
	}
	if !strings.HasSuffix(withKeyword.Statement, "UNION") {
		t.Errorf("set operation statement = %v, want UNION suffix", withKeyword.Statement)
	}
}

// TestSQLiteTypeMapping verifies type mapping works with real SQLite.
func TestSQLiteTypeMapping(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	dialect := NewDialect()

	tests := []struct {
		storage sqlgen.StorageType
		want    string
	}{
		{sqlgen.StorageInt64, "INTEGER"},
		{sqlgen.StorageVarchar, "TEXT"},
		{sqlgen.StorageText, "TEXT"},
		{sqlgen.StorageDateTimeTZ, "TEXT"},
		{sqlgen.StorageDecimal, "TEXT"},
		{sqlgen.StorageJSON, "TEXT"},
	}

	for _, tt := range tests {
		got, err := dialect.PhysicalType(tt.storage)
		if err != nil {
			t.Errorf("PhysicalType(%v) error: %v", tt.storage, err)
			continue
		}
		if got != tt.want {
			t.Errorf("PhysicalType(%v) = %v, want %v", tt.storage, got, tt.want)
		}
	}
}

// TestSQLiteTransaction verifies transaction behavior.
func TestSQLiteTransaction(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE orders (id INTEGER PRIMARY KEY, amount REAL)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	dialect := NewDialect()
	ctx := context.Background()

	// Start transaction
	tx, err := dialect.BeginTransaction(ctx, db)
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	insertSQL, err := dialect.Insert(ctx, sqlgen.InsertOptions{
		Table:   "orders",
		Columns: []string{"id", "amount"},
		Values:  map[string]any{"id": 1, "amount": 50.0},
	})
	if err != nil {
		t.Fatalf("Insert error: %v", err)
	}

	_, err = tx.Exec(insertSQL.Statement, insertSQL.Args...)
	if err != nil {
		t.Fatalf("INSERT in transaction failed: %v", err)
	}

	// Rollback
	err = tx.Rollback()
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify rollback
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM orders").Scan(&count)
	if err != nil {
		t.Fatalf("COUNT query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("COUNT after rollback = %d, want 0", count)
	}
}

// TestSQLiteNULLHandling verifies NULL value handling.
func TestSQLiteNULLHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE orders (id INTEGER PRIMARY KEY, amount REAL, status TEXT)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	dialect := NewDialect()
	ctx := context.Background()

	// Insert with NULL
	insertSQL, err := dialect.Insert(ctx, sqlgen.InsertOptions{
		Table:   "orders",
		Columns: []string{"id", "amount", "status"},
		Values:  map[string]any{"id": 1, "amount": nil, "status": "active"},
	})
	if err != nil {
		t.Fatalf("Insert error: %v", err)
	}

	_, err = db.Exec(insertSQL.Statement, insertSQL.Args...)
	if err != nil {
		t.Fatalf("INSERT with NULL failed: %v", err)
	}

	// Select with IS NULL
	selectSQL, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:   "orders",
		Columns: []string{"id"},
		Where:   []sqlgen.Condition{{Column: "amount", Operator: "IS NULL"}},
	})
	if err != nil {
		t.Fatalf("Select error: %v", err)
	}

	var id int
	err = db.QueryRow(selectSQL.Statement, selectSQL.Args...).Scan(&id)
	if err != nil {
		t.Fatalf("SELECT with IS NULL failed: %v", err)
	}
	if id != 1 {
		t.Errorf("SELECT with IS NULL returned id=%d, want 1", id)
	}
}

// TestSQLiteErrorConversion verifies DB errors are converted to TEI failures.
func TestSQLiteErrorConversion(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE orders (id INTEGER PRIMARY KEY, amount REAL NOT NULL)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	dialect := NewDialect()
	ctx := context.Background()

	// Try to insert NULL into NOT NULL column
	insertSQL, err := dialect.Insert(ctx, sqlgen.InsertOptions{
		Table:   "orders",
		Columns: []string{"id", "amount"},
		Values:  map[string]any{"id": 1, "amount": nil},
	})
	if err != nil {
		t.Fatalf("Insert error: %v", err)
	}

	_, err = db.Exec(insertSQL.Statement, insertSQL.Args...)
	if err == nil {
		t.Fatal("expected NOT NULL constraint error, got nil")
	}

	// Convert to TEI failure
	failure := sqlgen.ToDatabaseFailure("sqlite", err)
	if failure == nil {
		t.Fatal("ToDatabaseFailure returned nil")
	}

	if failure.Kind() != "business_rejection" {
		t.Errorf("expected business_rejection, got %s", failure.Kind())
	}
	if !errors.Is(failure.FailureCause(), err) {
		t.Error("original error not preserved in FailureCause()")
	}
}

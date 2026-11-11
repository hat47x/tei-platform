package sqlite

import (
	"context"
	"database/sql"
	"errors"
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

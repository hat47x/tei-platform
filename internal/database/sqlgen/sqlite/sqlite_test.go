package sqlite

import (
	"context"
	"errors"
	"strings"
	"testing"

	sqlgen "tei/internal/database/sqlgen"
)

func TestSQLitePhysicalType(t *testing.T) {
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
			t.Errorf("PhysicalType(%v) error = %v", tt.storage, err)
			continue
		}
		if got != tt.want {
			t.Errorf("PhysicalType(%v) = %v, want %v", tt.storage, got, tt.want)
		}
	}
}

func TestSQLiteCanStore(t *testing.T) {
	dialect := NewDialect()
	if !dialect.CanStore(sqlgen.StorageInt64) {
		t.Error("CanStore(StorageInt64) = false, want true")
	}
	if dialect.CanStore("unknown") {
		t.Error("CanStore(unknown) = true, want false")
	}
}

func TestSQLitePlaceholderStyle(t *testing.T) {
	dialect := NewDialect()
	if dialect.PlaceholderStyle() != sqlgen.PlaceholderPositionalQuestion {
		t.Errorf("PlaceholderStyle() = %v, want positional-question", dialect.PlaceholderStyle())
	}
}

func TestSQLiteSelect(t *testing.T) {
	dialect := NewDialect()
	ctx := context.Background()

	t.Run("basic select", func(t *testing.T) {
		opts := sqlgen.SelectOptions{
			Table:   "orders",
			Columns: []string{"id", "amount"},
		}
		sql, err := dialect.Select(ctx, opts)
		if err != nil {
			t.Fatalf("Select error = %v", err)
		}
		if !strings.Contains(sql.Statement, "SELECT id, amount FROM orders") {
			t.Errorf("Select statement = %v, want to contain SELECT id, amount FROM orders", sql.Statement)
		}
	})

	t.Run("select with where and order by", func(t *testing.T) {
		opts := sqlgen.SelectOptions{
			Table:   "orders",
			Columns: []string{"id", "amount"},
			Where: []sqlgen.Condition{
				{Column: "status", Operator: "=", Value: "active"},
				{LogicalOp: "AND", Column: "amount", Operator: ">", Value: 100},
			},
			OrderBy: []sqlgen.OrderClause{
				{Column: "amount", Desc: true},
			},
			Limit: intPtr(10),
		}
		sql, err := dialect.Select(ctx, opts)
		if err != nil {
			t.Fatalf("Select error = %v", err)
		}
		if !strings.Contains(sql.Statement, "WHERE status = ? AND amount > ?") {
			t.Errorf("Select statement = %v, want WHERE with placeholders", sql.Statement)
		}
		if !strings.Contains(sql.Statement, "ORDER BY amount DESC") {
			t.Errorf("Select statement = %v, want ORDER BY", sql.Statement)
		}
		if !strings.Contains(sql.Statement, "LIMIT 10") {
			t.Errorf("Select statement = %v, want LIMIT", sql.Statement)
		}
	})

	t.Run("select with join", func(t *testing.T) {
		opts := sqlgen.SelectOptions{
			Table:   "orders",
			Columns: []string{"orders.id", "customers.name"},
			Joins: []sqlgen.JoinClause{
				{Kind: "INNER", Table: "customers", On: []sqlgen.Condition{{Column: "orders.customer_id", Operator: "=", Value: "customers.id"}}},
			},
		}
		sql, err := dialect.Select(ctx, opts)
		if err != nil {
			t.Fatalf("Select error = %v", err)
		}
		if !strings.Contains(sql.Statement, "INNER JOIN customers ON orders.customer_id = ?") {
			t.Errorf("Select statement = %v, want JOIN", sql.Statement)
		}
	})

	t.Run("select with group by and having", func(t *testing.T) {
		opts := sqlgen.SelectOptions{
			Table:    "orders",
			Columns:  []string{"customer_id", "COUNT(*) as cnt"},
			GroupBy:  []string{"customer_id"},
			Having:   []sqlgen.Condition{{Column: "COUNT(*)", Operator: ">", Value: 5}},
		}
		sql, err := dialect.Select(ctx, opts)
		if err != nil {
			t.Fatalf("Select error = %v", err)
		}
		if !strings.Contains(sql.Statement, "GROUP BY customer_id") {
			t.Errorf("Select statement = %v, want GROUP BY", sql.Statement)
		}
		if !strings.Contains(sql.Statement, "HAVING COUNT(*) > ?") {
			t.Errorf("Select statement = %v, want HAVING", sql.Statement)
		}
	})
}

func TestSQLiteInsert(t *testing.T) {
	dialect := NewDialect()
	ctx := context.Background()

	opts := sqlgen.InsertOptions{
		Table:   "orders",
		Columns: []string{"id", "amount"},
		Values:  map[string]any{"id": 1, "amount": 100.5},
	}
	sql, err := dialect.Insert(ctx, opts)
	if err != nil {
		t.Fatalf("Insert error = %v", err)
	}
	if !strings.Contains(sql.Statement, "INSERT INTO orders (id, amount) VALUES (?, ?)") {
		t.Errorf("Insert statement = %v, want INSERT with placeholders", sql.Statement)
	}
	if len(sql.Args) != 2 {
		t.Errorf("Insert args = %v, want 2 args", sql.Args)
	}
}

func TestSQLiteUpdate(t *testing.T) {
	dialect := NewDialect()
	ctx := context.Background()

	opts := sqlgen.UpdateOptions{
		Table:       "orders",
		Assignments: map[string]any{"amount": 200.5, "status": "shipped"},
		Where:       []sqlgen.Condition{{Column: "id", Operator: "=", Value: 1}},
	}
	sql, err := dialect.Update(ctx, opts)
	if err != nil {
		t.Fatalf("Update error = %v", err)
	}
	if !strings.Contains(sql.Statement, "UPDATE orders SET") || !strings.Contains(sql.Statement, "WHERE id = ?") {
		t.Errorf("Update statement = %v, want UPDATE with placeholders", sql.Statement)
	}
}

func TestSQLiteDelete(t *testing.T) {
	dialect := NewDialect()
	ctx := context.Background()

	opts := sqlgen.DeleteOptions{
		Table: "orders",
		Where: []sqlgen.Condition{{Column: "id", Operator: "=", Value: 1}},
	}
	sql, err := dialect.Delete(ctx, opts)
	if err != nil {
		t.Fatalf("Delete error = %v", err)
	}
	if !strings.Contains(sql.Statement, "DELETE FROM orders WHERE id = ?") {
		t.Errorf("Delete statement = %v, want DELETE with placeholder", sql.Statement)
	}
}

func TestSQLiteConstraintDetection(t *testing.T) {
	dialect := NewDialect()

	t.Run("unique violation", func(t *testing.T) {
		err := errors.New("UNIQUE constraint failed: orders.id")
		if !dialect.IsUniqueViolation(err) {
			t.Error("IsUniqueViolation = false, want true")
		}
	})

	t.Run("foreign key violation", func(t *testing.T) {
		err := errors.New("FOREIGN KEY constraint failed")
		if !dialect.IsForeignKeyViolation(err) {
			t.Error("IsForeignKeyViolation = false, want true")
		}
	})

	t.Run("not null violation", func(t *testing.T) {
		err := errors.New("NOT NULL constraint failed: orders.amount")
		if !dialect.IsNullViolation(err) {
			t.Error("IsNullViolation = false, want true")
		}
	})
}

func TestSQLiteBeginTransaction(t *testing.T) {
	dialect := NewDialect()
	ctx := context.Background()

	// This test requires a real database connection, so we just verify the method signature
	// exists and returns an error when db is nil
	_, err := dialect.BeginTransaction(ctx, nil)
	if err == nil {
		t.Error("BeginTransaction with nil db should return error")
	}
}

func intPtr(i int) *int {
	return &i
}

// Ensure Dialect implements the common Dialect interface.
var _ sqlgen.Dialect = (*Dialect)(nil)

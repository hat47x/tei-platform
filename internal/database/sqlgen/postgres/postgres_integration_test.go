package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	sqlgen "tei/internal/database/sqlgen"
)

// TestPostgresIntegration verifies PostgreSQL dialect SQL generation.
// Note: This test only verifies SQL generation, not execution against a real
// PostgreSQL database. The generated SQL uses PostgreSQL-specific syntax
// ($1, $2, ... placeholders, RETURNING, FOR UPDATE, etc.) that may not be
// compatible with other databases.
func TestPostgresIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}

	dialect := NewDialect()
	ctx := context.Background()

	// Test INSERT with PostgreSQL-style placeholders
	insertSQL, err := dialect.Insert(ctx, sqlgen.InsertOptions{
		Table:   "orders",
		Columns: []string{"id", "amount", "status"},
		Values:  map[string]any{"id": 1, "amount": 100.5, "status": "active"},
	})
	if err != nil {
		t.Fatalf("Insert error: %v", err)
	}

	// Verify PostgreSQL-style placeholders ($1, $2, ...)
	if !strings.Contains(insertSQL.Statement, "$1") || !strings.Contains(insertSQL.Statement, "$2") {
		t.Errorf("Insert statement should use $1, $2 placeholders: %s", insertSQL.Statement)
	}

	// Test SELECT with WHERE, ORDER BY, LIMIT
	selectSQL, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:   "orders",
		Columns: []string{"id", "amount", "status"},
		Where:   []sqlgen.Condition{{Column: "status", Operator: "=", Value: "active"}},
		OrderBy: []sqlgen.OrderClause{{Column: "amount", Desc: true}},
		Limit:   intPtrIntegration(10),
	})
	if err != nil {
		t.Fatalf("Select error: %v", err)
	}

	if !strings.Contains(selectSQL.Statement, "SELECT id, amount, status FROM orders WHERE status = $1 ORDER BY amount DESC LIMIT 10") {
		t.Errorf("Select statement = %v, want expected PostgreSQL SELECT", selectSQL.Statement)
	}

	// Test UPDATE with RETURNING
	updateSQL, err := dialect.Update(ctx, sqlgen.UpdateOptions{
		Table:       "orders",
		Assignments: map[string]any{"status": "shipped"},
		Where:       []sqlgen.Condition{{Column: "id", Operator: "=", Value: 1}},
		Returning:   []string{"id"},
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	if !strings.Contains(updateSQL.Statement, "UPDATE orders SET status = $1 WHERE id = $2 RETURNING id") {
		t.Errorf("Update statement = %v, want expected PostgreSQL UPDATE", updateSQL.Statement)
	}

	// Test DELETE
	deleteSQL, err := dialect.Delete(ctx, sqlgen.DeleteOptions{
		Table: "orders",
		Where: []sqlgen.Condition{{Column: "id", Operator: "=", Value: 1}},
	})
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	if !strings.Contains(deleteSQL.Statement, "DELETE FROM orders WHERE id = $1") {
		t.Errorf("Delete statement = %v, want expected PostgreSQL DELETE", deleteSQL.Statement)
	}

	// Test JOIN with ON condition
	joinSQL, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:   "orders",
		Columns: []string{"orders.id", "customers.name"},
		Joins: []sqlgen.JoinClause{
			{
				Kind:  "INNER",
				Table: "customers",
				On:    []sqlgen.Condition{{Column: "orders.customer_id", Operator: "=", Value: "customers.id"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("Join Select error: %v", err)
	}

	if !strings.Contains(joinSQL.Statement, "INNER JOIN customers ON orders.customer_id = $1") {
		t.Errorf("Join Select statement = %v, want expected JOIN", joinSQL.Statement)
	}

	// Test GROUP BY and HAVING
	groupSQL, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:   "orders",
		Columns: []string{"status", "COUNT(*) as cnt"},
		GroupBy: []string{"status"},
		Having:  []sqlgen.Condition{{Column: "COUNT(*)", Operator: ">", Value: 5}},
	})
	if err != nil {
		t.Fatalf("Group By Select error: %v", err)
	}

	if !strings.Contains(groupSQL.Statement, "GROUP BY status") {
		t.Errorf("Group By Select statement = %v, want GROUP BY", groupSQL.Statement)
	}
	if !strings.Contains(groupSQL.Statement, "HAVING COUNT(*) > $1") {
		t.Errorf("Group By Select statement = %v, want HAVING", groupSQL.Statement)
	}

	// Test FOR UPDATE
	forUpdateSQL, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:     "orders",
		Columns:   []string{"id"},
		ForUpdate: true,
	})
	if err != nil {
		t.Fatalf("For Update Select error: %v", err)
	}
	if !strings.Contains(forUpdateSQL.Statement, "FOR UPDATE") {
		t.Errorf("For Update Select statement = %v, want FOR UPDATE", forUpdateSQL.Statement)
	}

	// Test FOR SHARE
	forShareSQL, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:    "orders",
		Columns:  []string{"id"},
		ForShare: true,
	})
	if err != nil {
		t.Fatalf("For Share Select error: %v", err)
	}
	if !strings.Contains(forShareSQL.Statement, "FOR SHARE") {
		t.Errorf("For Share Select statement = %v, want FOR SHARE", forShareSQL.Statement)
	}
}

// TestPostgresConstraintDetection verifies constraint error detection.
func TestPostgresConstraintDetectionIntegration(t *testing.T) {
	dialect := NewDialect()

	tests := []struct {
		name       string
		err        error
		checkFunc  func(error) bool
		shouldPass bool
	}{
		{
			name:       "unique violation",
			err:        errors.New("pq: duplicate key value violates unique constraint \"orders_id_key\""),
			checkFunc:  dialect.IsUniqueViolation,
			shouldPass: true,
		},
		{
			name:       "foreign key violation",
			err:        errors.New("pq: insert or update on table \"orders\" violates foreign key constraint"),
			checkFunc:  dialect.IsForeignKeyViolation,
			shouldPass: true,
		},
		{
			name:       "not null violation",
			err:        errors.New("pq: null value in column \"amount\" violates not-null constraint"),
			checkFunc:  dialect.IsNullViolation,
			shouldPass: true,
		},
		{
			name:       "check violation",
			err:        errors.New("pq: check constraint \"orders_amount_check\" violated"),
			checkFunc:  dialect.IsCheckViolation,
			shouldPass: true,
		},
		{
			name:       "generic error not unique",
			err:        errors.New("some other error"),
			checkFunc:  dialect.IsUniqueViolation,
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.checkFunc(tt.err)
			if result != tt.shouldPass {
				t.Errorf("%s: got %v, want %v", tt.name, result, tt.shouldPass)
			}
		})
	}
}

// TestPostgresTypeMapping verifies type mapping is consistent.
func TestPostgresTypeMapping(t *testing.T) {
	dialect := NewDialect()

	tests := []struct {
		storage sqlgen.StorageType
		want    string
	}{
		{sqlgen.StorageInt64, "bigint"},
		{sqlgen.StorageVarchar, "varchar(255)"},
		{sqlgen.StorageText, "text"},
		{sqlgen.StorageDateTimeTZ, "timestamptz"},
		{sqlgen.StorageDecimal, "numeric(18,2)"},
		{sqlgen.StorageJSON, "jsonb"},
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

func intPtrIntegration(i int) *int {
	return &i
}

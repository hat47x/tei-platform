package mysql

import (
	"context"
	"errors"
	"strings"
	"testing"

	sqlgen "tei/internal/database/sqlgen"
)

// TestMySQLIntegration verifies MySQL dialect SQL generation.
func TestMySQLIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}

	dialect := NewDialect()
	ctx := context.Background()

	// Test INSERT with MySQL-style placeholders (?)
	insertSQL, err := dialect.Insert(ctx, sqlgen.InsertOptions{
		Table:   "orders",
		Columns: []string{"id", "amount", "status"},
		Values:  map[string]any{"id": 1, "amount": 100.5, "status": "active"},
	})
	if err != nil {
		t.Fatalf("Insert error: %v", err)
	}

	if !strings.Contains(insertSQL.Statement, "?") {
		t.Errorf("Insert statement should use ? placeholders: %s", insertSQL.Statement)
	}

	// Test SELECT with WHERE, ORDER BY, LIMIT
	selectSQL, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:    "orders",
		Columns:  []string{"id", "amount", "status"},
		Where:    []sqlgen.Condition{{Column: "status", Operator: "=", Value: "active"}},
		OrderBy:  []sqlgen.OrderClause{{Column: "amount", Desc: true}},
		Limit:    intPtr(10),
	})
	if err != nil {
		t.Fatalf("Select error: %v", err)
	}

	if !strings.Contains(selectSQL.Statement, "SELECT id, amount, status FROM orders WHERE status = ? ORDER BY amount DESC LIMIT 10") {
		t.Errorf("Select statement = %v, want expected MySQL SELECT", selectSQL.Statement)
	}

	// Test UPDATE (no RETURNING)
	updateSQL, err := dialect.Update(ctx, sqlgen.UpdateOptions{
		Table:       "orders",
		Assignments: map[string]any{"status": "shipped"},
		Where:       []sqlgen.Condition{{Column: "id", Operator: "=", Value: 1}},
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	if !strings.Contains(updateSQL.Statement, "UPDATE orders SET status = ? WHERE id = ?") {
		t.Errorf("Update statement = %v, want expected MySQL UPDATE", updateSQL.Statement)
	}

	// Test DELETE
	deleteSQL, err := dialect.Delete(ctx, sqlgen.DeleteOptions{
		Table: "orders",
		Where: []sqlgen.Condition{{Column: "id", Operator: "=", Value: 1}},
	})
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	if !strings.Contains(deleteSQL.Statement, "DELETE FROM orders WHERE id = ?") {
		t.Errorf("Delete statement = %v, want expected MySQL DELETE", deleteSQL.Statement)
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

	if !strings.Contains(joinSQL.Statement, "INNER JOIN customers ON orders.customer_id = ?") {
		t.Errorf("Join Select statement = %v, want expected JOIN", joinSQL.Statement)
	}

	// Test GROUP BY and HAVING
	groupSQL, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:    "orders",
		Columns:  []string{"status", "COUNT(*) as cnt"},
		GroupBy:  []string{"status"},
		Having:   []sqlgen.Condition{{Column: "COUNT(*)", Operator: ">", Value: 5}},
	})
	if err != nil {
		t.Fatalf("Group By Select error: %v", err)
	}

	if !strings.Contains(groupSQL.Statement, "GROUP BY status") {
		t.Errorf("Group By Select statement = %v, want GROUP BY", groupSQL.Statement)
	}
	if !strings.Contains(groupSQL.Statement, "HAVING COUNT(*) > ?") {
		t.Errorf("Group By Select statement = %v, want HAVING", groupSQL.Statement)
	}

	// Test CTE (WITH)
	cteSQL, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:   "orders",
		Columns: []string{"id", "amount"},
		CTE:     "recent AS (SELECT * FROM orders WHERE amount > 0)",
		Where:   []sqlgen.Condition{{Column: "recent.amount", Operator: ">", Value: 0}},
	})
	if err != nil {
		t.Fatalf("CTE Select error: %v", err)
	}
	if !strings.Contains(cteSQL.Statement, "WITH recent AS (SELECT * FROM orders WHERE amount > 0)") {
		t.Errorf("CTE Select statement = %v, want WITH clause", cteSQL.Statement)
	}

	// Test WINDOW
	windowSQL, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:    "orders",
		Columns:  []string{"id", "amount", "SUM(amount) OVER w AS sum_amount"},
		Window:   "w AS (PARTITION BY customer_id ORDER BY created_at)",
	})
	if err != nil {
		t.Fatalf("Window Select error: %v", err)
	}
	if !strings.Contains(windowSQL.Statement, "WINDOW w AS (PARTITION BY customer_id ORDER BY created_at)") {
		t.Errorf("Window Select statement = %v, want WINDOW clause", windowSQL.Statement)
	}

	// Test UNION
	unionSQL, err := dialect.Select(ctx, sqlgen.SelectOptions{
		Table:        "orders",
		Columns:      []string{"id", "amount"},
		SetOperation: sqlgen.SetUnion,
	})
	if err != nil {
		t.Fatalf("Union Select error: %v", err)
	}
	if !strings.Contains(unionSQL.Statement, "UNION") {
		t.Errorf("Union Select statement = %v, want UNION", unionSQL.Statement)
	}
}

// TestMySQLConstraintDetection verifies constraint error detection.
func TestMySQLConstraintDetection(t *testing.T) {
	dialect := NewDialect()

	tests := []struct {
		name       string
		err        error
		checkFunc  func(error) bool
		shouldPass bool
	}{
		{
			name:       "unique violation",
			err:        errors.New("Error 1062: Duplicate entry '1' for key 'PRIMARY'"),
			checkFunc:  dialect.IsUniqueViolation,
			shouldPass: true,
		},
		{
			name:       "foreign key violation",
			err:        errors.New("Cannot add or update a child row: a foreign key constraint fails"),
			checkFunc:  dialect.IsForeignKeyViolation,
			shouldPass: true,
		},
		{
			name:       "not null violation",
			err:        errors.New("Column 'amount' cannot be null"),
			checkFunc:  dialect.IsNullViolation,
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

// TestMySQLTypeMapping verifies type mapping is consistent.
func TestMySQLTypeMapping(t *testing.T) {
	dialect := NewDialect()

	tests := []struct {
		storage sqlgen.StorageType
		want    string
	}{
		{sqlgen.StorageInt64, "BIGINT"},
		{sqlgen.StorageVarchar, "VARCHAR(255)"},
		{sqlgen.StorageText, "TEXT"},
		{sqlgen.StorageDateTimeTZ, "DATETIME(6)"},
		{sqlgen.StorageDecimal, "DECIMAL(18,2)"},
		{sqlgen.StorageJSON, "JSON"},
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

func intPtr(i int) *int {
	return &i
}

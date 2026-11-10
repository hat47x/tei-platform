package standard

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func openDB(t *testing.T) *sql.DB {
	t.Helper()
	c, err := sql.Open("sqlite", "file:test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRoundTripSQLite(t *testing.T) {
	cases := []struct {
		name     string
		sql      string
		args     []any
		setup    func(t *testing.T, db *sql.DB)
		expected [][]interface{}
	}{
		{
			name: "simple_where_order_limit",
			sql:  "SELECT order_id, customer_id FROM orders WHERE status = 'OPEN' ORDER BY order_id DESC LIMIT 2",
			setup: func(t *testing.T, db *sql.DB) {
				mustExec(t, db, "CREATE TABLE orders (order_id INTEGER, customer_id TEXT, status TEXT)")
				mustExec(t, db, "INSERT INTO orders VALUES (1, 'A', 'OPEN'), (2, 'B', 'OPEN'), (3, 'C', 'CLOSED')")
			},
			expected: [][]interface{}{{2, "B"}, {1, "A"}},
		},
		{
			name: "join_filter",
			sql:  "SELECT o.order_id, c.name FROM orders o JOIN customers c ON o.customer_id = c.id WHERE o.amount >= 15",
			setup: func(t *testing.T, db *sql.DB) {
				mustExec(t, db, "CREATE TABLE orders (order_id INTEGER, customer_id INTEGER, amount REAL)")
				mustExec(t, db, "CREATE TABLE customers (id INTEGER, name TEXT)")
				mustExec(t, db, "INSERT INTO customers VALUES (1, 'A'), (2, 'B')")
				mustExec(t, db, "INSERT INTO orders VALUES (1, 1, 10), (2, 1, 20), (3, 2, 5)")
			},
			expected: [][]interface{}{{2, "A"}},
		},
		{
			name: "group_by_having",
			sql:  "SELECT customer_id, SUM(amount) AS total FROM orders GROUP BY customer_id HAVING SUM(amount) > 15",
			setup: func(t *testing.T, db *sql.DB) {
				mustExec(t, db, "CREATE TABLE orders (order_id INTEGER, customer_id INTEGER, amount REAL)")
				mustExec(t, db, "INSERT INTO orders VALUES (1, 1, 10), (2, 1, 20), (3, 2, 5)")
			},
			expected: [][]interface{}{{1, float64(30)}},
		},
		{
			name: "case_null",
			sql:  "SELECT order_id, CASE WHEN amount IS NULL THEN 0 ELSE amount END AS amount FROM orders",
			setup: func(t *testing.T, db *sql.DB) {
				mustExec(t, db, "CREATE TABLE orders (order_id INTEGER, amount REAL)")
				mustExec(t, db, "INSERT INTO orders VALUES (1, 10), (2, NULL)")
			},
			expected: [][]interface{}{{1, float64(10)}, {2, float64(0)}},
		},
		{
			name: "in_subquery",
			sql:  "SELECT order_id FROM orders WHERE customer_id IN (SELECT id FROM customers WHERE active = 1)",
			setup: func(t *testing.T, db *sql.DB) {
				mustExec(t, db, "CREATE TABLE orders (order_id INTEGER, customer_id INTEGER)")
				mustExec(t, db, "CREATE TABLE customers (id INTEGER, active INTEGER)")
				mustExec(t, db, "INSERT INTO customers VALUES (1, 1), (2, 0)")
				mustExec(t, db, "INSERT INTO orders VALUES (1, 1), (2, 2)")
			},
			expected: [][]interface{}{{1}},
		},
		{
			name: "is_null",
			sql:  "SELECT order_id FROM orders WHERE amount IS NULL",
			setup: func(t *testing.T, db *sql.DB) {
				mustExec(t, db, "CREATE TABLE orders (order_id INTEGER, amount REAL)")
				mustExec(t, db, "INSERT INTO orders VALUES (1, 10), (2, NULL)")
			},
			expected: [][]interface{}{{2}},
		},
		{
			name: "distinct",
			sql:  "SELECT DISTINCT customer_id FROM orders",
			setup: func(t *testing.T, db *sql.DB) {
				mustExec(t, db, "CREATE TABLE orders (order_id INTEGER, customer_id INTEGER)")
				mustExec(t, db, "INSERT INTO orders VALUES (1, 1), (2, 1), (3, 2)")
			},
			expected: [][]interface{}{{1}, {2}},
		},
		{
			name: "cte",
			sql:  "WITH active_customers AS (SELECT id FROM customers WHERE active = 1) SELECT order_id FROM orders WHERE customer_id IN (SELECT id FROM active_customers)",
			setup: func(t *testing.T, db *sql.DB) {
				mustExec(t, db, "CREATE TABLE orders (order_id INTEGER, customer_id INTEGER)")
				mustExec(t, db, "CREATE TABLE customers (id INTEGER, active INTEGER)")
				mustExec(t, db, "INSERT INTO customers VALUES (1, 1), (2, 0)")
				mustExec(t, db, "INSERT INTO orders VALUES (1, 1), (2, 2)")
			},
			expected: [][]interface{}{{1}},
		},
		{
			name: "aggregate_count",
			sql:  "SELECT COUNT(*) AS cnt, SUM(amount) AS total FROM orders WHERE status = 'OPEN'",
			setup: func(t *testing.T, db *sql.DB) {
				mustExec(t, db, "CREATE TABLE orders (order_id INTEGER, amount REAL, status TEXT)")
				mustExec(t, db, "INSERT INTO orders VALUES (1, 10, 'OPEN'), (2, 20, 'OPEN'), (3, 30, 'CLOSED')")
			},
			expected: [][]interface{}{{int64(2), float64(30)}},
		},
		{
			name: "limit_offset",
			sql:  "SELECT order_id FROM orders ORDER BY order_id LIMIT 2 OFFSET 1",
			setup: func(t *testing.T, db *sql.DB) {
				mustExec(t, db, "CREATE TABLE orders (order_id INTEGER)")
				mustExec(t, db, "INSERT INTO orders VALUES (1), (2), (3)")
			},
			expected: [][]interface{}{{2}, {3}},
		},
		{
			name: "exists",
			sql:  "SELECT order_id FROM orders WHERE EXISTS (SELECT 1 FROM customers WHERE customers.id = orders.customer_id AND customers.active = 1)",
			setup: func(t *testing.T, db *sql.DB) {
				mustExec(t, db, "CREATE TABLE orders (order_id INTEGER, customer_id INTEGER)")
				mustExec(t, db, "CREATE TABLE customers (id INTEGER, active INTEGER)")
				mustExec(t, db, "INSERT INTO customers VALUES (1, 1), (2, 0)")
				mustExec(t, db, "INSERT INTO orders VALUES (1, 1), (2, 2), (3, 1)")
			},
			expected: [][]interface{}{{1}, {3}},
		},
		{
			name: "not_exists",
			sql:  "SELECT order_id FROM orders WHERE NOT EXISTS (SELECT 1 FROM customers WHERE customers.id = orders.customer_id AND customers.active = 1)",
			setup: func(t *testing.T, db *sql.DB) {
				mustExec(t, db, "CREATE TABLE orders (order_id INTEGER, customer_id INTEGER)")
				mustExec(t, db, "CREATE TABLE customers (id INTEGER, active INTEGER)")
				mustExec(t, db, "INSERT INTO customers VALUES (1, 1), (2, 0)")
				mustExec(t, db, "INSERT INTO orders VALUES (1, 1), (2, 2), (3, 1)")
			},
			expected: [][]interface{}{{2}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := openDB(t)
			defer db.Close()
			tc.setup(t, db)

			originalRows, err := db.Query(tc.sql, tc.args...)
			if err != nil {
				t.Fatalf("original query failed: %v", err)
			}
			defer originalRows.Close()
			original := queryRows(t, originalRows)

			q, diags := ParseSQLToStandard(tc.sql, "")
			for _, d := range diags {
				if d.Severity == SeverityError {
					t.Fatalf("parse error: %s", d.Message)
				}
			}
			generated, gdiags := StandardToSQL(q)
			for _, d := range gdiags {
				if d.Severity == SeverityError {
					t.Fatalf("generate error: %s", d.Message)
				}
			}

			genRows, err := db.Query(generated)
			if err != nil {
				t.Fatalf("generated query failed: %v generated SQL: %s", err, generated)
			}
			defer genRows.Close()
			generatedRows := queryRows(t, genRows)

			if len(original) != len(generatedRows) {
				t.Fatalf("row count mismatch: original=%d generated=%d original=%v generated=%v sql=%s", len(original), len(generatedRows), original, generatedRows, generated)
			}
			for i := range original {
				if !rowsEqual(original[i], generatedRows[i]) {
					t.Fatalf("row %d mismatch: original=%v generated=%v sql=%s", i, original[i], generatedRows[i], generated)
				}
			}
		})
	}
}

func queryRows(t *testing.T, rows *sql.Rows) [][]interface{} {
	t.Helper()
	var out [][]interface{}
	for rows.Next() {
		cols, err := rows.Columns()
		if err != nil {
			t.Fatal(err)
		}
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatal(err)
		}
		out = append(out, vals)
	}
	return out
}

func rowsEqual(a, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !valueEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func valueEqual(a, b interface{}) bool {
	if a == nil || b == nil {
		return a == b
	}
	switch av := a.(type) {
	case int64:
		bv, ok := b.(int64)
		return ok && av == bv
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case []byte:
		bv, ok := b.([]byte)
		return ok && string(av) == string(bv)
	}
	return false
}

func mustExec(t *testing.T, db *sql.DB, sql string) {
	t.Helper()
	if _, err := db.Exec(sql); err != nil {
		t.Fatal(err)
	}
}

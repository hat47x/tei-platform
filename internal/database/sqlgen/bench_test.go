package sqlgen_test

import (
	"context"
	"testing"

	"tei/internal/database/sqlgen"
	"tei/internal/database/sqlgen/postgres"
	"tei/internal/database/sqlgen/sqlite"
)

// BenchmarkPostgresSelect benchmarks PostgreSQL SELECT generation.
func BenchmarkPostgresSelect(b *testing.B) {
	dialect := postgres.NewDialect()
	ctx := context.Background()
	opts := sqlgen.SelectOptions{
		Table:    "orders",
		Columns:  []string{"id", "amount", "status"},
		Where:    []sqlgen.Condition{{Column: "status", Operator: "=", Value: "active"}},
		OrderBy:  []sqlgen.OrderClause{{Column: "amount", Desc: true}},
		Limit:    intPtr(100),
		Offset:   intPtr(0),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := dialect.Select(ctx, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSQLiteSelect benchmarks SQLite SELECT generation.
func BenchmarkSQLiteSelect(b *testing.B) {
	dialect := sqlite.NewDialect()
	ctx := context.Background()
	opts := sqlgen.SelectOptions{
		Table:    "orders",
		Columns:  []string{"id", "amount", "status"},
		Where:    []sqlgen.Condition{{Column: "status", Operator: "=", Value: "active"}},
		OrderBy:  []sqlgen.OrderClause{{Column: "amount", Desc: true}},
		Limit:    intPtr(100),
		Offset:   intPtr(0),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := dialect.Select(ctx, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPostgresInsert benchmarks PostgreSQL INSERT generation.
func BenchmarkPostgresInsert(b *testing.B) {
	dialect := postgres.NewDialect()
	ctx := context.Background()
	opts := sqlgen.InsertOptions{
		Table:   "orders",
		Columns: []string{"id", "amount", "status"},
		Values:  map[string]any{"id": 1, "amount": 100.5, "status": "active"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := dialect.Insert(ctx, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSQLiteInsert benchmarks SQLite INSERT generation.
func BenchmarkSQLiteInsert(b *testing.B) {
	dialect := sqlite.NewDialect()
	ctx := context.Background()
	opts := sqlgen.InsertOptions{
		Table:   "orders",
		Columns: []string{"id", "amount", "status"},
		Values:  map[string]any{"id": 1, "amount": 100.5, "status": "active"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := dialect.Insert(ctx, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func intPtr(i int) *int {
	return &i
}

package driver

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"tei/internal/database/persistence"
	"tei/internal/database/sqlgen"
	"tei/internal/query"
	"tei/internal/semantic"
)

func TestBuildRelationQuerySQL(t *testing.T) {
	resolved := query.ResolvedRelationQuery{
		Query: query.RelationQuery{
			ID:          "order_entry.order_with_customer",
			Application: "order_entry",
			Datasource:  "master",
			Root:        "sales.order",
			Relations:   []query.RelationStep{{Relation: "sales.order.customer", JoinKind: "LEFT", Alias: "cust"}},
			ResultColumns: []query.ResultColumnMapping{
				{Column: "order_id", Attribute: "sales.order.order_id"},
				{Column: "name", Attribute: "customer.customer.name"},
			},
		},
		RootData: semantic.Data{ID: "sales.order"},
		Steps: []query.ResolvedRelationStep{
			{
				Step:      query.RelationStep{Relation: "sales.order.customer", JoinKind: "LEFT", Alias: "cust"},
				Relation:  semantic.Relation{ID: "sales.order.customer", Keys: []semantic.RelationKey{{SourceAttribute: "sales.order.customer_id", TargetAttribute: "customer.customer.customer_id"}}},
				Source:    semantic.Data{ID: "sales.order"},
				Target:    semantic.Data{ID: "customer.customer"},
				JoinAlias: "cust",
			},
		},
	}
	mappings := []persistence.Mapping{
		{
			SemanticData: "sales.order",
			Table:        "orders",
			Columns: []persistence.ColumnMapping{
				{Attribute: "sales.order.order_id", Column: "id"},
				{Attribute: "sales.order.customer_id", Column: "customer_id"},
			},
		},
		{
			SemanticData: "customer.customer",
			Table:        "customers",
			Columns: []persistence.ColumnMapping{
				{Attribute: "customer.customer.customer_id", Column: "id"},
				{Attribute: "customer.customer.name", Column: "name"},
			},
		},
	}
	builder := RelationQuerySQL{
		Dialect:  newFakeDialect(),
		Resolved: resolved,
		Mappings: mappings,
	}
	sql, err := builder.BuildSelect()
	if err != nil {
		t.Fatalf("BuildSelect error: %v", err)
	}
	want := "SELECT orders.id, customers.name FROM orders LEFT JOIN customers AS cust ON orders.customer_id = customers.id"
	if sql.Statement != want {
		t.Errorf("SQL = %q, want %q", sql.Statement, want)
	}
}

func TestBuildRelationQuerySQLWithFiltersAndSorts(t *testing.T) {
	resolved := query.ResolvedRelationQuery{
		Query: query.RelationQuery{
			ID:          "order_entry.filtered",
			Application: "order_entry",
			Datasource:  "master",
			Root:        "sales.order",
			Relations:   []query.RelationStep{{Relation: "sales.order.customer"}},
			Filters:     []query.Filter{{Expression: "orders.status = 'active'"}, {Expression: "customers.name IS NOT NULL"}},
			Sorts:       []query.Sort{{Expression: "orders.created_at desc"}},
			PageSize:    10,
			PageOffset:  20,
			ResultColumns: []query.ResultColumnMapping{
				{Column: "order_id", Attribute: "sales.order.order_id"},
				{Column: "name", Attribute: "customer.customer.name"},
			},
		},
		RootData: semantic.Data{ID: "sales.order"},
		Steps: []query.ResolvedRelationStep{
			{
				Relation: semantic.Relation{ID: "sales.order.customer", Keys: []semantic.RelationKey{{SourceAttribute: "sales.order.customer_id", TargetAttribute: "customer.customer.customer_id"}}},
				Source:   semantic.Data{ID: "sales.order"},
				Target:   semantic.Data{ID: "customer.customer"},
			},
		},
	}
	mappings := []persistence.Mapping{
		{
			SemanticData: "sales.order",
			Table:        "orders",
			Columns: []persistence.ColumnMapping{
				{Attribute: "sales.order.order_id", Column: "id"},
				{Attribute: "sales.order.customer_id", Column: "customer_id"},
			},
		},
		{
			SemanticData: "customer.customer",
			Table:        "customers",
			Columns: []persistence.ColumnMapping{
				{Attribute: "customer.customer.customer_id", Column: "id"},
				{Attribute: "customer.customer.name", Column: "name"},
			},
		},
	}
	builder := RelationQuerySQL{
		Dialect:  newFakeDialect(),
		Resolved: resolved,
		Mappings: mappings,
	}
	sql, err := builder.BuildSelect()
	if err != nil {
		t.Fatalf("BuildSelect error: %v", err)
	}
	want := "SELECT orders.id, customers.name FROM orders LEFT JOIN customers AS rel_0 ON orders.customer_id = customers.id WHERE orders.status = 'active' AND customers.name IS NOT NULL ORDER BY orders.created_at DESC LIMIT 10 OFFSET 20"
	if sql.Statement != want {
		t.Errorf("SQL = %q, want %q", sql.Statement, want)
	}
}

type fakeDialect struct{ selectColumns []string }

func newFakeDialect() *fakeDialect { return &fakeDialect{} }

func (d *fakeDialect) Select(_ context.Context, opts sqlgen.SelectOptions) (sqlgen.SQL, error) {
	var sb strings.Builder
	sb.WriteString("SELECT ")
	sb.WriteString(strings.Join(opts.Columns, ", "))
	sb.WriteString(" FROM ")
	sb.WriteString(opts.Table)
	for _, j := range opts.Joins {
		sb.WriteString(" ")
		sb.WriteString(j.Kind)
		sb.WriteString(" JOIN ")
		sb.WriteString(j.Table)
		if j.Alias != "" {
			sb.WriteString(" AS ")
			sb.WriteString(j.Alias)
		}
		if len(j.On) > 0 {
			sb.WriteString(" ON ")
			for i, c := range j.On {
				if i > 0 {
					sb.WriteString(" AND ")
				}
				sb.WriteString(c.Column)
				sb.WriteString(" = ")
				sb.WriteString(c.Value.(string))
			}
		}
	}
	if len(opts.Where) > 0 {
		sb.WriteString(" WHERE ")
		for i, c := range opts.Where {
			if i > 0 {
				sb.WriteString(" AND ")
			}
			sb.WriteString(c.Expression)
		}
	}
	if len(opts.OrderBy) > 0 {
		sb.WriteString(" ORDER BY ")
		for i, o := range opts.OrderBy {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(o.Column)
			if o.Desc {
				sb.WriteString(" DESC")
			}
		}
	}
	if opts.Limit != nil {
		sb.WriteString(" LIMIT ")
		sb.WriteString(fmt.Sprintf("%d", *opts.Limit))
	}
	if opts.Offset != nil {
		sb.WriteString(" OFFSET ")
		sb.WriteString(fmt.Sprintf("%d", *opts.Offset))
	}
	return sqlgen.SQL{Statement: sb.String()}, nil
}
func (d *fakeDialect) Insert(_ context.Context, _ sqlgen.InsertOptions) (sqlgen.SQL, error) { return sqlgen.SQL{}, nil }
func (d *fakeDialect) Update(_ context.Context, _ sqlgen.UpdateOptions) (sqlgen.SQL, error) { return sqlgen.SQL{}, nil }
func (d *fakeDialect) Delete(_ context.Context, _ sqlgen.DeleteOptions) (sqlgen.SQL, error) { return sqlgen.SQL{}, nil }
func (d *fakeDialect) PhysicalType(_ sqlgen.StorageType) (string, error) { return "", nil }
func (d *fakeDialect) CanStore(_ sqlgen.StorageType) bool { return false }
func (d *fakeDialect) PlaceholderStyle() sqlgen.PlaceholderStyle { return sqlgen.PlaceholderPositionalDollar }
func (d *fakeDialect) Name() string { return "fake" }
func (d *fakeDialect) BeginTransaction(_ context.Context, _ *sql.DB) (*sql.Tx, error) { return nil, nil }
func (d *fakeDialect) IsUniqueViolation(_ error) bool { return false }
func (d *fakeDialect) IsForeignKeyViolation(_ error) bool { return false }
func (d *fakeDialect) IsNullViolation(_ error) bool { return false }
func (d *fakeDialect) IsCheckViolation(_ error) bool { return false }

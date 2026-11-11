package postgresquery

import (
	"context"
	"database/sql"
	"testing"

	"tei/internal/application"
	"tei/internal/database/persistence"
	"tei/internal/query"
	"tei/internal/semantic"
)

func TestExecuteRelationQueryValidatesDatasource(t *testing.T) {
	executor := Executor{
		DB:       nil,
		Database: "master_data",
	}
	resolved := query.ResolvedRelationQuery{
		Query: query.RelationQuery{
			ID:          "order_entry.order_with_customer",
			Application: "order_entry",
			Datasource:  "master",
		},
		RootData: semantic.Data{ID: "sales.order"},
		Steps: []query.ResolvedRelationStep{
			{
				Relation: semantic.Relation{ID: "sales.order.customer"},
				Source:   semantic.Data{ID: "sales.order"},
				Target:   semantic.Data{ID: "customer.customer"},
			},
		},
	}
	_, err := executor.ExecuteRelationQuery(context.Background(), resolved, application.DatabaseRequirement{TargetDatabase: "other_db"}, nil)
	if err == nil {
		t.Fatal("expected error for mismatched datasource")
	}
}

func TestExecuteRelationQueryRejectsLazyMode(t *testing.T) {
	mockDB := &mockQueryer{err: context.DeadlineExceeded}
	executor := Executor{
		DB:       mockDB,
		Database: "master_data",
		Mappings: []persistence.Mapping{
			{SemanticData: "sales.order", Table: "orders", Columns: []persistence.ColumnMapping{
				{Attribute: "sales.order.order_id", Column: "id"},
				{Attribute: "sales.order.customer_id", Column: "customer_id"},
			}},
			{SemanticData: "customer.customer", Table: "customers", Columns: []persistence.ColumnMapping{
				{Attribute: "customer.customer.customer_id", Column: "id"},
				{Attribute: "customer.customer.name", Column: "name"},
			}},
		},
	}
	resolved := query.ResolvedRelationQuery{
		Query: query.RelationQuery{
			ID:          "order_entry.lazy",
			Application: "order_entry",
			Datasource:  "master",
			FetchMode:   query.RelationFetchModeLazy,
			Relations:   []query.RelationStep{{Relation: "sales.order.customer"}},
			ResultColumns: []query.ResultColumnMapping{
				{Column: "order_id", Attribute: "sales.order.order_id"},
				{Column: "name", Attribute: "customer.customer.name"},
			},
		},
		RootData: semantic.Data{ID: "sales.order"},
		Steps: []query.ResolvedRelationStep{
			{
				Relation: semantic.Relation{
					ID:         "sales.order.customer",
					Keys:       []semantic.RelationKey{{SourceAttribute: "sales.order.customer_id", TargetAttribute: "customer.customer.customer_id"}},
				},
				Source:   semantic.Data{ID: "sales.order"},
				Target:   semantic.Data{ID: "customer.customer"},
			},
		},
		SQL: "SELECT 1",
	}
	_, err := executor.ExecuteRelationQuery(context.Background(), resolved, application.DatabaseRequirement{TargetDatabase: "master_data"}, nil)
	if err == nil {
		t.Fatal("expected error for lazy mode")
	}
	if msg := err.Error(); !contains(msg, "lazy fetchingは未実装です") {
		t.Fatalf("unexpected error message: %s", msg)
	}
}

func TestExecuteRelationQueryRejectsBatchWithoutSize(t *testing.T) {
	mockDB := &mockQueryer{err: context.DeadlineExceeded}
	executor := Executor{
		DB:       mockDB,
		Database: "master_data",
		Mappings: []persistence.Mapping{
			{SemanticData: "sales.order", Table: "orders", Columns: []persistence.ColumnMapping{
				{Attribute: "sales.order.order_id", Column: "id"},
				{Attribute: "sales.order.customer_id", Column: "customer_id"},
			}},
			{SemanticData: "customer.customer", Table: "customers", Columns: []persistence.ColumnMapping{
				{Attribute: "customer.customer.customer_id", Column: "id"},
				{Attribute: "customer.customer.name", Column: "name"},
			}},
		},
	}
	resolved := query.ResolvedRelationQuery{
		Query: query.RelationQuery{
			ID:          "order_entry.batch",
			Application: "order_entry",
			Datasource:  "master",
			FetchMode:   query.RelationFetchModeBatch,
			Relations:   []query.RelationStep{{Relation: "sales.order.customer"}},
			ResultColumns: []query.ResultColumnMapping{
				{Column: "order_id", Attribute: "sales.order.order_id"},
				{Column: "name", Attribute: "customer.customer.name"},
			},
		},
		RootData: semantic.Data{ID: "sales.order"},
		Steps: []query.ResolvedRelationStep{
			{
				Relation: semantic.Relation{
					ID:         "sales.order.customer",
					Keys:       []semantic.RelationKey{{SourceAttribute: "sales.order.customer_id", TargetAttribute: "customer.customer.customer_id"}},
				},
				Source:   semantic.Data{ID: "sales.order"},
				Target:   semantic.Data{ID: "customer.customer"},
			},
		},
		SQL: "SELECT 1",
	}
	_, err := executor.ExecuteRelationQuery(context.Background(), resolved, application.DatabaseRequirement{TargetDatabase: "master_data"}, nil)
	if err == nil {
		t.Fatal("expected error for batch without size")
	}
}

type mockQueryer struct {
	err error
}

func (m *mockQueryer) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return nil, m.err
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

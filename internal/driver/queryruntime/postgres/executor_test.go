package postgresquery

import (
	"context"
	"testing"

	"tei/internal/application"
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

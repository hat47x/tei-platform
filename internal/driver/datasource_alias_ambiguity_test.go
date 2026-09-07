package driver_test

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/semantic"
)

func TestResolverFailsClosedWhenDatasourceAliasIsAmbiguous(t *testing.T) {
	world := ir.World{
		Applications: []application.Application{
			{
				ID: "app_a",
				Databases: []application.DatabaseRequirement{
					{Alias: "sales", TargetDatabase: "sales_core", Access: database.AccessRead},
					{Alias: "sales", TargetDatabase: "archive_sales", Access: database.AccessRead},
				},
				Patterns: []application.Pattern{
					{ID: "app_a.browse", Kind: application.PatternBrowse, Data: "sales.order", Query: "app_a.query"},
				},
			},
		},
		Queries: []query.EmbeddedSQL{
			{ID: "app_a.query", Application: "app_a", Datasource: "sales"},
		},
		Semantics: []semantic.Namespace{
			{ID: "sales", Data: []semantic.Data{{ID: "sales.order"}}},
		},
	}

	resolver := driver.NewResolver(world)
	_, err := resolver.ResolvePattern("app_a", "app_a.browse")
	if err == nil {
		t.Fatal("duplicate datasource aliasから先頭要素を黙って選択しました")
	}
	if !strings.Contains(err.Error(), "複数定義されています") {
		t.Fatalf("duplicate datasource aliasをmissingとして診断しました: %v", err)
	}
}

func TestResolverReportsMissingDatasourceAliasSeparatelyFromAmbiguity(t *testing.T) {
	world := ir.World{
		Applications: []application.Application{
			{
				ID: "app_a",
				Databases: []application.DatabaseRequirement{
					{Alias: "master", TargetDatabase: "master_data", Access: database.AccessRead},
				},
				Patterns: []application.Pattern{
					{ID: "app_a.browse", Kind: application.PatternBrowse, Data: "sales.order", Query: "app_a.query"},
				},
			},
		},
		Queries: []query.EmbeddedSQL{
			{ID: "app_a.query", Application: "app_a", Datasource: "sales"},
		},
		Semantics: []semantic.Namespace{
			{ID: "sales", Data: []semantic.Data{{ID: "sales.order"}}},
		},
	}

	resolver := driver.NewResolver(world)
	_, err := resolver.ResolvePattern("app_a", "app_a.browse")
	if err == nil {
		t.Fatal("missing datasource aliasを解決済みとして扱いました")
	}
	if !strings.Contains(err.Error(), "定義されていません") {
		t.Fatalf("missing datasource aliasの診断が失われました: %v", err)
	}
	if strings.Contains(err.Error(), "複数定義されています") {
		t.Fatalf("missing datasource aliasをambiguityとして診断しました: %v", err)
	}
}

func TestReferencePatternFailsClosedWhenDatasourceAliasIsAmbiguous(t *testing.T) {
	world := ir.World{
		Applications: []application.Application{
			{
				ID: "app_a",
				Databases: []application.DatabaseRequirement{
					{Alias: "master", TargetDatabase: "master_data", Access: database.AccessRead},
					{Alias: "master", TargetDatabase: "archive_master", Access: database.AccessRead},
				},
				Patterns: []application.Pattern{
					{ID: "app_a.reference", Kind: application.PatternReference, Relation: "sales.order.customer", Datasource: "master"},
				},
			},
		},
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Data: []semantic.Data{
					{
						ID: "sales.order",
						Relations: []semantic.Relation{
							{ID: "sales.order.customer", Target: "customer.customer", Cardinality: semantic.CardinalityOne},
						},
					},
				},
			},
			{ID: "customer", Data: []semantic.Data{{ID: "customer.customer"}}},
		},
	}

	resolver := driver.NewResolver(world)
	_, err := resolver.ResolvePattern("app_a", "app_a.reference")
	if err == nil {
		t.Fatal("reference Patternがduplicate datasource aliasから一件を黙って選択しました")
	}
	if !strings.Contains(err.Error(), "複数定義されています") {
		t.Fatalf("reference Patternのduplicate datasource aliasをmissingとして診断しました: %v", err)
	}
}

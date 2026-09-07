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

package application

import "testing"

func TestDatabaseByAliasResolvesUniqueAlias(t *testing.T) {
	app := Application{
		Databases: []DatabaseRequirement{
			{Alias: "sales", TargetDatabase: "sales_core"},
			{Alias: "master", TargetDatabase: "master_data"},
		},
	}

	got, ok := app.DatabaseByAlias("sales")
	if !ok {
		t.Fatal("unique datasource aliasを解決できません")
	}
	if got.TargetDatabase != "sales_core" {
		t.Fatalf("別datasourceを返しました: got=%s want=sales_core", got.TargetDatabase)
	}
}

func TestDatabaseByAliasFailsClosedWhenAliasIsMissing(t *testing.T) {
	app := Application{
		Databases: []DatabaseRequirement{
			{Alias: "sales", TargetDatabase: "sales_core"},
		},
	}

	if _, ok := app.DatabaseByAlias("master"); ok {
		t.Fatal("存在しないdatasource aliasを解決済みとして扱いました")
	}
}

func TestDatabaseByAliasFailsClosedWhenAliasIsAmbiguous(t *testing.T) {
	app := Application{
		Databases: []DatabaseRequirement{
			{Alias: "sales", TargetDatabase: "sales_core"},
			{Alias: "sales", TargetDatabase: "archive_sales"},
		},
	}

	if _, ok := app.DatabaseByAlias("sales"); ok {
		t.Fatal("duplicate datasource aliasから先頭要素を黙って選択しました")
	}
}

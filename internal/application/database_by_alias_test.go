package application

import "testing"

func TestDatabaseByAliasKeepsDeclarationLookupForDuplicateAlias(t *testing.T) {
	app := Application{
		Databases: []DatabaseRequirement{
			{Alias: "sales", TargetDatabase: "sales_core"},
			{Alias: "sales", TargetDatabase: "archive_sales"},
		},
	}

	if _, ok := app.DatabaseByAlias("sales"); !ok {
		t.Fatal("存在するdatasource aliasを未定義として扱いました")
	}
}

func TestDatabaseByAliasExactResolvesUniqueAlias(t *testing.T) {
	app := Application{
		Databases: []DatabaseRequirement{
			{Alias: "sales", TargetDatabase: "sales_core"},
			{Alias: "master", TargetDatabase: "master_data"},
		},
	}

	got, ok := app.DatabaseByAliasExact("sales")
	if !ok {
		t.Fatal("unique datasource aliasを解決できません")
	}
	if got.TargetDatabase != "sales_core" {
		t.Fatalf("別datasourceを返しました: got=%s want=sales_core", got.TargetDatabase)
	}
}

func TestDatabaseByAliasExactFailsClosedWhenAliasIsMissing(t *testing.T) {
	app := Application{
		Databases: []DatabaseRequirement{
			{Alias: "sales", TargetDatabase: "sales_core"},
		},
	}

	if _, ok := app.DatabaseByAliasExact("master"); ok {
		t.Fatal("存在しないdatasource aliasを解決済みとして扱いました")
	}
}

func TestDatabaseByAliasExactFailsClosedWhenAliasIsAmbiguous(t *testing.T) {
	app := Application{
		Databases: []DatabaseRequirement{
			{Alias: "sales", TargetDatabase: "sales_core"},
			{Alias: "sales", TargetDatabase: "archive_sales"},
		},
	}

	if _, ok := app.DatabaseByAliasExact("sales"); ok {
		t.Fatal("duplicate datasource aliasから一件を黙って選択しました")
	}
}

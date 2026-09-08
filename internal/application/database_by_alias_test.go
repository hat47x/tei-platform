package application

import "testing"

func TestHasDatabaseAliasReturnsTrueForDuplicateDeclaration(t *testing.T) {
	app := Application{
		Databases: []DatabaseRequirement{
			{Alias: "sales", TargetDatabase: "sales_core"},
			{Alias: "sales", TargetDatabase: "archive_sales"},
		},
	}

	if !app.HasDatabaseAlias("sales") {
		t.Fatal("duplicateを含む既存datasource alias宣言を未定義として扱いました")
	}
}

func TestHasDatabaseAliasReturnsFalseWhenAliasIsMissing(t *testing.T) {
	app := Application{
		Databases: []DatabaseRequirement{
			{Alias: "sales", TargetDatabase: "sales_core"},
		},
	}

	if app.HasDatabaseAlias("master") {
		t.Fatal("存在しないdatasource alias宣言を存在すると扱いました")
	}
}

func TestDatabaseByAliasKeepsDeclarationLookupForDuplicateAliasWithoutReturningFirstMatch(t *testing.T) {
	app := Application{
		Databases: []DatabaseRequirement{
			{Alias: "sales", TargetDatabase: "sales_core"},
			{Alias: "sales", TargetDatabase: "archive_sales"},
		},
	}

	got, ok := app.DatabaseByAlias("sales")
	if !ok {
		t.Fatal("存在するdatasource aliasを未定義として扱いました")
	}
	if got.Alias != "" || got.TargetDatabase != "" || got.Access != "" || got.RequiredContracts != nil {
		t.Fatalf("duplicate datasource aliasから一件目の内容を返しました: %#v", got)
	}
}

func TestDatabaseByAliasKeepsUniqueLookupValue(t *testing.T) {
	app := Application{
		Databases: []DatabaseRequirement{
			{Alias: "sales", TargetDatabase: "sales_core"},
			{Alias: "master", TargetDatabase: "master_data"},
		},
	}

	got, ok := app.DatabaseByAlias("sales")
	if !ok {
		t.Fatal("unique datasource aliasを未定義として扱いました")
	}
	if got.TargetDatabase != "sales_core" {
		t.Fatalf("unique datasource aliasの既存lookup値が変わりました: got=%s want=sales_core", got.TargetDatabase)
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

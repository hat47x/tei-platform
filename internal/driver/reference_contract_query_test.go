package driver_test

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func TestResolveReferenceContractQueryUsesExplicitInvocation(t *testing.T) {
	world := referenceContractQueryWorld()
	resolver := driver.NewResolver(world)

	resolved, err := resolver.ResolveReferenceContractQuery("order_entry", "order_entry.customer_reference")
	if err != nil {
		t.Fatalf("明示したReference Contract Queryを解決できません: %v", err)
	}
	if resolved.Datasource.Alias != "master" || resolved.Database.ID != "master_data" {
		t.Fatalf("Datasource / Database境界が変わっています: datasource=%#v database=%#v", resolved.Datasource, resolved.Database)
	}
	if resolved.Contract.ID != "master_data.customer" || resolved.Contract.Version.Major != 2 || resolved.Contract.Version.Minor != 4 {
		t.Fatalf("Contract identity / versionを保持できていません: %#v", resolved.Contract)
	}
	if resolved.Query.Name != "find_by_id" {
		t.Fatalf("明示したContract Queryを解決していません: %#v", resolved.Query)
	}
	if len(resolved.Invocation.Inputs) != 1 ||
		resolved.Invocation.Inputs[0].TargetAttribute != "customer.customer.customer_id" ||
		resolved.Invocation.Inputs[0].Input != "id" {
		t.Fatalf("input bindingを変更しています: %#v", resolved.Invocation.Inputs)
	}
}

func TestResolveReferenceContractQueryRejectsMissingInvocation(t *testing.T) {
	world := referenceContractQueryWorld()
	world.Applications[0].Patterns[0].ContractQuery = nil

	_, err := driver.NewResolver(world).ResolveReferenceContractQuery("order_entry", "order_entry.customer_reference")
	if err == nil {
		t.Fatal("Contract Query呼び出しがないreference Patternを暗黙に補完しています")
	}
	if !strings.Contains(err.Error(), "Contract Query呼び出しが定義されていません") {
		t.Fatalf("不足しているauthoringを判別できないerrorです: %v", err)
	}
}

func TestResolveReferenceContractQueryRejectsContractOutsideDatasourceRequirement(t *testing.T) {
	world := referenceContractQueryWorld()
	world.Applications[0].Databases[0].RequiredContracts = nil

	_, err := driver.NewResolver(world).ResolveReferenceContractQuery("order_entry", "order_entry.customer_reference")
	if err == nil {
		t.Fatal("Applicationが要求していないContractをDriverが利用しています")
	}
	if !strings.Contains(err.Error(), "要求していません") {
		t.Fatalf("Datasource requirement違反を判別できないerrorです: %v", err)
	}
}

func TestResolveReferenceContractQueryRejectsIncompatibleMajorVersion(t *testing.T) {
	world := referenceContractQueryWorld()
	world.Applications[0].Databases[0].RequiredContracts[0].CompatibleMajor = 3

	_, err := driver.NewResolver(world).ResolveReferenceContractQuery("order_entry", "order_entry.customer_reference")
	if err == nil {
		t.Fatal("互換majorが一致しないContractをDriverが利用しています")
	}
	if !strings.Contains(err.Error(), "互換majorは3") {
		t.Fatalf("Contract version不整合を判別できないerrorです: %v", err)
	}
}

func TestResolveReferenceContractQueryRejectsUnknownQuery(t *testing.T) {
	world := referenceContractQueryWorld()
	world.Applications[0].Patterns[0].ContractQuery.Query = "find_by_name"

	_, err := driver.NewResolver(world).ResolveReferenceContractQuery("order_entry", "order_entry.customer_reference")
	if err == nil {
		t.Fatal("存在しないContract Queryを名前から推測して補っています")
	}
	if !strings.Contains(err.Error(), "Query find_by_nameがありません") {
		t.Fatalf("存在しないContract Queryを判別できないerrorです: %v", err)
	}
}

func TestResolveReferenceContractQueryRejectsContractFromAnotherDatabase(t *testing.T) {
	world := referenceContractQueryWorld()
	world.Databases[0].Contracts = nil
	world.Databases = append(world.Databases, database.Database{
		ID: "other_master",
		Contracts: []database.Contract{
			{
				ID:      "master_data.customer",
				Version: database.ContractVersion{Major: 2, Minor: 4},
				Queries: []database.QueryContract{{Name: "find_by_id"}},
			},
		},
	})

	_, err := driver.NewResolver(world).ResolveReferenceContractQuery("order_entry", "order_entry.customer_reference")
	if err == nil {
		t.Fatal("別Databaseにある同名Contractを暗黙に利用しています")
	}
	if !strings.Contains(err.Error(), "Database master_data") || !strings.Contains(err.Error(), "Contract master_data.customerが見つかりません") {
		t.Fatalf("Datasourceのtarget Database境界を判別できないerrorです: %v", err)
	}
}

func referenceContractQueryWorld() ir.World {
	return ir.World{
		Applications: []application.Application{
			{
				ID: "order_entry",
				Databases: []application.DatabaseRequirement{
					{
						Alias:          "master",
						TargetDatabase: "master_data",
						Access:         database.AccessRead,
						RequiredContracts: []database.ContractRequirement{
							{ContractID: "master_data.customer", CompatibleMajor: 2},
						},
					},
				},
				Patterns: []application.Pattern{
					{
						ID:         "order_entry.customer_reference",
						Kind:       application.PatternReference,
						Relation:   "sales.order.customer",
						Datasource: "master",
						ContractQuery: &application.ContractQueryInvocation{
							Contract: "master_data.customer",
							Query:    "find_by_id",
							Inputs: []application.ContractQueryInputBinding{
								{TargetAttribute: "customer.customer.customer_id", Input: "id"},
							},
						},
					},
				},
			},
		},
		Databases: []database.Database{
			{
				ID: "master_data",
				Contracts: []database.Contract{
					{
						ID:      "master_data.customer",
						Version: database.ContractVersion{Major: 2, Minor: 4},
						Queries: []database.QueryContract{{Name: "find_by_id"}},
					},
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
			{
				ID:   "customer",
				Data: []semantic.Data{{ID: "customer.customer"}},
			},
		},
	}
}

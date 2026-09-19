package validation

import (
	"testing"

	"tei/internal/database"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/realization"
	"tei/internal/semantic"
)

func contractQuerySQLValidationWorld() ir.World {
	return ir.World{
		Databases: []database.Database{
			{
				ID: "master_data",
				Contracts: []database.Contract{
					{
						ID:      "master_data.customer",
						Version: database.ContractVersion{Major: 2, Minor: 4},
						Queries: []database.QueryContract{
							{
								Name:              "find_by_id",
								Inputs:            []database.QueryInput{{Name: "id", Type: "customer.customer_id"}},
								Result:            "customer.customer",
								ResultCardinality: database.QueryResultZeroOrOne,
							},
						},
					},
				},
			},
		},
		Semantics: []semantic.Namespace{
			{
				ID: "customer",
				Data: []semantic.Data{
					{
						ID: "customer.customer",
						Attributes: []semantic.Attribute{
							{ID: "customer.customer.customer_id", Symbol: "customerId", TypeRef: "customer.customer_id"},
							{ID: "customer.customer.name", Symbol: "name", TypeRef: "std.text"},
						},
					},
				},
			},
		},
		ContractQuerySQLRealizations: []realization.ContractQuerySQLRealization{
			{
				Target:   realization.ContractQueryTarget{Database: "master_data", Contract: "master_data.customer", Query: "find_by_id"},
				Provider: environment.ProviderOracle,
				Source:   "select ...",
				Inputs:   []realization.ContractQueryInputParameterMapping{{Input: "id", Parameter: "customer_id"}},
				ResultColumns: []query.ResultColumnMapping{
					{Column: "CUSTOMER_ID", Attribute: "customer.customer.customer_id"},
					{Column: "NAME", Attribute: "customer.customer.name"},
				},
			},
		},
	}
}

func TestValidateContractQuerySQLRealizationsAcceptsExplicitConsistentRealization(t *testing.T) {
	diagnostics := ValidateContractQuerySQLRealizations(contractQuerySQLValidationWorld())
	if len(diagnostics) != 0 {
		t.Fatalf("整合したSQL realizationに診断が出ています: %#v", diagnostics)
	}
}

func TestValidateContractQuerySQLRealizationsRejectsUnknownTargetQuery(t *testing.T) {
	world := contractQuerySQLValidationWorld()
	world.ContractQuerySQLRealizations[0].Target.Query = "find_by_name"

	diagnostics := ValidateContractQuerySQLRealizations(world)
	if !hasCode(diagnostics, "UNKNOWN_CONTRACT_QUERY_SQL_REALIZATION_QUERY") {
		t.Fatalf("存在しないContract Query targetを拒否していません: %#v", diagnostics)
	}
}

func TestValidateContractQuerySQLRealizationsRejectsInputMappingGap(t *testing.T) {
	world := contractQuerySQLValidationWorld()
	world.ContractQuerySQLRealizations[0].Inputs = nil

	diagnostics := ValidateContractQuerySQLRealizations(world)
	if !hasCode(diagnostics, "INVALID_CONTRACT_QUERY_SQL_REALIZATION") {
		t.Fatalf("Query input→物理parameter mapping欠如を拒否していません: %#v", diagnostics)
	}
}

func TestValidateContractQuerySQLRealizationsRejectsAttributeOutsideResultData(t *testing.T) {
	world := contractQuerySQLValidationWorld()
	world.ContractQuerySQLRealizations[0].ResultColumns[1].Attribute = "sales.order.order_id"

	diagnostics := ValidateContractQuerySQLRealizations(world)
	if !hasCode(diagnostics, "CONTRACT_QUERY_SQL_RESULT_ATTRIBUTE_OUTSIDE_RESULT_DATA") {
		t.Fatalf("Query result Data外のSemantic attributeへのmappingを拒否していません: %#v", diagnostics)
	}
}

func TestValidateContractQuerySQLRealizationsRejectsDuplicateIdentity(t *testing.T) {
	world := contractQuerySQLValidationWorld()
	world.ContractQuerySQLRealizations = append(world.ContractQuerySQLRealizations, world.ContractQuerySQLRealizations[0])

	diagnostics := ValidateContractQuerySQLRealizations(world)
	if !hasCode(diagnostics, "DUPLICATE_CONTRACT_QUERY_SQL_REALIZATION") {
		t.Fatalf("同一target + providerのSQL realization重複を拒否していません: %#v", diagnostics)
	}
}

func hasCode(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

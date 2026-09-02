package validation

import (
	"testing"

	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func TestValidateDatabaseContractQuerySignaturesAcceptsFindByID(t *testing.T) {
	world := databaseContractQuerySignatureWorld(database.QueryContract{
		Name: "find_by_id",
		Inputs: []database.QueryInput{
			{Name: "id", Type: "customer.customer_id"},
		},
		Result:            "customer.customer",
		ResultCardinality: database.QueryResultZeroOrOne,
	})

	if got := ValidateDatabaseContractQuerySignatures(world); len(got) != 0 {
		t.Fatalf("正しいDatabase Contract Query signatureでDiagnosticが発生しました: %#v", got)
	}
}

func TestValidateDatabaseContractQuerySignaturesKeepsLegacyQueryValid(t *testing.T) {
	world := databaseContractQuerySignatureWorld(database.QueryContract{Name: "find_by_id"})

	if got := ValidateDatabaseContractQuerySignatures(world); len(got) != 0 {
		t.Fatalf("signature未定義の既存Queryを拒否してはいけません: %#v", got)
	}
}

func TestValidateDatabaseContractQuerySignaturesRejectsInvalidReferences(t *testing.T) {
	tests := []struct {
		name  string
		query database.QueryContract
		code  string
	}{
		{
			name: "unknown input type",
			query: database.QueryContract{
				Name: "find_by_id",
				Inputs: []database.QueryInput{
					{Name: "id", Type: "customer.missing_id"},
				},
				Result:            "customer.customer",
				ResultCardinality: database.QueryResultZeroOrOne,
			},
			code: "UNKNOWN_DATABASE_CONTRACT_QUERY_INPUT_TYPE",
		},
		{
			name: "unknown result data",
			query: database.QueryContract{
				Name: "find_by_id",
				Inputs: []database.QueryInput{
					{Name: "id", Type: "customer.customer_id"},
				},
				Result:            "customer.missing",
				ResultCardinality: database.QueryResultZeroOrOne,
			},
			code: "UNKNOWN_DATABASE_CONTRACT_QUERY_RESULT",
		},
		{
			name: "result cardinality required",
			query: database.QueryContract{
				Name: "find_by_id",
				Inputs: []database.QueryInput{
					{Name: "id", Type: "customer.customer_id"},
				},
				Result: "customer.customer",
			},
			code: "DATABASE_CONTRACT_QUERY_RESULT_CARDINALITY_REQUIRED",
		},
		{
			name: "invalid result cardinality",
			query: database.QueryContract{
				Name: "find_by_id",
				Inputs: []database.QueryInput{
					{Name: "id", Type: "customer.customer_id"},
				},
				Result:            "customer.customer",
				ResultCardinality: database.QueryResultCardinality("unknown"),
			},
			code: "INVALID_DATABASE_CONTRACT_QUERY_RESULT_CARDINALITY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnostics := ValidateDatabaseContractQuerySignatures(databaseContractQuerySignatureWorld(tt.query))
			if !containsDatabaseContractQuerySignatureDiagnostic(diagnostics, tt.code) {
				t.Fatalf("Diagnostic %sがありません: %#v", tt.code, diagnostics)
			}
			for _, diagnostic := range diagnostics {
				if diagnostic.Code != tt.code {
					continue
				}
				if diagnostic.Subject == nil || diagnostic.Subject.Kind != "database-contract-query" || diagnostic.Subject.ID != "master_data.customer/find_by_id" {
					t.Fatalf("DiagnosticがDatabase Contract Queryへ結び付いていません: %#v", diagnostic)
				}
			}
		})
	}
}

func TestValidateDatabaseContractQuerySignaturesRejectsDuplicateInputs(t *testing.T) {
	world := databaseContractQuerySignatureWorld(database.QueryContract{
		Name: "find_by_id",
		Inputs: []database.QueryInput{
			{Name: "id", Type: "customer.customer_id"},
			{Name: "id", Type: "customer.customer_id"},
		},
		Result:            "customer.customer",
		ResultCardinality: database.QueryResultZeroOrOne,
	})

	diagnostics := ValidateDatabaseContractQuerySignatures(world)
	if !containsDatabaseContractQuerySignatureDiagnostic(diagnostics, "DUPLICATE_DATABASE_CONTRACT_QUERY_INPUT") {
		t.Fatalf("重複inputを拒否していません: %#v", diagnostics)
	}
}

func databaseContractQuerySignatureWorld(query database.QueryContract) ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "customer",
				Types: []semantic.Type{
					{ID: "customer.customer_id", Symbol: "CustomerId", Kind: semantic.TypeAlias, Underlying: "std.identifier"},
				},
				Data: []semantic.Data{
					{ID: "customer.customer", Symbol: "Customer"},
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
						Queries: []database.QueryContract{query},
					},
				},
			},
		},
	}
}

func containsDatabaseContractQuerySignatureDiagnostic(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

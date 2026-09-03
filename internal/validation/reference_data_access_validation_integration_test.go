package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func TestValidateIncludesApplicationDataAccessContracts(t *testing.T) {
	tests := []struct {
		name  string
		world func() ir.World
		code  string
	}{
		{
			name: "Semantic Relation key",
			world: func() ir.World {
				return relationKeyWorld(semantic.RelationKey{
					SourceAttribute: "sales.order.order_id",
					TargetAttribute: "customer.customer.customer_id",
				})
			},
			code: "SEMANTIC_RELATION_KEY_TYPE_MISMATCH",
		},
		{
			name: "Database Contract Query signature",
			world: func() ir.World {
				return databaseContractQuerySignatureWorld(database.QueryContract{
					Name: "find_by_id",
					Inputs: []database.QueryInput{
						{Name: "id", Type: "customer.missing_id"},
					},
					Result:            "customer.customer",
					ResultCardinality: database.QueryResultZeroOrOne,
				})
			},
			code: "UNKNOWN_DATABASE_CONTRACT_QUERY_INPUT_TYPE",
		},
		{
			name: "reference invocation",
			world: func() ir.World {
				pattern := PatternReferenceInvocationFixture()
				pattern.Kind = application.PatternBrowse
				return referenceContractQueryWorld(pattern)
			},
			code: "CONTRACT_QUERY_INVOCATION_REQUIRES_REFERENCE_PATTERN",
		},
		{
			name: "reference cross-layer",
			world: func() ir.World {
				world := referenceContractQueryCrossWorld()
				world.Semantics[0].Data[0].Relations[0].Keys = nil
				return world
			},
			code: "REFERENCE_CONTRACT_QUERY_RELATION_KEY_REQUIRED",
		},
		{
			name: "header-detail Contract Query binding",
			world: func() ir.World {
				world := headerDetailContractQueryBindingWorld()
				world.Databases[0].Contracts[0].Queries[0].ResultCardinality = database.QueryResultZeroOrOne
				return world
			},
			code: "HEADER_DETAIL_BINDING_RESULT_CARDINALITY_NOT_MANY",
		},
		{
			name: "Contract Query SQL realization",
			world: func() ir.World {
				world := contractQuerySQLValidationWorld()
				world.ContractQuerySQLRealizations[0].Inputs = nil
				return world
			},
			code: "INVALID_CONTRACT_QUERY_SQL_REALIZATION",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnostics := Validate(tt.world())
			if !containsApplicationDataAccessDiagnostic(diagnostics, tt.code) {
				t.Fatalf("通常のValidate入口からDiagnostic %sを取得できません: %#v", tt.code, diagnostics)
			}
		})
	}
}

func containsApplicationDataAccessDiagnostic(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// The remaining resolution codes: a reference Pattern whose datasource points
// at a Database or Contract that is declared twice, and the version agreement
// between a header-detail binding and its Contract (#4032).
func TestReferenceAndHeaderDetailResolutionCodesAreProduced(t *testing.T) {
	t.Run("ambiguous reference datasource database", func(t *testing.T) {
		world := referenceContractQueryWorld(validReferencePattern())
		world.Databases = append(world.Databases, world.Databases[0])
		assertHasDiagnostic(t, ValidateReferenceContractQueryInvocations(world), "AMBIGUOUS_REFERENCE_CONTRACT_QUERY_DATABASE")
	})

	t.Run("ambiguous reference contract", func(t *testing.T) {
		world := referenceContractQueryWorld(validReferencePattern())
		world.Databases[0].Contracts = append(world.Databases[0].Contracts, world.Databases[0].Contracts[0])
		assertHasDiagnostic(t, ValidateReferenceContractQueryInvocations(world), "AMBIGUOUS_REFERENCE_CONTRACT_QUERY_CONTRACT")
	})

	t.Run("header-detail contract major version mismatch", func(t *testing.T) {
		world := headerDetailContractQueryBindingWorld()
		world.Databases[0].Contracts[0].Version = database.ContractVersion{Major: 2}
		assertHasDiagnostic(t, ValidateHeaderDetailContractQueryBindings(world), "HEADER_DETAIL_BINDING_CONTRACT_MAJOR_VERSION_MISMATCH")
	})

	t.Run("effect rule changes unknown data", func(t *testing.T) {
		world := effectRuleWorld()
		world.Semantics[0].EffectRules[0].EffectData = "sales.missing"
		assertHasDiagnostic(t, Validate(world), "EFFECT_RULE_UNKNOWN_DATA")
	})

	t.Run("effect rule delegates to unknown capability", func(t *testing.T) {
		world := effectRuleWorld()
		world.Semantics[0].EffectRules[0].ViaCapability = "sales.missing"
		assertHasDiagnostic(t, Validate(world), "EFFECT_RULE_UNKNOWN_VIA")
	})
}

func validReferencePattern() application.Pattern {
	return application.Pattern{
		ID:         "order_entry.customer",
		Kind:       application.PatternReference,
		Datasource: "master",
		ContractQuery: &application.ContractQueryInvocation{
			Contract: "master_data.customer",
			Query:    "find_by_id",
			Inputs: []application.ContractQueryInputBinding{
				{TargetAttribute: core.CanonicalID("customer.customer.customer_id"), Input: core.LocalName("id")},
			},
		},
	}
}

func unusedReferenceWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Types: []semantic.Type{
					{ID: "sales.order_id", Symbol: "OrderId", Kind: semantic.TypeAlias, Underlying: "std.identifier"},
				},
				Data: []semantic.Data{{ID: "sales.order_line"}},
			},
			{ID: "std", Types: []semantic.Type{{ID: "std.identifier", Symbol: "Identifier", Kind: semantic.TypePrimitive}}},
		},
		Databases: []database.Database{
			{
				ID: "sales_core",
				Contracts: []database.Contract{
					{
						ID:      "sales_core.order_line",
						Version: database.ContractVersion{Major: 1},
						Queries: []database.QueryContract{
							{
								Name:              "find_by_order",
								Inputs:            []database.QueryInput{{Name: "order_id", Type: "sales.order_id"}},
								Result:            "sales.order_line",
								ResultCardinality: database.QueryResultMany,
							},
						},
					},
				},
			},
		},
		Applications: []application.Application{
			{
				ID: "order_entry",
				Databases: []application.DatabaseRequirement{
					{
						Alias:          "sales",
						TargetDatabase: "sales_core",
						Access:         database.AccessRead,
						RequiredContracts: []database.ContractRequirement{
							{ContractID: "sales_core.order_line", CompatibleMajor: 1},
						},
					},
				},
				Patterns: []application.Pattern{
					{
						ID: "order_entry.order_lines", Kind: application.PatternReference,
						Datasource: "sales",
						ContractQuery: &application.ContractQueryInvocation{
							Contract: "sales_core.order_line",
							Query:    "find_by_order",
							Inputs: []application.ContractQueryInputBinding{
								{TargetAttribute: core.CanonicalID("sales.order_line.order_id"), Input: core.LocalName("order_id")},
							},
						},
					},
				},
			},
		},
	}
}

func effectRuleWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Capabilities: []semantic.Capability{
					{ID: "sales.submit", Symbol: "submit"},
					{ID: "sales.notify", Symbol: "notify"},
				},
				Data: []semantic.Data{{ID: "sales.order"}},
				EffectRules: []semantic.EffectRule{
					{
						ID:            "sales.submit.notify",
						OnCapability:  "sales.submit",
						Stage:         "after-commit",
						EffectData:    "sales.order",
						ViaCapability: "sales.notify",
					},
				},
			},
		},
	}
}

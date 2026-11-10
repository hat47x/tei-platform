package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
)

// A reference Pattern declares the Database Contract Query it invokes. The
// validator reports one code per missing or malformed part of that
// declaration, so each code is produced here by breaking one part of an
// accepted world (#4032).
func TestReferenceContractQueryCodesAreProduced(t *testing.T) {
	valid := func() ir.World {
		return ir.World{
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
									{TargetAttribute: "sales.order.order_id", Input: "order_id"},
								},
							},
						},
					},
				},
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
									Name:   "find_by_order",
									Inputs: []database.QueryInput{{Name: "order_id", Type: "std.identifier"}},
									Result: "sales.order_line",
								},
							},
						},
					},
				},
			},
		}
	}

	cases := []struct {
		name   string
		mutate func(*ir.World)
		want   string
	}{
		{
			name: "contract required",
			mutate: func(w *ir.World) {
				w.Applications[0].Patterns[0].ContractQuery.Contract = ""
			},
			want: "REFERENCE_CONTRACT_QUERY_CONTRACT_REQUIRED",
		},
		{
			name: "contract malformed",
			mutate: func(w *ir.World) {
				w.Applications[0].Patterns[0].ContractQuery.Contract = core.CanonicalID("bad contract")
			},
			want: "INVALID_REFERENCE_CONTRACT_QUERY_CONTRACT",
		},
		{
			name: "query name required",
			mutate: func(w *ir.World) {
				w.Applications[0].Patterns[0].ContractQuery.Query = ""
			},
			want: "REFERENCE_CONTRACT_QUERY_NAME_REQUIRED",
		},
		{
			name: "query name malformed",
			mutate: func(w *ir.World) {
				w.Applications[0].Patterns[0].ContractQuery.Query = core.LocalName("bad query")
			},
			want: "INVALID_REFERENCE_CONTRACT_QUERY_NAME",
		},
		{
			name: "datasource required",
			mutate: func(w *ir.World) {
				w.Applications[0].Patterns[0].Datasource = ""
			},
			want: "REFERENCE_CONTRACT_QUERY_DATASOURCE_REQUIRED",
		},
		{
			name: "target attribute required",
			mutate: func(w *ir.World) {
				w.Applications[0].Patterns[0].ContractQuery.Inputs[0].TargetAttribute = ""
			},
			want: "REFERENCE_CONTRACT_QUERY_TARGET_ATTRIBUTE_REQUIRED",
		},
		{
			name: "input required",
			mutate: func(w *ir.World) {
				w.Applications[0].Patterns[0].ContractQuery.Inputs[0].Input = ""
			},
			want: "REFERENCE_CONTRACT_QUERY_INPUT_REQUIRED",
		},
		{
			name: "input malformed",
			mutate: func(w *ir.World) {
				w.Applications[0].Patterns[0].ContractQuery.Inputs[0].Input = core.LocalName("bad input")
			},
			want: "INVALID_REFERENCE_CONTRACT_QUERY_INPUT",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			world := valid()
			testCase.mutate(&world)
			for _, diagnostic := range ValidateReferenceContractQueryInvocations(world) {
				if diagnostic.Code == testCase.want {
					return
				}
			}
			var codes []string
			for _, diagnostic := range ValidateReferenceContractQueryInvocations(world) {
				codes = append(codes, diagnostic.Code)
			}
			t.Fatalf("expected %s, got %v", testCase.want, codes)
		})
	}
}

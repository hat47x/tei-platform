package validation

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/semantic"
)

// A Database Contract Query that declares a signature is checked field by
// field. These codes are the contract a Query author reads, so each one is
// produced here by breaking one field of an accepted declaration (#4032).
func TestDatabaseContractQuerySignatureCodesAreProduced(t *testing.T) {
	valid := func() ir.World {
		return ir.World{
			Semantics: []semantic.Namespace{
				{
					ID: "sales",
					Types: []semantic.Type{
						{ID: "sales.order_id", Symbol: "OrderId", Kind: semantic.TypeAlias, Underlying: "std.identifier"},
					},
					Data: []semantic.Data{
						{ID: "sales.order_line"},
					},
				},
				{
					ID:    "std",
					Types: []semantic.Type{{ID: "std.identifier", Symbol: "Identifier", Kind: semantic.TypePrimitive}},
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
		}
	}

	cases := []struct {
		name   string
		mutate func(*ir.World)
		want   string
	}{
		{
			name: "input name required",
			mutate: func(w *ir.World) {
				w.Databases[0].Contracts[0].Queries[0].Inputs[0].Name = ""
			},
			want: "DATABASE_CONTRACT_QUERY_INPUT_NAME_REQUIRED",
		},
		{
			name: "input name malformed",
			mutate: func(w *ir.World) {
				w.Databases[0].Contracts[0].Queries[0].Inputs[0].Name = core.LocalName("bad name")
			},
			want: "INVALID_DATABASE_CONTRACT_QUERY_INPUT_NAME",
		},
		{
			name: "input type required",
			mutate: func(w *ir.World) {
				w.Databases[0].Contracts[0].Queries[0].Inputs[0].Type = ""
			},
			want: "DATABASE_CONTRACT_QUERY_INPUT_TYPE_REQUIRED",
		},
		{
			name: "result required",
			mutate: func(w *ir.World) {
				w.Databases[0].Contracts[0].Queries[0].Result = ""
			},
			want: "DATABASE_CONTRACT_QUERY_RESULT_REQUIRED",
		},
		{
			name: "duplicate input",
			mutate: func(w *ir.World) {
				query := &w.Databases[0].Contracts[0].Queries[0]
				query.Inputs = append(query.Inputs, query.Inputs[0])
			},
			want: "DUPLICATE_DATABASE_CONTRACT_QUERY_INPUT",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			world := valid()
			testCase.mutate(&world)
			for _, diagnostic := range ValidateDatabaseContractQuerySignatures(world) {
				if diagnostic.Code == testCase.want {
					return
				}
			}
			var codes []string
			for _, diagnostic := range ValidateDatabaseContractQuerySignatures(world) {
				codes = append(codes, diagnostic.Code)
			}
			t.Fatalf("expected %s, got %v", testCase.want, codes)
		})
	}
}

// The header-detail binding maps one source attribute and one Query input to
// each input binding. Reusing either one is reported, because the mapping would
// be ambiguous (#4032).
func TestHeaderDetailBindingDuplicateInputCodesAreProduced(t *testing.T) {
	cases := []struct {
		name    string
		inputs  []struct{ attribute, input string }
		want    string
		comment string
	}{
		{
			name: "duplicate source attribute",
			inputs: []struct{ attribute, input string }{
				{"sales.order.order_id", "order_id"},
				{"sales.order.order_id", "customer_id"},
			},
			want:    "DUPLICATE_HEADER_DETAIL_BINDING_SOURCE_ATTRIBUTE",
			comment: "one attribute cannot feed two Query inputs",
		},
		{
			name: "duplicate query input",
			inputs: []struct{ attribute, input string }{
				{"sales.order.order_id", "order_id"},
				{"sales.order.customer_id", "order_id"},
			},
			want:    "DUPLICATE_HEADER_DETAIL_BINDING_INPUT",
			comment: "one Query input cannot be filled twice",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			world := headerDetailContractQueryBindingWorld()
			world.HeaderDetailContractQueryBindings[0].Inputs = nil
			for _, input := range testCase.inputs {
				world.HeaderDetailContractQueryBindings[0].Inputs = append(
					world.HeaderDetailContractQueryBindings[0].Inputs,
					realization.HeaderDetailContractQueryInputBinding{SourceAttribute: core.CanonicalID(input.attribute), Input: core.LocalName(input.input)},
				)
			}
			found := false
			for _, diagnostic := range ValidateHeaderDetailContractQueryBindings(world) {
				if diagnostic.Code == testCase.want {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("%s: expected %s", testCase.comment, testCase.want)
			}
		})
	}
}

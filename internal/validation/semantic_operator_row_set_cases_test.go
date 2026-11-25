package validation

import (
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

// A row-set example states one table of rows and one answer. The validator
// checks every cell of every row and the answer word, and refuses an operator
// whose examples use both forms.
//
// Before this, the one-value checks read Value and Inputs, which a row set does
// not set, so a table validated as an empty value and a real row-set example
// could never be stated at all.

func rowSetCaseWorld() ir.World {
	return ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Operators: []semantic.Operator{{
			ID: "sales.every_line_at_most", Symbol: "every_line_at_most",
			Input: semantic.OperatorInputNumber,
			Inputs: []semantic.OperatorInputDeclaration{
				{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
				{Name: "limit", Type: semantic.OperatorInputNumber, OfEachRow: true},
			},
			RefutesOnSomeRows: true,
			Cases: []semantic.OperatorCase{
				{Rows: []map[string]string{
					{"value": "2", "limit": "10"},
					{"value": "5", "limit": "5"},
				}, RowExpect: semantic.RowExpectSatisfied},
				{Rows: []map[string]string{
					{"value": "2", "limit": "10"},
					{"value": "5", "limit": "4"},
				}, RowExpect: semantic.RowExpectNotSatisfied},
			},
		}},
	}}}
}

func TestARowSetExampleIsAccepted(t *testing.T) {
	if diagnostics := Validate(rowSetCaseWorld()); len(diagnostics) != 0 {
		t.Fatalf("the baseline row-set world is rejected: %+v", diagnostics)
	}
}

func TestRowSetExampleRefusals(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ir.World)
		want   string
	}{
		{"a cell that is not a number", func(w *ir.World) {
			w.Semantics[0].Operators[0].Cases[0].Rows[1]["limit"] = "none"
		}, "INVALID_SEMANTIC_OPERATOR_CASE"},
		{"an empty cell", func(w *ir.World) {
			w.Semantics[0].Operators[0].Cases[0].Rows[0]["value"] = ""
		}, "INVALID_SEMANTIC_OPERATOR_CASE"},
		{"an answer this reader does not know", func(w *ir.World) {
			w.Semantics[0].Operators[0].Cases[0].RowExpect = "undecided"
		}, "INVALID_SEMANTIC_OPERATOR_CASE"},
		{"the same table twice", func(w *ir.World) {
			w.Semantics[0].Operators[0].Cases[1].RowExpect = semantic.RowExpectSatisfied
			w.Semantics[0].Operators[0].Cases[1].Rows = []map[string]string{
				{"value": "2", "limit": "10"},
				{"value": "5", "limit": "5"},
			}
		}, "DUPLICATE_SEMANTIC_OPERATOR_CASE"},
		{"a row-set example beside a one-record example", func(w *ir.World) {
			w.Semantics[0].Operators[0].Cases = append(w.Semantics[0].Operators[0].Cases,
				semantic.OperatorCase{Value: "5", Satisfied: true})
		}, "TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := rowSetCaseWorld()
			tc.mutate(&world)
			var codes []string
			found := false
			for _, diagnostic := range Validate(world) {
				codes = append(codes, diagnostic.Code)
				found = found || diagnostic.Code == tc.want &&
					diagnostic.Subject != nil && diagnostic.Subject.ID == "sales.every_line_at_most"
			}
			if !found {
				t.Fatalf("want %s on sales.every_line_at_most, got %v", tc.want, codes)
			}
		})
	}
}

// A repeated row inside one table is two rows, not two examples, so the table
// is still one example and is accepted.
func TestARowSetExampleKeepsARepeatedRow(t *testing.T) {
	world := rowSetCaseWorld()
	world.Semantics[0].Operators[0].Cases = []semantic.OperatorCase{{
		Rows: []map[string]string{
			{"value": "5", "limit": "4"},
			{"value": "5", "limit": "4"},
		}, RowExpect: semantic.RowExpectNotSatisfied,
	}}
	for _, diagnostic := range Validate(world) {
		if strings.Contains(diagnostic.Code, "OPERATOR_CASE") {
			t.Fatalf("a table with two equal rows was refused: %s %s", diagnostic.Code, diagnostic.Message)
		}
	}
}

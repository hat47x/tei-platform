//go:build !tei_order_only && !tei_repair_only

package operations

import (
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// A selection that resolves by name can still read a different *shape* than the
// operator declares: an operator whose input reads every row of a relation
// asked of an implementation that decides from one record's values, or the
// reverse. The evaluator refuses both when it is asked, so validate must refuse
// them without waiting for a read.
//
// This check had no test. Breaking it in CheckImplementationMatchesContract
// (`if rowScoped && !hasRows` -> `if false && ...`) left the suite green, so the
// refusal could have been dropped without anyone noticing. It matters more now
// that a row-scoped implementation ships (#5916): a Workspace can select it.
func rowScopedSelectionCheckAvailable() []validationeval.NamedImplementation {
	return []validationeval.NamedImplementation{
		{ID: "test.number", Input: semantic.OperatorInputNumber,
			Number: func(*big.Rat) (bool, error) { return true, nil }},
		{ID: "test.rows", Input: semantic.OperatorInputNumber,
			NumberRows: func(map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
				return validationeval.TriYes, nil
			}},
	}
}

// rowsContract declares one operator whose single input reads every row.
func rowsContract(id core.CanonicalID) semantic.Operator {
	return semantic.Operator{
		ID: id, Symbol: "all_rows", Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
		},
	}
}

func TestCheckOperatorSelectionRefusesARowReaderForAnOperatorOfOneRecord(t *testing.T) {
	// The reverse of the case above: the operator's input does not read every
	// row, so its answer is about one record. A row-reading implementation
	// would answer a question about the relation's rows instead.
	loaded := selectionCheckLoaded(
		selectionCheckWorld(semantic.Operator{
			ID: "stock.one_record", Symbol: "one_record", Input: semantic.OperatorInputNumber,
			Inputs: []semantic.OperatorInputDeclaration{
				{Name: "value", Type: semantic.OperatorInputNumber},
			},
		}),
		[]workspace.OperatorImplementationSelection{{Operator: "stock.one_record", Implementation: "test.rows"}},
	)
	codes := selectionCheckCodes(loaded)
	if len(codes) != 1 || codes[0] != OperatorSelectionUnresolvedCode {
		t.Fatalf("codes=%v, want exactly the selection finding", codes)
	}
	finding := CheckOperatorSelection(loaded, sourceindex.Index{}, rowScopedSelectionCheckAvailable())[0]
	if !strings.Contains(finding.Diagnostic.Message, "rows of a relation") {
		t.Fatalf("the finding does not say the operator reads no rows: %q", finding.Diagnostic.Message)
	}
}

func TestCheckOperatorSelectionRefusesARowScopedOperatorJudgedFromOneRecord(t *testing.T) {
	loaded := selectionCheckLoaded(
		selectionCheckWorld(rowsContract("stock.rows_op")),
		[]workspace.OperatorImplementationSelection{{Operator: "stock.rows_op", Implementation: "test.number"}},
	)
	codes := selectionCheckCodes(loaded)
	if len(codes) != 1 || codes[0] != OperatorSelectionUnresolvedCode {
		t.Fatalf("codes=%v, want exactly the selection finding", codes)
	}
	finding := CheckOperatorSelection(loaded, sourceindex.Index{}, rowScopedSelectionCheckAvailable())[0]
	if finding.Diagnostic.Message == "" {
		t.Fatalf("the finding carries no message")
	}
}

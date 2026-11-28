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

// everyDayNotBefore declares an operator whose input reads a column of days.
func everyDayNotBefore() semantic.Operator {
	return semantic.Operator{
		ID: "stock.every_day_not_before", Symbol: "every_day_not_before",
		Input: semantic.OperatorInputDate,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputDate, OfEachRow: true},
			{Name: "deadline", Type: semantic.OperatorInputDate},
		},
	}
}

// A Workspace selects a row implementation by name, so the type of the column it
// decides is what keeps a Workspace from being answered from a value nobody
// read: a contract that reads a column of days is not decided by the
// implementation that decides a column of numbers, and validate must refuse the
// selection rather than wait for a read.
func TestCheckOperatorSelectionRefusesANumberRowImplementationForAColumnOfDays(t *testing.T) {
	contract := everyDayNotBefore()
	loaded := selectionCheckLoaded(
		selectionCheckWorld(contract),
		[]workspace.OperatorImplementationSelection{{Operator: string(contract.ID), Implementation: "tei.number.every-at-most"}},
	)
	findings := CheckOperatorSelection(loaded, sourceindex.Index{}, validationeval.BuiltinImplementations())
	if len(findings) != 1 {
		t.Fatalf("findings=%+v, want exactly the selection finding", findings)
	}
	message := findings[0].Diagnostic.Message
	if !strings.Contains(message, "date") || !strings.Contains(message, "number") {
		t.Fatalf("the finding does not name the two types: %q", message)
	}
}

// The same selection of the implementation that decides a column of days is
// honoured, so a Workspace that states it passes validate.
func TestCheckOperatorSelectionAcceptsTheShippedImplementationOfAColumnOfDays(t *testing.T) {
	contract := everyDayNotBefore()
	contract.RowCases = []semantic.OperatorRowCase{{
		Rows: []map[string]string{
			{"value": "2026-04-01", "deadline": "2026-04-01"},
			{"value": "2026-03-31", "deadline": "2026-04-05"},
		},
		Answer: semantic.OperatorRowNotSatisfied,
	}}
	loaded := selectionCheckLoaded(
		selectionCheckWorld(contract),
		[]workspace.OperatorImplementationSelection{{Operator: string(contract.ID), Implementation: "tei.day.every-not-before"}},
	)
	if findings := CheckOperatorSelection(loaded, sourceindex.Index{}, validationeval.BuiltinImplementations()); len(findings) != 0 {
		t.Fatalf("findings=%+v, want none", findings)
	}
}

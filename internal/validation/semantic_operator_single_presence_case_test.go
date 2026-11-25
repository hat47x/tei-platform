package validation

import (
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

// A single-input operator that reads presence states one of two words, so the
// validator reads the single-value form's cell against that input's type. The
// expectations are written from what the declaration says the input reads, not
// from the code.
func TestASinglePresenceInputExampleIsReadByItsType(t *testing.T) {
	world := func(cell string) ir.World {
		return ir.World{Semantics: []semantic.Namespace{{
			ID: "sales",
			Operators: []semantic.Operator{{
				ID: "sales.item_present", Symbol: "item_present",
				Input: semantic.OperatorInputPresence,
				Cases: []semantic.OperatorCase{{Inputs: map[string]string{"value": cell}, Satisfied: true}},
			}},
		}}}
	}
	for _, cell := range []string{semantic.OperatorCasePresencePresent, semantic.OperatorCasePresenceAbsent} {
		if diagnostics := Validate(world(cell)); len(diagnostics) != 0 {
			t.Fatalf("cell %q was refused: %+v", cell, diagnostics)
		}
	}
	// A decimal is not a presence word, and a word that is neither is refused
	// at the cell. Both name what the cell has to be.
	for _, cell := range []string{"5", "yes", "true"} {
		var found bool
		for _, diagnostic := range Validate(world(cell)) {
			if diagnostic.Code != "INVALID_SEMANTIC_OPERATOR_CASE" {
				continue
			}
			found = true
			if !strings.Contains(diagnostic.Message, semantic.OperatorCasePresencePresent) ||
				!strings.Contains(diagnostic.Message, semantic.OperatorCasePresenceAbsent) {
				t.Fatalf("the finding does not name the two words: %q", diagnostic.Message)
			}
		}
		if !found {
			t.Fatalf("cell %q was accepted for a presence input", cell)
		}
	}
}

// A single-input number operator keeps the decimal spelling. The presence words
// are not read as decimals, so a word in a number cell is refused the way it
// was before.
func TestASingleNumberInputExampleKeepsTheDecimal(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Operators: []semantic.Operator{{
			ID: "sales.non_negative", Symbol: "non_negative",
			Input: semantic.OperatorInputNumber,
			Cases: []semantic.OperatorCase{{Inputs: map[string]string{"value": "present"}, Satisfied: true}},
		}},
	}}}
	var found bool
	for _, diagnostic := range Validate(world) {
		if diagnostic.Code == "INVALID_SEMANTIC_OPERATOR_CASE" {
			found = true
		}
	}
	if !found {
		t.Fatal("a presence word in a number cell was accepted")
	}
}

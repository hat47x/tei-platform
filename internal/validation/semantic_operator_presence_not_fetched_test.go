package validation

import (
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

// Section 8.3 says the writing for "not read" does not depend on the input's
// type. A presence input's cell otherwise states one of two words, so the shared
// word has to be read as the value nobody read rather than refused as a third
// presence word.
//
// The expectations are drawn from that section's table, not from the code.
func TestAPresenceInputExampleCarriesTheNotReadWord(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Operators: []semantic.Operator{{
			ID: "sales.holds", Symbol: "holds",
			Input: semantic.OperatorInputPresence,
			Cases: []semantic.OperatorCase{{
				Inputs:    map[string]string{"value": "not-fetched"},
				Satisfied: false,
			}},
		}},
	}}}
	if diagnostics := Validate(world); len(diagnostics) != 0 {
		t.Fatalf("the not-read word was refused in a presence cell: %+v", diagnostics)
	}
}

// The two presence words keep their own refusal: a decimal in a presence cell
// is not a value that input can read.
func TestAPresenceInputExampleStillRefusesADecimal(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Operators: []semantic.Operator{{
			ID: "sales.holds", Symbol: "holds",
			Input: semantic.OperatorInputPresence,
			Cases: []semantic.OperatorCase{{
				Inputs:    map[string]string{"value": "5"},
				Satisfied: true,
			}},
		}},
	}}}
	var found bool
	for _, diagnostic := range Validate(world) {
		if diagnostic.Code != "INVALID_SEMANTIC_OPERATOR_CASE" {
			continue
		}
		found = true
		if !strings.Contains(diagnostic.Message, semantic.OperatorCasePresencePresent) {
			t.Fatalf("the refusal does not name the words the input reads: %s", diagnostic.Message)
		}
	}
	if !found {
		t.Fatal("a decimal was accepted in a presence cell")
	}
}

// The multi-input form reads the shared word the same way, so a table that
// names its inputs can state that a presence input was not read.
func TestAPresenceInputInAMultiInputExampleCarriesTheNotReadWord(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Operators: []semantic.Operator{{
			ID: "sales.mixed", Symbol: "mixed",
			Input: semantic.OperatorInputNumber,
			Inputs: []semantic.OperatorInputDeclaration{
				{Name: "value", Type: semantic.OperatorInputNumber},
				{Name: "limit", Type: semantic.OperatorInputPresence},
			},
			Cases: []semantic.OperatorCase{{
				Inputs:    map[string]string{"value": "5", "limit": "not-fetched"},
				Satisfied: false,
			}},
		}},
	}}}
	if diagnostics := Validate(world); len(diagnostics) != 0 {
		t.Fatalf("the not-read word was refused in a presence cell: %+v", diagnostics)
	}
}

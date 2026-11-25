package validation

import (
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

// The presence-word refusal at the single-value form must be reachable: with it
// removed a decimal reaches the Workspace as if it were a presence cell. This
// test names that boundary rather than the code that implements it.
func TestASinglePresenceInputRefusesAWordItDoesNotKnow(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Operators: []semantic.Operator{{
			ID: "sales.item_present", Symbol: "item_present",
			Input: semantic.OperatorInputPresence,
			Cases: []semantic.OperatorCase{{Inputs: map[string]string{"value": "yes"}, Satisfied: true}},
		}},
	}}}
	var found bool
	for _, diagnostic := range Validate(world) {
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
		t.Fatal("a word that is neither present nor absent was accepted")
	}
}

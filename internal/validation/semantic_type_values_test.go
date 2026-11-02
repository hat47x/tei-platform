package validation_test

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/validation"
)

func TestSemanticTypeValuesAreNonEmptyAndUnique(t *testing.T) {
	world := ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Types: []semantic.Type{{
				ID:         "sales.status",
				Symbol:     "Status",
				Kind:       semantic.TypeAlias,
				Underlying: "std.code",
				Values:     []string{"open", "open", ""},
			}},
		}},
	}
	codes := map[string]bool{}
	for _, diagnostic := range validation.Validate(world) {
		codes[diagnostic.Code] = true
	}
	if !codes["SEMANTIC_TYPE_VALUE_DUPLICATE"] || !codes["SEMANTIC_TYPE_VALUE_EMPTY"] {
		t.Fatalf("codes=%v", codes)
	}
}

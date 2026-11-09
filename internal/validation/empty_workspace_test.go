package validation_test

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/validation"
)

// TestEmptyWorkspaceGetsAWarningButNoError pins #2822: ok=true alone must not
// be read as "model declarations were loaded and checked". The standard std
// prelude is present in every World, so it is excluded.
func TestEmptyWorkspaceGetsAWarningButNoError(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{ID: "std", Types: []semantic.Type{{ID: "std.text", Kind: semantic.TypePrimitive}}}}}
	ds := validation.Validate(world)
	found := false
	for _, d := range ds {
		if d.Code == "WORKSPACE_DECLARES_NO_MODEL_ELEMENTS" {
			found = true
			if d.Severity != validation.Warning {
				t.Fatalf("empty workspace severity=%s want warning", d.Severity)
			}
		}
	}
	if !found {
		t.Fatalf("empty workspace produced no warning: %#v", ds)
	}
	if validation.HasErrors(ds) {
		t.Fatalf("empty workspace warning was treated as an error: %#v", ds)
	}
}

// TestWorkspaceWithDeclarationsHasNoEmptinessWarning is the positive control.
func TestWorkspaceWithDeclarationsHasNoEmptinessWarning(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{
		{ID: "std", Types: []semantic.Type{{ID: "std.text", Kind: semantic.TypePrimitive}}},
		{ID: "sales", Data: []semantic.Data{{ID: "sales.order"}}},
	}}
	for _, d := range validation.Validate(world) {
		if d.Code == "WORKSPACE_DECLARES_NO_MODEL_ELEMENTS" {
			t.Fatalf("declared workspace got an emptiness warning: %#v", d)
		}
	}
}

package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
)

// recordPatternWithHook builds a world with one record pattern that declares one
// hook, so a scope diagnostic can be read from a real pattern.
//
// A record pattern is used because the hook declaration is checked on that pattern
// path; a browse pattern's interaction carries the item list, not the hooks.
func recordPatternWithHook(scope string) ir.World {
	world := validApplicationPatternWorld()
	world.Applications[0].Patterns = []application.Pattern{{
		ID:   "order_entry.order_record",
		Kind: application.PatternRecord,
		Data: "sales.order",
		Interaction: &application.Interaction{
			Hooks: []application.Hook{{Phase: "before-update", Scope: scope}},
		},
	}}
	return world
}

func hookScopeCodes(w ir.World) []string {
	var out []string
	for _, diagnostic := range validateApplicationPatterns(w) {
		switch diagnostic.Code {
		case "APPLICATION_PATTERN_INTERACTION_HOOK_SCOPE_MISMATCH",
			"APPLICATION_PATTERN_INTERACTION_HOOK_SCOPE_UNSUPPORTED":
			out = append(out, diagnostic.Code)
		}
	}
	return out
}

// TestHookScopeFitsPattern checks the mapping directly: each kind names the units
// its contract already works with, so a scope the kind does not look at is refused.
func TestHookScopeFitsPattern(t *testing.T) {
	cases := []struct {
		kind  application.PatternKind
		scope string
		want  bool
	}{
		{application.PatternBrowse, "row", true},
		{application.PatternBrowse, "record", true},
		{application.PatternBrowse, "header", false},
		// browse and header-detail cannot declare an interaction at all, so their
		// entries document the mapping, not a reachable declaration.
		{application.PatternRecord, "record", true},
		// A record pattern can also decide per row: RunRows applies a row scope, and
		// the interaction contract exists only on record patterns, so refusing row
		// here would make the whole row scope undeclarable.
		{application.PatternRecord, "row", true},
		{application.PatternHeaderDetail, "header", true},
		{application.PatternHeaderDetail, "row", true},
		{application.PatternHeaderDetail, "record", false},
		{application.PatternCommand, "item", true},
		{application.PatternCommand, "row", false},
		{application.PatternReference, "record", true},
	}
	for _, c := range cases {
		got := scopeFitsPattern(c.kind, c.scope)
		if got != c.want {
			t.Fatalf("kind=%s scope=%s got=%v want=%v", c.kind, c.scope, got, c.want)
		}
	}
}

// TestHookScopeAcceptsAMatchingUnit checks that a scope the pattern looks at
// produces no diagnostic.
func TestHookScopeAcceptsAMatchingUnit(t *testing.T) {
	for _, scope := range []string{"record", "row"} {
		if codes := hookScopeCodes(recordPatternWithHook(scope)); len(codes) != 0 {
			t.Fatalf("scope %s: codes=%v", scope, codes)
		}
	}
	// A hook without a scope keeps working: the pattern's own unit is the default.
	if codes := hookScopeCodes(recordPatternWithHook("")); len(codes) != 0 {
		t.Fatalf("no scope: codes=%v", codes)
	}
}

// TestHookScopeRefusesAMismatch checks the diagnostic: a header-scoped hook on a
// browse pattern is refused, because the browse pattern never looks at a header.
func TestHookScopeRefusesAMismatch(t *testing.T) {
	codes := hookScopeCodes(recordPatternWithHook("header"))
	if len(codes) != 1 || codes[0] != "APPLICATION_PATTERN_INTERACTION_HOOK_SCOPE_MISMATCH" {
		t.Fatalf("codes=%v", codes)
	}
}

// TestHookScopeRefusesAnUnsupportedWord checks that a word outside the vocabulary is
// refused, so a misspelling does not read as "no scope".
func TestHookScopeRefusesAnUnsupportedWord(t *testing.T) {
	codes := hookScopeCodes(recordPatternWithHook("column"))
	if len(codes) != 1 || codes[0] != "APPLICATION_PATTERN_INTERACTION_HOOK_SCOPE_UNSUPPORTED" {
		t.Fatalf("codes=%v", codes)
	}
}

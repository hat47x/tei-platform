package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
)

// The application Pattern validator has one code per rule. These cases take a
// fixture that already passes and break one rule, so the code that answers to
// that rule is produced rather than merely declared (#4032).
func TestApplicationPatternCodesAreProduced(t *testing.T) {
	t.Run("aggregates belong to header-detail only", func(t *testing.T) {
		w := headerDetailPatternWorld()
		pattern := &w.Applications[0].Patterns[0]
		pattern.Kind = application.PatternBrowse
		pattern.Aggregates = []application.Aggregate{{Name: "total", Function: "sum", Attribute: "sales.order.amount"}}
		assertHasDiagnostic(t, validateApplicationPatterns(w), "APPLICATION_PATTERN_AGGREGATES_NOT_ALLOWED")
	})

	t.Run("aggregate name required", func(t *testing.T) {
		w := headerDetailPatternWorld()
		pattern := &w.Applications[0].Patterns[0]
		pattern.Aggregates = []application.Aggregate{{Function: "sum", Attribute: "sales.order.amount"}}
		assertHasDiagnostic(t, validateApplicationPatterns(w), "APPLICATION_PATTERN_AGGREGATE_NAME_REQUIRED")
	})

	t.Run("filter operators required", func(t *testing.T) {
		w := filterWorld()
		w.Applications[0].Patterns[0].Filters[0].Operators = nil
		assertHasDiagnostic(t, validateApplicationPatterns(w), "APPLICATION_PATTERN_FILTER_OPERATORS_REQUIRED")
	})

	t.Run("hook produce must be known", func(t *testing.T) {
		w := hookWorld()
		interactionRecordPattern(&w).Interaction.Hooks[0].Produces = []core.CanonicalID{"sales.order.missing"}
		assertHasDiagnostic(t, validateApplicationPatterns(w), "UNKNOWN_APPLICATION_PATTERN_INTERACTION_HOOK_PRODUCE")
	})

	t.Run("hook navigation must be known", func(t *testing.T) {
		w := hookWorld()
		interactionRecordPattern(&w).Interaction.Navigations = []application.InteractionNavigation{{From: "order_entry.missing"}}
		assertHasDiagnostic(t, validateApplicationPatterns(w), "UNKNOWN_APPLICATION_PATTERN_INTERACTION_NAVIGATION")
	})
}

func assertHasDiagnostic(t *testing.T, diagnostics []Diagnostic, want string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == want {
			return
		}
	}
	var codes []string
	for _, diagnostic := range diagnostics {
		codes = append(codes, diagnostic.Code)
	}
	t.Fatalf("expected %s, got %v", want, codes)
}

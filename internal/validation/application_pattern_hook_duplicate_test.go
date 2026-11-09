package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
)

func TestValidateInteractionHookRejectsDuplicatePhase(t *testing.T) {
	w := hookWorld()
	pattern := interactionRecordPattern(&w)
	pattern.Interaction.Hooks = append(pattern.Interaction.Hooks, application.Hook{
		Phase:   "before-update",
		Reads:   []core.CanonicalID{"sales.order.order_id"},
		Failure: "reject",
	})
	if diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_HOOK_PHASE_DUPLICATE") == nil {
		t.Fatal("two hooks with the same phase were accepted")
	}
}

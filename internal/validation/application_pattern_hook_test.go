package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func hookWorld() ir.World {
	w := applicationPatternInteractionWorld()
	w.Semantics[0].Capabilities = append(w.Semantics[0].Capabilities, semantic.Capability{ID: "sales.submit", Symbol: "submit"})
	pattern := interactionRecordPattern(&w)
	pattern.Interaction.Hooks = []application.Hook{{
		Phase:    "before-update",
		Reads:    []core.CanonicalID{"sales.order.order_id"},
		Uses:     []core.CanonicalID{"sales.submit"},
		Produces: []core.CanonicalID{"sales.order.customer_id"},
		Failure:  "reject",
	}}
	w.Applications[0].SemanticUses = []core.CanonicalID{"sales", "customer"}
	// A hook reaches a capability through the Command boundary, so the fixture
	// declares the command pattern that exposes it. Without it the declaration
	// would pass validation but never run.
	w.Applications[0].Patterns = append(w.Applications[0].Patterns, application.Pattern{
		ID:         "order_entry.submit",
		Kind:       application.PatternCommand,
		Capability: "sales.submit",
	})
	return w
}

func TestValidateInteractionHook(t *testing.T) {
	for _, diagnostic := range validateApplicationPatterns(hookWorld()) {
		t.Fatalf("unexpected diagnostic: %s %s", diagnostic.Code, diagnostic.Message)
	}
}

func TestValidateInteractionHookRejectsUnsupportedPhase(t *testing.T) {
	w := hookWorld()
	interactionRecordPattern(&w).Interaction.Hooks[0].Phase = "during-update"
	if diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_HOOK_PHASE_UNSUPPORTED") == nil {
		t.Fatal("unsupported hook phase was accepted")
	}
}

func TestValidateInteractionHookRejectsUnknownRead(t *testing.T) {
	w := hookWorld()
	interactionRecordPattern(&w).Interaction.Hooks[0].Reads = []core.CanonicalID{"sales.order.missing"}
	if diagnosticByCode(validateApplicationPatterns(w), "UNKNOWN_APPLICATION_PATTERN_INTERACTION_HOOK_READ") == nil {
		t.Fatal("unknown hook read was accepted")
	}
}

func TestValidateInteractionHookRejectsUnknownUse(t *testing.T) {
	w := hookWorld()
	interactionRecordPattern(&w).Interaction.Hooks[0].Uses = []core.CanonicalID{"sales.missing"}
	if diagnosticByCode(validateApplicationPatterns(w), "UNKNOWN_APPLICATION_PATTERN_INTERACTION_HOOK_USE") == nil {
		t.Fatal("unknown hook capability was accepted")
	}
}

func TestValidateInteractionHookRejectsUnsupportedFailure(t *testing.T) {
	w := hookWorld()
	interactionRecordPattern(&w).Interaction.Hooks[0].Failure = "ignore"
	if diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_HOOK_FAILURE_UNSUPPORTED") == nil {
		t.Fatal("unsupported hook failure was accepted")
	}
}

func TestValidateInteractionHookRejectsUseWithoutCommand(t *testing.T) {
	w := hookWorld()
	// Drop the command pattern that exposes sales.submit. The hook still names a
	// real capability, but nothing in the application can run it.
	keep := w.Applications[0].Patterns[:0]
	for _, pattern := range w.Applications[0].Patterns {
		if pattern.Kind == application.PatternCommand {
			continue
		}
		keep = append(keep, pattern)
	}
	w.Applications[0].Patterns = keep
	if diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_HOOK_USE_WITHOUT_COMMAND") == nil {
		t.Fatal("a hook capability without a command pattern was accepted")
	}
}

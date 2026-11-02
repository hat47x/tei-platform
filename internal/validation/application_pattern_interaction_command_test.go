package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func interactionRecordPattern(w *ir.World) *application.Pattern {
	for i := range w.Applications[0].Patterns {
		if w.Applications[0].Patterns[i].ID == "order_entry.order_record" {
			return &w.Applications[0].Patterns[i]
		}
	}
	return nil
}

func TestValidateInteractionCommand(t *testing.T) {
	w := applicationPatternInteractionWorld()
	w.Semantics[0].Capabilities = []semantic.Capability{{ID: "sales.confirm_order", Symbol: "confirm_order"}}
	pattern := interactionRecordPattern(&w)
	if pattern == nil {
		t.Fatal("record pattern not found")
	}
	pattern.Interaction.Commands = []application.InteractionCommand{{Capability: "sales.confirm_order"}}
	w.Applications[0].SemanticUses = []core.CanonicalID{"sales", "customer"}

	diagnostics := validateApplicationPatterns(w)
	if diagnostic := diagnosticByCode(diagnostics, "UNKNOWN_APPLICATION_PATTERN_INTERACTION_COMMAND"); diagnostic != nil {
		t.Fatalf("diagnostics=%#v", diagnostics)
	}
	if diagnostic := diagnosticByCode(diagnostics, "APPLICATION_PATTERN_INTERACTION_COMMAND_SEMANTIC_NOT_USED"); diagnostic != nil {
		t.Fatalf("diagnostics=%#v", diagnostics)
	}
}

func TestValidateInteractionCommandRejectsUnknownCapability(t *testing.T) {
	w := applicationPatternInteractionWorld()
	pattern := interactionRecordPattern(&w)
	pattern.Interaction.Commands = []application.InteractionCommand{{Capability: "sales.missing"}}

	diagnostics := validateApplicationPatterns(w)
	if diagnosticByCode(diagnostics, "UNKNOWN_APPLICATION_PATTERN_INTERACTION_COMMAND") == nil {
		t.Fatalf("diagnostics=%#v", diagnostics)
	}
}

func TestValidateInteractionCommandRejectsUnusedNamespace(t *testing.T) {
	w := applicationPatternInteractionWorld()
	w.Semantics[0].Capabilities = []semantic.Capability{{ID: "sales.confirm_order", Symbol: "confirm_order"}}
	pattern := interactionRecordPattern(&w)
	pattern.Interaction.Commands = []application.InteractionCommand{{Capability: "sales.confirm_order"}}
	w.Applications[0].SemanticUses = []core.CanonicalID{"customer"}

	diagnostics := validateApplicationPatterns(w)
	if diagnosticByCode(diagnostics, "APPLICATION_PATTERN_INTERACTION_COMMAND_SEMANTIC_NOT_USED") == nil {
		t.Fatalf("diagnostics=%#v", diagnostics)
	}
}

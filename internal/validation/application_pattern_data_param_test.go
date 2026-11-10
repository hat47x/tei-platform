package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// dataIdentifierWorld declares a capability whose parameter names sales.order, and
// two attributes of that Data: its identifier and an ordinary value.
func dataIdentifierWorld(boundAttribute string) ir.World {
	w := applicationPatternInteractionWorld()
	w.Semantics[0].Data[0].Attributes = []semantic.Attribute{
		{ID: "sales.order.order_id", TypeRef: "std.identifier"},
		{ID: "sales.order.status", TypeRef: "std.text"},
	}
	w.Semantics[0].Capabilities = append(w.Semantics[0].Capabilities, semantic.Capability{
		ID:     "sales.submit",
		Symbol: "submit",
		Inputs: []semantic.Parameter{{Name: "order", Type: "sales.order"}},
	})
	pattern := interactionRecordPattern(&w)
	pattern.Interaction.Commands = []application.InteractionCommand{{
		Capability: "sales.submit",
		Inputs:     []application.CommandInputBinding{{Name: "order", Attribute: core.CanonicalID(boundAttribute)}},
	}}
	w.Applications[0].SemanticUses = []core.CanonicalID{"sales", "customer"}
	return w
}

// TestDataParameterBoundToTheIdentifierIsAccepted checks the shape every existing
// example uses.
func TestDataParameterBoundToTheIdentifierIsAccepted(t *testing.T) {
	w := dataIdentifierWorld("sales.order.order_id")
	if diagnostic := diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_DATA_MISMATCH"); diagnostic != nil {
		t.Fatalf("the identifier binding was refused: %s", diagnostic.Message)
	}
}

// TestDataParameterBoundToAnOrdinaryAttributeIsAccepted records what the check
// currently allows.
//
// A parameter typed `sales.order` is fed by one of that Data's attributes. The
// check only requires the attribute to belong to the Data, so a plain value such
// as `sales.order.status` is accepted as well. Every example binds the
// identifier instead, and the parameter is a whole Data: receiving a status
// string where a record is expected is a shape the Command boundary does not
// inspect. The convention is not enforced, so this test states the current
// behaviour rather than blessing it.
func TestDataParameterBoundToAnOrdinaryAttributeIsAccepted(t *testing.T) {
	w := dataIdentifierWorld("sales.order.status")
	if diagnostic := diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_DATA_MISMATCH"); diagnostic != nil {
		t.Fatalf("binding an ordinary attribute was refused: %s", diagnostic.Message)
	}
}

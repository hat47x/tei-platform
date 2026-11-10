package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// conversionWorld binds an attribute of one declared type to a capability
// parameter of another, so a test can see which pairs the check admits.
func conversionWorld(attributeType, parameterType string, declareAlias bool) ir.World {
	w := applicationPatternInteractionWorld()
	w.Semantics[0].Data[0].Attributes = []semantic.Attribute{
		{ID: "sales.order.order_id", TypeRef: "std.identifier"},
		{ID: "sales.order.amount", TypeRef: core.CanonicalID(attributeType)},
	}
	if declareAlias {
		w.Semantics[0].Types = []semantic.Type{
			{ID: "sales.order_amount", Symbol: "order_amount", Kind: semantic.TypeAlias, Underlying: "std.decimal"},
		}
	}
	w.Semantics[0].Capabilities = append(w.Semantics[0].Capabilities, semantic.Capability{
		ID:     "sales.submit",
		Symbol: "submit",
		Inputs: []semantic.Parameter{{Name: "amount", Type: core.CanonicalID(parameterType)}},
	})
	pattern := interactionRecordPattern(&w)
	pattern.Interaction.Commands = []application.InteractionCommand{{
		Capability: "sales.submit",
		Inputs:     []application.CommandInputBinding{{Name: "amount", Attribute: "sales.order.amount"}},
	}}
	w.Applications[0].SemanticUses = []core.CanonicalID{"sales", "customer"}
	return w
}

// TestAcceptedPrimitiveBindingsShareTheSameUnderlyingType pins the property that
// makes a runtime value conversion unnecessary.
//
// The check admits a primitive-to-primitive binding only when both sides resolve
// to the same std primitive. An alias is followed to what it ultimately names, so
// an alias of std.decimal agrees with std.decimal and nothing else. Two bindings
// that both resolve to std.decimal carry the same lexical form, which is the
// form the Command boundary already validates against the semantic type: there is
// no pair the check admits whose value would need converting.
func TestAcceptedPrimitiveBindingsShareTheSameUnderlyingType(t *testing.T) {
	accepted := []struct {
		name        string
		attribute   string
		parameter   string
		declareType bool
	}{
		{"same primitive", "std.decimal", "std.decimal", false},
		{"alias of the same primitive", "sales.order_amount", "std.decimal", true},
		{"primitive to its alias", "std.decimal", "sales.order_amount", true},
	}
	for _, c := range accepted {
		t.Run(c.name, func(t *testing.T) {
			w := conversionWorld(c.attribute, c.parameter, c.declareType)
			if diagnostic := diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_TYPE_MISMATCH"); diagnostic != nil {
				t.Fatalf("an admitted binding was refused: %s", diagnostic.Message)
			}
		})
	}
}

// TestDifferentPrimitivesAreRefusedSoNoConversionIsReachable checks the other
// side: a pair that would need converting is refused, not converted.
func TestDifferentPrimitivesAreRefusedSoNoConversionIsReachable(t *testing.T) {
	refused := []struct {
		name      string
		attribute string
		parameter string
	}{
		{"decimal to text", "std.decimal", "std.text"},
		{"date to datetime", "std.date", "std.datetime"},
		{"identifier to decimal", "std.identifier", "std.decimal"},
	}
	for _, c := range refused {
		t.Run(c.name, func(t *testing.T) {
			w := conversionWorld(c.attribute, c.parameter, false)
			if diagnostic := diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_TYPE_MISMATCH"); diagnostic == nil {
				t.Fatalf("%s to %s was admitted, so a conversion would be reachable", c.attribute, c.parameter)
			}
		})
	}
}

package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// typeBindingWorld declares a record pattern whose interaction command binds an
// attribute to a capability parameter, plus the types both sides name. The
// shapes are the ones the binding check has to tell apart.
func typeBindingWorld(attributeType string, parameterType string) ir.World {
	w := applicationPatternInteractionWorld()
	w.Semantics[0].Data[0].Attributes = []semantic.Attribute{
		{ID: "sales.order.order_id", TypeRef: "std.identifier"},
		{ID: "sales.order.ordered_at", TypeRef: core.CanonicalID(attributeType)},
	}
	w.Semantics[0].Types = []semantic.Type{
		{ID: "sales.order_id", Symbol: "order_id", Kind: semantic.TypeAlias, Underlying: "std.identifier"},
	}
	w.Semantics[0].Capabilities = append(w.Semantics[0].Capabilities, semantic.Capability{
		ID:     "sales.submit",
		Symbol: "submit",
		Inputs: []semantic.Parameter{{Name: "at", Type: core.CanonicalID(parameterType)}},
	})
	pattern := interactionRecordPattern(&w)
	pattern.Interaction.Commands = []application.InteractionCommand{{
		Capability: "sales.submit",
		Inputs:     []application.CommandInputBinding{{Name: "at", Attribute: "sales.order.ordered_at"}},
	}}
	w.Applications[0].SemanticUses = []core.CanonicalID{"sales", "customer"}
	return w
}

// TestValidateBindingAcceptsTheSamePrimitiveType checks the agreeing case.
func TestValidateBindingAcceptsTheSamePrimitiveType(t *testing.T) {
	w := typeBindingWorld("std.datetime", "std.datetime")
	if diagnostic := diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_TYPE_MISMATCH"); diagnostic != nil {
		t.Fatalf("an agreeing binding was refused: %s", diagnostic.Message)
	}
}

// TestValidateBindingAcceptsAnAliasOfTheSameType checks that aliases compare by
// what they ultimately name, so an alias of a primitive agrees with it.
func TestValidateBindingAcceptsAnAliasOfTheSameType(t *testing.T) {
	w := typeBindingWorld("sales.order_id", "std.identifier")
	if diagnostic := diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_TYPE_MISMATCH"); diagnostic != nil {
		t.Fatalf("an alias of the same type was refused: %s", diagnostic.Message)
	}
}

// TestValidateBindingRefusesDifferentPrimitiveTypes checks the defect the rule
// exists for: a primitive attribute bound to a parameter of another primitive.
func TestValidateBindingRefusesDifferentPrimitiveTypes(t *testing.T) {
	w := typeBindingWorld("std.datetime", "std.date")
	diagnostic := diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_TYPE_MISMATCH")
	if diagnostic == nil {
		t.Fatal("a primitive type mismatch was accepted")
	}
	for _, want := range []string{"std.date", "std.datetime", "sales.order.ordered_at"} {
		if !containsString(diagnostic.Message, want) {
			t.Fatalf("the diagnostic does not name %q: %s", want, diagnostic.Message)
		}
	}
}

// TestValidateBindingLeavesADataParameterAlone checks that a parameter naming a
// Data is not compared as a type: a capability that takes `sales.order` is
// commonly bound to that data's identifier, and that correspondence is not a
// type equality.
func TestValidateBindingLeavesADataParameterAlone(t *testing.T) {
	w := typeBindingWorld("std.identifier", "sales.order")
	if diagnostic := diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_TYPE_MISMATCH"); diagnostic != nil {
		t.Fatalf("a data-typed parameter was compared as a primitive: %s", diagnostic.Message)
	}
}

// TestValidateBindingLeavesADataAttributeAlone checks the other direction.
func TestValidateBindingLeavesADataAttributeAlone(t *testing.T) {
	w := typeBindingWorld("sales.order", "std.datetime")
	if diagnostic := diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_TYPE_MISMATCH"); diagnostic != nil {
		t.Fatalf("a data-typed attribute was compared as a primitive: %s", diagnostic.Message)
	}
}

// dataBindingWorld declares a record pattern whose interaction command binds a
// Data-typed parameter to another Data's attribute.
func dataBindingWorld(attribute string) ir.World {
	w := applicationPatternInteractionWorld()
	w.Semantics[0].Data = append(w.Semantics[0].Data, semantic.Data{
		ID:         "sales.customer",
		Attributes: []semantic.Attribute{{ID: "sales.customer.customer_id", TypeRef: "std.identifier"}},
	})
	w.Semantics[0].Capabilities = append(w.Semantics[0].Capabilities, semantic.Capability{
		ID:     "sales.submit",
		Symbol: "submit",
		Inputs: []semantic.Parameter{{Name: "order", Type: "sales.order"}},
	})
	pattern := interactionRecordPattern(&w)
	pattern.Interaction.Commands = []application.InteractionCommand{{
		Capability: "sales.submit",
		Inputs:     []application.CommandInputBinding{{Name: "order", Attribute: core.CanonicalID(attribute)}},
	}}
	w.Applications[0].SemanticUses = []core.CanonicalID{"sales", "customer"}
	return w
}

// TestValidateBindingAcceptsAnAttributeOfTheParameterData checks that a
// Data-typed parameter bound to one of that Data's own attributes is accepted.
func TestValidateBindingAcceptsAnAttributeOfTheParameterData(t *testing.T) {
	w := dataBindingWorld("sales.order.order_id")
	if diagnostic := diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_DATA_MISMATCH"); diagnostic != nil {
		t.Fatalf("an attribute of the parameter's own data was refused: %s", diagnostic.Message)
	}
}

// TestValidateBindingRefusesAnotherDatasAttribute checks the defect: a
// `sales.order` parameter fed by a `sales.customer` attribute supplies a value
// of the wrong record, and nothing downstream inspects the record identity.
func TestValidateBindingRefusesAnotherDatasAttribute(t *testing.T) {
	w := dataBindingWorld("sales.customer.customer_id")
	diagnostic := diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_DATA_MISMATCH")
	if diagnostic == nil {
		t.Fatal("an attribute of another data was accepted for a data-typed parameter")
	}
	for _, want := range []string{"sales.order", "sales.customer.customer_id"} {
		if !containsString(diagnostic.Message, want) {
			t.Fatalf("the diagnostic does not name %q: %s", want, diagnostic.Message)
		}
	}
}

// containsString reports whether a string holds a substring.
func containsString(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

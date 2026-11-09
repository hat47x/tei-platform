package validation

import (
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// roleTypeWorld builds a world with the primitive types a role resolves to and
// one data whose attributes carry the roles under test. The world is built
// directly because the semantic file parser reads one namespace per file and
// the standard primitives live in the "std" namespace.
func roleTypeWorld(t *testing.T, attributes []semantic.Attribute, extraTypes ...semantic.Type) ir.World {
	t.Helper()
	types := []semantic.Type{
		{ID: "std.date", Symbol: "date", Kind: semantic.TypePrimitive},
		{ID: "std.decimal", Symbol: "decimal", Kind: semantic.TypePrimitive},
		{ID: "std.integer", Symbol: "integer", Kind: semantic.TypePrimitive},
		{ID: "std.code", Symbol: "code", Kind: semantic.TypePrimitive},
		{ID: "std.identifier", Symbol: "identifier", Kind: semantic.TypePrimitive},
		{ID: "std.text", Symbol: "text", Kind: semantic.TypePrimitive},
	}
	types = append(types, extraTypes...)
	return ir.World{Semantics: []semantic.Namespace{
		{ID: "std", Types: types},
		{ID: "tax", Data: []semantic.Data{{ID: "tax.rate", Symbol: "Rate", Attributes: attributes}}},
	}}
}

func attribute(id core.CanonicalID, typeRef core.CanonicalID, role string) semantic.Attribute {
	return semantic.Attribute{ID: id, Symbol: "value", TypeRef: typeRef, Role: role}
}

// roleTypeMismatch returns the mismatch diagnostics in the world.
func roleTypeMismatch(w ir.World) []Diagnostic {
	var out []Diagnostic
	for _, diagnostic := range Validate(w) {
		if diagnostic.Code == "SEMANTIC_ATTRIBUTE_ROLE_TYPE_MISMATCH" {
			out = append(out, diagnostic)
		}
	}
	return out
}

// TestValidateRoleTypeAcceptsMatchingPrimitive checks that a role on the right
// primitive produces no mismatch, so the check does not fire on correct
// declarations.
func TestValidateRoleTypeAcceptsMatchingPrimitive(t *testing.T) {
	w := roleTypeWorld(t, []semantic.Attribute{
		attribute("tax.rate.applies_from", "std.date", "tax-applies-from"),
		attribute("tax.rate.value", "std.decimal", "tax-value"),
	})
	if diagnostics := roleTypeMismatch(w); len(diagnostics) != 0 {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
}

// TestValidateRoleTypeReportsADateOnADecimal checks that a mismatch names the
// role, the expected primitive, and what the type resolved to. X-TEA read rates
// by column name (D:Session.java L2315–2334), so a column holding the wrong kind
// of value was only discovered when the number came out wrong.
func TestValidateRoleTypeReportsADateOnADecimal(t *testing.T) {
	w := roleTypeWorld(t, []semantic.Attribute{
		attribute("tax.rate.value", "std.date", "tax-value"),
	})
	diagnostics := roleTypeMismatch(w)
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
	if diagnostics[0].Message == "" || diagnostics[0].Subject == nil {
		t.Fatalf("diagnostic lost its message or subject: %+v", diagnostics[0])
	}
}

// TestValidateRoleTypeFollowsAnAlias checks that a role on an alias resolves to
// the underlying primitive, so a declared alias is not refused for being an
// alias.
func TestValidateRoleTypeFollowsAnAlias(t *testing.T) {
	alias := semantic.Type{ID: "tax.rate_value", Symbol: "rate_value", Kind: semantic.TypeAlias, Underlying: "std.decimal"}
	w := roleTypeWorld(t, []semantic.Attribute{
		attribute("tax.rate.value", "tax.rate_value", "tax-value"),
	}, alias)
	if diagnostics := roleTypeMismatch(w); len(diagnostics) != 0 {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
}

// TestValidateRoleTypeRefusesAnAliasToTheWrongPrimitive checks that following
// the alias still reports a mismatch.
func TestValidateRoleTypeRefusesAnAliasToTheWrongPrimitive(t *testing.T) {
	alias := semantic.Type{ID: "tax.rate_date", Symbol: "rate_date", Kind: semantic.TypeAlias, Underlying: "std.date"}
	w := roleTypeWorld(t, []semantic.Attribute{
		attribute("tax.rate.value", "tax.rate_date", "tax-value"),
	}, alias)
	if diagnostics := roleTypeMismatch(w); len(diagnostics) != 1 {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
}

// TestValidateRoleTypeAcceptsAnAnyTypeRole checks that a role that states no
// primitive accepts a resolvable type, because the declaration says what the
// value means and not how it is stored.
func TestValidateRoleTypeAcceptsAnAnyTypeRole(t *testing.T) {
	w := roleTypeWorld(t, []semantic.Attribute{
		attribute("tax.rate.category", "std.code", "tax-category"),
		attribute("tax.rate.rounding", "std.text", "tax-rounding"),
	})
	if diagnostics := roleTypeMismatch(w); len(diagnostics) != 0 {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
}

// TestValidateRoleTypeLeavesAnUnknownTypeToItsOwnCheck checks that an
// unresolvable type is not reported twice: the unknown type is its own
// diagnostic, and the role check does not guess a primitive for it.
func TestValidateRoleTypeLeavesAnUnknownTypeToItsOwnCheck(t *testing.T) {
	w := roleTypeWorld(t, []semantic.Attribute{
		attribute("tax.rate.value", "tax.missing", "tax-value"),
	})
	var unknown, mismatch bool
	for _, diagnostic := range Validate(w) {
		switch diagnostic.Code {
		case "UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE":
			unknown = true
		case "SEMANTIC_ATTRIBUTE_ROLE_TYPE_MISMATCH":
			mismatch = true
		}
	}
	if !unknown || mismatch {
		t.Fatalf("unknown=%v mismatch=%v", unknown, mismatch)
	}
}

// TestValidateRoleTypeAcceptsEveryDeclaredRolePair checks the whole vocabulary:
// each role with the primitive it requires is accepted, so the map and the
// vocabulary do not disagree.
func TestValidateRoleTypeAcceptsEveryDeclaredRolePair(t *testing.T) {
	for role, family := range rolePrimitives {
		if family == "" {
			continue
		}
		w := roleTypeWorld(t, []semantic.Attribute{
			attribute("tax.rate.value", core.CanonicalID("std."+family), role),
		})
		if diagnostics := roleTypeMismatch(w); len(diagnostics) != 0 {
			t.Fatalf("role %s with std.%s: %+v", role, family, diagnostics)
		}
	}
}

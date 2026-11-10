package validation

import (
	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/subject"
)

// rolePrimitives is the primitive type family each standard business role must
// resolve to. The role vocabulary says which attribute a binder reads; this map
// says what the attribute must hold, so a declaration that names the right role
// on the wrong type is reported instead of binding a date as a decimal.
//
// An empty entry means the role accepts any resolvable type: a category is a
// code or text, a prefix is text, and a rule name is text. The check reports
// only what the declaration can contradict.
var rolePrimitives = map[string]string{
	"tax-applies-from":             "date",
	"tax-applies-until":            "date",
	"tax-category":                 "",
	"tax-value":                    "decimal",
	"tax-rounding":                 "",
	"currency-rate-currency":       "",
	"currency-rate-fiscal-year":    "integer",
	"currency-rate-month-sequence": "integer",
	"currency-rate-middle":         "decimal",
	"currency-rate-buying":         "decimal",
	"currency-rate-selling":        "decimal",
	"currency-rate-fallback":       "", // any type: a rule name
	"currency-rate-granularity":    "", // any type: a granularity name
	"numbering-counter":            "integer",
	"numbering-digits":             "integer",
	"numbering-prefix":             "",
	"numbering-check":              "",
	"numbering-overflow":           "",
	"holiday-date":                 "date",
	"holiday-calendar":             "",           // any type: a calendar identity is a code or text
	"setting-key":                  "identifier", // a setting is named by an identifier
	"setting-value":                "",           // any type: a setting value is what the key holds
	"setting-user":                 "identifier", // a user is named by an identifier
}

// validateAttributeRoleTypes reports an attribute whose role requires a
// primitive family it does not have.
//
// An attribute whose type cannot be resolved is left to
// UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE; this check does not guess a primitive for it,
// so one mistake is reported once.
func validateAttributeRoleTypes(attributes []semantic.Attribute, types map[core.CanonicalID]semantic.Type) []Diagnostic {
	var out []Diagnostic
	for _, attribute := range attributes {
		want, known := rolePrimitives[attribute.Role]
		if !known || want == "" {
			continue
		}
		if _, ok := types[attribute.TypeRef]; !ok {
			continue
		}
		got, ok := primitiveOf(attribute.TypeRef, types)
		if !ok {
			continue
		}
		if got == want {
			continue
		}
		out = append(out, errFor(
			subject.Subject{Kind: "semantic-attribute", ID: string(attribute.ID)},
			"SEMANTIC_ATTRIBUTE_ROLE_TYPE_MISMATCH",
			"attribute %s declares role %s which requires a %s type, but its type %s resolves to %s",
			attribute.ID, attribute.Role, want, attribute.TypeRef, got,
		))
	}
	return out
}

// primitiveOf follows the alias chain to a primitive family name. The name is
// the primitive type ID without its namespace, so "std.date" yields "date".
func primitiveOf(typeRef core.CanonicalID, types map[core.CanonicalID]semantic.Type) (string, bool) {
	current := typeRef
	for depth := 0; depth < 16; depth++ {
		typ, ok := types[current]
		if !ok {
			return "", false
		}
		switch typ.Kind {
		case semantic.TypePrimitive:
			return primitiveFamily(current), true
		case semantic.TypeAlias:
			if typ.Underlying == "" {
				return "", false
			}
			current = typ.Underlying
		default:
			return "", false
		}
	}
	return "", false
}

// primitiveFamily returns the last segment of a canonical primitive type ID.
func primitiveFamily(id core.CanonicalID) string {
	text := string(id)
	for index := len(text) - 1; index >= 0; index-- {
		if text[index] == '.' {
			return text[index+1:]
		}
	}
	return text
}

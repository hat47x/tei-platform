package semantic

import "tei/internal/core"

// followAlias reports whether id is one of roots, or an alias that leads to
// one. A cycle of aliases leads nowhere.
func followAlias(types map[core.CanonicalID]Type, id core.CanonicalID, roots ...core.CanonicalID) bool {
	seen := map[core.CanonicalID]bool{}
	for id != "" && !seen[id] {
		for _, root := range roots {
			if id == root {
				return true
			}
		}
		seen[id] = true
		typ, known := types[id]
		if !known || typ.Kind != TypeAlias {
			return false
		}
		id = typ.Underlying
	}
	return false
}

// IsNumberType reports whether a type is a number: std.integer, std.decimal,
// std.money, or an alias of one. It is the rule the validator applies to a
// rule that reads a number.
func IsNumberType(types map[core.CanonicalID]Type, id core.CanonicalID) bool {
	return followAlias(types, id, "std.integer", "std.decimal", "std.money")
}

// IsCalendarDateType reports whether a type is a calendar date: std.date or
// an alias of it. A moment in time (std.datetime) is not one.
func IsCalendarDateType(types map[core.CanonicalID]Type, id core.CanonicalID) bool {
	return followAlias(types, id, "std.date")
}

// IsTextType reports whether a type is a text: std.text or an alias of it. A
// text is read as a text and never as a number: the number 7 is not the text
// "7", and reading one for the other would answer a different question.
func IsTextType(types map[core.CanonicalID]Type, id core.CanonicalID) bool {
	return followAlias(types, id, "std.text")
}

// IsTruthType reports whether a type is a truth: std.truth or an alias of it.
// A truth is read as a truth and never as the number 1 or 0.
func IsTruthType(types map[core.CanonicalID]Type, id core.CanonicalID) bool {
	return followAlias(types, id, "std.truth")
}

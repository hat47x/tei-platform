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

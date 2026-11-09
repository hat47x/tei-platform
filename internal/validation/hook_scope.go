package validation

import "tei/internal/application"

// scopeFitsPattern reports whether a pattern kind looks at the unit a hook
// declares.
//
// The mapping is deliberately narrow: each kind names the units its contract
// already works with, so a scope is refused when the pattern has no such unit.
// A browse pattern shows rows and one record at a time; a record pattern works on
// one record; a header-detail pattern works on a header and its rows; a command
// pattern works on items. Declaring a scope the kind does not look at leaves a hook
// that looks correct and never applies.
func scopeFitsPattern(kind application.PatternKind, scope string) bool {
	switch kind {
	case application.PatternBrowse:
		return scope == "row" || scope == "record"
	case application.PatternRecord:
		return scope == "record"
	case application.PatternHeaderDetail:
		return scope == "header" || scope == "row"
	case application.PatternCommand:
		return scope == "item" || scope == "record"
	case application.PatternReference:
		return scope == "record"
	default:
		// A kind this function does not know is not narrowed: refusing every scope
		// for it would make a new kind unusable before anyone decided what it
		// looks at.
		return true
	}
}

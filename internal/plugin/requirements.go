package plugin

import (
	"fmt"

	"tei/internal/core"
)

// ValidateRequirements refuses repeated dependency identities, even when their
// compatible majors are the same. A duplicate is an ambiguous declaration,
// not a request for the resolver to select or collapse dependencies.
func ValidateRequirements(requires []Requirement) error {
	seen := make(map[core.CanonicalID]struct{}, len(requires))
	var firstDuplicate core.CanonicalID
	hasDuplicate := false
	for _, req := range requires {
		if _, duplicate := seen[req.Plugin]; duplicate {
			// Report the lexically smallest duplicate so declaration order does
			// not affect the diagnostic for the same invalid requirement set.
			if !hasDuplicate || req.Plugin < firstDuplicate {
				firstDuplicate = req.Plugin
				hasDuplicate = true
			}
		}
		seen[req.Plugin] = struct{}{}
	}
	if hasDuplicate {
		return fmt.Errorf("plugin requirement %s appears more than once", firstDuplicate)
	}
	return nil
}

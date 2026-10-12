package plugin

import (
	"fmt"
	"sort"

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

// ValidateRequirementEntries checks the local shape of every dependency.
// This is separate from duplicate-identity diagnostics: a malformed ID or
// compatible major is not itself a duplicate declaration. Sorting a copy
// makes the first reported invalid dependency independent of source order.
func ValidateRequirementEntries(requires []Requirement) error {
	ordered := append([]Requirement(nil), requires...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Plugin != ordered[j].Plugin {
			return ordered[i].Plugin < ordered[j].Plugin
		}
		return ordered[i].CompatibleMajor < ordered[j].CompatibleMajor
	})
	for _, req := range ordered {
		parsed, err := core.ParseCanonicalID(string(req.Plugin))
		if err != nil {
			return fmt.Errorf("plugin requirement %q: %w", req.Plugin, err)
		}
		if parsed != req.Plugin {
			return fmt.Errorf("plugin requirement %q must be a canonical id without surrounding whitespace", req.Plugin)
		}
		if req.CompatibleMajor < 1 {
			return fmt.Errorf("plugin requirement %s compatible major must be >= 1, got %d", req.Plugin, req.CompatibleMajor)
		}
	}
	return nil
}

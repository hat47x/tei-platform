package plugin

import (
	"fmt"
	"sort"

	"tei/internal/core"
)

// ValidateUniqueRequirements refuses repeated dependency identities, even when their
// compatible majors are the same. A duplicate is an ambiguous declaration,
// not a request for the resolver to select or collapse dependencies.
func ValidateUniqueRequirements(requires []Requirement) error {
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

// ValidateRequirementValues rejects invalid dependency identities and
// nonpositive compatible majors. Diagnostics are independent of input order.
func ValidateRequirementValues(requires []Requirement) error {
	var invalidIDs, invalidMajors []string
	for _, req := range requires {
		parsed, err := core.ParseCanonicalID(string(req.Plugin))
		if err != nil || parsed != req.Plugin {
			invalidIDs = append(invalidIDs, fmt.Sprintf("plugin requirement %q has invalid canonical id", req.Plugin))
		}
		if req.CompatibleMajor < 1 {
			invalidMajors = append(invalidMajors, fmt.Sprintf("plugin requirement %q has invalid compatible major %d (must be positive)", req.Plugin, req.CompatibleMajor))
		}
	}
	sort.Strings(invalidIDs)
	sort.Strings(invalidMajors)
	if len(invalidIDs) != 0 {
		return fmt.Errorf("%s", invalidIDs[0])
	}
	if len(invalidMajors) != 0 {
		return fmt.Errorf("%s", invalidMajors[0])
	}
	return nil
}

// ValidateRequirements checks both intra-declaration uniqueness and the
// constituent requirement values. World validation calls the two helpers
// separately to emit distinct diagnostic codes.
func ValidateRequirements(requires []Requirement) error {
	if err := ValidateUniqueRequirements(requires); err != nil {
		return err
	}
	return ValidateRequirementValues(requires)
}

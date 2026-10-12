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
	for _, req := range requires {
		if _, duplicate := seen[req.Plugin]; duplicate {
			return fmt.Errorf("plugin requirement %s appears more than once", req.Plugin)
		}
		seen[req.Plugin] = struct{}{}
	}
	return nil
}

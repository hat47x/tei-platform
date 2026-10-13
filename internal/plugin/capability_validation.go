package plugin

import (
	"fmt"

	"tei/internal/core"
)

// ValidateUniqueCapabilities rejects duplicate (kind, ID) pairs before a
// definition is normalized or packaged. Different kinds may use the same ID.
// Report the lowest duplicate identity to avoid declaration-order diagnostics.
func ValidateUniqueCapabilities(provides []Capability) error {
	type identity struct {
		kind CapabilityKind
		id   core.CanonicalID
	}
	seen := make(map[identity]struct{}, len(provides))
	var first identity
	hasDuplicate := false
	for _, c := range provides {
		key := identity{kind: c.Kind, id: c.ID}
		if _, duplicate := seen[key]; duplicate {
			if !hasDuplicate || key.kind < first.kind || (key.kind == first.kind && key.id < first.id) {
				first = key
				hasDuplicate = true
			}
		}
		seen[key] = struct{}{}
	}
	if hasDuplicate {
		return fmt.Errorf("capability %s %s appears more than once", first.kind, first.id)
	}
	return nil
}

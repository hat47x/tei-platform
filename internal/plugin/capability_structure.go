package plugin

import (
	"fmt"
	"sort"

	"tei/internal/environment"
	"tei/internal/permission"
)

// ValidateUniqueCapabilityPermissions rejects repeated permissions within one
// capability, identifying the same duplicate regardless of declaration order.
func ValidateUniqueCapabilityPermissions(specs []permission.Spec) error {
	seen := make(map[string]struct{}, len(specs))
	duplicates := []string{}
	for _, spec := range specs {
		key := spec.Key()
		if _, exists := seen[key]; exists {
			duplicates = append(duplicates, key)
		}
		seen[key] = struct{}{}
	}
	if len(duplicates) == 0 {
		return nil
	}
	sort.Strings(duplicates)
	return fmt.Errorf("permission %q appears more than once", duplicates[0])
}

// ValidateCapabilityStructure checks local declarations independently of a
// World. Referenced environments and runtime grants are validated elsewhere.
func ValidateCapabilityStructure(provides []Capability) error {
	if err := ValidateUniqueCapabilities(provides); err != nil {
		return err
	}
	for _, c := range provides {
		if !c.Kind.Valid() {
			return fmt.Errorf("capability %s has invalid kind %q", c.ID, c.Kind)
		}
		for _, a := range c.Actions {
			if !a.Valid() {
				return fmt.Errorf("capability %s has invalid action %q", c.ID, a)
			}
		}
		for _, o := range c.Observations {
			if !o.Valid() {
				return fmt.Errorf("capability %s has invalid observation %q", c.ID, o)
			}
		}
		for _, e := range c.Evidence {
			if !e.Valid() {
				return fmt.Errorf("capability %s has invalid evidence %q", c.ID, e)
			}
		}
		for _, p := range c.DatabaseProviders {
			if !p.Known() || p == environment.ProviderGeneric {
				return fmt.Errorf("capability %s has invalid database provider %q", c.ID, p)
			}
		}
		for _, spec := range c.Permissions {
			if err := spec.Validate(); err != nil {
				return fmt.Errorf("capability %s: %w", c.ID, err)
			}
		}
		if err := ValidateUniqueCapabilityPermissions(c.Permissions); err != nil {
			return fmt.Errorf("capability %s: %w", c.ID, err)
		}
	}
	return nil
}

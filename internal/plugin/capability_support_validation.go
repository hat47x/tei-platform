package plugin

import (
	"fmt"
	"sort"
)

// ValidateUniqueCapabilitySupports treats each declared support collection as
// a set. Repetitions do not increase runtime capabilities, but they change the
// definition fingerprint, so reject them instead of silently deduplicating.
func ValidateUniqueCapabilitySupports(capability Capability) error {
	sets := []struct {
		name   string
		values []string
	}{
		{name: "actions", values: supportStrings(capability.Actions)},
		{name: "observations", values: supportStrings(capability.Observations)},
		{name: "evidence", values: supportStrings(capability.Evidence)},
		{name: "databaseProviders", values: supportStrings(capability.DatabaseProviders)},
		{name: "migrationTools", values: supportStrings(capability.MigrationTools)},
		{name: "features", values: supportStrings(capability.Features)},
	}
	for _, set := range sets {
		seen := make(map[string]struct{}, len(set.values))
		var duplicates []string
		for _, value := range set.values {
			if _, exists := seen[value]; exists {
				duplicates = append(duplicates, value)
			}
			seen[value] = struct{}{}
		}
		if len(duplicates) != 0 {
			sort.Strings(duplicates)
			return fmt.Errorf("capability %s %s value %q appears more than once", capability.ID, set.name, duplicates[0])
		}
	}
	return nil
}

func supportStrings[T ~string](items []T) []string {
	values := make([]string, len(items))
	for i, value := range items {
		values[i] = string(value)
	}
	return values
}

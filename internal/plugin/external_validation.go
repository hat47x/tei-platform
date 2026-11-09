package plugin

import (
	"fmt"
	"sort"
	"strings"
)

// ValidScope reports whether an external requirement declares a supported
// lifecycle phase.
func (r ExternalRequirement) ValidScope() bool {
	switch r.Scope {
	case "editor", "build", "test", "runtime", "migration":
		return true
	default:
		return false
	}
}

// ValidKind reports whether the external dependency category is supported.
func (r ExternalRequirement) ValidKind() bool {
	switch r.Kind {
	case "toolchain", "runtime", "language-package", "tool":
		return true
	default:
		return false
	}
}

func (r ExternalRequirement) ValidName() bool {
	return strings.TrimSpace(r.Name) != ""
}

// ValidateUniqueExternalRequirements rejects exact, redundant declarations.
// The three fields form an identity; distinct phases or kinds are meaningful.
func ValidateUniqueExternalRequirements(requirements []ExternalRequirement) error {
	type key struct{ scope, kind, name string }
	seen := make(map[key]struct{}, len(requirements))
	duplicates := []key{}
	for _, r := range requirements {
		id := key{r.Scope, r.Kind, r.Name}
		if _, exists := seen[id]; exists {
			duplicates = append(duplicates, id)
		}
		seen[id] = struct{}{}
	}
	if len(duplicates) == 0 {
		return nil
	}
	sort.Slice(duplicates, func(i, j int) bool {
		if duplicates[i].scope != duplicates[j].scope {
			return duplicates[i].scope < duplicates[j].scope
		}
		if duplicates[i].kind != duplicates[j].kind {
			return duplicates[i].kind < duplicates[j].kind
		}
		return duplicates[i].name < duplicates[j].name
	})
	r := duplicates[0]
	return fmt.Errorf("external requirement %q/%q/%q appears more than once", r.scope, r.kind, r.name)
}

// ValidateExternalRequirements is the standalone materializer's local
// structural gate, shared with the world validator through the three field
// predicates and the uniqueness check.
func ValidateExternalRequirements(requirements []ExternalRequirement) error {
	for _, r := range requirements {
		if !r.ValidScope() {
			return fmt.Errorf("external requirement %q has invalid scope %q", r.Name, r.Scope)
		}
		if !r.ValidKind() {
			return fmt.Errorf("external requirement %q has invalid kind %q", r.Name, r.Kind)
		}
		if !r.ValidName() {
			return fmt.Errorf("external requirement name is required")
		}
	}
	return ValidateUniqueExternalRequirements(requirements)
}

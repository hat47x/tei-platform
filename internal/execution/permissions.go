package execution

import (
	"strings"

	"tei/internal/core"
	"tei/internal/permission"
)

// PermissionPreconditions adapts the declared permissions to the run plan: a
// capability whose requirements are not covered by the granted permissions is
// refused before it runs. This compares declarations; it does not sandbox the
// process, so it cannot stop an access the capability obtains another way.
func PermissionPreconditions(required func(core.CanonicalID) []permission.Spec, granted []permission.Spec) PreconditionFunc {
	return func(capabilityID core.CanonicalID) string {
		if required == nil {
			return ""
		}
		requirements := required(capabilityID)
		missing := permission.Missing(granted, requirements)
		if len(missing) == 0 {
			return ""
		}
		parts := make([]string, 0, len(missing))
		for _, requirement := range missing {
			parts = append(parts, string(requirement.Kind)+":"+string(requirement.Access)+" "+requirement.Scope)
		}
		return "the granted permissions do not cover " + strings.Join(parts, ", ")
	}
}

package execution

import (
	"strings"

	"tei/internal/access"
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

// GuardFromGrants builds the access guard for one capability from the
// environment grants. A grant belongs to the capability when both its kind and
// its identity match, so two capabilities that share an identity but differ in
// kind do not share permissions.
func GuardFromGrants(grants []permission.Grant, capabilityKind string, capabilityID core.CanonicalID, journal access.Journal) access.Guard {
	var granted []permission.Spec
	for _, grant := range grants {
		if grant.CapabilityKind != capabilityKind || grant.CapabilityID != capabilityID {
			continue
		}
		granted = append(granted, grant.Permission)
	}
	return access.Guard{Granted: granted, Journal: journal}
}

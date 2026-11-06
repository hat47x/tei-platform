package permission

import "strings"

// Covers reports whether a grant satisfies a requirement: the same kind, an
// access that includes the required access, and a scope that contains it. A
// scope contains another when it is equal or a path prefix of it; the prefix
// must end at a separator so "/var/data" does not cover "/var/database".
func Covers(grant, requirement Spec) bool {
	if grant.Kind != requirement.Kind {
		return false
	}
	if !accessCovers(grant.Access, requirement.Access) {
		return false
	}
	return scopeCovers(grant.Scope, requirement.Scope)
}

// accessCovers reports whether granted access includes required access.
func accessCovers(granted, required Access) bool {
	if granted == required {
		return true
	}
	if granted == AccessReadWrite {
		return required == AccessRead || required == AccessWrite
	}
	return false
}

// scopeCovers reports whether the granted scope contains the required scope.
func scopeCovers(granted, required string) bool {
	granted = strings.TrimSpace(granted)
	required = strings.TrimSpace(required)
	if granted == "" || required == "" {
		return false
	}
	if granted == required {
		return true
	}
	// Both separators are accepted so a scope declared with one style still
	// covers paths resolved with the other.
	for _, separator := range []string{"/", "\\", ".", ":"} {
		if strings.HasPrefix(required, granted+separator) {
			return true
		}
	}
	return false
}

// Missing returns the requirements that no grant covers. The result keeps the
// order of the requirements and drops none, so a caller can report every gap.
func Missing(granted, required []Spec) []Spec {
	var missing []Spec
	for _, requirement := range required {
		covered := false
		for _, grant := range granted {
			if Covers(grant, requirement) {
				covered = true
				break
			}
		}
		if !covered {
			missing = append(missing, requirement)
		}
	}
	return missing
}

package core

import "strings"

// ValidNumericVersion accepts the intentionally small M0 release-version
// surface: numeric major.minor.patch. Release identity is independent from Git
// identity and may later grow richer compatibility metadata without changing
// this helper's callers.
func ValidNumericVersion(v string) bool {
	parts := strings.Split(strings.TrimSpace(v), ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

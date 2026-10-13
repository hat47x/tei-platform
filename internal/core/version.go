package core

import "tei/core/identity"

// ValidNumericVersion forwards to the single implementation of the legacy M0
// release version surface in core/identity.
func ValidNumericVersion(v string) bool { return identity.ValidNumericVersion(v) }

//go:build tei_api_only && tei_order_only && !tei_repair_only

package main

import (
	"fmt"
	"regexp"
)

// Set by the diagnostic Builder from its exact clean Git checkout with -X.
// The executable refuses to announce an arbitrary caller-supplied T as its
// own build revision. This plain build-time string is NOT a signature,
// accepted model identity, trusted Builder identity or deployment authority.
var referenceAPISourceRevision string

var diagnosticTargetRevision = regexp.MustCompile(`^[0-9a-f]{40}$`)

func validateReferenceBuildRevision(requested string) error {
	if !diagnosticTargetRevision.MatchString(referenceAPISourceRevision) {
		return fmt.Errorf("diagnostic API build revision T was not pinned by the builder")
	}
	if requested != referenceAPISourceRevision {
		return fmt.Errorf("diagnostic API requested Target build revision T differs from compiled build")
	}
	return nil
}

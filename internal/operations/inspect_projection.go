package operations

import (
	"encoding/json"
	"errors"

	"tei/internal/sourceindex"
)

// ValidateInspectionProjection checks the observable shape of an Inspector
// response before a CLI or Studio adapter reports success. It is deliberately
// not a check of Provider identity, source completeness, current Disclosure,
// an accepted meaning, or an artifact actually invoked by the Host.
//
// Both successful transports and selected Inspector implementations should
// apply this same contract check. A result with the wrong subject or Effect
// must not be accepted merely because it contains valid JSON.
func ValidateInspectionProjection(requested sourceindex.Subject, value SubjectInspection) error {
	if value.Subject != requested ||
		value.SchemaVersion != "tei.operations.inspect/v0" ||
		value.Effect != "read-only" ||
		!json.Valid(value.Canonical) ||
		value.WorldFingerprint == "" ||
		value.SourceIndexFingerprint == "" {
		return errors.New("inspection service returned an inconsistent projection")
	}
	return nil
}

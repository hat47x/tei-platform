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
	// Every currently supported Canonical Subject kind is an object with
	// an exact top-level ID. The response wrapper's Subject alone cannot
	// attest that the returned Canonical bytes belong to that Subject:
	// a selected Provider could return another Subject's valid JSON.
	var identity struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(value.Canonical, &identity); err != nil || identity.ID != requested.ID {
		return errors.New("inspection service returned a Canonical object for another subject")
	}
	return nil
}

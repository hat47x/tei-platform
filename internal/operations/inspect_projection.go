package operations

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"

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
	if !canonicalIDMatchesExactField(value.Canonical, requested.ID) {
		return errors.New("inspection service returned a Canonical object for another subject")
	}
	return nil
}

// canonicalIDMatchesExactField requires exactly one top-level, lowercase
// "id" field. Encoding/json's struct matching is case-insensitive, and
// unmarshalling into a map silently overwrites duplicated keys. Neither is
// sufficient to qualify an untrusted Provider's Canonical identity.
func canonicalIDMatchesExactField(raw json.RawMessage, requestedID string) bool {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return false
	}
	found := false
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return false
		}
		name, ok := token.(string)
		if !ok {
			return false
		}
		var field json.RawMessage
		if err := decoder.Decode(&field); err != nil {
			return false
		}
		if name != "id" {
			// Reject keys that Go's case-insensitive struct decoding could
			// treat as the canonical ID, even though they are not "id".
			if strings.EqualFold(name, "id") {
				return false
			}
			continue
		}
		if found {
			return false
		}
		var actualID string
		if err := json.Unmarshal(field, &actualID); err != nil || actualID != requestedID {
			return false
		}
		found = true
	}
	closing, err := decoder.Token()
	return found && err == nil && closing == json.Delim('}')
}

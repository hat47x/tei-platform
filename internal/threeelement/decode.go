package threeelement

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// DecodeModel reads one three-element analysis model document.
//
// A model file states which version of the model contract it follows, and that
// version is the only one this package knows how to read. Before this function
// existed, every reader called json.Unmarshal directly: an unknown
// schemaVersion was read as if it were v0, and a field the contract does not
// define was dropped without a trace, so a model that carried meaning this
// package cannot interpret produced the same ModelFingerprint as a model that
// never carried it.
//
// The rules are therefore explicit:
//
//   - schemaVersion must be exactly the version this package implements. A
//     missing or unknown version is refused instead of guessed.
//   - a field outside the contract is refused. Under a known version, those
//     fields have no defined meaning, and silently dropping them makes two
//     different documents indistinguishable.
//   - a leading UTF-8 BOM is an encoding detail, not JSON text, so it is
//     trimmed first (Windows editors add one often enough that refusing it
//     would report a broken model for a byte-order mark).
//
// Decoding does not compute the fingerprint; the caller does that after adding
// anything it needs to the model, because the fingerprint itself is part of the
// model content that is excluded from its own hash.
func DecodeModel(raw []byte) (ThreeElementModel, error) {
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var model ThreeElementModel
	if err := decoder.Decode(&model); err != nil {
		return ThreeElementModel{}, err
	}
	// A trailing document would make the file say two things; refuse it rather
	// than read the first one and ignore the rest.
	if err := ensureEndOfDocument(decoder); err != nil {
		return ThreeElementModel{}, err
	}
	if got := model.SchemaVersion; got != schemaVersion {
		return ThreeElementModel{}, fmt.Errorf("model schemaVersion %q is not %q, the only version this reader implements",
			got, schemaVersion)
	}
	return model, nil
}

// ensureEndOfDocument reports whether anything other than whitespace follows the
// model document.
func ensureEndOfDocument(decoder *json.Decoder) error {
	var extra json.Token
	token, err := decoder.Token()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	extra = token
	return fmt.Errorf("model document has content after the model object: %v", extra)
}

// SchemaVersion returns the model contract version this package reads and
// writes. Callers that build tests or fixtures from the package use it instead
// of repeating the literal, so a fixture cannot claim a version the reader
// refuses.
func SchemaVersion() string {
	return schemaVersion
}

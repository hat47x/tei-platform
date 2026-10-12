package xeadsource

import (
	"fmt"
	"reflect"
)

// CrossViewSourceReconciledAccounting produces public aggregate data only after
// independently rebuilding a reported projection from the caller-supplied
// original source bytes and the versioned transcription contract.
//
// CrossViewPublicAccounting verifies the internal consistency of a projection;
// it cannot prove that counts or omitted nodes match the source. This function
// closes that gap for *supplied bytes* without publishing individual legacy
// identifiers or source tokens. The caller remains responsible for proving
// that the bytes really came from the intended external source.
//
// Successful reconciliation neither accepts Canonical meaning nor proves
// equivalence to the old runtime, ownership, transaction or authority.
func CrossViewSourceReconciledAccounting(
	format Format,
	source []byte,
	contract []byte,
	reported CrossViewProjection,
) (CrossViewAccounting, error) {
	if _, err := CrossViewPublicAccounting(reported); err != nil {
		return CrossViewAccounting{}, fmt.Errorf("invalid reported cross-view projection: %w", err)
	}
	rebuilt, err := BuildCrossViewProjection(format, source, contract)
	if err != nil {
		return CrossViewAccounting{}, fmt.Errorf("cannot rebuild cross-view source evidence: %w", err)
	}
	if !reflect.DeepEqual(reported, rebuilt) {
		return CrossViewAccounting{}, fmt.Errorf("reported cross-view projection differs from source-derived projection")
	}
	// Return an aggregate made from the newly inspected source, not from the
	// caller-controlled reported object. The aggregate does not contain any
	// source IDs or individual provenance tokens.
	return CrossViewPublicAccounting(rebuilt)
}

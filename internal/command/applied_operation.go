package command

import (
	"tei/internal/core"
	"tei/internal/semantic"
)

// AppliedOperation is one data operation an executor actually applied.
// A reference executor reports the effect kind, the data, and the record key;
// it does not report the SQL statement text, because it does not issue one.
type AppliedOperation struct {
	Kind semantic.DataEffectKind `json:"kind"`
	Data core.CanonicalID        `json:"data"`
	Key  string                  `json:"key,omitempty"`
}

// MatchesDeclared reports whether the operation belongs to the declared effect.
// A declared effect names one kind and one data; a cascade delete applies the
// same kind to related data, so callers compare per operation instead of
// assuming every applied operation names the declared data.
func (o AppliedOperation) MatchesDeclared(declared *semantic.DataEffect) bool {
	if declared == nil {
		return false
	}
	return o.Kind == declared.Kind && o.Data == declared.Data
}

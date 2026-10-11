package languagereference

import (
	"tei/internal/core"
	"tei/internal/ir"
)

// ExpandRelationReference retains the small output-type adapter for the E2
// prototype. Callers needing to preserve the relation identity and key
// mapping should use InspectPinnedRelationExpansion instead. Neither call
// executes a join or accepts a Semantic World.
func ExpandRelationReference(w ir.World, sourceID, relationID core.CanonicalID, call PinnedCall, catalog []OperatorContract) (string, error) {
	candidate, err := InspectPinnedRelationExpansion(w, sourceID, relationID, call, catalog)
	if err != nil {
		return "", err
	}
	return candidate.OutputType, nil
}

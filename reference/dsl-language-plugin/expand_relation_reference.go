package languagereference

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/ir"
)

// ExpandRelationReference validates a relation against TEI's multi-namespace
// Semantic World and an exact pinned pure Operator. It does not execute a
// join, infer missing records, or authorize an extension.
func ExpandRelationReference(w ir.World, sourceID, relationID core.CanonicalID, call PinnedCall, catalog []OperatorContract) (string, error) {
	if err := ValidateRelationKeys(w, sourceID, relationID); err != nil {
		return "", err
	}
	relation, err := resolveWorldRelation(w, sourceID, relationID)
	if err != nil {
		return "", err
	}
	if len(relation.Keys) == 0 {
		return "", fmt.Errorf("relation %q has no executable key mapping", relationID)
	}
	if call.Contract.ID != "std.op.expand" {
		return "", fmt.Errorf("expected std.op.expand contract")
	}
	input := "rows/" + string(sourceID)
	expectedOutput := "rows/" + string(relation.Target)
	if call.Contract.InputType != input || call.Contract.OutputType != expectedOutput {
		return "", fmt.Errorf("expand contract does not match semantic relation endpoints")
	}
	return ComposePinned(input, []PinnedCall{call}, catalog)
}

package languagereference

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// PinnedRelationStep binds an authored relation identity to an externally
// supplied operator pin. The same operator signature can implement different
// relations, so the relation identity must survive lowering.
type PinnedRelationStep struct {
	RelationID core.CanonicalID
	Call       PinnedCall
}

// PinnedRelationPathCandidate retains the semantic meaning of a multi-hop
// expansion. It is an unaccepted, detached reference candidate, not a query
// plan, permission grant, or a substitute for Semantic Set Lock verification.
type PinnedRelationPathCandidate struct {
	Shape      RelationPathShape
	Expansions []RelationExpansionCandidate
	OutputType string
}

// InspectPinnedRelationPath checks a sequence of relation-bound expand calls.
// A one-to-many hop permanently widens the possible multiplicity; it does
// not prove that any target row exists. Missing matches and NULL handling
// remain unspecified and must be addressed before executable lowering.
func InspectPinnedRelationPath(w ir.World, sourceID core.CanonicalID, steps []PinnedRelationStep, catalog []OperatorContract) (PinnedRelationPathCandidate, error) {
	if sourceID == "" || len(steps) == 0 {
		return PinnedRelationPathCandidate{}, fmt.Errorf("pinned relation path requires a source and at least one step")
	}
	result := PinnedRelationPathCandidate{
		Shape: RelationPathShape{
			SourceID: sourceID, TargetID: sourceID,
			Multiplicity: semantic.CardinalityOne,
		},
		Expansions: make([]RelationExpansionCandidate, 0, len(steps)),
	}
	current := sourceID
	for i, step := range steps {
		if step.RelationID == "" {
			return PinnedRelationPathCandidate{}, fmt.Errorf("step %d: relation identity is required", i)
		}
		expansion, err := InspectPinnedRelationExpansion(w, current, step.RelationID, step.Call, catalog)
		if err != nil {
			return PinnedRelationPathCandidate{}, fmt.Errorf("step %d: %w", i, err)
		}
		if expansion.SourceID != current {
			return PinnedRelationPathCandidate{}, fmt.Errorf("step %d: relation source mismatch", i)
		}
		if expansion.Cardinality == semantic.CardinalityMany {
			result.Shape.Multiplicity = semantic.CardinalityMany
		}
		result.Expansions = append(result.Expansions, expansion)
		current = expansion.TargetID
	}
	result.Shape.TargetID = current
	result.OutputType = result.Expansions[len(result.Expansions)-1].OutputType
	return result, nil
}

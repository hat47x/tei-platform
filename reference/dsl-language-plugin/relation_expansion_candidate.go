package languagereference

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/validation"
)

// RelationExpansionCandidate keeps the semantic relation identity alongside
// the operator pin. The output type alone cannot distinguish two relations
// with the same source and target Data.
//
// The candidate is a detached snapshot for inspection, not an accepted
// execution plan or an independent authority over Semantic relations.
type RelationExpansionCandidate struct {
	SourceID    core.CanonicalID
	RelationID  core.CanonicalID
	TargetID    core.CanonicalID
	Cardinality semantic.Cardinality
	Keys        []semantic.RelationKey
	Operator    OperatorContract
	OutputType  string
}

// InspectPinnedRelationExpansion validates a declared relation and an exact
// pure operator pin. It does not issue a Semantic Lock or choose a provider.
// Only callers with a previously accepted World can give these values
// authoritative meaning.
func InspectPinnedRelationExpansion(w ir.World, sourceID, relationID core.CanonicalID, call PinnedCall, catalog []OperatorContract) (RelationExpansionCandidate, error) {
	// Use the existing canonical validator exactly once, then retain the
	// checked Relation identity and its authored key mapping.
	relation, err := checkedWorldRelation(w, sourceID, relationID, validation.ValidateSemanticRelationKeys(w))
	if err != nil {
		return RelationExpansionCandidate{}, err
	}
	if len(relation.Keys) == 0 {
		return RelationExpansionCandidate{}, fmt.Errorf("relation %q has no executable key mapping", relationID)
	}
	if call.Contract.ID != "std.op.expand" {
		return RelationExpansionCandidate{}, fmt.Errorf("expected std.op.expand contract")
	}
	input := "rows/" + string(sourceID)
	expectedOutput := "rows/" + string(relation.Target)
	if call.Contract.InputType != input || call.Contract.OutputType != expectedOutput {
		return RelationExpansionCandidate{}, fmt.Errorf("expand contract does not match semantic relation endpoints")
	}
	output, err := ComposePinned(input, []PinnedCall{call}, catalog)
	if err != nil {
		return RelationExpansionCandidate{}, err
	}
	return RelationExpansionCandidate{
		SourceID:    sourceID,
		RelationID:  relationID,
		TargetID:    relation.Target,
		Cardinality: relation.Cardinality,
		Keys:        append([]semantic.RelationKey(nil), relation.Keys...),
		Operator:    call.Contract,
		OutputType:  output,
	}, nil
}

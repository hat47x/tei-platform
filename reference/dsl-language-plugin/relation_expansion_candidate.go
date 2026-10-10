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
	return inspectPinnedRelationExpansionChecked(w, sourceID, relationID, call, catalog, validation.ValidateSemanticRelationKeys(w))
}

// inspectPinnedRelationExpansionChecked consumes diagnostics from the existing
// canonical validator. A multi-hop caller may compute this read-only snapshot
// once for the whole accepted World, rather than rerunning validation per hop.
func inspectPinnedRelationExpansionChecked(w ir.World, sourceID, relationID core.CanonicalID, call PinnedCall, catalog []OperatorContract, diagnostics []validation.Diagnostic) (RelationExpansionCandidate, error) {
	relation, err := checkedWorldRelation(w, sourceID, relationID, diagnostics)
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
	if call.InputType != input || !expandTypesMatch(call.Contract, sourceID, relation.Target) {
		return RelationExpansionCandidate{}, fmt.Errorf("expand contract does not match semantic relation endpoints")
	}
	// A parametric signature is pinned once and applied to each validated
	// Relation independently. Its two type variables are bound only from
	// the canonical source and target, never from provider registration order.
	output := expectedOutput
	if isParametricExpand(call.Contract) {
		if _, err := ResolveOperator(call.Contract, catalog); err != nil {
			return RelationExpansionCandidate{}, err
		}
	} else {
		resolvedOutput, err := ComposePinned(input, []PinnedCall{call}, catalog)
		if err != nil {
			return RelationExpansionCandidate{}, err
		}
		output = resolvedOutput
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

const (
	expandSourceTypeVariable = "rows/$Source"
	expandTargetTypeVariable = "rows/$Target"
)

// This Reference recognizes only the exact two-variable expand shape.
// It does not implement general-purpose type inference or issue pins.
func isParametricExpand(contract OperatorContract) bool {
	return contract.InputType == expandSourceTypeVariable &&
		contract.OutputType == expandTargetTypeVariable
}

func expandTypesMatch(contract OperatorContract, sourceID, targetID core.CanonicalID) bool {
	if isParametricExpand(contract) {
		return true
	}
	return contract.InputType == "rows/"+string(sourceID) &&
		contract.OutputType == "rows/"+string(targetID)
}

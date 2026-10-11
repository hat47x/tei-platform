package languagereference

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/validation"
)

// RelationPathShape describes a finite chain of semantic relations. Multiplicity
// is the possible number of target records per starting record; it does not
// express nullability or prove that any target record exists.
type RelationPathShape struct {
	SourceID     core.CanonicalID
	TargetID     core.CanonicalID
	Multiplicity semantic.Cardinality
}

// TraceRelationPath validates each hop using the canonical Semantic World.
// A relation may cross namespaces; keyless conceptual relations cannot
// participate in this executable-shape reference. No records are read.
func TraceRelationPath(w ir.World, sourceID core.CanonicalID, relationIDs []core.CanonicalID) (RelationPathShape, error) {
	if sourceID == "" || len(relationIDs) == 0 {
		return RelationPathShape{}, fmt.Errorf("relation path requires source and steps")
	}
	// Canonical key diagnostics are independent of the traversal order.
	// Build them once and associate them with each hop's Relation identity.
	diagnostics := validation.ValidateSemanticRelationKeys(w)
	result := RelationPathShape{SourceID: sourceID, TargetID: sourceID, Multiplicity: semantic.CardinalityOne}
	current := sourceID
	for step, relationID := range relationIDs {
		relation, err := checkedWorldRelation(w, current, relationID, diagnostics)
		if err != nil {
			return RelationPathShape{}, fmt.Errorf("step %d: %w", step, err)
		}
		if len(relation.Keys) == 0 {
			return RelationPathShape{}, fmt.Errorf("step %d: relation %q has no key mapping", step, relationID)
		}
		if relation.Cardinality == semantic.CardinalityMany {
			result.Multiplicity = semantic.CardinalityMany
		}
		current = relation.Target
	}
	result.TargetID = current
	return result, nil
}

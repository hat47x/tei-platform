package languagereference

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// RelationPathShape describes a finite chain of semantic relations. Multiplicity
// is the possible number of target records per starting record; it does not
// express nullability or prove that any target record exists.
type RelationPathShape struct {
	SourceID     core.CanonicalID
	TargetID     core.CanonicalID
	Multiplicity semantic.Cardinality
}

// TraceRelationPath validates each hop before propagating multiplicity.
// Conceptual relations without keys are valid model elements, but cannot
// participate in this executable-shape reference. No data is read or joined.
func TraceRelationPath(ns semantic.Namespace, sourceID core.CanonicalID, relationIDs []core.CanonicalID) (RelationPathShape, error) {
	if ns.ID == "" || sourceID == "" || len(relationIDs) == 0 {
		return RelationPathShape{}, fmt.Errorf("relation path requires namespace, source and steps")
	}
	result := RelationPathShape{SourceID: sourceID, TargetID: sourceID, Multiplicity: semantic.CardinalityOne}
	current := sourceID
	for step, relationID := range relationIDs {
		if err := ValidateRelationKeys(ns, current, relationID); err != nil {
			return RelationPathShape{}, fmt.Errorf("step %d: %w", step, err)
		}
		var source *semantic.Data
		for i := range ns.Data {
			if ns.Data[i].ID == current {
				source = &ns.Data[i]
				break
			}
		}
		var relation *semantic.Relation
		for i := range source.Relations {
			if source.Relations[i].ID == relationID {
				relation = &source.Relations[i]
				break
			}
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

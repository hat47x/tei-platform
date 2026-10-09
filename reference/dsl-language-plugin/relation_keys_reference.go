package languagereference

import (
	"fmt"
	"tei/internal/core"
	"tei/internal/semantic"
)

// ValidateRelationKeys checks a relation's target and explicit key pairs
// against an already accepted namespace. Empty keys remain valid for a
// conceptual relation, but do not authorize a runtime join.
func ValidateRelationKeys(ns semantic.Namespace, sourceID, relationID core.CanonicalID) error {
	var source *semantic.Data
	for i := range ns.Data {
		if ns.Data[i].ID == sourceID {
			if source != nil {
				return fmt.Errorf("duplicate source data %q", sourceID)
			}
			source = &ns.Data[i]
		}
	}
	if source == nil {
		return fmt.Errorf("unknown source data %q", sourceID)
	}
	var relation *semantic.Relation
	for i := range source.Relations {
		if source.Relations[i].ID == relationID {
			if relation != nil {
				return fmt.Errorf("duplicate relation %q", relationID)
			}
			relation = &source.Relations[i]
		}
	}
	if relation == nil {
		return fmt.Errorf("unknown relation %q", relationID)
	}
	if !relation.Cardinality.Valid() {
		return fmt.Errorf("invalid relation cardinality")
	}
	var target *semantic.Data
	for i := range ns.Data {
		if ns.Data[i].ID == relation.Target {
			if target != nil {
				return fmt.Errorf("duplicate target data %q", relation.Target)
			}
			target = &ns.Data[i]
		}
	}
	if target == nil {
		return fmt.Errorf("unknown relation target %q", relation.Target)
	}
	seenSource := map[core.CanonicalID]bool{}
	seenTarget := map[core.CanonicalID]bool{}
	for _, key := range relation.Keys {
		if key.SourceAttribute == "" || key.TargetAttribute == "" {
			return fmt.Errorf("empty relation key attribute")
		}
		if seenSource[key.SourceAttribute] || seenTarget[key.TargetAttribute] {
			return fmt.Errorf("duplicate relation key attribute")
		}
		seenSource[key.SourceAttribute] = true
		seenTarget[key.TargetAttribute] = true
		sourceCount, targetCount := 0, 0
		var sourceType, targetType core.CanonicalID
		for _, a := range source.Attributes {
			if a.ID == key.SourceAttribute {
				sourceCount++
				sourceType = a.TypeRef
			}
		}
		for _, a := range target.Attributes {
			if a.ID == key.TargetAttribute {
				targetCount++
				targetType = a.TypeRef
			}
		}
		if sourceCount != 1 || targetCount != 1 {
			return fmt.Errorf("relation key references missing or duplicate attribute")
		}
		if sourceType == "" || targetType == "" || sourceType != targetType {
			return fmt.Errorf("relation key attribute type mismatch")
		}
	}
	return nil
}

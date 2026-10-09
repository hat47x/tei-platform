package languagereference

import (
	"fmt"
	"tei/internal/core"
	"tei/internal/semantic"
)

// SemanticPath is a reference-only typed path in an externally accepted
// namespace. It does not create or accept new canonical identities.
type SemanticPath struct {
	DataID      core.CanonicalID
	AttributeID core.CanonicalID
	RelationID  core.CanonicalID
}

// ValidateSemanticPath checks the actual canonical model instead of inferring
// attributes or relation cardinality from a rows/<id> string.
func ValidateSemanticPath(ns semantic.Namespace, path SemanticPath) (semantic.Cardinality, error) {
	if ns.ID == "" || path.DataID == "" {
		return "", fmt.Errorf("missing semantic namespace or data")
	}
	var data *semantic.Data
	for i := range ns.Data {
		if ns.Data[i].ID == path.DataID {
			if data != nil {
				return "", fmt.Errorf("duplicate semantic data %q", path.DataID)
			}
			data = &ns.Data[i]
		}
	}
	if data == nil {
		return "", fmt.Errorf("unknown semantic data %q", path.DataID)
	}
	if path.AttributeID != "" && path.RelationID != "" {
		return "", fmt.Errorf("attribute and relation require separate paths")
	}
	if path.AttributeID != "" {
		matches := 0
		for _, a := range data.Attributes {
			if a.ID == path.AttributeID {
				matches++
			}
		}
		if matches != 1 {
			return "", fmt.Errorf("attribute %q: expected one definition, got %d", path.AttributeID, matches)
		}
		return semantic.CardinalityOne, nil
	}
	if path.RelationID != "" {
		matches := 0
		var cardinality semantic.Cardinality
		for _, r := range data.Relations {
			if r.ID == path.RelationID {
				matches++
				cardinality = r.Cardinality
			}
		}
		if matches != 1 {
			return "", fmt.Errorf("relation %q: expected one definition, got %d", path.RelationID, matches)
		}
		if !cardinality.Valid() {
			return "", fmt.Errorf("relation %q has invalid cardinality", path.RelationID)
		}
		return cardinality, nil
	}
	return semantic.CardinalityMany, nil
}

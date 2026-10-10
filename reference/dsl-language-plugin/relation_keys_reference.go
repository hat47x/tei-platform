package languagereference

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/validation"
)

// resolveWorldRelation locates an unambiguous relation and its endpoints in
// the existing multi-namespace semantic world. It does not resolve or accept
// Semantic Extensions, and must be given an already accepted world.
func resolveWorldRelation(w ir.World, sourceID, relationID core.CanonicalID) (semantic.Relation, error) {
	if sourceID == "" || relationID == "" {
		return semantic.Relation{}, fmt.Errorf("relation requires source and relation identities")
	}
	var source *semantic.Data
	var sourceNamespace core.CanonicalID
	for i := range w.Semantics {
		for j := range w.Semantics[i].Data {
			if w.Semantics[i].Data[j].ID != sourceID {
				continue
			}
			if source != nil {
				return semantic.Relation{}, fmt.Errorf("duplicate source data %q", sourceID)
			}
			source = &w.Semantics[i].Data[j]
			sourceNamespace = w.Semantics[i].ID
		}
	}
	if source == nil || sourceNamespace == "" {
		return semantic.Relation{}, fmt.Errorf("missing source data %q or its namespace", sourceID)
	}
	var relation *semantic.Relation
	for i := range source.Relations {
		if source.Relations[i].ID == relationID {
			if relation != nil {
				return semantic.Relation{}, fmt.Errorf("duplicate relation %q", relationID)
			}
			relation = &source.Relations[i]
		}
	}
	if relation == nil {
		return semantic.Relation{}, fmt.Errorf("unknown relation %q", relationID)
	}
	// The canonical identity belongs to the whole Semantic World, not just
	// the source Data. A collision elsewhere could mix unrelated diagnostics
	// because the canonical validator reports relation errors by ID.
	relationMatches := 0
	for _, ns := range w.Semantics {
		for _, data := range ns.Data {
			for _, candidate := range data.Relations {
				if candidate.ID == relationID {
					relationMatches++
				}
			}
		}
	}
	if relationMatches != 1 {
		return semantic.Relation{}, fmt.Errorf("relation %q: expected one definition across semantic world, got %d", relationID, relationMatches)
	}
	if !relation.Cardinality.Valid() {
		return semantic.Relation{}, fmt.Errorf("relation %q has invalid cardinality", relationID)
	}
	targetMatches := 0
	for _, ns := range w.Semantics {
		for _, data := range ns.Data {
			if data.ID == relation.Target {
				if ns.ID == "" {
					return semantic.Relation{}, fmt.Errorf("target %q has no semantic namespace", relation.Target)
				}
				targetMatches++
			}
		}
	}
	if targetMatches != 1 {
		return semantic.Relation{}, fmt.Errorf("relation %q: expected one target %q, got %d", relationID, relation.Target, targetMatches)
	}
	return *relation, nil
}

// checkedWorldRelation reuses diagnostics produced by TEI's canonical key
// validator. It only owns identity lookup and association of existing errors
// with a requested Relation; it does not define a second key rule set.
func checkedWorldRelation(w ir.World, sourceID, relationID core.CanonicalID, diagnostics []validation.Diagnostic) (semantic.Relation, error) {
	relation, err := resolveWorldRelation(w, sourceID, relationID)
	if err != nil {
		return semantic.Relation{}, err
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == validation.Error &&
			diagnostic.Subject != nil &&
			diagnostic.Subject.Kind == "semantic-relation" &&
			diagnostic.Subject.ID == string(relationID) {
			return semantic.Relation{}, fmt.Errorf("relation %q: %s: %s", relationID, diagnostic.Code, diagnostic.Message)
		}
	}
	return relation, nil
}

// ValidateRelationKeys uses TEI's existing Semantic Relation key validator for
// key ownership, type identity and mapping constraints. Its only additional
// responsibility is identifying an unambiguous source/relation/target in the
// supplied world. Keyless conceptual relations remain valid here.
func ValidateRelationKeys(w ir.World, sourceID, relationID core.CanonicalID) error {
	_, err := checkedWorldRelation(w, sourceID, relationID, validation.ValidateSemanticRelationKeys(w))
	return err
}

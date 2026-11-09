package exampleslide

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"tei/internal/ir"
	"tei/internal/semantic"
)

// CatalogFromWorld translates the checked semantic World into a read-only
// diagram vocabulary. It does not infer physical foreign keys, requiredness,
// identifier uniqueness, or unexpressed business constraints.
//
// Revision hashes the serialized semantic namespaces (not application/runtime
// state). The caller must use this exact revision in its Example before Build.
func CatalogFromWorld(world ir.World) (Catalog, error) {
	encoded, err := json.Marshal(world.Semantics)
	if err != nil {
		return Catalog{}, fmt.Errorf("encode semantic revision: %w", err)
	}
	catalog := Catalog{
		Revision: fmt.Sprintf("sha256:%x", sha256.Sum256(encoded)),
		Tables:   make(map[string]Table),
	}

	types := make(map[string]semantic.Type)
	for _, ns := range world.Semantics {
		for _, typ := range ns.Types {
			id := string(typ.ID)
			if id == "" {
				return Catalog{}, fmt.Errorf("empty semantic type ID")
			}
			if _, found := types[id]; found {
				return Catalog{}, fmt.Errorf("duplicate semantic type %q", id)
			}
			types[id] = typ
		}
	}

	// One pass for Data identity and attributes; a second for relations.
	for _, ns := range world.Semantics {
		for _, data := range ns.Data {
			id := string(data.ID)
			if id == "" {
				return Catalog{}, fmt.Errorf("empty Data ID")
			}
			if _, found := catalog.Tables[id]; found {
				return Catalog{}, fmt.Errorf("duplicate Data ID %q", id)
			}
			var table Table
			for _, attr := range data.Attributes {
				kind, err := catalogType(types, string(attr.TypeRef), map[string]bool{})
				if err != nil {
					return Catalog{}, fmt.Errorf("%s.%s: %w", id, attr.ID, err)
				}
				table.Fields = append(table.Fields, Field{
					ID: string(attr.ID), Type: kind, SemanticType: string(attr.TypeRef),
					// Attribute contains no required/visible flag. Both must be
					// explicit presentation or validation decisions elsewhere.
					Visible: true,
				})
			}
			catalog.Tables[id] = table
		}
	}

	for _, ns := range world.Semantics {
		for _, data := range ns.Data {
			for _, relation := range data.Relations {
				r := SemanticRelation{
					ID: string(relation.ID), SourceTable: string(data.ID),
					TargetTable: string(relation.Target),
					Cardinality: string(relation.Cardinality),
				}
				for _, key := range relation.Keys {
					r.Keys = append(r.Keys, SemanticRelationKey{
						SourceField: string(key.SourceAttribute),
						TargetField: string(key.TargetAttribute),
					})
				}
				catalog.Relations = append(catalog.Relations, r)
			}
		}
	}
	if err := validateCatalog(catalog); err != nil {
		return Catalog{}, fmt.Errorf("world projection invalid: %w", err)
	}
	return catalog, nil
}

// catalogType checks only what the semantic type can prove. "opaque" means
// full domain/value validation remains unresolved; it is not a success claim.
func catalogType(types map[string]semantic.Type, id string, visited map[string]bool) (string, error) {
	if visited[id] {
		return "", fmt.Errorf("cyclic type aliases involving %q", id)
	}
	typ, found := types[id]
	if !found {
		return "", fmt.Errorf("unknown semantic type %q", id)
	}
	switch typ.Kind {
	case semantic.TypeAlias:
		if typ.Underlying == "" {
			return "", fmt.Errorf("alias %q has no underlying type", id)
		}
		visited[id] = true
		return catalogType(types, string(typ.Underlying), visited)
	case semantic.TypePrimitive:
		switch id {
		case "std.integer":
			return "integer", nil
		case "std.boolean":
			return "boolean", nil
		case "std.text", "std.code":
			return "string", nil
		case "std.decimal", "std.money", "std.identifier",
			"std.date", "std.datetime", "std.time", "std.json", "std.binary":
			return "opaque", nil
		default:
			return "", fmt.Errorf("unmapped primitive %q", id)
		}
	default:
		return "", fmt.Errorf("unsupported semantic type kind %q", typ.Kind)
	}
}

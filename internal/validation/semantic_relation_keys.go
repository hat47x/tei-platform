package validation

import (
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/subject"
)

type semanticRelationKeyAttributeRef struct {
	Owner core.CanonicalID
	Type  core.CanonicalID
}

// ValidateSemanticRelationKeysは、Relationが明示したinstance traversal用keyについて、
// source / targetのSemantic attributeが正しいDataに属し、同じSemantic型を共有することを確認する。
// attribute名、物理column、Foreign Key、基底primitive型から対応を推測しない。
func ValidateSemanticRelationKeys(w ir.World) []Diagnostic {
	attributes := map[core.CanonicalID]semanticRelationKeyAttributeRef{}
	dataIDs := map[core.CanonicalID]struct{}{}

	for _, ns := range w.Semantics {
		for _, data := range ns.Data {
			dataIDs[data.ID] = struct{}{}
			for _, attribute := range data.Attributes {
				attributes[attribute.ID] = semanticRelationKeyAttributeRef{
					Owner: data.ID,
					Type:  attribute.TypeRef,
				}
			}
		}
	}

	var out []Diagnostic
	for _, ns := range w.Semantics {
		for _, sourceData := range ns.Data {
			for _, relation := range sourceData.Relations {
				if len(relation.Keys) == 0 {
					continue
				}

				relationSubject := subject.Subject{Kind: "semantic-relation", ID: string(relation.ID)}
				seenPairs := map[string]struct{}{}

				for _, key := range relation.Keys {
					pair := string(key.SourceAttribute) + "\x00" + string(key.TargetAttribute)
					if _, exists := seenPairs[pair]; exists {
						out = append(out, errFor(
							relationSubject,
							"DUPLICATE_SEMANTIC_RELATION_KEY",
							"relation %s declares the same key mapping %s -> %s more than once",
							relation.ID,
							key.SourceAttribute,
							key.TargetAttribute,
						))
						continue
					}
					seenPairs[pair] = struct{}{}

					if key.SourceAttribute == "" {
						out = append(out, errFor(
							relationSubject,
							"SEMANTIC_RELATION_KEY_SOURCE_REQUIRED",
							"relation %s has a key mapping without a source attribute",
							relation.ID,
						))
					}
					if key.TargetAttribute == "" {
						out = append(out, errFor(
							relationSubject,
							"SEMANTIC_RELATION_KEY_TARGET_REQUIRED",
							"relation %s has a key mapping without a target attribute",
							relation.ID,
						))
					}

					sourceAttribute, sourceFound := attributes[key.SourceAttribute]
					if key.SourceAttribute != "" && !sourceFound {
						out = append(out, errFor(
							relationSubject,
							"UNKNOWN_SEMANTIC_RELATION_KEY_SOURCE_ATTRIBUTE",
							"relation %s refers to unknown source attribute %s",
							relation.ID,
							key.SourceAttribute,
						))
					} else if sourceFound && sourceAttribute.Owner != sourceData.ID {
						out = append(out, errFor(
							relationSubject,
							"SEMANTIC_RELATION_KEY_SOURCE_PARENT_MISMATCH",
							"relation %s is declared under data %s but source key attribute %s belongs to data %s",
							relation.ID,
							sourceData.ID,
							key.SourceAttribute,
							sourceAttribute.Owner,
						))
					}

					targetAttribute, targetFound := attributes[key.TargetAttribute]
					if key.TargetAttribute != "" && !targetFound {
						out = append(out, errFor(
							relationSubject,
							"UNKNOWN_SEMANTIC_RELATION_KEY_TARGET_ATTRIBUTE",
							"relation %s refers to unknown target attribute %s",
							relation.ID,
							key.TargetAttribute,
						))
					} else if targetFound {
						if _, targetDataExists := dataIDs[relation.Target]; targetDataExists && targetAttribute.Owner != relation.Target {
							out = append(out, errFor(
								relationSubject,
								"SEMANTIC_RELATION_KEY_TARGET_PARENT_MISMATCH",
								"relation %s targets data %s but target key attribute %s belongs to data %s",
								relation.ID,
								relation.Target,
								key.TargetAttribute,
								targetAttribute.Owner,
							))
						}
					}

					if sourceFound && targetFound && sourceAttribute.Owner == sourceData.ID && targetAttribute.Owner == relation.Target && sourceAttribute.Type != targetAttribute.Type {
						out = append(out, errFor(
							relationSubject,
							"SEMANTIC_RELATION_KEY_TYPE_MISMATCH",
							"relation %s maps source attribute %s of type %s to target attribute %s of type %s; relation keys require the same semantic type identity",
							relation.ID,
							key.SourceAttribute,
							sourceAttribute.Type,
							key.TargetAttribute,
							targetAttribute.Type,
						))
					}
				}
			}
		}
	}

	return out
}

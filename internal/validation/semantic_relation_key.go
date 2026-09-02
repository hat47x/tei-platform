package validation

import (
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/subject"
)

type semanticRelationKeyAttributeRef struct {
	Owner core.CanonicalID
	Type  core.CanonicalID
}

// ValidateSemanticRelationKeysは、Relationに明示されたinstance key対応が
// source / targetのSemantic Dataと矛盾していないかを検査する。
// key対応が未定義の概念Relationはこの検査では拒否せず、実際にinstance traversalを
// 必要とする側がkey未定義を実行不能として扱える余地を残す。
func ValidateSemanticRelationKeys(w ir.World) []Diagnostic {
	return validateSemanticRelationKeys(w)
}

func validateSemanticRelationKeys(w ir.World) []Diagnostic {
	attributes := map[core.CanonicalID]semanticRelationKeyAttributeRef{}
	for _, ns := range w.Semantics {
		for _, data := range ns.Data {
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
				relationSubject := subject.Subject{Kind: "semantic-relation", ID: string(relation.ID)}
				seenSources := map[core.CanonicalID]struct{}{}
				seenTargets := map[core.CanonicalID]struct{}{}

				for _, key := range relation.Keys {
					sourceRef, sourceExists := attributes[key.SourceAttribute]
					targetRef, targetExists := attributes[key.TargetAttribute]

					if key.SourceAttribute == "" {
						out = append(out, errFor(
							relationSubject,
							"SEMANTIC_RELATION_KEY_SOURCE_REQUIRED",
							"semantic relation %s has an instance key without a source attribute",
							relation.ID,
						))
					} else if !sourceExists {
						out = append(out, errFor(
							relationSubject,
							"UNKNOWN_SEMANTIC_RELATION_KEY_SOURCE_ATTRIBUTE",
							"semantic relation %s refers to unknown source key attribute %s",
							relation.ID,
							key.SourceAttribute,
						))
					} else if sourceRef.Owner != sourceData.ID {
						out = append(out, errFor(
							relationSubject,
							"SEMANTIC_RELATION_KEY_SOURCE_OUTSIDE_SOURCE_DATA",
							"semantic relation %s uses source key attribute %s outside source data %s",
							relation.ID,
							key.SourceAttribute,
							sourceData.ID,
						))
					}

					if key.TargetAttribute == "" {
						out = append(out, errFor(
							relationSubject,
							"SEMANTIC_RELATION_KEY_TARGET_REQUIRED",
							"semantic relation %s has an instance key without a target attribute",
							relation.ID,
						))
					} else if !targetExists {
						out = append(out, errFor(
							relationSubject,
							"UNKNOWN_SEMANTIC_RELATION_KEY_TARGET_ATTRIBUTE",
							"semantic relation %s refers to unknown target key attribute %s",
							relation.ID,
							key.TargetAttribute,
						))
					} else if targetRef.Owner != relation.Target {
						out = append(out, errFor(
							relationSubject,
							"SEMANTIC_RELATION_KEY_TARGET_OUTSIDE_TARGET_DATA",
							"semantic relation %s uses target key attribute %s outside target data %s",
							relation.ID,
							key.TargetAttribute,
							relation.Target,
						))
					}

					if key.SourceAttribute != "" {
						if _, duplicate := seenSources[key.SourceAttribute]; duplicate {
							out = append(out, errFor(
								relationSubject,
								"DUPLICATE_SEMANTIC_RELATION_KEY_SOURCE_ATTRIBUTE",
								"semantic relation %s uses source key attribute %s more than once",
								relation.ID,
								key.SourceAttribute,
							))
						} else {
							seenSources[key.SourceAttribute] = struct{}{}
						}
					}
					if key.TargetAttribute != "" {
						if _, duplicate := seenTargets[key.TargetAttribute]; duplicate {
							out = append(out, errFor(
								relationSubject,
								"DUPLICATE_SEMANTIC_RELATION_KEY_TARGET_ATTRIBUTE",
								"semantic relation %s uses target key attribute %s more than once",
								relation.ID,
								key.TargetAttribute,
							))
						} else {
							seenTargets[key.TargetAttribute] = struct{}{}
						}
					}

					if sourceExists && targetExists && sourceRef.Owner == sourceData.ID && targetRef.Owner == relation.Target && sourceRef.Type != targetRef.Type {
						out = append(out, errFor(
							relationSubject,
							"SEMANTIC_RELATION_KEY_TYPE_MISMATCH",
							"semantic relation %s maps source attribute %s of type %s to target attribute %s of type %s",
							relation.ID,
							key.SourceAttribute,
							sourceRef.Type,
							key.TargetAttribute,
							targetRef.Type,
						))
					}
				}
			}
		}
	}

	return out
}

package validation

import (
	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/subject"
)

type persistenceRelationRef struct {
	Source   core.CanonicalID
	Relation semantic.Relation
}

type persistenceAttributeRef struct {
	Owner core.CanonicalID
}

// ValidatePersistenceRelationStorageMappingsは、Database-ownedなRelation Storageが
// Semantic Relationのsource / targetと整合していることを確認する。
//
// physical columnやForeign Key、RelationKeyから不足した意味を補完しない。
// keylessなRelationでも、Relationそのものとsource Semantic attributeが明示されていれば検証できる。
func ValidatePersistenceRelationStorageMappings(w ir.World) []Diagnostic {
	relations := make(map[core.CanonicalID]persistenceRelationRef)
	attributes := make(map[core.CanonicalID]persistenceAttributeRef)

	for _, ns := range w.Semantics {
		for _, data := range ns.Data {
			for _, attribute := range data.Attributes {
				attributes[attribute.ID] = persistenceAttributeRef{Owner: data.ID}
			}
			for _, relation := range data.Relations {
				relations[relation.ID] = persistenceRelationRef{
					Source:   data.ID,
					Relation: relation,
				}
			}
		}
	}

	var out []Diagnostic
	for _, mapping := range w.Persistence {
		if len(mapping.RelationStorage) == 0 {
			continue
		}

		mappingSubject := subject.Subject{Kind: "persistence-mapping", ID: string(mapping.ID)}
		if err := persistence.ValidateRelationStorageMappings(mapping); err != nil {
			out = append(out, errFor(
				mappingSubject,
				"INVALID_PERSISTENCE_RELATION_STORAGE",
				"Persistence Mapping %sのRelation Storageが不正です: %v",
				mapping.ID,
				err,
			))
			continue
		}

		for _, storage := range mapping.RelationStorage {
			relationRef, found := relations[storage.Relation]
			if !found {
				out = append(out, errFor(
					mappingSubject,
					"UNKNOWN_PERSISTENCE_RELATION_STORAGE_RELATION",
					"Persistence Mapping %sのRelation Storageは存在しないSemantic Relation %sを参照しています",
					mapping.ID,
					storage.Relation,
				))
				continue
			}

			if relationRef.Relation.Target != mapping.SemanticData {
				out = append(out, errFor(
					mappingSubject,
					"PERSISTENCE_RELATION_STORAGE_TARGET_MISMATCH",
					"Persistence Mapping %sはSemantic Data %sを保存しますが、Relation %sのtargetは%sです",
					mapping.ID,
					mapping.SemanticData,
					storage.Relation,
					relationRef.Relation.Target,
				))
			}

			for _, component := range storage.Components {
				attributeRef, found := attributes[component.SourceAttribute]
				if !found {
					out = append(out, errFor(
						mappingSubject,
						"UNKNOWN_PERSISTENCE_RELATION_STORAGE_SOURCE_ATTRIBUTE",
						"Persistence Mapping %sのRelation %sは存在しないsource attribute %sを物理保存へ対応付けています",
						mapping.ID,
						storage.Relation,
						component.SourceAttribute,
					))
					continue
				}
				if attributeRef.Owner != relationRef.Source {
					out = append(out, errFor(
						mappingSubject,
						"PERSISTENCE_RELATION_STORAGE_SOURCE_ATTRIBUTE_PARENT_MISMATCH",
						"Relation %sはSemantic Data %sから定義されていますが、物理保存componentのsource attribute %sは%sに属しています",
						storage.Relation,
						relationRef.Source,
						component.SourceAttribute,
						attributeRef.Owner,
					))
				}
			}
		}
	}

	return out
}

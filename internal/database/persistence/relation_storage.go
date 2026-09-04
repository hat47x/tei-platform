package persistence

import (
	"fmt"
	"strings"

	"tei/internal/core"
)

// ValidateRelationStorageMappingsは、Relation-ownedな物理support columnの局所形状を検査する。
//
// RelationやSourceAttributeが実際にWorld内のSemantic要素として存在するか、Relation source / targetと
// Mapping.SemanticDataが整合するかはWorld-level Validationの責務とする。ここではPersistence Mapping単体で
// 判定できるidentity形式、component構造、物理列衝突だけを検査する。
func ValidateRelationStorageMappings(mapping Mapping) error {
	seenRelations := make(map[core.CanonicalID]struct{}, len(mapping.RelationStorage))
	seenPhysicalColumns := make(map[string]struct{}, len(mapping.Columns))
	for _, column := range mapping.Columns {
		name := strings.TrimSpace(column.Column)
		if name != "" {
			seenPhysicalColumns[name] = struct{}{}
		}
	}

	for _, storage := range mapping.RelationStorage {
		if _, err := core.ParseCanonicalID(string(storage.Relation)); err != nil {
			return fmt.Errorf("Persistence Mapping %sのRelation Storageに不正なRelation ID %qがあります: %w", mapping.ID, storage.Relation, err)
		}
		if _, exists := seenRelations[storage.Relation]; exists {
			return fmt.Errorf("Persistence Mapping %sでRelation %sのRelation Storageが重複しています", mapping.ID, storage.Relation)
		}
		seenRelations[storage.Relation] = struct{}{}

		if len(storage.Components) == 0 {
			return fmt.Errorf("Persistence Mapping %sのRelation %sに物理保存componentがありません", mapping.ID, storage.Relation)
		}

		seenSourceAttributes := make(map[core.CanonicalID]struct{}, len(storage.Components))
		for _, component := range storage.Components {
			if _, err := core.ParseCanonicalID(string(component.SourceAttribute)); err != nil {
				return fmt.Errorf("Persistence Mapping %sのRelation %sに不正なsource attribute ID %qがあります: %w", mapping.ID, storage.Relation, component.SourceAttribute, err)
			}
			if _, exists := seenSourceAttributes[component.SourceAttribute]; exists {
				return fmt.Errorf("Persistence Mapping %sのRelation %sでsource attribute %sが重複しています", mapping.ID, storage.Relation, component.SourceAttribute)
			}
			seenSourceAttributes[component.SourceAttribute] = struct{}{}

			column := strings.TrimSpace(component.Column)
			if column == "" {
				return fmt.Errorf("Persistence Mapping %sのRelation %sに物理列名のないcomponentがあります", mapping.ID, storage.Relation)
			}
			if _, exists := seenPhysicalColumns[column]; exists {
				return fmt.Errorf("Persistence Mapping %sで物理列 %qがSemantic attribute列または別Relation Storageと重複しています", mapping.ID, column)
			}
			seenPhysicalColumns[column] = struct{}{}

			if !component.Storage.Valid() {
				return fmt.Errorf("Persistence Mapping %sのRelation %sの物理列 %qに不正なstorage type %qがあります", mapping.ID, storage.Relation, column, component.Storage)
			}
			if !component.Nullability.Valid() {
				return fmt.Errorf("Persistence Mapping %sのRelation %sの物理列 %qに不正なnullability %qがあります", mapping.ID, storage.Relation, column, component.Nullability)
			}
		}
	}

	return nil
}

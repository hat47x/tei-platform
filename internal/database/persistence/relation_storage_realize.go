package persistence

import (
	"fmt"

	"tei/internal/database/physical"
)

// realizeRelationStorageColumnsは、Relation-ownedなsupport columnだけをphysical schemaへ投影する。
//
// support columnをSemantic attribute列へ変換せず、Primary Key / Unique / Index / Foreign Keyへも
// 暗黙参加させない。それらが必要な場合は、Database-ownedな別の明示authoringで判断する。
func realizeRelationStorageColumns(
	mapping Mapping,
	startOrdinal int,
	policy PhysicalRealizationPolicy,
) ([]physical.Column, error) {
	if err := ValidateRelationStorageMappings(mapping); err != nil {
		return nil, err
	}

	columns := make([]physical.Column, 0)
	ordinal := startOrdinal
	for _, storage := range mapping.RelationStorage {
		for _, component := range storage.Components {
			physicalType, err := policy.PhysicalType(component.Storage)
			if err != nil {
				return nil, fmt.Errorf(
					"mapping %s relation %s source attribute %s: %w",
					mapping.ID,
					storage.Relation,
					component.SourceAttribute,
					err,
				)
			}
			ordinal++
			columns = append(columns, physical.Column{
				Name:         component.Column,
				PhysicalType: physicalType,
				Nullable:     component.Nullability == Optional,
				Ordinal:      ordinal,
			})
		}
	}
	return columns, nil
}

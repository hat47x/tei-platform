package persistence

import (
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/core"
)

func relationStorageMappingFixture() Mapping {
	return Mapping{
		ID:           "sales_core.order_line",
		Database:     "sales_core",
		SemanticData: "sales.order_line",
		Table:        "public.order_lines",
		Columns: []ColumnMapping{
			{
				Attribute:   "sales.order_line.order_line_id",
				Column:      "order_line_id",
				Storage:     StorageInt64,
				Nullability: Required,
			},
		},
		RelationStorage: []RelationStorageMapping{
			{
				Relation: "sales.order.lines",
				Components: []RelationStorageComponentMapping{
					{
						SourceAttribute: "sales.order.order_id",
						Column:          "order_id",
						Storage:         StorageInt64,
						Nullability:     Required,
					},
				},
			},
		},
	}
}

func TestValidateRelationStorageMappingsAcceptsDatabaseOwnedSupportColumn(t *testing.T) {
	mapping := relationStorageMappingFixture()
	if err := ValidateRelationStorageMappings(mapping); err != nil {
		t.Fatalf("keyless Relationの物理support columnを検証できません: %v", err)
	}

	storage, ok := mapping.RelationStorageByRelation("sales.order.lines")
	if !ok || len(storage.Components) != 1 {
		t.Fatalf("Relation identityから物理保存realizationを取得できません: %#v", storage)
	}
	if storage.Components[0].SourceAttribute != "sales.order.order_id" || storage.Components[0].Column != "order_id" {
		t.Fatalf("source Semantic identityと物理列の対応が変わっています: %#v", storage.Components[0])
	}
}

func TestValidateRelationStorageMappingsRejectsPhysicalColumnCollision(t *testing.T) {
	mapping := relationStorageMappingFixture()
	mapping.RelationStorage[0].Components[0].Column = "order_line_id"

	err := ValidateRelationStorageMappings(mapping)
	if err == nil || !strings.Contains(err.Error(), "order_line_id") || !strings.Contains(err.Error(), "重複") {
		t.Fatalf("Semantic attribute列とRelation Storageの物理列衝突を拒否していません: %v", err)
	}
}

func TestValidateRelationStorageMappingsRejectsDuplicateRelationAndSourceComponent(t *testing.T) {
	t.Run("relation", func(t *testing.T) {
		mapping := relationStorageMappingFixture()
		mapping.RelationStorage = append(mapping.RelationStorage, mapping.RelationStorage[0])

		err := ValidateRelationStorageMappings(mapping)
		if err == nil || !strings.Contains(err.Error(), "Relation Storageが重複") {
			t.Fatalf("同じSemantic Relationの物理保存realization重複を拒否していません: %v", err)
		}
	})

	t.Run("source attribute", func(t *testing.T) {
		mapping := relationStorageMappingFixture()
		mapping.RelationStorage[0].Components = append(
			mapping.RelationStorage[0].Components,
			RelationStorageComponentMapping{
				SourceAttribute: "sales.order.order_id",
				Column:          "parent_order_id",
				Storage:         StorageInt64,
				Nullability:     Required,
			},
		)

		err := ValidateRelationStorageMappings(mapping)
		if err == nil || !strings.Contains(err.Error(), "source attribute") || !strings.Contains(err.Error(), "重複") {
			t.Fatalf("同じsource attributeのcomponent重複を拒否していません: %v", err)
		}
	})
}

func TestValidateRelationStorageMappingsRequiresExplicitComponentsAndPhysicalShape(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Mapping)
		want   string
	}{
		{
			name: "component missing",
			mutate: func(mapping *Mapping) {
				mapping.RelationStorage[0].Components = nil
			},
			want: "componentがありません",
		},
		{
			name: "invalid source identity",
			mutate: func(mapping *Mapping) {
				mapping.RelationStorage[0].Components[0].SourceAttribute = core.CanonicalID("OrderId")
			},
			want: "source attribute ID",
		},
		{
			name: "column missing",
			mutate: func(mapping *Mapping) {
				mapping.RelationStorage[0].Components[0].Column = "   "
			},
			want: "物理列名",
		},
		{
			name: "storage invalid",
			mutate: func(mapping *Mapping) {
				mapping.RelationStorage[0].Components[0].Storage = StorageType("unknown")
			},
			want: "storage type",
		},
		{
			name: "nullability invalid",
			mutate: func(mapping *Mapping) {
				mapping.RelationStorage[0].Components[0].Nullability = Nullability("sometimes")
			},
			want: "nullability",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapping := relationStorageMappingFixture()
			test.mutate(&mapping)
			err := ValidateRelationStorageMappings(mapping)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Relation Storageの不完全な局所形状を拒否していません: %v", err)
			}
		})
	}
}

func TestRelationStorageJSONPreservesExistingMappingWhenUnusedAndCompositeOrderWhenUsed(t *testing.T) {
	legacy := relationStorageMappingFixture()
	legacy.RelationStorage = nil
	encoded, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("Persistence MappingをJSON化できません: %v", err)
	}
	if strings.Contains(string(encoded), "relationStorage") {
		t.Fatalf("Relation Storage未使用時に既存Canonical JSONへ空fieldを追加しています: %s", encoded)
	}

	mapping := relationStorageMappingFixture()
	mapping.RelationStorage[0].Components = append(
		mapping.RelationStorage[0].Components,
		RelationStorageComponentMapping{
			SourceAttribute: "sales.order.tenant_id",
			Column:          "tenant_id",
			Storage:         StorageVarchar,
			Nullability:     Required,
		},
	)
	encoded, err = json.Marshal(mapping)
	if err != nil {
		t.Fatalf("Relation StorageをJSON化できません: %v", err)
	}
	var decoded Mapping
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Relation StorageをJSON round-tripできません: %v", err)
	}
	components := decoded.RelationStorage[0].Components
	if len(components) != 2 || components[0].SourceAttribute != "sales.order.order_id" || components[1].SourceAttribute != "sales.order.tenant_id" {
		t.Fatalf("composite source identityのcomponent順序が変わっています: %#v", components)
	}
}

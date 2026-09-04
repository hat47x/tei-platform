package persistence_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/database/providers/postgres"
)

func TestRealizeDesiredAddsRelationOwnedSupportColumnAfterSemanticColumns(t *testing.T) {
	mapping := persistence.Mapping{
		ID:           "sales_core.order_line",
		Database:     "sales_core",
		SemanticData: "sales.order_line",
		Table:        "public.order_lines",
		Columns: []persistence.ColumnMapping{
			{
				Attribute:   "sales.order_line.order_line_id",
				Column:      "order_line_id",
				Storage:     persistence.StorageInt64,
				Nullability: persistence.Required,
			},
			{
				Attribute:   "sales.order_line.line_number",
				Column:      "line_number",
				Storage:     persistence.StorageInt64,
				Nullability: persistence.Required,
			},
		},
		RelationStorage: []persistence.RelationStorageMapping{
			{
				Relation: "sales.order.lines",
				Components: []persistence.RelationStorageComponentMapping{
					{
						SourceAttribute: "sales.order.order_id",
						Column:          "order_id",
						Storage:         persistence.StorageInt64,
						Nullability:     persistence.Required,
					},
				},
			},
		},
	}

	snapshot, err := persistence.RealizeDesired(
		[]persistence.Mapping{mapping},
		"sales_core",
		postgres.RealizationPolicy{},
	)
	if err != nil {
		t.Fatalf("Relation-ownedな物理support columnをDesired Schemaへ投影できません: %v", err)
	}
	if len(snapshot.Tables) != 1 {
		t.Fatalf("想定外のtable数です: %#v", snapshot.Tables)
	}
	columns := snapshot.Tables[0].Columns
	if len(columns) != 3 {
		t.Fatalf("Relation Storageの物理列が追加されていません: %#v", columns)
	}
	if columns[0].Name != "order_line_id" || columns[0].Ordinal != 1 {
		t.Fatalf("既存Semantic attribute列の順序が変わっています: %#v", columns)
	}
	if columns[1].Name != "line_number" || columns[1].Ordinal != 2 {
		t.Fatalf("既存Semantic attribute列の順序が変わっています: %#v", columns)
	}
	if columns[2].Name != "order_id" || columns[2].Ordinal != 3 {
		t.Fatalf("Relation-owned support columnのphysical identityまたは順序が変わっています: %#v", columns[2])
	}
	if columns[2].PhysicalType != "pg_catalog.int8" || columns[2].Nullable {
		t.Fatalf("Relation-owned support columnのstorage契約が反映されていません: %#v", columns[2])
	}
}

func TestRealizeDesiredDoesNotPromoteRelationStorageToSemanticPrimaryKey(t *testing.T) {
	mapping := persistence.Mapping{
		ID:           "sales_core.order_line",
		Database:     "sales_core",
		SemanticData: "sales.order_line",
		Table:        "public.order_lines",
		Columns: []persistence.ColumnMapping{
			{
				Attribute:   "sales.order_line.order_line_id",
				Column:      "order_line_id",
				Storage:     persistence.StorageInt64,
				Nullability: persistence.Required,
			},
		},
		PrimaryKey: []core.CanonicalID{"sales.order_line.order_line_id"},
		RelationStorage: []persistence.RelationStorageMapping{
			{
				Relation: "sales.order.lines",
				Components: []persistence.RelationStorageComponentMapping{
					{
						SourceAttribute: "sales.order.order_id",
						Column:          "order_id",
						Storage:         persistence.StorageInt64,
						Nullability:     persistence.Required,
					},
				},
			},
		},
	}

	snapshot, err := persistence.RealizeDesired(
		[]persistence.Mapping{mapping},
		"sales_core",
		postgres.RealizationPolicy{},
	)
	if err != nil {
		t.Fatalf("Desired Schemaを生成できません: %v", err)
	}
	primaryKey := snapshot.Tables[0].PrimaryKey
	if len(primaryKey) != 1 || primaryKey[0] != "order_line_id" {
		t.Fatalf("Relation Storage columnがSemantic Primary Keyへ暗黙参加しています: %#v", primaryKey)
	}
}

func TestRealizeDesiredRejectsRelationStorageColumnCollisionBeforePhysicalProjection(t *testing.T) {
	mapping := persistence.Mapping{
		ID:           "sales_core.order_line",
		Database:     "sales_core",
		SemanticData: "sales.order_line",
		Table:        "public.order_lines",
		Columns: []persistence.ColumnMapping{
			{
				Attribute:   "sales.order_line.order_line_id",
				Column:      "order_line_id",
				Storage:     persistence.StorageInt64,
				Nullability: persistence.Required,
			},
		},
		RelationStorage: []persistence.RelationStorageMapping{
			{
				Relation: "sales.order.lines",
				Components: []persistence.RelationStorageComponentMapping{
					{
						SourceAttribute: "sales.order.order_id",
						Column:          "order_line_id",
						Storage:         persistence.StorageInt64,
						Nullability:     persistence.Required,
					},
				},
			},
		},
	}

	if _, err := persistence.RealizeDesired(
		[]persistence.Mapping{mapping},
		"sales_core",
		postgres.RealizationPolicy{},
	); err == nil {
		t.Fatal("Semantic attribute列とRelation Storageの物理列衝突をDesired Schemaへ流しています")
	}
}

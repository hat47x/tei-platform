package postgres_test

import (
	"testing"

	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
)

func TestRelationStorageSupportColumnFlowsThroughSchemaDiffAndMigrationSQL(t *testing.T) {
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
						Column:          "order_id",
						Storage:         persistence.StorageInt64,
						Nullability:     persistence.Required,
					},
				},
			},
		},
	}

	desired, err := persistence.RealizeDesired(
		[]persistence.Mapping{mapping},
		"sales_core",
		postgres.RealizationPolicy{},
	)
	if err != nil {
		t.Fatalf("Relation Storageを含むDesired Schemaを生成できません: %v", err)
	}

	actual := physical.SchemaSnapshot{
		Database: "sales_core",
		Tables: []physical.Table{
			{
				Schema: "public",
				Name:   "order_lines",
				Kind:   physical.TableKindTable,
				Columns: []physical.Column{
					{
						Name:         "order_line_id",
						PhysicalType: "pg_catalog.int8",
						Nullable:     false,
						Ordinal:      1,
					},
				},
			},
		},
	}

	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatalf("Relation Storage追加前のActual Schemaとの差分を計算できません: %v", err)
	}
	if len(diff.Changes) != 1 {
		t.Fatalf("Relation Storage support column以外のmigration差分を生成しています: %#v", diff.Changes)
	}
	change := diff.Changes[0]
	if change.Kind != schemadiff.AddColumn || change.Table != "public.order_lines" || change.Column == nil {
		t.Fatalf("Relation Storage support columnをadd-columnとして保持できていません: %#v", change)
	}
	if change.Column.Name != "order_id" || change.Column.PhysicalType != "pg_catalog.int8" || change.Column.Nullable {
		t.Fatalf("Relation Storage support columnの物理契約がmigration差分で変わっています: %#v", change.Column)
	}

	statements, err := postgres.RenderMigrationSQL(diff)
	if err != nil {
		t.Fatalf("Relation Storage support columnのPostgreSQL migration SQLを生成できません: %v", err)
	}
	want := `ALTER TABLE "public"."order_lines" ADD COLUMN "order_id" pg_catalog.int8 NOT NULL;`
	if len(statements) != 1 || statements[0] != want {
		t.Fatalf("Relation Storage support columnのmigration SQLが期待値と異なります: %#v", statements)
	}
}

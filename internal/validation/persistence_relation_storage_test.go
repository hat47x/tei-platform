package validation

import (
	"testing"

	"tei/internal/database/persistence"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func persistenceRelationStorageWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Data: []semantic.Data{
					{
						ID: "sales.order",
						Attributes: []semantic.Attribute{
							{ID: "sales.order.order_id", TypeRef: "sales.order_id"},
						},
						Relations: []semantic.Relation{
							{
								ID:          "sales.order.lines",
								Target:      "sales.order_line",
								Cardinality: semantic.CardinalityMany,
							},
						},
					},
					{
						ID: "sales.order_line",
						Attributes: []semantic.Attribute{
							{ID: "sales.order_line.order_line_id", TypeRef: "sales.order_line_id"},
						},
					},
				},
			},
		},
		Persistence: []persistence.Mapping{
			{
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
			},
		},
	}
}

func TestValidatePersistenceRelationStorageMappingsAcceptsKeylessManyRelation(t *testing.T) {
	world := persistenceRelationStorageWorld()
	if len(world.Semantics[0].Data[0].Relations[0].Keys) != 0 {
		t.Fatal("fixtureがkeyless Relationではありません")
	}

	if diagnostics := ValidatePersistenceRelationStorageMappings(world); len(diagnostics) != 0 {
		t.Fatalf("keylessなmany RelationのDatabase-owned storageを拒否しています: %#v", diagnostics)
	}
}

func TestValidatePersistenceRelationStorageMappingsRejectsUnknownRelation(t *testing.T) {
	world := persistenceRelationStorageWorld()
	world.Persistence[0].RelationStorage[0].Relation = "sales.order.unknown"

	diagnostics := ValidatePersistenceRelationStorageMappings(world)
	assertDiagnosticCode(t, diagnostics, "UNKNOWN_PERSISTENCE_RELATION_STORAGE_RELATION")
}

func TestValidatePersistenceRelationStorageMappingsRequiresRelationTargetToMatchMappingData(t *testing.T) {
	world := persistenceRelationStorageWorld()
	world.Persistence[0].SemanticData = "sales.order"

	diagnostics := ValidatePersistenceRelationStorageMappings(world)
	assertDiagnosticCode(t, diagnostics, "PERSISTENCE_RELATION_STORAGE_TARGET_MISMATCH")
}

func TestValidatePersistenceRelationStorageMappingsRequiresSourceAttributeFromRelationSourceData(t *testing.T) {
	world := persistenceRelationStorageWorld()
	world.Persistence[0].RelationStorage[0].Components[0].SourceAttribute = "sales.order_line.order_line_id"

	diagnostics := ValidatePersistenceRelationStorageMappings(world)
	assertDiagnosticCode(t, diagnostics, "PERSISTENCE_RELATION_STORAGE_SOURCE_ATTRIBUTE_PARENT_MISMATCH")
}

func TestValidatePersistenceRelationStorageMappingsRejectsUnknownSourceAttribute(t *testing.T) {
	world := persistenceRelationStorageWorld()
	world.Persistence[0].RelationStorage[0].Components[0].SourceAttribute = "sales.order.unknown"

	diagnostics := ValidatePersistenceRelationStorageMappings(world)
	assertDiagnosticCode(t, diagnostics, "UNKNOWN_PERSISTENCE_RELATION_STORAGE_SOURCE_ATTRIBUTE")
}

func TestValidatePersistenceRelationStorageMappingsPropagatesLocalShapeErrors(t *testing.T) {
	world := persistenceRelationStorageWorld()
	world.Persistence[0].RelationStorage[0].Components[0].Column = "order_line_id"

	diagnostics := ValidatePersistenceRelationStorageMappings(world)
	assertDiagnosticCode(t, diagnostics, "INVALID_PERSISTENCE_RELATION_STORAGE")
}

func TestValidateIncludesPersistenceRelationStorageMappings(t *testing.T) {
	world := persistenceRelationStorageWorld()
	world.Persistence[0].RelationStorage[0].Components[0].SourceAttribute = "sales.order_line.order_line_id"

	diagnostics := Validate(world)
	assertDiagnosticCode(t, diagnostics, "PERSISTENCE_RELATION_STORAGE_SOURCE_ATTRIBUTE_PARENT_MISMATCH")
}

func assertDiagnosticCode(t *testing.T, diagnostics []Diagnostic, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("Diagnostic %sがありません: %#v", code, diagnostics)
}

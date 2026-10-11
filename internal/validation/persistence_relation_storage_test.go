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

// All Relation Storage checks must run on the same default path as the CLI.
// The specialist validator is not a substitute for Workspace validation.
func TestDefaultValidationRejectsMalformedRelationStorageMapping(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ir.World)
		code   string
	}{
		{
			name: "unknown semantic relation",
			mutate: func(world *ir.World) {
				world.Persistence[0].RelationStorage[0].Relation = "sales.order.unknown"
			},
			code: "UNKNOWN_PERSISTENCE_RELATION_STORAGE_RELATION",
		},
		{
			name: "storage targets wrong semantic data",
			mutate: func(world *ir.World) {
				world.Persistence[0].SemanticData = "sales.order"
			},
			code: "PERSISTENCE_RELATION_STORAGE_TARGET_MISMATCH",
		},
		{
			name: "source attribute belongs to wrong data",
			mutate: func(world *ir.World) {
				world.Persistence[0].RelationStorage[0].Components[0].SourceAttribute =
					"sales.order_line.order_line_id"
			},
			code: "PERSISTENCE_RELATION_STORAGE_SOURCE_ATTRIBUTE_PARENT_MISMATCH",
		},
		{
			name: "missing source attribute",
			mutate: func(world *ir.World) {
				world.Persistence[0].RelationStorage[0].Components[0].SourceAttribute =
					"sales.order.unknown"
			},
			code: "UNKNOWN_PERSISTENCE_RELATION_STORAGE_SOURCE_ATTRIBUTE",
		},
		{
			name: "invalid local storage shape",
			mutate: func(world *ir.World) {
				world.Persistence[0].RelationStorage[0].Components[0].Column = "order_line_id"
			},
			code: "INVALID_PERSISTENCE_RELATION_STORAGE",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			world := persistenceRelationStorageWorld()
			tc.mutate(&world)
			count := 0
			for _, d := range Validate(world) {
				if d.Code != tc.code {
					continue
				}
				if d.Severity != Error || d.Subject == nil ||
					d.Subject.Kind != "persistence-mapping" ||
					d.Subject.ID != "sales_core.order_line" {
					t.Fatalf("Relation Storage error has no source-resolvable Persistence identity: %+v", d)
				}
				count++
			}
			if count != 1 {
				t.Fatalf("default Validation must report %s exactly once, got %d", tc.code, count)
			}
		})
	}
}

func TestDefaultValidationAcceptsExplicitValidRelationStorage(t *testing.T) {
	world := persistenceRelationStorageWorld()
	for _, d := range Validate(world) {
		switch d.Code {
		case "INVALID_PERSISTENCE_RELATION_STORAGE",
			"UNKNOWN_PERSISTENCE_RELATION_STORAGE_RELATION",
			"PERSISTENCE_RELATION_STORAGE_TARGET_MISMATCH",
			"UNKNOWN_PERSISTENCE_RELATION_STORAGE_SOURCE_ATTRIBUTE",
			"PERSISTENCE_RELATION_STORAGE_SOURCE_ATTRIBUTE_PARENT_MISMATCH":
			t.Fatalf("valid explicit Relation Storage rejected: %+v", d)
		}
	}
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

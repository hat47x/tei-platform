package ercrossview

import (
	"encoding/json"
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func sampleWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{ID: "sales", Data: []semantic.Data{
			{
				ID: "sales.order_line", Symbol: "OrderLine",
				Attributes: []semantic.Attribute{
					{ID: "sales.order_line.row_uuid", Symbol: "rowUuid", TypeRef: "std.identifier"},
					{ID: "sales.order_line.order_number", Symbol: "orderNumber", TypeRef: "std.code"},
					{ID: "sales.order_line.line_number", Symbol: "lineNumber", TypeRef: "std.integer"},
					{ID: "sales.order_line.item_code", Symbol: "itemCode", TypeRef: "std.code"},
				},
				CandidateKeys: []semantic.CandidateKey{{
					ID: "sales.order_line.business_line", Symbol: "businessLine",
					Attributes: []core.CanonicalID{"sales.order_line.order_number", "sales.order_line.line_number"},
				}},
				FunctionalDependencies: []semantic.FunctionalDependency{{
					ID: "sales.order_line.item_by_line", Symbol: "itemByLine",
					Determinants: []core.CanonicalID{"sales.order_line.order_number", "sales.order_line.line_number"},
					Dependents:   []core.CanonicalID{"sales.order_line.item_code"},
				}},
				Relations: []semantic.Relation{{
					ID: "sales.order_line.parent", Symbol: "parent", Target: "sales.order",
					Cardinality: semantic.CardinalityOne,
					Keys: []semantic.RelationKey{{
						SourceAttribute: "sales.order_line.order_number",
						TargetAttribute: "sales.order.order_number",
					}},
				}},
			},
			{ID: "sales.order", Symbol: "Order",
				Attributes: []semantic.Attribute{{ID: "sales.order.order_number", Symbol: "orderNumber", TypeRef: "std.code"}}},
		}}},
		Persistence: []persistence.Mapping{
			{
				ID: "store.sales_line", Database: "store", SemanticData: "sales.order_line", Table: "order_lines",
				Columns: []persistence.ColumnMapping{
					{Attribute: "sales.order_line.row_uuid", Column: "id", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
					{Attribute: "sales.order_line.order_number", Column: "order_no", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
					{Attribute: "sales.order_line.line_number", Column: "line_no", Storage: persistence.StorageInt64, Nullability: persistence.Required},
					{Attribute: "sales.order_line.item_code", Column: "item_code", Storage: persistence.StorageVarchar, Nullability: persistence.Optional},
				},
				PrimaryKey: []core.CanonicalID{"sales.order_line.row_uuid"},
				UniqueConstraints: []persistence.UniqueConstraintMapping{{
					Name:       "business_line_unique",
					Attributes: []core.CanonicalID{"sales.order_line.order_number", "sales.order_line.line_number"},
				}},
				ForeignKeys: []persistence.ForeignKeyMapping{{
					Name: "fk_parent", Attributes: []core.CanonicalID{"sales.order_line.order_number"},
					ReferencedMapping:    "store.sales_order",
					ReferencedAttributes: []core.CanonicalID{"sales.order.order_number"},
				}},
			},
		},
	}
}

func TestProjectionSeparatesLogicalCandidateKeyFromPhysicalPrimaryKey(t *testing.T) {
	world := sampleWorld()
	baseline, err := json.Marshal(world)
	if err != nil {
		t.Fatal(err)
	}
	view, err := Project(world)
	if err != nil {
		t.Fatal(err)
	}
	if view.Schema != "tei.reference.er-physical-crossview/v0" ||
		len(view.StructureRevision) != 71 || len(view.Data) != 2 ||
		len(view.Relations) != 1 || len(view.Tables) != 1 || len(view.KeyComparisons) != 2 {
		t.Fatalf("incomplete projection: %+v", view)
	}
	orderLine := view.Data[1]
	if orderLine.ID != "sales.order_line" ||
		!reflect.DeepEqual(orderLine.PhysicalMappingIDs, []string{"store.sales_line"}) ||
		!reflect.DeepEqual(orderLine.CandidateKeys[0].Attributes,
			[]string{"sales.order_line.order_number", "sales.order_line.line_number"}) ||
		!reflect.DeepEqual(orderLine.Dependencies[0].Dependents, []string{"sales.order_line.item_code"}) {
		t.Fatalf("logical ER view missing explicit declarations: %+v", orderLine)
	}
	table := view.Tables[0]
	if table.Name != "order_lines" ||
		!reflect.DeepEqual(table.PrimaryKey, []string{"sales.order_line.row_uuid"}) ||
		table.Columns[0].Name != "id" ||
		len(table.ForeignKeys) != 1 || table.ForeignKeys[0].ReferencedMapping != "store.sales_order" {
		t.Fatalf("physical authored table changed or inferred: %+v", table)
	}
	if view.KeyComparisons[0].ConstraintKind != "primary-key" ||
		view.KeyComparisons[0].Status != "distinct-attribute-set" ||
		view.KeyComparisons[1].ConstraintKind != "unique" ||
		view.KeyComparisons[1].Status != "same-attribute-set" {
		t.Fatalf("logical and physical key identities conflated: %+v", view.KeyComparisons)
	}
	for _, edge := range view.Relations {
		if edge.ID != "sales.order_line.parent" || len(edge.Keys) != 1 ||
			edge.Keys[0].SourceAttribute != "sales.order_line.order_number" {
			t.Fatalf("semantic join relation was lost: %+v", edge)
		}
	}
	if len(view.Unresolved) != 2 ||
		view.Unresolved[0].Code != "FK_REFERENCED_MAPPING_NOT_PRESENT" ||
		view.Unresolved[0].Subject != "store.sales_line:fk_parent->store.sales_order" ||
		view.Unresolved[1].Code != "NO_EXPLICIT_PHYSICAL_MAPPING" ||
		view.Unresolved[1].Subject != "sales.order" {
		t.Fatalf("unmapped Data or missing FK target was falsely inferred away: %+v", view.Unresolved)
	}
	after, err := json.Marshal(world)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(baseline, after) {
		t.Fatal("read-only projection mutated World")
	}
	again, err := Project(world)
	if err != nil || !reflect.DeepEqual(view, again) {
		t.Fatalf("projection was not deterministic: %+v %v", again, err)
	}
}

func TestProjectionWithoutExplicitKeysNeverInfersThemFromPK(t *testing.T) {
	world := sampleWorld()
	world.Semantics[0].Data[0].CandidateKeys = nil
	world.Semantics[0].Data[0].FunctionalDependencies = nil
	view, err := Project(world)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Data[1].CandidateKeys) != 0 || len(view.Data[1].Dependencies) != 0 ||
		len(view.KeyComparisons) != 0 || len(view.Tables[0].PrimaryKey) != 1 {
		t.Fatalf("physical PK or index was treated as an implicit business key: %+v", view)
	}
}

func TestProjectionSupportsMultipleMappingsAndDoesNotInferForeignKeys(t *testing.T) {
	world := sampleWorld()
	other := world.Persistence[0]
	other.ID = "archive.sales_line"
	other.Database = "archive"
	other.Table = "archive_lines"
	other.PrimaryKey = nil
	other.UniqueConstraints = nil
	other.ForeignKeys = nil
	world.Persistence = append(world.Persistence, other)
	// A declared Semantic Relation by itself never adds a physical FK.
	world.Persistence[0].ForeignKeys = nil
	view, err := Project(world)
	if err != nil {
		t.Fatal(err)
	}
	if got := view.Data[1].PhysicalMappingIDs; !reflect.DeepEqual(got, []string{"archive.sales_line", "store.sales_line"}) {
		t.Fatalf("many physical mappings were collapsed: %v", got)
	}
	for _, table := range view.Tables {
		if len(table.ForeignKeys) != 0 {
			t.Fatalf("physical FK was inferred from Semantic Relation: %+v", table)
		}
	}
	if len(view.Relations) != 1 {
		t.Fatalf("logical relation lost on physical divergence: %+v", view.Relations)
	}
}

func TestProjectionReportsUnresolvedExternalMappingButDoesNotInventEvidence(t *testing.T) {
	world := sampleWorld()
	world.Persistence[0].Columns[0].Attribute = "sales.order_line.missing"
	world.Persistence[0].PrimaryKey = []core.CanonicalID{"sales.order_line.row_uuid"}
	world.Persistence[0].SemanticData = "sales.missing"
	view, err := Project(world)
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]bool{}
	for _, finding := range view.Unresolved {
		codes[finding.Code] = true
	}
	for _, code := range []string{
		"MAPPING_DATA_NOT_PRESENT",
		"MAPPED_ATTRIBUTE_NOT_IN_DATA",
		"PK_ATTRIBUTE_HAS_NO_COLUMN",
		"NO_EXPLICIT_PHYSICAL_MAPPING",
	} {
		if !codes[code] {
			t.Fatalf("missing %s unresolved finding: %+v", code, view.Unresolved)
		}
	}
}

func TestProjectionRejectsDuplicateDataMappingAndColumns(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*ir.World)
	}{
		{"duplicate Data", func(w *ir.World) { w.Semantics[0].Data = append(w.Semantics[0].Data, w.Semantics[0].Data[0]) }},
		{"duplicate Mapping", func(w *ir.World) { w.Persistence = append(w.Persistence, w.Persistence[0]) }},
		{"duplicate column attribute", func(w *ir.World) { w.Persistence[0].Columns[1].Attribute = w.Persistence[0].Columns[0].Attribute }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			world := sampleWorld()
			tc.edit(&world)
			if _, err := Project(world); err == nil {
				t.Fatal("ambiguous projection input was accepted")
			}
		})
	}
}

func TestAttributeComparisonDoesNotAcceptDuplicatesAsMatching(t *testing.T) {
	if sameSet([]string{"a", "b"}, []string{"a", "a"}) ||
		sameSet([]string{"a", "a"}, []string{"a", "a"}) ||
		sameSet(nil, nil) ||
		!sameSet([]string{"b", "a"}, []string{"a", "b"}) {
		t.Fatal("set equality confused duplicated or absent key components")
	}
}

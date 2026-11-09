package ercrossview

import (
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
)

func hasFinding(findings []Unresolved, code, subject string) bool {
	for _, item := range findings {
		if item.Code == code && item.Subject == subject {
			return true
		}
	}
	return false
}

func TestCrossviewSuppressesFalseMatchingConstraintWhenColumnsAreNotMapped(t *testing.T) {
	world := sampleWorld()
	// Even when the declared PK and logical business key list the same
	// semantic attributes, a missing ColumnMapping is a distinct problem.
	world.Persistence[0].PrimaryKey = []core.CanonicalID{
		"sales.order_line.order_number", "sales.order_line.line_number",
	}
	world.Persistence[0].Columns = world.Persistence[0].Columns[:2] // line_number has no physical column.
	out, err := Project(world)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.KeyComparisons) != 2 {
		t.Fatalf("expected PK and UNIQUE comparisons: %+v", out.KeyComparisons)
	}
	for _, c := range out.KeyComparisons {
		if c.Status != "unresolved-physical-constraint" {
			t.Fatalf("unmapped declared PK/UNIQUE was falsely reported as %q: %+v", c.Status, c)
		}
	}
	for _, item := range []struct{ code, subject string }{
		{"PK_ATTRIBUTE_HAS_NO_COLUMN", "store.sales_line:sales.order_line.line_number"},
		{"UNIQUE_ATTRIBUTE_HAS_NO_COLUMN", "store.sales_line:business_line_unique:sales.order_line.line_number"},
	} {
		if !hasFinding(out.Unresolved, item.code, item.subject) {
			t.Fatalf("missing physical mapping evidence %s %s: %+v", item.code, item.subject, out.Unresolved)
		}
	}
}

func TestCrossviewDetectsDuplicateAndEmptyPhysicalUniqueDeclarations(t *testing.T) {
	world := sampleWorld()
	world.Persistence[0].UniqueConstraints = []persistence.UniqueConstraintMapping{
		{Name: "same_twice", Attributes: []core.CanonicalID{
			"sales.order_line.order_number", "sales.order_line.order_number",
		}},
		{Name: "empty"},
	}
	out, err := Project(world)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFinding(out.Unresolved, "UNIQUE_ATTRIBUTE_REPEATED",
		"store.sales_line:same_twice:sales.order_line.order_number") ||
		!hasFinding(out.Unresolved, "UNIQUE_ATTRIBUTES_EMPTY", "store.sales_line:empty") {
		t.Fatalf("duplicate or empty UNIQUE silently trusted: %+v", out.Unresolved)
	}
	for _, c := range out.KeyComparisons {
		if c.ConstraintKind == "unique" && c.Status == "same-attribute-set" {
			t.Fatalf("invalid physical UNIQUE was equated to a business identity: %+v", c)
		}
	}
}

func TestCrossviewForeignKeyOnlyLinksExplicitValidMappingTargets(t *testing.T) {
	world := sampleWorld()
	// The reference fixture has a declared FK pointing to a mapping absent
	// from this World. It must remain unresolved, not be synthesized.
	before, err := Project(world)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFinding(before.Unresolved, "FK_REFERENCED_MAPPING_NOT_PRESENT",
		"store.sales_line:fk_parent->store.sales_order") {
		t.Fatalf("unavailable physical FK mapping was inferred: %+v", before.Unresolved)
	}
	world.Persistence = append(world.Persistence, persistence.Mapping{
		ID: "store.sales_order", Database: "store", SemanticData: "sales.order", Table: "orders",
		Columns: []persistence.ColumnMapping{{
			Attribute: "sales.order.order_number", Column: "order_no", Storage: persistence.StorageVarchar, Nullability: persistence.Required,
		}},
	})
	valid, err := Project(world)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range valid.Unresolved {
		if f.Code == "FK_REFERENCED_MAPPING_NOT_PRESENT" ||
			f.Code == "FK_REFERENCED_ATTRIBUTE_HAS_NO_COLUMN" {
			t.Fatalf("explicitly mapped FK target still unresolved: %+v", valid.Unresolved)
		}
	}
	if len(valid.Tables) != 2 ||
		valid.Tables[0].MappingID != "store.sales_line" ||
		len(valid.Tables[0].ForeignKeys) != 1 ||
		valid.Tables[0].ForeignKeys[0].ReferencedMapping != "store.sales_order" {
		t.Fatalf("FK was fabricated or lost when mapping became available: %+v", valid.Tables)
	}

	// The referenced Mapping exists, but its declared Attribute has no
	// target ColumnMapping. The FK must still be indeterminate.
	world.Persistence[0].ForeignKeys[0].ReferencedAttributes =
		[]core.CanonicalID{"sales.order.missing"}
	invalid, err := Project(world)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFinding(invalid.Unresolved, "FK_REFERENCED_ATTRIBUTE_HAS_NO_COLUMN",
		"store.sales_line:fk_parent:sales.order.missing") {
		t.Fatalf("referenced attribute without column was trusted: %+v", invalid.Unresolved)
	}
}

func TestCrossviewDetectsLocalFKMissingAndAmbiguousAttributePairs(t *testing.T) {
	world := sampleWorld()
	world.Persistence[0].ForeignKeys = []persistence.ForeignKeyMapping{{
		Name: "malformed", Attributes: []core.CanonicalID{
			"sales.order_line.missing", "sales.order_line.missing",
		},
		ReferencedMapping: "store.sales_order", ReferencedAttributes: []core.CanonicalID{"sales.order.order_number"},
	}}
	out, err := Project(world)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []struct{ code, subject string }{
		{"FK_ATTRIBUTE_HAS_NO_COLUMN", "store.sales_line:malformed:sales.order_line.missing"},
		{"FK_ATTRIBUTE_REPEATED", "store.sales_line:malformed:sales.order_line.missing"},
		{"FK_ATTRIBUTE_COUNT_MISMATCH", "store.sales_line:malformed"},
		{"FK_REFERENCED_MAPPING_NOT_PRESENT", "store.sales_line:malformed->store.sales_order"},
	} {
		if !hasFinding(out.Unresolved, expected.code, expected.subject) {
			t.Fatalf("missing %s/%s: %+v", expected.code, expected.subject, out.Unresolved)
		}
	}
	again, err := Project(world)
	if err != nil || !reflect.DeepEqual(out, again) {
		t.Fatalf("invalid yet read-only evidence report is unstable: %+v %v", again, err)
	}
}

func TestPhysicalConstraintCompletenessDoesNotClaimLiveEnforcement(t *testing.T) {
	for _, tc := range []struct {
		attributes []string
		complete   bool
	}{
		{[]string{"data.a", "data.b"}, true},
		{[]string{"data.a", "data.a"}, false},
		{[]string{"data.a", "data.missing"}, false},
		{nil, false},
	} {
		table := TableView{Columns: []ColumnView{
			{Attribute: "data.a", Name: "a"}, {Attribute: "data.b", Name: "b"},
		}}
		if got := constraintHasMappedColumns(table, tc.attributes); got != tc.complete {
			t.Fatalf("declared ColumnMapping completeness %v: got %v want %v", tc.attributes, got, tc.complete)
		}
	}
}

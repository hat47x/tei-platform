package schematosemantic_test

import (
	"testing"
	"time"

	"tei/internal/database/physical"
	"tei/internal/database/schematosemantic"
	"tei/internal/semantic"
)

func TestProjectBasicTable(t *testing.T) {
	snap := physical.SchemaSnapshot{
		Database:    "sales",
		CapturedAt:  "2026-11-11T00:00:00Z",
		Fingerprint: "fp-1",
		Tables: []physical.Table{
			{
				Schema: "public",
				Name:   "products",
				Columns: []physical.Column{
					{Name: "product_id", PhysicalType: "int", Nullable: false, Ordinal: 1},
					{Name: "name", PhysicalType: "text", Nullable: false, Ordinal: 2},
					{Name: "price", PhysicalType: "numeric", Nullable: true, Ordinal: 3},
				},
				PrimaryKey: []string{"product_id"},
			},
		},
	}

	result, err := schematosemantic.Project(snap, schematosemantic.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}

	if result.Namespace.ID != "sales" {
		t.Fatalf("namespace id=%q", result.Namespace.ID)
	}
	if len(result.Namespace.Data) != 1 {
		t.Fatalf("data count=%d", len(result.Namespace.Data))
	}

	data := result.Namespace.Data[0]
	if data.ID != "public_products" {
		t.Fatalf("data id=%q", data.ID)
	}
	if len(data.Attributes) != 3 {
		t.Fatalf("attributes=%#v", data.Attributes)
	}
	if data.Attributes[0].Symbol != "product_id" || !data.Attributes[0].Required {
		t.Fatalf("product_id attr=%#v", data.Attributes[0])
	}
	if data.Attributes[2].Required {
		t.Fatalf("price should be nullable")
	}

	if len(data.CandidateKeys) != 1 {
		t.Fatalf("candidate keys=%#v", data.CandidateKeys)
	}
	if data.CandidateKeys[0].Symbol != "PK_products_structural_candidate" {
		t.Fatalf("pk symbol=%q", data.CandidateKeys[0].Symbol)
	}
	if len(data.CandidateKeys[0].Attributes) != 1 || data.CandidateKeys[0].Attributes[0] != "public_products_product_id" {
		t.Fatalf("pk attrs=%#v", data.CandidateKeys[0].Attributes)
	}

	if result.Provenance.ContentFingerprint == "" {
		t.Fatal("missing content fingerprint")
	}
	if result.Provenance.CapturedAt != "2026-11-11T00:00:00Z" {
		t.Fatalf("capturedAt=%q", result.Provenance.CapturedAt)
	}
}

func TestProjectRelationsFromForeignKey(t *testing.T) {
	snap := physical.SchemaSnapshot{
		Database:   "sales",
		CapturedAt: time.Now().UTC().Format(time.RFC3339),
		Tables: []physical.Table{
			{
				Schema: "public",
				Name:   "orders",
				Columns: []physical.Column{
					{Name: "order_id", PhysicalType: "int", Nullable: false, Ordinal: 1},
					{Name: "customer_id", PhysicalType: "int", Nullable: false, Ordinal: 2},
				},
				PrimaryKey: []string{"order_id"},
				ForeignKeys: []physical.ForeignKey{
					{
						Name:              "orders_customer_fk",
						Columns:           []string{"customer_id"},
						ReferencedTable:   "customers",
						ReferencedColumns: []string{"customer_id"},
					},
				},
			},
			{
				Schema: "public",
				Name:   "customers",
				Columns: []physical.Column{
					{Name: "customer_id", PhysicalType: "int", Nullable: false, Ordinal: 1},
					{Name: "name", PhysicalType: "text", Nullable: false, Ordinal: 2},
				},
				PrimaryKey: []string{"customer_id"},
			},
		},
	}

	result, err := schematosemantic.Project(snap, schematosemantic.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}

	var orders *semantic.Data
	for i := range result.Namespace.Data {
		if result.Namespace.Data[i].ID == "public_orders" {
			orders = &result.Namespace.Data[i]
			break
		}
	}
	if orders == nil {
		t.Fatal("public_orders not found in semantic model")
	}
	if len(orders.Relations) != 1 {
		t.Fatalf("relations=%#v", orders.Relations)
	}
	if orders.Relations[0].Target != "public_customers" {
		t.Fatalf("relation target=%q", orders.Relations[0].Target)
	}
	if len(orders.Relations[0].Keys) != 1 || orders.Relations[0].Keys[0].SourceAttribute != "public_orders_customer_id" {
		t.Fatalf("relation keys=%#v", orders.Relations[0].Keys)
	}
}

func TestProjectDiagnostics(t *testing.T) {
	snap := physical.SchemaSnapshot{
		Database:   "test",
		CapturedAt: time.Now().UTC().Format(time.RFC3339),
		Tables: []physical.Table{
			{
				Schema:  "main",
				Name:    "no_pk",
				Columns: []physical.Column{{Name: "id", PhysicalType: "int", Nullable: false, Ordinal: 1}},
			},
			{
				Schema:     "main",
				Name:       "dangling_fk",
				Columns:    []physical.Column{{Name: "id", PhysicalType: "int", Nullable: false, Ordinal: 1}},
				PrimaryKey: []string{"id"},
				ForeignKeys: []physical.ForeignKey{
					{
						Name:              "dangling_fk",
						Columns:           []string{"parent_id"},
						ReferencedTable:   "missing_parent",
						ReferencedColumns: []string{"id"},
					},
				},
			},
		},
	}

	result, err := schematosemantic.Project(snap, schematosemantic.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}

	var codes []string
	for _, d := range result.Diagnostics {
		codes = append(codes, d.Code)
	}
	if len(codes) != 3 { // no-candidate-key, physical-pk-requires-owner-acceptance (x2), dangling-foreign-key
		t.Fatalf("diagnostics=%#v", codes)
	}
}

func TestProjectWithoutOwnerAcceptance(t *testing.T) {
	snap := physical.SchemaSnapshot{
		Database:   "test",
		CapturedAt: time.Now().UTC().Format(time.RFC3339),
		Tables: []physical.Table{
			{
				Schema:     "public",
				Name:       "t",
				Columns:    []physical.Column{{Name: "id", PhysicalType: "int", Nullable: false, Ordinal: 1}},
				PrimaryKey: []string{"id"},
			},
		},
	}

	result, err := schematosemantic.Project(snap, schematosemantic.Options{RequireOwnerAcceptance: false})
	if err != nil {
		t.Fatal(err)
	}

	if result.Namespace.Data[0].CandidateKeys[0].Symbol != "PK_t" {
		t.Fatalf("pk symbol=%q", result.Namespace.Data[0].CandidateKeys[0].Symbol)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics=%#v", result.Diagnostics)
	}
}

func TestProjectCircularReference(t *testing.T) {
	snap := physical.SchemaSnapshot{
		Database:   "test",
		CapturedAt: time.Now().UTC().Format(time.RFC3339),
		Tables: []physical.Table{
			{
				Schema:     "public",
				Name:       "categories",
				Columns:    []physical.Column{{Name: "id", PhysicalType: "int", Nullable: false, Ordinal: 1}, {Name: "parent_id", PhysicalType: "int", Nullable: true, Ordinal: 2}},
				PrimaryKey: []string{"id"},
				ForeignKeys: []physical.ForeignKey{
					{
						Name:              "categories_parent_fk",
						Columns:           []string{"parent_id"},
						ReferencedTable:   "categories",
						ReferencedColumns: []string{"id"},
					},
				},
			},
		},
	}

	result, err := schematosemantic.Project(snap, schematosemantic.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Namespace.Data) != 1 {
		t.Fatalf("expected 1 data entity, got %d", len(result.Namespace.Data))
	}
	categories := result.Namespace.Data[0]
	if len(categories.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(categories.Relations))
	}
	if categories.Relations[0].Target != "public_categories" {
		t.Fatalf("relation target=%q", categories.Relations[0].Target)
	}
}

func TestProjectNullableAttributes(t *testing.T) {
	snap := physical.SchemaSnapshot{
		Database:   "test",
		CapturedAt: time.Now().UTC().Format(time.RFC3339),
		Tables: []physical.Table{
			{
				Schema:     "public",
				Name:       "items",
				Columns: []physical.Column{
					{Name: "id", PhysicalType: "int", Nullable: false, Ordinal: 1},
					{Name: "name", PhysicalType: "text", Nullable: false, Ordinal: 2},
					{Name: "description", PhysicalType: "text", Nullable: true, Ordinal: 3},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	result, err := schematosemantic.Project(snap, schematosemantic.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}

	items := result.Namespace.Data[0]
	if len(items.Attributes) != 3 {
		t.Fatalf("attributes=%#v", items.Attributes)
	}
	if !items.Attributes[0].Required {
		t.Fatalf("id should be required")
	}
	if !items.Attributes[1].Required {
		t.Fatalf("name should be required")
	}
	if items.Attributes[2].Required {
		t.Fatalf("description should be nullable")
	}
}

func TestProjectSchemaDiffIntegration(t *testing.T) {
	oldSnap := physical.SchemaSnapshot{
		Database:   "sales",
		CapturedAt: time.Now().UTC().Format(time.RFC3339),
		Tables: []physical.Table{
			{
				Schema:     "public",
				Name:       "products",
				Columns:    []physical.Column{{Name: "id", PhysicalType: "int", Nullable: false, Ordinal: 1}, {Name: "name", PhysicalType: "text", Nullable: false, Ordinal: 2}},
				PrimaryKey: []string{"id"},
			},
		},
	}

	newSnap := physical.SchemaSnapshot{
		Database:   "sales",
		CapturedAt: time.Now().UTC().Format(time.RFC3339),
		Tables: []physical.Table{
			{
				Schema:     "public",
				Name:       "products",
				Columns: []physical.Column{
					{Name: "id", PhysicalType: "int", Nullable: false, Ordinal: 1},
					{Name: "name", PhysicalType: "text", Nullable: false, Ordinal: 2},
					{Name: "price", PhysicalType: "numeric", Nullable: true, Ordinal: 3},
				},
				PrimaryKey: []string{"id"},
			},
		},
	}

	oldResult, err := schematosemantic.Project(oldSnap, schematosemantic.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	newResult, err := schematosemantic.Project(newSnap, schematosemantic.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}

	if len(oldResult.Namespace.Data[0].Attributes) == len(newResult.Namespace.Data[0].Attributes) {
		t.Fatalf("expected attribute count to change after schema change")
	}
}

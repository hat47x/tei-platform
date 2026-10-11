package exampleslide

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

func checkedWorld(t *testing.T) ir.World {
	t.Helper()
	data, err := os.ReadFile("../../reference/world.json")
	if err != nil {
		t.Fatal(err)
	}
	var world ir.World
	if err := json.Unmarshal(data, &world); err != nil {
		t.Fatal(err)
	}
	return world
}

func TestCatalogFromCheckedWorld(t *testing.T) {
	world := checkedWorld(t)
	catalog, err := CatalogFromWorld(world)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(catalog.Revision, "sha256:") {
		t.Fatalf("missing semantic revision: %s", catalog.Revision)
	}
	order, exists := catalog.Tables["sales.order"]
	if !exists || len(order.Fields) < 2 {
		t.Fatal("real semantic sales.order missing")
	}
	if !hasField(order, "sales.order.customer_id") {
		t.Fatal("the real Canonical attribute identity was lost")
	}
	if catalog.Tables["sales.order_line"].Fields[0].ID == "" {
		t.Fatal("real order line attributes missing")
	}
	var foundCustomerKey, foundKeylessMany bool
	for _, relation := range catalog.Relations {
		switch relation.ID {
		case "sales.order.customer":
			foundCustomerKey = relation.Cardinality == "one" &&
				relation.SourceTable == "sales.order" &&
				relation.TargetTable == "customer.customer" &&
				len(relation.Keys) == 1 &&
				relation.Keys[0].SourceField == "sales.order.customer_id" &&
				relation.Keys[0].TargetField == "customer.customer.customer_id"
		case "sales.order.lines":
			foundKeylessMany = relation.Cardinality == "many" &&
				relation.SourceTable == "sales.order" &&
				relation.TargetTable == "sales.order_line" &&
				len(relation.Keys) == 0
		}
	}
	if !foundCustomerKey || !foundKeylessMany {
		t.Fatal("semantic Relation/RelationKey or keyless relation lost")
	}
	for _, field := range order.Fields {
		if field.RefTable != "" {
			t.Fatal("adapter invented a physical/record-ID reference")
		}
		if field.Required {
			t.Fatal("adapter invented requiredness not present in the semantic Attribute")
		}
	}

	example := Example{ID: "view-semantics", ModelRevision: catalog.Revision}
	diagram := Diagram{ID: "sales-er", Placements: []Placement{
		{ID: "customer", Table: "customer.customer", X: 10, Y: 30},
		{ID: "order", Table: "sales.order", X: 330, Y: 30},
		{ID: "lines", Table: "sales.order_line", X: 650, Y: 30},
	}}
	projection, err := Build(Input{
		Catalog: catalog, Example: example, Diagram: diagram,
		Presentation: Presentation{Frames: []Frame{{ID: "base", Stage: "base"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(projection.Slides) != 1 || len(projection.Slides[0].Connections) != 2 {
		t.Fatalf("lost real-world semantic connections: %+v", projection)
	}
	for _, edge := range projection.Slides[0].Connections {
		switch edge.RelationID {
		case "sales.order.customer":
			if edge.Parent != "customer" || edge.Child != "order" {
				t.Fatalf("wrong reference-direction projection: %+v", edge)
			}
		case "sales.order.lines":
			if edge.Parent != "order" || edge.Child != "lines" {
				t.Fatalf("wrong multi-relation projection: %+v", edge)
			}
		default:
			t.Fatalf("unexpected derived connection: %+v", edge)
		}
	}
}

func TestWorldRevisionTracksSemanticChange(t *testing.T) {
	world := checkedWorld(t)
	first, err := CatalogFromWorld(world)
	if err != nil {
		t.Fatal(err)
	}
	world.Semantics[0].Data[0].Symbol += "Changed"
	second, err := CatalogFromWorld(world)
	if err != nil {
		t.Fatal(err)
	}
	if first.Revision == second.Revision {
		t.Fatal("semantic change did not invalidate the example")
	}
}

func TestAdapterFailsClosedOnUnknownType(t *testing.T) {
	world := checkedWorld(t)
	world.Semantics[0].Data[0].Attributes[0].TypeRef = "external.missing"
	if _, err := CatalogFromWorld(world); err == nil || !strings.Contains(err.Error(), "unknown semantic type") {
		t.Fatalf("wanted missing type error, got %v", err)
	}
}

func TestAdapterRejectsAliasCycle(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{
		Types: []semantic.Type{
			{ID: "fixture.one", Kind: semantic.TypeAlias, Underlying: "fixture.two"},
			{ID: "fixture.two", Kind: semantic.TypeAlias, Underlying: "fixture.one"},
		},
		Data: []semantic.Data{{
			ID: "fixture.item",
			Attributes: []semantic.Attribute{{ID: "fixture.item.value", TypeRef: "fixture.one"}},
		}},
	}}}
	if _, err := CatalogFromWorld(world); err == nil || !strings.Contains(err.Error(), "cyclic type aliases") {
		t.Fatalf("wanted alias-cycle error, got %v", err)
	}
}


func TestActualWorldIllustrativeJoinAndUnknownKeylessRelation(t *testing.T) {
	catalog, err := CatalogFromWorld(checkedWorld(t))
	if err != nil {
		t.Fatal(err)
	}
	input := Input{
		Catalog: catalog,
		Example: Example{
			ID: "synthetic-order-join",
			ModelRevision: catalog.Revision,
			// The illustrative customer table is claimed complete only within
			// this fabricated snapshot, not in the real provider database.
			CompleteTables: []string{"customer.customer"},
			Base: []Record{
				{ID: "cust-synthetic", Table: "customer.customer",
					Values: map[string]json.RawMessage{
						"customer.customer.customer_id": json.RawMessage("\"C-1\""),
					}},
				{ID: "order-synthetic", Table: "sales.order",
					Values: map[string]json.RawMessage{
						"sales.order.customer_id": json.RawMessage("\"C-1\""),
					}},
			},
		},
		Diagram: Diagram{ID: "order-example", Placements: []Placement{
			{ID: "c", Table: "customer.customer", X: 10, Y: 30},
			{ID: "o", Table: "sales.order", X: 320, Y: 30},
			{ID: "l", Table: "sales.order_line", X: 630, Y: 30},
		}},
		Presentation: Presentation{Frames: []Frame{{ID: "base", Stage: "base"}}},
	}
	projection, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	findings := projection.Slides[0].RelationFindings
	foundKeyed, foundKeyless := false, false
	for _, finding := range findings {
		switch finding.RelationID {
		case "sales.order.customer":
			foundKeyed = true
			if finding.Result != "supported" || len(finding.MatchedTargetIDs) != 1 ||
				finding.MatchedTargetIDs[0] != "cust-synthetic" {
				t.Fatalf("real keyed relation not qualified: %+v", finding)
			}
		case "sales.order.lines":
			foundKeyless = true
			if finding.Result != "unresolved" || finding.Reason != "relation-keys-not-declared" {
				t.Fatalf("invented relation keys for OrderLine: %+v", finding)
			}
		}
	}
	if !foundKeyed || !foundKeyless {
		t.Fatalf("incomplete source-to-target assessments: %+v", findings)
	}
}

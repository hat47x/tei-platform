package threeelement

import "testing"

// A Data-to-Data projection that drops the relation identity cannot say which
// row came from which declaration. Two relations between the same pair of Data
// elements are different declarations even when every other projected field
// agrees, so the identity has to travel with the row (method contract section
// 33.6 SDM-01: a Table/Graph switch must not merge same-name nodes or parallel
// edges).
//
// The expectation is fixed by hand here: the model declares relations r-2 and
// r-1 in that order on purpose, and both point at the same target with the same
// kind. Neither the input order nor the sort may decide which row is r-1.
func TestDataDataCrossReferenceKeepsTheRelationIdentity(t *testing.T) {
	model := NewThreeElementModel("parallel", "Parallel relations")
	model.DataElements = []DataElement{
		{ID: "order", Name: "Order"},
		{
			ID: "customer", Name: "Customer",
			Relations: []DataRelation{
				{ID: "r-2", TargetID: "order", Kind: "many-to-one", Directed: true},
				{ID: "r-1", TargetID: "order", Kind: "many-to-one", Directed: true},
			},
		},
	}

	rows := GenerateDataDataCrossReference(model)
	if len(rows) != 2 {
		t.Fatalf("two declared relations must project as two rows, got %+v", rows)
	}
	// Total order: source, target, then relation identity.
	if rows[0].RelationID != "r-1" || rows[1].RelationID != "r-2" {
		t.Fatalf("rows are not ordered by relation identity: %+v", rows)
	}
	for _, row := range rows {
		if row.SourceID != "customer" || row.TargetID != "order" {
			t.Fatalf("relation %s lost its endpoints: %+v", row.RelationID, row)
		}
		if row.Kind != "many-to-one" || !row.Directed {
			t.Fatalf("relation %s lost its kind or direction: %+v", row.RelationID, row)
		}
	}
}

// The same relation id may appear under two different Data elements. Each
// element owns its relations, so the identity is the pair (source, relation id)
// and not the relation id alone. A projection that sorted or de-duplicated by
// relation id would silently lose one of the two declarations.
func TestDataDataCrossReferenceQualifiesTheRelationIdentityBySource(t *testing.T) {
	model := NewThreeElementModel("qualified", "Qualified relation identity")
	model.DataElements = []DataElement{
		{ID: "order", Name: "Order"},
		{ID: "invoice", Name: "Invoice"},
		{
			ID: "customer", Name: "Customer",
			Relations: []DataRelation{
				{ID: "r-1", TargetID: "order", Kind: "many-to-one", Directed: true},
				{ID: "r-1", TargetID: "invoice", Kind: "many-to-one", Directed: true},
			},
		},
	}

	rows := GenerateDataDataCrossReference(model)
	if len(rows) != 2 {
		t.Fatalf("two declarations sharing a relation id must both survive, got %+v", rows)
	}
	// Ordered by target: order before invoice is not the point; both rows must
	// exist and be distinguishable by source+relation id, which they are only
	// because the target differs as well. The row for the same target is what
	// the ordering test above covers.
	seen := map[string]bool{}
	for _, row := range rows {
		if row.SourceID != "customer" || row.RelationID != "r-1" {
			t.Fatalf("relation lost its source or identity: %+v", row)
		}
		seen[row.TargetID] = true
	}
	if !seen["order"] || !seen["invoice"] {
		t.Fatalf("a declaration was dropped: %+v", rows)
	}
}

// Two models that declare the same relations in a different order are the same
// model content for this projection. The rows must come out identical, or a
// Consumer comparing two projections would report a change that is only a
// declaration-order difference.
func TestDataDataCrossReferenceOrderIsIndependentOfDeclarationOrder(t *testing.T) {
	build := func(first, second string) ThreeElementModel {
		model := NewThreeElementModel("order", "Declaration order")
		model.DataElements = []DataElement{
			{ID: "order", Name: "Order"},
			{
				ID: "customer", Name: "Customer",
				Relations: []DataRelation{
					{ID: first, TargetID: "order", Kind: "many-to-one", Directed: true},
					{ID: second, TargetID: "order", Kind: "many-to-one", Directed: true},
				},
			},
		}
		return model
	}

	a := GenerateDataDataCrossReference(build("r-1", "r-2"))
	b := GenerateDataDataCrossReference(build("r-2", "r-1"))
	if len(a) != 2 || len(b) != 2 {
		t.Fatalf("expected two rows from each model, got %d and %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("row %d depends on declaration order:\n  %+v\n  %+v", i, a[i], b[i])
		}
	}
}

// The projection is read-only and must not reorder the model it was given.
// Sorting the output is a projection property; the source arrays keep their
// declared order so an editor can still find the declaration it read.
func TestDataDataCrossReferenceDoesNotReorderTheModel(t *testing.T) {
	model := NewThreeElementModel("purity", "Projection purity")
	model.DataElements = []DataElement{
		{
			ID: "customer", Name: "Customer",
			Relations: []DataRelation{
				{ID: "r-9", TargetID: "order", Kind: "one-to-many"},
				{ID: "r-1", TargetID: "order", Kind: "one-to-many"},
			},
		},
		{ID: "order", Name: "Order"},
	}

	GenerateDataDataCrossReference(model)
	relations := model.DataElements[0].Relations
	if relations[0].ID != "r-9" || relations[1].ID != "r-1" {
		t.Fatalf("the projection reordered the source model: %+v", relations)
	}
}

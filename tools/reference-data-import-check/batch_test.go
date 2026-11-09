package dataimportcheck

import (
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func batchWorld() ir.World {
	return ir.World{Semantics: []semantic.Namespace{{ID: "s",
		Types: []semantic.Type{
			{ID: "std.integer", Kind: semantic.TypePrimitive},
			{ID: "std.text", Kind: semantic.TypePrimitive},
		},
		Data: []semantic.Data{
			{ID: "s.customer", Symbol: "Customer", Attributes: []semantic.Attribute{
				{ID: "s.customer.customerId", Symbol: "customerId", TypeRef: "std.integer"},
				{ID: "s.customer.name", Symbol: "name", TypeRef: "std.text"},
			}, CandidateKeys: []semantic.CandidateKey{{ID: "s.customer.k", Symbol: "k",
				Attributes: []core.CanonicalID{"s.customer.customerId"}}}},
			{ID: "s.order", Symbol: "Order", Attributes: []semantic.Attribute{
				{ID: "s.order.orderId", Symbol: "orderId", TypeRef: "std.integer"},
				{ID: "s.order.customerId", Symbol: "customerId", TypeRef: "std.integer"},
			}, CandidateKeys: []semantic.CandidateKey{{ID: "s.order.k", Symbol: "k",
				Attributes: []core.CanonicalID{"s.order.orderId"}}},
				Relations: []semantic.Relation{{ID: "s.order.customer", Target: "s.customer",
					Keys: []semantic.RelationKey{{SourceAttribute: "s.order.customerId", TargetAttribute: "s.customer.customerId"}}}}},
			{ID: "s.free", Symbol: "Free", Attributes: []semantic.Attribute{
				{ID: "s.free.value", Symbol: "value", TypeRef: "std.text"},
			}},
		}}}}
}

func cleanBatch() []TableInput {
	return []TableInput{
		{Data: "s.customer", Role: "initial", TSV: "customerId\tname\n1\tA\n2\tB\n"},
		{Data: "s.order", Role: "operational", TSV: "orderId\tcustomerId\n10\t1\n11\t2\n"},
	}
}

func TestBatchCleanRowsAndReferencesPass(t *testing.T) {
	rep, err := CheckBatch(batchWorld(), cleanBatch())
	if err != nil {
		t.Fatal(err)
	}
	if !rep.OK || !rep.ReferencesOK {
		t.Fatalf("report = %+v", rep)
	}
	if len(rep.References) != 0 {
		t.Fatalf("references = %+v", rep.References)
	}
	for _, table := range rep.Tables {
		for _, row := range table.Rows {
			if row.Status != "ok" {
				t.Errorf("%s row %d = %q (%v)", table.Data, row.Row, row.Status, row.Codes)
			}
		}
	}
	if !reflect.DeepEqual(rep.Boundary.Initial, []string{"s.customer"}) ||
		!reflect.DeepEqual(rep.Boundary.Operational, []string{"s.order"}) {
		t.Errorf("boundary = %+v", rep.Boundary)
	}
}

func TestMissingReferenceTargetIsRejected(t *testing.T) {
	batch := cleanBatch()
	batch[1].TSV = "orderId\tcustomerId\n10\t1\n11\t9\n"
	rep, err := CheckBatch(batchWorld(), batch)
	if err != nil {
		t.Fatal(err)
	}
	if rep.OK || rep.ReferencesOK {
		t.Fatalf("report = %+v", rep)
	}
	if len(rep.References) != 1 {
		t.Fatalf("references = %+v", rep.References)
	}
	finding := rep.References[0]
	if finding.Row != 3 || finding.Code != "reference-target-missing" || finding.Attribute != "s.order.customerId" {
		t.Fatalf("finding = %+v", finding)
	}
	var row RowOutcome
	for _, candidate := range rep.Tables[1].Rows {
		if candidate.Row == 3 {
			row = candidate
		}
	}
	if row.Status != "rejected" || !containsString(row.Codes, "reference-target-missing") {
		t.Fatalf("row outcome = %+v", row)
	}
}

func TestMissingTargetTableIsUnresolved(t *testing.T) {
	rep, err := CheckBatch(batchWorld(), []TableInput{{Data: "s.order", Role: "operational", TSV: "orderId\tcustomerId\n10\t1\n"}})
	if err != nil {
		t.Fatal(err)
	}
	if rep.ReferencesOK || rep.OK {
		t.Fatalf("report = %+v", rep)
	}
	if !containsString(rep.Unchecked, "reference-table-not-provided:s.customer") {
		t.Fatalf("unchecked = %+v", rep.Unchecked)
	}
}

func TestPerRowOutcomesExplainEachRow(t *testing.T) {
	batch := cleanBatch()
	batch[1].TSV = "orderId\tcustomerId\n10\t1\n11\tx\n"
	rep, err := CheckBatch(batchWorld(), batch)
	if err != nil {
		t.Fatal(err)
	}
	rows := rep.Tables[1].Rows
	if len(rows) != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	if rows[0].Status != "ok" {
		t.Errorf("row 2 = %+v", rows[0])
	}
	if rows[1].Status != "rejected" {
		t.Fatalf("row 3 = %+v", rows[1])
	}
	if !containsString(rows[1].Codes, "type-mismatch") || !containsString(rows[1].Codes, "reference-target-missing") {
		t.Errorf("row 3 codes = %v", rows[1].Codes)
	}
}

func TestInitialAndOperationalBoundary(t *testing.T) {
	batch := append(cleanBatch(), TableInput{Data: "s.free", TSV: "value\nx\n"})
	rep, err := CheckBatch(batchWorld(), batch)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rep.Boundary.Undeclared, []string{"s.free"}) {
		t.Errorf("boundary = %+v", rep.Boundary)
	}
	if !containsString(rep.Unchecked, "role-not-declared") {
		t.Errorf("unchecked = %+v", rep.Unchecked)
	}
}

func TestReferenceKeyColumnMissingIsUnresolved(t *testing.T) {
	batch := cleanBatch()
	batch[0].TSV = "name\nA\nB\n"
	rep, err := CheckBatch(batchWorld(), batch)
	if err != nil {
		t.Fatal(err)
	}
	if rep.ReferencesOK {
		t.Fatalf("report = %+v", rep)
	}
	if !containsString(rep.Unchecked, "reference-key-column-missing:s.order.customer") {
		t.Fatalf("unchecked = %+v", rep.Unchecked)
	}
}

func TestUnknownDataInBatchIsError(t *testing.T) {
	if _, err := CheckBatch(batchWorld(), []TableInput{{Data: "s.unknown", TSV: "x\n1\n"}}); err == nil {
		t.Fatal("expected an unknown Data to fail")
	}
}

func TestBatchDoesNotChangeInput(t *testing.T) {
	batch := cleanBatch()
	original := cleanBatch()
	_, err := CheckBatch(batchWorld(), batch)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(batch, original) {
		t.Fatal("CheckBatch changed the input")
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

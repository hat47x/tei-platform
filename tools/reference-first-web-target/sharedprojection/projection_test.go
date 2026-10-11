package sharedprojection

import (
	"encoding/json"
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
)

func TestMappedRowsKeepCanonicalValuesAndColumnOrder(t *testing.T) {
	attributeID := core.MustCanonicalID("customer.customer.customer_id")
	attributeName := core.MustCanonicalID("customer.customer.name")
	rows, err := ProjectMappedRows(queryresult.MappedResult{
		Columns: []queryresult.Column{
			{Raw: "name", Attribute: attributeName},
			{Raw: "customer_id", Attribute: attributeID},
		},
		Rows: [][]any{
			{"Reference Customer 11", int64(11)},
			{nil, int64(12)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []SemanticRowProjection{
		{Attributes: []SemanticValueProjection{
			{Attribute: string(attributeName), Value: "Reference Customer 11"},
			{Attribute: string(attributeID), Value: int64(11)},
		}},
		{Attributes: []SemanticValueProjection{
			{Attribute: string(attributeName), Value: nil},
			{Attribute: string(attributeID), Value: int64(12)},
		}},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("source column order, null or identity changed: got=%+v want=%+v", rows, want)
	}
	raw, err := json.Marshal(rows[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"attributes":[{"attribute":"customer.customer.name","value":"Reference Customer 11"},{"attribute":"customer.customer.customer_id","value":11}]}` {
		t.Fatalf("JSON projection contract drift: %s", raw)
	}
}

func TestMappedRowsRejectWidthMismatch(t *testing.T) {
	for _, values := range [][]any{
		{},
		{int64(11), "extra"},
	} {
		_, err := ProjectMappedRows(queryresult.MappedResult{
			Columns: []queryresult.Column{{
				Raw: "customer_id",
				Attribute: core.MustCanonicalID("customer.customer.customer_id"),
			}},
			Rows: [][]any{values},
		})
		if err == nil {
			t.Fatalf("mismatched values %v were accepted", values)
		}
	}
}

func TestContractRowsSortCanonicalKeys(t *testing.T) {
	attributeID := core.MustCanonicalID("customer.customer.customer_id")
	attributeName := core.MustCanonicalID("customer.customer.name")
	rows := ProjectContractRows([]driver.ContractQueryRow{
		{Attributes: map[core.CanonicalID]any{
			attributeName: "Reference Customer 11",
			attributeID: int64(11),
		}},
	})
	want := []SemanticRowProjection{
		{Attributes: []SemanticValueProjection{
			{Attribute: string(attributeID), Value: int64(11)},
			{Attribute: string(attributeName), Value: "Reference Customer 11"},
		}},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("Contract Query key order/values changed: got=%+v want=%+v", rows, want)
	}
}

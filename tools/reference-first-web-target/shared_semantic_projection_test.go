package main

import (
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
)

// Both Application runners consume these projections. Their shared
// representation must preserve nil and column order without inventing
// a second record identity or changing Canonical attribute names.
func TestSharedProjectionRetainsMappedValuesAndSourceColumnOrder(t *testing.T) {
	customerID := core.MustCanonicalID("customer.customer.customer_id")
	customerName := core.MustCanonicalID("customer.customer.name")
	mapped := queryresult.MappedResult{
		Columns: []queryresult.Column{
			{Raw: "name", Attribute: customerName},
			{Raw: "customer_id", Attribute: customerID},
		},
		Rows: [][]any{
			{"Reference Customer 11", int64(11)},
			{nil, int64(12)},
		},
	}
	rows, err := projectMappedRows(mapped)
	if err != nil {
		t.Fatal(err)
	}
	expected := []semanticRowProjection{
		{Attributes: []semanticValueProjection{
			{Attribute: string(customerName), Value: "Reference Customer 11"},
			{Attribute: string(customerID), Value: int64(11)},
		}},
		{Attributes: []semanticValueProjection{
			{Attribute: string(customerName), Value: nil},
			{Attribute: string(customerID), Value: int64(12)},
		}},
	}
	if !reflect.DeepEqual(rows, expected) {
		t.Fatalf("shared mapped projection changed shape/order/null: actual=%+v expected=%+v", rows, expected)
	}
	if got := reflect.TypeOf(semanticRowProjection{}); got.NumField() != 1 ||
		got.Field(0).Name != "Attributes" {
		t.Fatalf("shared row projection acquired a second identity source: %v", got)
	}
}

func TestSharedProjectionRejectsMismatchedMappedRowWidth(t *testing.T) {
	_, err := projectMappedRows(queryresult.MappedResult{
		Columns: []queryresult.Column{{
			Raw: "customer_id", Attribute: core.MustCanonicalID("customer.customer.customer_id"),
		}},
		Rows: [][]any{{int64(11), "extra"}},
	})
	if err == nil {
		t.Fatal("mapped row with extra values silently accepted")
	}
}

func TestSharedContractProjectionSortsCanonicalAttributes(t *testing.T) {
	customerID := core.MustCanonicalID("customer.customer.customer_id")
	customerName := core.MustCanonicalID("customer.customer.name")
	rows := projectContractRows([]driver.ContractQueryRow{
		{Attributes: map[core.CanonicalID]any{
			customerName: "Reference Customer 11",
			customerID:   int64(11),
		}},
	})
	expected := []semanticRowProjection{
		{Attributes: []semanticValueProjection{
			{Attribute: string(customerID), Value: int64(11)},
			{Attribute: string(customerName), Value: "Reference Customer 11"},
		}},
	}
	if !reflect.DeepEqual(rows, expected) {
		t.Fatalf("shared Contract Query projection changed canonical ordering: actual=%+v expected=%+v", rows, expected)
	}
}

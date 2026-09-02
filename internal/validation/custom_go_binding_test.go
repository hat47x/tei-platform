package validation_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/validation"
)

func TestCustomGoBindingIdentityIsApplicationAndCapabilityPair(t *testing.T) {
	world := ir.World{
		Applications: []application.Application{
			{ID: "first_app"},
			{ID: "second_app"},
		},
		Bindings: []realization.CustomGoBinding{
			{Application: "first_app", Capability: "sales.order.submit"},
			{Application: "second_app", Capability: "sales.order.submit"},
		},
	}

	for _, diagnostic := range validation.Validate(world) {
		if diagnostic.Code == "DUPLICATE_CUSTOM_GO_BINDING" {
			t.Fatalf("同じCapabilityを異なるApplicationがBindingしても衝突してはいけません: %#v", diagnostic)
		}
	}
}

func TestDuplicateCustomGoBindingForSameApplicationAndCapabilityIsRejected(t *testing.T) {
	world := ir.World{
		Applications: []application.Application{{ID: "order_entry"}},
		Bindings: []realization.CustomGoBinding{
			{Application: "order_entry", Capability: "sales.order.submit"},
			{Application: "order_entry", Capability: "sales.order.submit"},
		},
	}

	for _, diagnostic := range validation.Validate(world) {
		if diagnostic.Code == "DUPLICATE_CUSTOM_GO_BINDING" {
			return
		}
	}
	t.Fatal("同じApplication + CapabilityのBinding重複が報告されませんでした")
}

package runtimepackaging

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/workspace"
)

func TestRealOrderEntryCommandBinding(t *testing.T) {
	loaded, err := workspace.Load("../workspace")
	if err != nil {
		t.Fatal(err)
	}
	out, err := ObserveCommandRealizations(loaded.World, "order_entry")
	if err != nil {
		t.Fatal(err)
	}
	var found *CommandRealizationObservation
	for i := range out {
		if out[i].PatternID == "order_entry.submit_order" {
			found = &out[i]
		}
	}
	if found == nil || found.CapabilityID != "sales.order.submit" ||
		found.BindingIdentity != "order_entry/sales.order.submit" ||
		found.ExpectedAdapterFilename != "sales__order__submit.bind.go" ||
		found.ExpectedHookName != "TeiBindSalesOrderSubmit" {
		t.Fatalf("unexpected submit-order realization observation: %+v", found)
	}
}

func commandWorld() ir.World {
	return ir.World{
		Applications: []application.Application{{
			ID: "app",
			Patterns: []application.Pattern{{ID: "app.submit", Kind: application.PatternCommand, Capability: "order.submit"}},
		}},
		Bindings: []realization.CustomGoBinding{{Application: "app", Capability: "order.submit"}},
	}
}

func TestCommandObservationRejectsMissingOrAmbiguousRealization(t *testing.T) {
	x := commandWorld()
	x.Bindings = nil
	if _, err := ObserveCommandRealizations(x, "app"); err == nil || !strings.Contains(err.Error(), "not defined") {
		t.Fatalf("missing binding was not rejected: %v", err)
	}
	x = commandWorld()
	x.Bindings = append(x.Bindings, x.Bindings[0])
	if _, err := ObserveCommandRealizations(x, "app"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("duplicate binding was not rejected: %v", err)
	}
}

func TestCommandObservationDoesNotUseAnotherApplicationBinding(t *testing.T) {
	x := commandWorld()
	x.Bindings[0].Application = "another"
	if _, err := ObserveCommandRealizations(x, "app"); err == nil {
		t.Fatal("application-scoped binding must not leak across applications")
	}
}

func TestCommandObservationRejectsDuplicatePattern(t *testing.T) {
	x := commandWorld()
	x.Applications[0].Patterns = append(x.Applications[0].Patterns, x.Applications[0].Patterns[0])
	if _, err := ObserveCommandRealizations(x, "app"); err == nil {
		t.Fatal("duplicate pattern must be rejected")
	}
}

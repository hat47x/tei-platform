package main

import (
	"testing"

	"tei/internal/semantic"
)

func TestCommandInspectionReadsExplicitCapabilityDataEffect(t *testing.T) {
	w := referenceWorkspace(t)
	var updated bool
	for i := range w.Semantics {
		for j := range w.Semantics[i].Capabilities {
			c := &w.Semantics[i].Capabilities[j]
			if c.ID == "sales.order.submit" {
				c.Effect = &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"}
				updated = true
			}
		}
	}
	if !updated {
		t.Fatal("reference Capability not found")
	}
	report, err := InspectExplicitPattern(w, "order_entry", "order_entry.submit_order")
	if err != nil {
		t.Fatal(err)
	}
	if report.DeclaredData != "sales.order" || report.DataOperation != "create" ||
		report.Executable || report.Authority != "declared-references-only" ||
		!includes(report.Resolved, "data-operation-link") ||
		includes(report.Unverified, "data-operation-link") ||
		!includes(report.Unverified, "independent-database-read") {
		t.Fatalf("declared effect was either ignored or promoted to execution: %+v", report)
	}

	// The command effect must not be inferred from capability output or name.
	for i := range w.Semantics {
		for j := range w.Semantics[i].Capabilities {
			if w.Semantics[i].Capabilities[j].ID == "sales.order.submit" {
				w.Semantics[i].Capabilities[j].Effect = nil
			}
		}
	}
	legacy, err := InspectExplicitPattern(w, "order_entry", "order_entry.submit_order")
	if err != nil {
		t.Fatal(err)
	}
	if legacy.DeclaredData != "" || legacy.DataOperation != "" ||
		!includes(legacy.Unverified, "data-operation-link") {
		t.Fatalf("legacy Capability inferred an effect: %+v", legacy)
	}
}

func TestCommandInspectionRejectsInvalidEffectWithoutExecutor(t *testing.T) {
	w := referenceWorkspace(t)
	for i := range w.Semantics {
		for j := range w.Semantics[i].Capabilities {
			if w.Semantics[i].Capabilities[j].ID == "sales.order.submit" {
				w.Semantics[i].Capabilities[j].Effect = &semantic.DataEffect{
					Kind: semantic.DataEffectUpdate, Data: "sales.unknown",
				}
			}
		}
	}
	if result, err := InspectExplicitPattern(w, "order_entry", "order_entry.submit_order"); err == nil {
		t.Fatalf("invalid effect was promoted to declared reference: %+v", result)
	}
}

package main

import (
	"strings"
	"testing"
)

func TestOrderApprovalAndWarehousePreparationDemonstration(t *testing.T) {
	const root = "../../reference/business-system-design/dfd-order-approval/"
	p, err := run(
		root+"order-approval-procedure.json",
		root+"review-proposal.json",
		root+"procedure-set.json",
		"../../reference/workspace",
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range p.Diagnostics {
		if d.Severity == "error" {
			t.Fatalf("demo should be a reviewable source case: %+v", d)
		}
	}
	if p.ProcedureID != "order-entry.approval-and-shipment-prep" {
		t.Fatalf("unexpected procedure %s", p.ProcedureID)
	}
	if len(p.Steps) != 8 || len(p.Triggers) != 3 || len(p.Information.InformationFlows) != 5 || len(p.Handoffs) != 1 {
		t.Fatalf("demo projection incomplete: steps=%d triggers=%d information=%d handoffs=%d",
			len(p.Steps), len(p.Triggers), len(p.Information.InformationFlows), len(p.Handoffs))
	}
	manualCount, functionCount, exceptionCount := 0, 0, 0
	for _, s := range p.Steps {
		if s.NoApplicationFunction {
			manualCount++
		} else {
			functionCount++
		}
		if s.ID == "APP-10" {
			if !strings.Contains(s.Label, "call the customer") || !s.NoApplicationFunction {
				t.Fatal("manual telephone confirmation was replaced by a fake Function")
			}
		}
		if s.ID == "APP-20" && !s.NoApplicationFunction {
			t.Fatal("approval decision must not require a Function")
		}
	}
	for _, flow := range p.Flows {
		if flow.Kind == "exception" {
			exceptionCount++
		}
		if flow.Authority != "proposal" {
			t.Fatal("business exception promoted to accepted order process")
		}
	}
	if manualCount != 6 || functionCount != 2 || exceptionCount != 1 {
		t.Fatalf("unexpected process distribution manual=%d function=%d exception=%d", manualCount, functionCount, exceptionCount)
	}
	h := p.Handoffs[0]
	if h.From.Step != "APP-50" || h.To.Procedure != "order-entry.warehouse-preparation" || h.To.Step != "WH-00" {
		t.Fatalf("warehouse Handoff lost: %+v", h)
	}
	if p.RoleAuthorizationBinding != "not-inferred" {
		t.Fatal("business Role was promoted to Authorization Role")
	}
	for _, tr := range p.Triggers {
		if tr.RuntimeBinding != "unbound" || tr.Authority != "proposal" {
			t.Fatal("trigger proposal promoted to runtime authority")
		}
	}
	if !strings.Contains(dot(p), "business Handoff (not runtime):") {
		t.Fatal("diagram conflated cross-Procedure Handoff with a sequence")
	}
}

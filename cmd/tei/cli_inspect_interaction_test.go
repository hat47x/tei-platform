package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestInspectCLIProjectsApplicationPatternInteraction(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: inspect_interaction\nsources:\n  - applications\n")
	writeWorkspaceFile(t, root, "applications/order-entry/application.tei", "application order_entry {\n}\n")
	writeWorkspaceFile(t, root, "applications/order-entry/patterns/order-record.tei", `pattern order_entry.order_record record {
    data sales.order
    interaction {
        item sales.order.order_id
        item sales.order.customer_id lookup order_entry.customer_reference customer.customer.customer_id
        navigate from order_entry.recent_orders
    }
}
`)

	var out bytes.Buffer
	code, err := inspectWorkspaceJSON(&out, root, "", "application-pattern", "order_entry.order_record", 0)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}

	var report cliInspectionReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if !report.OK || report.Result == nil {
		t.Fatalf("unexpected inspect report: %+v", report)
	}
	var canonical map[string]any
	if err := json.Unmarshal(report.Result.Canonical, &canonical); err != nil {
		t.Fatal(err)
	}
	interaction, ok := canonical["interaction"].(map[string]any)
	if !ok {
		t.Fatalf("interactionがCLI projectionにありません: %s", report.Result.Canonical)
	}
	items, ok := interaction["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("items=%#v", interaction["items"])
	}
	navigations, ok := interaction["navigations"].([]any)
	if !ok || len(navigations) != 1 {
		t.Fatalf("navigations=%#v", interaction["navigations"])
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func viewWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: view_command\nsources:\n  - semantic\n  - applications\n")
	writeWorkspaceFile(t, root, "semantic/sales/order.tei", `namespace sales

data Order {
    attribute orderId type std.identifier
    attribute customerId type customer.customer_id
    relation customer to customer.customer one
        key sales.order.customer_id -> customer.customer.customer_id
}
`)
	writeWorkspaceFile(t, root, "semantic/customer/customer.tei", `namespace customer

data Customer {
    attribute customerId type customer.customer_id
    attribute name type std.text
}
`)
	writeWorkspaceFile(t, root, "applications/order-entry/application.tei", `application order_entry {
    uses semantic sales
    uses semantic customer
}
`)
	writeWorkspaceFile(t, root, "applications/order-entry/patterns/customer-reference.tei", `pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
`)
	writeWorkspaceFile(t, root, "applications/order-entry/patterns/recent-orders.tei", `pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
`)
	writeWorkspaceFile(t, root, "applications/order-entry/patterns/order-record.tei", `pattern order_entry.order_record record {
    data sales.order
    interaction {
        item sales.order.order_id
        item sales.order.customer_id lookup order_entry.customer_reference customer.customer.customer_id
        navigate from order_entry.recent_orders
    }
}
`)
	return root
}

func TestViewCommandReturnsResolvedInteractionAsJSON(t *testing.T) {
	root := viewWorkspace(t)
	var out bytes.Buffer
	code, err := runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_record",
		"-format", "json",
	}, &out)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	var report cliViewReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if !report.OK || report.Result == nil {
		t.Fatalf("unexpected report: %+v", report)
	}
	if len(report.Result.Items) != 2 || report.Result.Items[1].Lookup == nil {
		t.Fatalf("items=%+v", report.Result.Items)
	}
	if report.Result.Items[1].Lookup.TargetType != "customer.customer_id" {
		t.Fatalf("lookup=%+v", report.Result.Items[1].Lookup)
	}
}

func TestViewCommandRendersTextForm(t *testing.T) {
	root := viewWorkspace(t)
	var out bytes.Buffer
	code, err := runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_record",
	}, &out)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	text := out.String()
	for _, want := range []string{
		"pattern order_entry.order_record (record)",
		"sales.order.customer_id : customer.customer_id lookup order_entry.customer_reference (reference)",
		"navigations:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("text=%q does not contain %q", text, want)
		}
	}
}

func TestViewCommandFailsClosedWithoutInteraction(t *testing.T) {
	root := viewWorkspace(t)
	var out bytes.Buffer
	code, err := runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.recent_orders",
	}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 {
		t.Fatalf("exit=%d body=%s", code, out.String())
	}
	var report cliViewReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.OK || report.Error == nil || report.Error.Code != "VIEW_FAILED" {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestViewCommandRejectsMissingFlags(t *testing.T) {
	var out bytes.Buffer
	code, err := runViewCommand([]string{"-workspace", t.TempDir()}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if code != 2 {
		t.Fatalf("exit=%d body=%s", code, out.String())
	}
	var report cliViewReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.OK || report.Error == nil || report.Error.Code != "INVALID_ARGUMENT" {
		t.Fatalf("unexpected report: %+v", report)
	}
}

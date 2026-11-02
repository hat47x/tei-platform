package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/applicationview"
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
	writeWorkspaceFile(t, root, "applications/order-entry/packages/list-to-record.tei", `package order_entry.list_to_record ^1 {
    application order_entry
    member browse order_entry.recent_orders
    member record order_entry.order_record
    requires semantic sales.order
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
	var report struct {
		OK     bool                 `json:"ok"`
		Result applicationview.View `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if !report.OK || len(report.Result.Items) != 2 || report.Result.Items[1].Lookup == nil {
		t.Fatalf("unexpected report: %+v", report)
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
		"packages: order_entry.list_to_record ^1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("text=%q does not contain %q", text, want)
		}
	}
}

func TestViewCommandRejectsUnsupportedKind(t *testing.T) {
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
	if report.OK || report.Error == nil || report.Error.Code != "VIEW_UNSUPPORTED" {
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

func TestViewCommandRendersHTMLForm(t *testing.T) {
	root := viewWorkspace(t)
	var out bytes.Buffer
	code, err := runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_record",
		"-format", "html",
	}, &out)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	html := out.String()
	for _, want := range []string{"<!doctype html>", `name="sales.order.order_id"`, "参照: order_entry.customer_reference"} {
		if !strings.Contains(html, want) {
			t.Fatalf("html does not contain %q: %s", want, html)
		}
	}
}

func headerDetailWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: header_detail_view\nsources:\n  - semantic\n  - applications\n")
	writeWorkspaceFile(t, root, "semantic/sales/order.tei", `namespace sales

data Order {
    attribute orderId type std.identifier
    attribute customerId type std.identifier
    relation lines to sales.order_line many
}

data OrderLine {
    attribute orderLineId type std.identifier
    attribute quantity type std.decimal display "数量"
}
`)
	writeWorkspaceFile(t, root, "applications/order-entry/application.tei", `application order_entry {
    uses semantic sales
}
`)
	writeWorkspaceFile(t, root, "applications/order-entry/patterns/order-lines.tei", `pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
`)
	return root
}

func TestViewCommandRendersHeaderDetail(t *testing.T) {
	root := headerDetailWorkspace(t)
	var out bytes.Buffer
	code, err := runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_lines",
		"-format", "html",
	}, &out)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	html := out.String()
	if !strings.Contains(html, "<table>") || !strings.Contains(html, "数量") {
		t.Fatalf("header-detail html=%s", html)
	}

	out.Reset()
	code, err = runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_lines",
	}, &out)
	if err != nil || code != 0 {
		t.Fatalf("text exit=%d err=%v body=%s", code, err, out.String())
	}
	if !strings.Contains(out.String(), "detail:") {
		t.Fatalf("text=%s", out.String())
	}
}

func TestViewCommandRendersPrintFormat(t *testing.T) {
	root := viewWorkspace(t)
	var out bytes.Buffer
	code, err := runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_record",
		"-format", "print",
	}, &out)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	if !strings.Contains(out.String(), "@page") || !strings.Contains(out.String(), "<table>") {
		t.Fatalf("print output=%s", out.String())
	}
}

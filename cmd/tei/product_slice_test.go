package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/applicationview"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func productSliceWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: product_slice\nformatVersion: 1\nsources:\n  - semantic\n  - applications\n")
	writeWorkspaceFile(t, root, "semantic/sales/order.tei", `namespace sales

type Status from std.code values { "open" "closed" }

data Order {
    attribute orderId type std.identifier display "受注番号"
    attribute customerId type customer.customer_id display "顧客"
    attribute status type sales.status display "状態"
    relation customer to customer.customer one
    relation lines to sales.order_line many
}

data OrderLine {
    attribute orderLineId type std.identifier display "行"
    attribute quantity type std.decimal display "数量" required
    attribute amount type std.money display "金額"
}

capability submit_order {
    input order sales.order
    output sales.order
    outcome submitted sales.order
    effect update sales.order
}
`)
	writeWorkspaceFile(t, root, "semantic/customer/customer.tei", `namespace customer

data Customer {
    attribute customerId type customer.customer_id display "顧客ID"
    attribute name type std.text display "顧客名"
}
`)
	writeWorkspaceFile(t, root, "applications/order-entry/application.tei", `application order_entry {
    uses semantic sales
    uses semantic customer
    binds package order_entry.list_to_record ^1
}
`)
	writeWorkspaceFile(t, root, "applications/order-entry/patterns/recent-orders.tei", `pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
    filter sales.order.status { eq ne } parameter status
    sort sales.order.status asc
    page-size 20
}
`)
	writeWorkspaceFile(t, root, "applications/order-entry/patterns/customer-reference.tei", `pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
`)
	writeWorkspaceFile(t, root, "applications/order-entry/queries/recent_orders.tei", `query order_entry.recent_orders {
    input status sales.status
    datasource sales
    sql postgres file "recent_orders.sql"
    result {
        column order_id -> sales.order.order_id
        column customer_id -> sales.order.customer_id
    }
}
`)
	writeWorkspaceFile(t, root, "applications/order-entry/queries/recent_orders.sql", "select order_id, customer_id from sales.order where status = :status\n")
	writeWorkspaceFile(t, root, "applications/order-entry/patterns/order-record.tei", `pattern order_entry.order_record record {
    data sales.order
    interaction {
        item sales.order.order_id
        item sales.order.customer_id lookup order_entry.customer_reference customer.customer.customer_id
        command sales.submit_order {
            input order sales.order.order_id
        }
        hook before-update {
            reads sales.order.status
            uses sales.submit_order
            produces sales.order.customer_id
            on-failure reject
        }
    }
}
`)
	writeWorkspaceFile(t, root, "applications/order-entry/patterns/submit-order.tei", `pattern order_entry.submit_order command {
    capability sales.submit_order
}
`)
	writeWorkspaceFile(t, root, "applications/order-entry/patterns/order-lines.tei", `pattern order_entry.order_lines header-detail {
    relation sales.order.lines
    page-break 3 repeat-header
}
`)
	writeWorkspaceFile(t, root, "applications/order-entry/packages/list-to-record.tei", `package order_entry.list_to_record ^1 {
    application order_entry
    member browse order_entry.recent_orders
    member record order_entry.order_record
    member reference order_entry.customer_reference
    member header-detail order_entry.order_lines
    requires semantic sales.order
    requires relation sales.order.customer
    projection screen
    conformance list_to_record
}
`)
	return root
}

func TestProductSliceLoadsValidatesAndViews(t *testing.T) {
	root := productSliceWorkspace(t)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if strings.HasPrefix(diagnostic.Code, "APPLICATION_PACKAGE_") || strings.HasPrefix(diagnostic.Code, "APPLICATION_PATTERN_INTERACTION_") {
			t.Fatalf("unexpected diagnostic: %s %s", diagnostic.Code, diagnostic.Message)
		}
	}

	var record bytes.Buffer
	code, err := runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_record",
		"-format", "json",
	}, &record)
	if err != nil || code != 0 {
		t.Fatalf("record view exit=%d err=%v body=%s", code, err, record.String())
	}
	var report struct {
		OK     bool                 `json:"ok"`
		Result applicationview.View `json:"result"`
	}
	if err := json.Unmarshal(record.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if !report.OK || len(report.Result.Items) != 2 || len(report.Result.Commands) != 1 || len(report.Result.Packages) != 1 {
		t.Fatalf("record view=%+v", report.Result)
	}
	if len(report.Result.Commands[0].Inputs) != 1 || report.Result.Commands[0].Inputs[0].Attribute != "sales.order.order_id" {
		t.Fatalf("command inputs=%+v", report.Result.Commands[0].Inputs)
	}
	if report.Result.Items[1].Lookup == nil || report.Result.Items[1].Lookup.TargetType != "customer.customer_id" {
		t.Fatalf("lookup=%+v", report.Result.Items[1].Lookup)
	}

	var planOut bytes.Buffer
	code, err = runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_record",
		"-command", "sales.submit_order",
		"-input", "order=ORD-1",
		"-format", "json",
	}, &planOut)
	if err != nil || code != 0 {
		t.Fatalf("command plan exit=%d err=%v body=%s", code, err, planOut.String())
	}
	var planReport struct {
		OK     bool                        `json:"ok"`
		Result applicationview.CommandPlan `json:"result"`
	}
	if err := json.Unmarshal(planOut.Bytes(), &planReport); err != nil {
		t.Fatal(err)
	}
	if !planReport.OK || !planReport.Result.Ready || len(planReport.Result.Inputs) != 1 || planReport.Result.Inputs[0].Value != "ORD-1" {
		t.Fatalf("command plan=%+v", planReport.Result)
	}

	var commandView bytes.Buffer
	code, err = runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.submit_order",
		"-format", "json",
	}, &commandView)
	if err != nil || code != 0 {
		t.Fatalf("command view exit=%d err=%v body=%s", code, err, commandView.String())
	}
	var commandReport struct {
		OK     bool                               `json:"ok"`
		Result applicationview.CommandPatternView `json:"result"`
	}
	if err := json.Unmarshal(commandView.Bytes(), &commandReport); err != nil {
		t.Fatal(err)
	}
	if !commandReport.OK || commandReport.Result.Capability != "sales.submit_order" || len(commandReport.Result.Outcomes) != 1 {
		t.Fatalf("command view=%+v", commandReport.Result)
	}

	var referenceRun bytes.Buffer
	code, err = runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_record",
		"-command", "sales.submit_order",
		"-input", "order=ORD-1",
		"-execute",
		"-format", "json",
	}, &referenceRun)
	if err != nil || code != 0 {
		t.Fatalf("reference run exit=%d err=%v body=%s", code, err, referenceRun.String())
	}
	var referenceReport struct {
		OK     bool                `json:"ok"`
		Result referenceCommandRun `json:"result"`
	}
	if err := json.Unmarshal(referenceRun.Bytes(), &referenceReport); err != nil {
		t.Fatal(err)
	}
	if !referenceReport.OK || referenceReport.Result.Mode != "reference" || !referenceReport.Result.Committed || referenceReport.Result.Outcome == nil {
		t.Fatalf("reference run=%+v", referenceReport.Result)
	}

	var periodStateFailure bytes.Buffer
	code, err = runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_record",
		"-command", "sales.submit_order",
		"-input", "order=ORD-1",
		"-execute",
		"-period-state", filepath.Join(root, "missing-period-state.json"),
	}, &periodStateFailure)
	if code == 0 || !strings.Contains(periodStateFailure.String(), "PERIOD_STATE_LOAD_FAILED") {
		t.Fatalf("period state failure exit=%d body=%s", code, periodStateFailure.String())
	}

	var unauthorized bytes.Buffer
	code, err = runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_record",
		"-command", "sales.submit_order",
		"-input", "order=ORD-1",
		"-execute",
		"-allow-capability", "sales.other",
	}, &unauthorized)
	if code == 0 || !strings.Contains(unauthorized.String(), "COMMAND_UNAUTHORIZED") {
		t.Fatalf("unauthorized run exit=%d body=%s", code, unauthorized.String())
	}

	var applicationView bytes.Buffer
	code, err = runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-format", "json",
	}, &applicationView)
	if err != nil || code != 0 {
		t.Fatalf("application view exit=%d err=%v body=%s", code, err, applicationView.String())
	}
	var applicationReport struct {
		OK     bool                            `json:"ok"`
		Result applicationview.ApplicationView `json:"result"`
	}
	if err := json.Unmarshal(applicationView.Bytes(), &applicationReport); err != nil {
		t.Fatal(err)
	}
	if !applicationReport.OK || len(applicationReport.Result.Patterns) < 5 || applicationReport.Result.SemanticFingerprint == "" {
		t.Fatalf("application view=%+v", applicationReport.Result)
	}

	var packageView bytes.Buffer
	code, err = runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-package", "order_entry.list_to_record",
		"-format", "json",
	}, &packageView)
	if err != nil || code != 0 {
		t.Fatalf("package view exit=%d err=%v body=%s", code, err, packageView.String())
	}
	var packageReport struct {
		OK     bool                        `json:"ok"`
		Result applicationview.PackageView `json:"result"`
	}
	if err := json.Unmarshal(packageView.Bytes(), &packageReport); err != nil {
		t.Fatal(err)
	}
	if !packageReport.OK || packageReport.Result.Version != "1" || !packageReport.Result.Bound || len(packageReport.Result.Members) != 4 {
		t.Fatalf("package view=%+v", packageReport.Result)
	}

	var recordHTML bytes.Buffer
	code, err = runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_record",
		"-format", "html",
	}, &recordHTML)
	if err != nil || code != 0 {
		t.Fatalf("record html exit=%d err=%v body=%s", code, err, recordHTML.String())
	}
	for _, want := range []string{"フック", "未使用の属性", "状態"} {
		if !strings.Contains(recordHTML.String(), want) {
			t.Fatalf("record html does not contain %s", want)
		}
	}

	var recordPrint bytes.Buffer
	code, err = runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_record",
		"-format", "print",
	}, &recordPrint)
	if err != nil || code != 0 {
		t.Fatalf("record print exit=%d err=%v body=%s", code, err, recordPrint.String())
	}
	if !strings.Contains(recordPrint.String(), "format: tei.print/v0") {
		t.Fatal("record print does not carry the receipt")
	}

	var browse bytes.Buffer
	code, err = runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.recent_orders",
		"-format", "html",
	}, &browse)
	if err != nil || code != 0 {
		t.Fatalf("browse view exit=%d err=%v body=%s", code, err, browse.String())
	}
	if !strings.Contains(browse.String(), "<th>受注番号</th>") || !strings.Contains(browse.String(), "<option>eq</option>") || !strings.Contains(browse.String(), "並び順") || !strings.Contains(browse.String(), "page-size: 20") || !strings.Contains(browse.String(), "状態 (status)") {
		t.Fatalf("browse html=%s", browse.String())
	}

	var detail bytes.Buffer
	code, err = runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_lines",
		"-format", "html",
	}, &detail)
	if err != nil || code != 0 {
		t.Fatalf("detail view exit=%d err=%v body=%s", code, err, detail.String())
	}
	for _, want := range []string{"<table>", "明細", "数量"} {
		if !strings.Contains(detail.String(), want) {
			t.Fatalf("detail html does not contain %q: %s", want, detail.String())
		}
	}

	var printed bytes.Buffer
	code, err = runViewCommand([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_lines",
		"-format", "print",
	}, &printed)
	if err != nil || code != 0 {
		t.Fatalf("print view exit=%d err=%v body=%s", code, err, printed.String())
	}
	if !strings.Contains(printed.String(), "@page") || !strings.Contains(printed.String(), "3行/ページ") {
		t.Fatalf("print output is not a print sheet: %s", printed.String())
	}
}

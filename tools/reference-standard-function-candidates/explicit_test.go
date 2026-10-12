package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/workspace"
)

func referenceWorkspace(t *testing.T) ir.World {
	t.Helper()
	loaded, err := workspace.Load("../../reference/workspace")
	if err != nil {
		t.Fatal(err)
	}
	return loaded.World
}

func TestExplicitBrowseUsesDeclaredQueryAndDatasource(t *testing.T) {
	w := referenceWorkspace(t)
	report, err := InspectExplicitPattern(w, "order_entry", "order_entry.recent_orders")
	if err != nil {
		t.Fatal(err)
	}
	if report.Authority != "declared-references-only" || report.Executable ||
		report.Kind != application.PatternBrowse || report.DeclaredData != "sales.order" ||
		report.Query != "order_entry.recent_orders" || report.Datasource != "sales" ||
		report.TargetDatabase != "sales_core" {
		t.Fatalf("browse did not preserve explicit identities: %+v", report)
	}
	if !reflect.DeepEqual(report.Resolved, []string{"data", "query", "datasource"}) ||
		!includes(report.Unverified, "current-authority") ||
		!includes(report.Unverified, "provider-session") {
		t.Fatalf("browse claimed more than declared contracts: %+v", report)
	}
	again, err := InspectExplicitPattern(w, "order_entry", "order_entry.recent_orders")
	if err != nil || !reflect.DeepEqual(report, again) {
		t.Fatalf("inspection is not deterministic: %+v, %v", again, err)
	}
}

func TestExplicitCommandHasBindingButNoImplicitDataOperation(t *testing.T) {
	w := referenceWorkspace(t)
	report, err := InspectExplicitPattern(w, "order_entry", "order_entry.submit_order")
	if err != nil {
		t.Fatal(err)
	}
	if report.Kind != application.PatternCommand || report.Capability != "sales.order.submit" ||
		report.BindingIdentity != "order_entry/sales.order.submit" ||
		report.DeclaredData != "" || report.Executable ||
		!includes(report.Unverified, "data-operation-link") ||
		!includes(report.Unverified, "independent-database-read") {
		t.Fatalf("command improperly promoted a declaration to execution: %+v", report)
	}

	w.Bindings = nil
	if _, err := InspectExplicitPattern(w, "order_entry", "order_entry.submit_order"); err == nil {
		t.Fatal("missing explicit Binding was silently inferred")
	}
}

func TestExplicitRecordCannotInferReadBinding(t *testing.T) {
	w := referenceWorkspace(t)
	report, err := InspectExplicitPattern(w, "order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatal(err)
	}
	if report.DeclaredData != "sales.order" || report.Query != "" ||
		report.Datasource != "" || !includes(report.Unverified, "read-query") ||
		report.Executable {
		t.Fatalf("record invented query or runtime authority: %+v", report)
	}
}

func TestExplicitInspectionRejectsUnsupportedAndAmbiguousSource(t *testing.T) {
	w := referenceWorkspace(t)
	if _, err := InspectExplicitPattern(w, "request_log", "order_entry.recent_orders"); err == nil {
		t.Fatal("wrong Application was accepted")
	}
	if _, err := InspectExplicitPattern(w, "order_entry", "missing_pattern"); err == nil {
		t.Fatal("missing Pattern was inferred")
	}
	if _, err := InspectExplicitPattern(w, "order_entry", "order_entry.order_lines"); err == nil {
		t.Fatal("header-detail was incorrectly treated as standard CRUD")
	}

	w = referenceWorkspace(t)
	for i := range w.Applications {
		if w.Applications[i].ID == "order_entry" {
			for _, pattern := range w.Applications[i].Patterns {
				if pattern.ID == "order_entry.recent_orders" {
					w.Applications[i].Patterns = append(w.Applications[i].Patterns, pattern)
					break
				}
			}
		}
	}
	if _, err := InspectExplicitPattern(w, "order_entry", "order_entry.recent_orders"); err == nil {
		t.Fatal("ambiguous Pattern was silently selected")
	}

	w = referenceWorkspace(t)
	for i := range w.Applications {
		if w.Applications[i].ID == "order_entry" {
			w.Applications[i].Databases = append(w.Applications[i].Databases, w.Applications[i].Databases[0])
		}
	}
	if _, err := InspectExplicitPattern(w, "order_entry", "order_entry.recent_orders"); err == nil {
		t.Fatal("ambiguous datasource was selected")
	}

	w = referenceWorkspace(t)
	for i := range w.Queries {
		if w.Queries[i].ID == "order_entry.recent_orders" {
			w.Queries[i].Application = "other_app"
		}
	}
	if _, err := InspectExplicitPattern(w, "order_entry", "order_entry.recent_orders"); err == nil {
		t.Fatal("cross-Application Query was accepted")
	}
}

func TestExplicitCLIDoesNotClaimRuntimeAndDoesNotChangeProjectionMode(t *testing.T) {
	args := []string{"-workspace", "../../reference/workspace", "-application", "order_entry", "-pattern", "order_entry.recent_orders"}
	var first, second bytes.Buffer
	if err := run(args, &first); err != nil {
		t.Fatal(err)
	}
	if err := run(args, &second); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("same explicit Pattern returned different bytes")
	}
	var report ExplicitContractReport
	if err := json.Unmarshal(first.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Executable || report.Authority != "declared-references-only" {
		t.Fatalf("CLI claimed runtime authority: %+v", report)
	}

	var rejected bytes.Buffer
	err := run([]string{"-workspace", "../../examples/first-workspace", "-application", "request_log", "-pattern", "request_log.fake"}, &rejected)
	if err == nil || rejected.Len() != 0 {
		t.Fatalf("unbound first Workspace Pattern was accepted: err=%v result=%s", err, rejected.String())
	}

	var plain bytes.Buffer
	if err := run([]string{"-workspace", "../../examples/first-workspace", "-application", "request_log"}, &plain); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plain.String(), "proposal-only") {
		t.Fatal("original candidate projection mode changed")
	}
}

func includes(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}

package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvironmentRowsHTMLPreservesObservedStatesAndDecisions(t *testing.T) {
	result := destinationRead{
		Application: "order_entry", Pattern: "order_entry.order_lines", Environment: "development", Relation: "sales.order.lines", Coverage: "partial",
		Rows: []destinationRow{
			{Key: "2", Fields: map[string]destinationField{
				"sales.order_line.quantity":   {State: "present", Value: "<script>alert(1)</script>"},
				"sales.order_line.unit_price": {State: "null"},
			}},
			{Key: "3", Fields: map[string]destinationField{
				"sales.order_line.quantity":   {State: "present", Value: "12.300"},
				"sales.order_line.unit_price": {State: "not-fetched"},
			}},
		},
		PerRow: map[string]map[string]string{
			"2": {"sales.order_line.quantity_positive": "not-satisfied"},
			"3": {"sales.order_line.quantity_positive": "satisfied"},
		},
		PerRelation:          map[string]string{"sales.order.lines.line_number_unique": "undecided"},
		PerRelationUndecided: map[string]string{"sales.order.lines.line_number_unique": "full relation not observed"},
	}
	page, err := renderEnvironmentRowsHTML(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{
		"<!doctype html>", `data-coverage="partial"`, `data-row-key="2"`, `data-row-key="3"`,
		`data-state="null">NULL`, `data-state="not-fetched">未取得`,
		`data-rule-id="sales.order_line.quantity_positive" data-answer="not-satisfied"`,
		`data-rule-id="sales.order_line.quantity_positive" data-answer="satisfied"`,
		`data-rule-id="sales.order.lines.line_number_unique" data-answer="undecided"`,
		"full relation not observed", "12.300", "&lt;script&gt;alert(1)&lt;/script&gt;",
	} {
		if !strings.Contains(page, needle) {
			t.Errorf("missing %q in HTML", needle)
		}
	}
	if strings.Contains(page, "<script>") {
		t.Fatal("unescaped value in HTML")
	}
	second, err := renderEnvironmentRowsHTML(result)
	if err != nil || page != second {
		t.Fatalf("HTML projection is not deterministic: %v", err)
	}
}

func TestEnvironmentRowsHTMLDoesNotInventMissingDecisions(t *testing.T) {
	result := destinationRead{
		Rows: []destinationRow{{Key: "1"}, {Key: "2"}},
		PerRow: map[string]map[string]string{
			"1": {"rule": "satisfied"},
			"2": {},
		},
	}
	if _, err := renderEnvironmentRowsHTML(result); err == nil {
		t.Fatal("a missing row decision must not be rendered as undecided")
	}
}

func TestEnvironmentRowsHTMLEmptyRowsDoNotImplyCompleteCoverage(t *testing.T) {
	page, err := renderEnvironmentRowsHTML(destinationRead{
		Application: "order_entry", Pattern: "order_entry.order_lines", Coverage: "partial",
		Rows: []destinationRow{}, PerRelation: map[string]string{"unique": "undecided"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page, "取得した行（0行）") ||
		!strings.Contains(page, `data-coverage="partial"`) ||
		!strings.Contains(page, `data-answer="undecided"`) {
		t.Fatalf("empty partial result was misrepresented: %s", page)
	}
}

func TestEnvironmentRuntimeHTMLRequiresExplicitDatabaseConnection(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	t.Setenv("TEI_VIEW_PG_DSN", "")
	var out bytes.Buffer
	exit, err := runViewCommand([]string{
		"-workspace", root, "-application", "order_entry", "-pattern", "order_entry.order_lines",
		"-environment", "development", "-input", "order_id=1001", "-format", "html",
	}, &out)
	if err != nil || exit == 0 {
		t.Fatalf("exit=%d err=%v output=%s", exit, err, out.String())
	}
	var report struct {
		OK    bool `json:"ok"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.OK || report.Error.Code != "VIEW_EXECUTION_FAILED" {
		t.Fatalf("unexpected failure report: %+v", report)
	}
}

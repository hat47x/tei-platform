package applicationview_test

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/applicationview"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func headerDetailWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Data: []semantic.Data{
				{
					ID: "sales.order",
					Attributes: []semantic.Attribute{
						{ID: "sales.order.order_id", Display: "受注番号", TypeRef: "std.identifier"},
						{ID: "sales.order.customer_id", Display: "顧客", TypeRef: "customer.customer_id"},
					},
					Relations: []semantic.Relation{{
						ID:          "sales.order.order_lines",
						Target:      "sales.order_line",
						Cardinality: semantic.CardinalityMany,
					}},
				},
				{
					ID: "sales.order_line",
					Attributes: []semantic.Attribute{
						{ID: "sales.order_line.line_no", Display: "行", TypeRef: "std.integer"},
						{ID: "sales.order_line.quantity", Display: "数量", TypeRef: "std.decimal"},
						{ID: "sales.order_line.amount", Display: "金額", TypeRef: "std.money"},
					},
				},
			},
		}},
		Applications: []application.Application{{
			ID:           "order_entry",
			SemanticUses: []core.CanonicalID{"sales"},
			Packages: []application.Package{{
				ID:      "order_entry.order_lines",
				Version: "1",
				Members: []application.PackageMember{
					{Kind: application.PatternHeaderDetail, Pattern: "order_entry.order_lines"},
				},
			}},
			Patterns: []application.Pattern{{
				ID:        "order_entry.order_lines",
				Kind:      application.PatternHeaderDetail,
				Relation:  "sales.order.order_lines",
				PageBreak: &application.PageBreak{Rows: 3, RepeatHeader: true},
				Aggregates: []application.Aggregate{{
					Name:      "total",
					Attribute: "sales.order_line.amount",
					Function:  "sum",
				}},
			}},
		}},
	}
}

func TestBuildHeaderDetailResolvesHeaderAndDetail(t *testing.T) {
	view, err := applicationview.BuildHeaderDetail(headerDetailWorld(), "order_entry", "order_entry.order_lines")
	if err != nil {
		t.Fatal(err)
	}
	if view.Kind != application.PatternHeaderDetail || view.Header != "sales.order" || view.Detail != "sales.order_line" {
		t.Fatalf("view=%+v", view)
	}
	if len(view.HeaderItems) != 2 || view.HeaderItems[0].Label != "受注番号" {
		t.Fatalf("header=%+v", view.HeaderItems)
	}
	if len(view.DetailColumns) != 3 || view.DetailColumns[1].Label != "数量" || view.DetailColumns[1].Input != "number" {
		t.Fatalf("detail=%+v", view.DetailColumns)
	}
}

func TestBuildHeaderDetailRejectsNonHeaderDetailPattern(t *testing.T) {
	world := headerDetailWorld()
	world.Applications[0].Patterns[0].Kind = application.PatternRecord
	if _, err := applicationview.BuildHeaderDetail(world, "order_entry", "order_entry.order_lines"); err == nil {
		t.Fatal("record pattern was accepted as header-detail")
	}
}

func TestRenderHeaderDetailHTML(t *testing.T) {
	view, err := applicationview.BuildHeaderDetail(headerDetailWorld(), "order_entry", "order_entry.order_lines")
	if err != nil {
		t.Fatal(err)
	}
	html, err := applicationview.RenderHeaderDetailHTML(view)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<table>", "<th>数量</th>", "受注番号"} {
		if !strings.Contains(html, want) {
			t.Fatalf("html does not contain %q: %s", want, html)
		}
	}
}

func TestBuildHeaderDetailCarriesPackages(t *testing.T) {
	view, err := applicationview.BuildHeaderDetail(headerDetailWorld(), "order_entry", "order_entry.order_lines")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Packages) != 1 || view.Packages[0].ID != "order_entry.order_lines" || view.Packages[0].Version != "1" {
		t.Fatalf("packages=%+v", view.Packages)
	}
}

func TestBuildHeaderDetailCarriesSemanticFingerprint(t *testing.T) {
	view, err := applicationview.BuildHeaderDetail(headerDetailWorld(), "order_entry", "order_entry.order_lines")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(view.SemanticFingerprint, "sha256:") {
		t.Fatalf("fingerprint=%q", view.SemanticFingerprint)
	}
}

func TestBuildHeaderDetailCarriesAggregates(t *testing.T) {
	view, err := applicationview.BuildHeaderDetail(headerDetailWorld(), "order_entry", "order_entry.order_lines")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Aggregates) != 1 {
		t.Fatalf("aggregates=%+v", view.Aggregates)
	}
	aggregate := view.Aggregates[0]
	if aggregate.Name != "total" || aggregate.Function != "sum" || aggregate.Label != "金額" || aggregate.Type != "std.money" {
		t.Fatalf("aggregate=%+v", aggregate)
	}
}

func TestBuildHeaderDetailCarriesPageBreak(t *testing.T) {
	view, err := applicationview.BuildHeaderDetail(headerDetailWorld(), "order_entry", "order_entry.order_lines")
	if err != nil {
		t.Fatal(err)
	}
	if view.PageBreak == nil || view.PageBreak.Rows != 3 || !view.PageBreak.RepeatHeader {
		t.Fatalf("page-break=%+v", view.PageBreak)
	}
	if len(view.DetailRows) != 3 {
		t.Fatalf("detail rows=%d", len(view.DetailRows))
	}
}

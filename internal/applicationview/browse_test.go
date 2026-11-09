package applicationview_test

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/applicationview"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/semantic"
)

func browseWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Data: []semantic.Data{{
				ID: "sales.order",
				Attributes: []semantic.Attribute{
					{ID: "sales.order.order_id", Display: "受注番号", TypeRef: "std.identifier"},
					{ID: "sales.order.customer_id", Display: "顧客", TypeRef: "customer.customer_id"},
					{ID: "sales.order.status", Display: "状態", TypeRef: "std.code"},
				},
			}},
		}},
		Queries: []query.EmbeddedSQL{{
			ID:          "order_entry.recent_orders",
			Application: "order_entry",
			Inputs:      []query.Input{{Name: "status", Type: "std.code"}},
			ResultColumns: []query.ResultColumnMapping{
				{Column: "order_id", Attribute: "sales.order.order_id"},
				{Column: "customer_id", Attribute: "sales.order.customer_id"},
			},
		}},
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID:    "order_entry.recent_orders",
				Kind:  application.PatternBrowse,
				Data:  "sales.order",
				Query: "order_entry.recent_orders",
				Filters: []application.Filter{{
					Attribute: "sales.order.status",
					Operators: []string{"eq", "ne"},
					Parameter: "status",
				}},
				Sorts: []application.Sort{{
					Attribute: "sales.order.order_id",
					Direction: "asc",
				}},
				PageSize: 20,
			}},
		}},
	}
}

func TestBuildBrowseResolvesQueryColumns(t *testing.T) {
	view, err := applicationview.BuildBrowse(browseWorld(), "order_entry", "order_entry.recent_orders")
	if err != nil {
		t.Fatal(err)
	}
	if view.Query != "order_entry.recent_orders" || len(view.Columns) != 2 {
		t.Fatalf("view=%+v", view)
	}
	if view.Columns[0].Label != "受注番号" || view.Columns[0].Type != "std.identifier" {
		t.Fatalf("column=%+v", view.Columns[0])
	}
	if len(view.Filters) != 1 || view.Filters[0].Label != "状態" || len(view.Filters[0].Operators) != 2 {
		t.Fatalf("filters=%+v", view.Filters)
	}
	if view.Filters[0].Parameter != "status" {
		t.Fatalf("filter parameter=%+v", view.Filters[0])
	}
	if view.PageSize != 20 {
		t.Fatalf("page-size=%d", view.PageSize)
	}
	if len(view.Sorts) != 1 || view.Sorts[0].Label != "受注番号" || view.Sorts[0].Direction != "asc" {
		t.Fatalf("sorts=%+v", view.Sorts)
	}
}

func TestBuildBrowseFailsClosedWithoutQueryColumns(t *testing.T) {
	world := browseWorld()
	world.Queries[0].ResultColumns = nil
	if _, err := applicationview.BuildBrowse(world, "order_entry", "order_entry.recent_orders"); err == nil {
		t.Fatal("query without result columns was accepted")
	}
}

func TestRenderBrowseHTMLAndPrint(t *testing.T) {
	view, err := applicationview.BuildBrowse(browseWorld(), "order_entry", "order_entry.recent_orders")
	if err != nil {
		t.Fatal(err)
	}
	html, err := applicationview.RenderBrowseHTML(view)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<th>受注番号</th>") || !strings.Contains(html, "page-size: 20") || !strings.Contains(html, "並び順") || !strings.Contains(html, "状態 (status)") || strings.Count(html, "<tr><td></td><td></td></tr>") != 5 {
		t.Fatalf("browse html=%s", html)
	}
	printed, err := applicationview.RenderBrowsePrintHTML(view)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(printed, "@page") || !strings.Contains(printed, "受注番号") || !strings.Contains(printed, "format: tei.print/v0") {
		t.Fatalf("browse print=%s", printed)
	}
}

package applicationview_test

import (
	"strings"
	"testing"

	"tei/internal/applicationview"
)

func TestRenderPrintHTMLCarriesTemplate(t *testing.T) {
	world := viewWorld()
	for i := range world.Applications[0].Patterns {
		if world.Applications[0].Patterns[i].ID == "order_entry.order_record" {
			world.Applications[0].Patterns[i].PrintTemplate = "a4-v1"
		}
	}
	view, err := applicationview.Build(world, "order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatal(err)
	}
	html, err := applicationview.RenderPrintHTML(view)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "template: a4-v1") {
		t.Fatalf("print html does not carry the template: %s", html)
	}
}

func TestRenderPrintHTML(t *testing.T) {
	view, err := applicationview.Build(viewWorld(), "order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatal(err)
	}
	html, err := applicationview.RenderPrintHTML(view)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"@page", "受注番号", "<table>", "format: tei.print/v0"} {
		if !strings.Contains(html, want) {
			t.Fatalf("print html does not contain %q: %s", want, html)
		}
	}
}

func TestRenderHeaderDetailPrintHTML(t *testing.T) {
	view, err := applicationview.BuildHeaderDetail(headerDetailWorld(), "order_entry", "order_entry.order_lines")
	if err != nil {
		t.Fatal(err)
	}
	html, err := applicationview.RenderHeaderDetailPrintHTML(view)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"@page", "明細", "<thead>", "数量", "集計", "3行/ページ", "format: tei.print/v0"} {
		if !strings.Contains(html, want) {
			t.Fatalf("print html does not contain %q: %s", want, html)
		}
	}
	if strings.Count(html, "<tr><td></td><td></td><td></td></tr>") != 3 {
		t.Fatalf("detail rows are not repeated: %s", html)
	}
}

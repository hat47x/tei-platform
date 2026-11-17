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
	for _, want := range []string{"@page", "明細", "<thead>", "数量", "集計", "3行ごとに面を分ける", "format: tei.print/v0"} {
		if !strings.Contains(html, want) {
			t.Fatalf("print html does not contain %q: %s", want, html)
		}
	}
	// 3行/ページの宣言でプレースホルダ5行を分けると、3行と2行の2面になる。
	// 分割しても行は落ちず、repeat-headerがあるので両面が見出しを描く。
	if got := strings.Count(html, "<tr><td></td><td></td><td></td></tr>"); got != 5 {
		t.Fatalf("detail rows=%d, want 5: %s", got, html)
	}
	if got := strings.Count(html, "<thead>"); got != 2 {
		t.Fatalf("thead count=%d, want 2 (見出し繰返し): %s", got, html)
	}
}

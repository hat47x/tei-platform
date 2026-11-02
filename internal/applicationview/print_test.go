package applicationview_test

import (
	"strings"
	"testing"

	"tei/internal/applicationview"
)

func TestRenderPrintHTML(t *testing.T) {
	view, err := applicationview.Build(viewWorld(), "order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatal(err)
	}
	html, err := applicationview.RenderPrintHTML(view)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"@page", "受注番号", "<table>"} {
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
	for _, want := range []string{"@page", "明細", "<thead>", "数量"} {
		if !strings.Contains(html, want) {
			t.Fatalf("print html does not contain %q: %s", want, html)
		}
	}
	if strings.Count(html, "<tr><td></td><td></td><td></td></tr>") != 5 {
		t.Fatalf("detail rows are not repeated: %s", html)
	}
}

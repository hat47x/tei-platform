package applicationview_test

import (
	"strings"
	"testing"

	"tei/internal/applicationview"
)

func TestRenderHTMLEscapesLabelsAndRendersInputs(t *testing.T) {
	world := viewWorld()
	world.Semantics[0].Data[0].Attributes[0].Display = `<script>alert(1)</script>`
	view, err := applicationview.Build(world, "order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatal(err)
	}
	html, err := applicationview.RenderHTML(view)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `type="text"`) || !strings.Contains(html, `name="sales.order.order_id"`) {
		t.Fatalf("input was not rendered: %s", html)
	}
	if !strings.Contains(html, "参照: order_entry.customer_reference") {
		t.Fatalf("lookup was not rendered: %s", html)
	}
	if strings.Contains(html, "<script>alert(1)</script>") {
		t.Fatal("label was not escaped")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Fatalf("escaped label missing: %s", html)
	}
}

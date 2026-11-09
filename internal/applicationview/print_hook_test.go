package applicationview_test

import (
	"strings"
	"testing"

	"tei/internal/applicationview"
)

// TestRenderPrintHTMLCarriesTheHooks checks that a printed record page shows the
// hook declarations of the same pattern as the screen. A hook that appears on
// one surface and not the other would make the two surfaces describe different
// contracts, which is what AC-2 asks to avoid.
func TestRenderPrintHTMLCarriesTheHooks(t *testing.T) {
	view, err := applicationview.Build(viewWorld(), "order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatal(err)
	}
	html, err := applicationview.RenderPrintHTML(view)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Hooks) == 0 {
		t.Fatalf("the fixture has no hooks: %+v", view.Hooks)
	}
	if !strings.Contains(html, "フック") || !strings.Contains(html, view.Hooks[0].Phase) {
		t.Fatalf("print html does not show the hook %q: %s", view.Hooks[0].Phase, html)
	}
	for _, use := range view.Hooks[0].Uses {
		if !strings.Contains(html, string(use)) {
			t.Fatalf("print html does not show the declared capability %q: %s", use, html)
		}
	}
}

// TestPrintHTMLWithoutHooksHasNoEmptySection checks that the section appears only
// when a hook is declared, so a page does not claim a contract it does not have.
func TestPrintHTMLWithoutHooksHasNoEmptySection(t *testing.T) {
	world := viewWorld()
	for i := range world.Applications[0].Patterns {
		if world.Applications[0].Patterns[i].ID == "order_entry.order_record" {
			world.Applications[0].Patterns[i].Interaction = nil
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
	if strings.Contains(html, "フック") {
		t.Fatalf("print html claims hooks it does not declare: %s", html)
	}
}

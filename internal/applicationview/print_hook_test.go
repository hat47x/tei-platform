package applicationview_test

import (
	"strings"
	"testing"

	"tei/internal/application"
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

// TestSurfacesCarryTheHookRowSource checks that the screen and the printed page
// both show where a row-scoped hook takes its rows from. A source that appears
// on one surface and not the other would leave two descriptions of the same
// declaration, which is the split AC-2 asks to avoid.
func TestSurfacesCarryTheHookRowSource(t *testing.T) {
	world := viewWorld()
	for i := range world.Applications[0].Patterns {
		pattern := &world.Applications[0].Patterns[i]
		if pattern.ID != "order_entry.order_record" || pattern.Interaction == nil {
			continue
		}
		pattern.Interaction.Hooks[0].Scope = "row"
		pattern.Interaction.Hooks[0].Rows = &application.HookRowSource{Query: "order_entry.order_list"}
	}
	view, err := applicationview.Build(world, "order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatal(err)
	}
	html, err := applicationview.RenderHTML(view)
	if err != nil {
		t.Fatal(err)
	}
	printed, err := applicationview.RenderPrintHTML(view)
	if err != nil {
		t.Fatal(err)
	}
	for name, rendered := range map[string]string{"html": html, "print": printed} {
		if !strings.Contains(rendered, "order_entry.order_list") {
			t.Fatalf("%s does not show the hook's row source: %s", name, rendered)
		}
		if !strings.Contains(rendered, "scope row") {
			t.Fatalf("%s does not show the hook's scope: %s", name, rendered)
		}
	}
}

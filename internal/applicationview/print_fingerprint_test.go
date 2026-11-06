package applicationview_test

import (
	"strings"
	"testing"

	"tei/internal/applicationview"
	"tei/internal/core"
)

// TestPrintHTMLCarriesTheSemanticFingerprint checks that a print surface names
// the semantic revision it was rendered from. Without it a printed page cannot
// be compared with the meaning it came from, which is what AC-7 asks for.
func TestPrintHTMLCarriesTheSemanticFingerprint(t *testing.T) {
	view, err := applicationview.Build(viewWorld(), "order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatal(err)
	}
	html, err := applicationview.RenderPrintHTML(view)
	if err != nil {
		t.Fatal(err)
	}
	if condition := `<meta name="tei-semantic-fingerprint" content="` + view.SemanticFingerprint; !strings.Contains(html, condition) {
		t.Fatalf("print html does not name the revision %q: %s", view.SemanticFingerprint, html)
	}
}

// TestHeaderDetailPrintHTMLCarriesTheSemanticFingerprint checks the same for a
// header-detail print surface, which has its own template.
func TestHeaderDetailPrintHTMLCarriesTheSemanticFingerprint(t *testing.T) {
	view, err := applicationview.BuildHeaderDetail(headerDetailWorld(), "order_entry", "order_entry.order_lines")
	if err != nil {
		t.Fatal(err)
	}
	html, err := applicationview.RenderHeaderDetailPrintHTML(view)
	if err != nil {
		t.Fatal(err)
	}
	if condition := `<meta name="tei-semantic-fingerprint" content="` + view.SemanticFingerprint; !strings.Contains(html, condition) {
		t.Fatalf("header-detail print html does not name the revision %q: %s", view.SemanticFingerprint, html)
	}
}

// TestBrowsePrintHTMLCarriesTheSemanticFingerprint checks the list surface.
func TestBrowsePrintHTMLCarriesTheSemanticFingerprint(t *testing.T) {
	world := browseWorld()
	// The fixture does not name the semantic it uses, so a fingerprint cannot
	// exist; name it so the print surface has a revision to carry.
	world.Applications[0].SemanticUses = []core.CanonicalID{"sales"}
	view, err := applicationview.BuildBrowse(world, "order_entry", "order_entry.recent_orders")
	if err != nil {
		t.Fatal(err)
	}
	html, err := applicationview.RenderBrowsePrintHTML(view)
	if err != nil {
		t.Fatal(err)
	}
	if condition := `<meta name="tei-semantic-fingerprint" content="` + view.SemanticFingerprint; !strings.Contains(html, condition) {
		t.Fatalf("browse print html does not name the revision %q: %s", view.SemanticFingerprint, html)
	}
}

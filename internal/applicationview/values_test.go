package applicationview_test

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/applicationview"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func valuesWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Types: []semantic.Type{{
				ID:         "sales.status",
				Symbol:     "Status",
				Kind:       semantic.TypeAlias,
				Underlying: "std.code",
				Values:     []string{"open", "closed"},
			}},
			Data: []semantic.Data{{
				ID: "sales.ticket",
				Attributes: []semantic.Attribute{{
					ID:      "sales.ticket.status",
					Symbol:  "status",
					TypeRef: "sales.status",
				}},
			}},
		}},
		Applications: []application.Application{{
			ID: "app",
			Patterns: []application.Pattern{{
				ID:   "app.ticket",
				Kind: application.PatternRecord,
				Data: "sales.ticket",
				Interaction: &application.Interaction{
					Items: []application.InteractionItem{{Attribute: "sales.ticket.status"}},
				},
			}},
		}},
	}
}

func TestBuildCarriesTypeValues(t *testing.T) {
	view, err := applicationview.Build(valuesWorld(), "app", "app.ticket")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Items) != 1 || len(view.Items[0].Values) != 2 {
		t.Fatalf("items=%+v", view.Items)
	}
	if view.Items[0].Values[0] != "open" || view.Items[0].Values[1] != "closed" {
		t.Fatalf("values=%v", view.Items[0].Values)
	}
}

func TestRenderHTMLUsesSelectForValues(t *testing.T) {
	view, err := applicationview.Build(valuesWorld(), "app", "app.ticket")
	if err != nil {
		t.Fatal(err)
	}
	html, err := applicationview.RenderHTML(view)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<select") || !strings.Contains(html, `<option value="open">open</option>`) {
		t.Fatalf("select was not rendered: %s", html)
	}
}

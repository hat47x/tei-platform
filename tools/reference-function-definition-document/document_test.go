package funcdoc

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func world() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{ID: "s",
			Data:        []semantic.Data{{ID: "s.order", Symbol: "Order"}},
			Validations: []semantic.Validation{{ID: "s.positive"}},
			Capabilities: []semantic.Capability{{
				ID: "s.place", Symbol: "place",
				Inputs:   []semantic.Parameter{{Name: "order", Type: "s.order"}},
				Output:   "s.order",
				Outcomes: []semantic.Outcome{{Name: "placed", Payload: "s.order"}},
				Effect:   &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "s.order"},
			}},
		}},
		Applications: []application.Application{
			{ID: "app.empty"},
			{ID: "app.main", SemanticUses: []core.CanonicalID{"s"}, Validations: []core.CanonicalID{"s.positive", "s.ghost"},
				Patterns: []application.Pattern{
					{ID: "app.main.browse", Kind: application.PatternBrowse, Data: "s.order"},
					{ID: "app.main.place", Kind: application.PatternCommand, Capability: "s.place"},
					{ID: "app.main.broken", Kind: application.PatternCommand, Capability: "s.gone"},
					{ID: "app.main.ghost", Kind: application.PatternBrowse, Data: "s.nowhere"},
				}},
		},
	}
}

func TestRendersDeclaredContentWithCapabilityDetails(t *testing.T) {
	doc, err := Render(world())
	if err != nil {
		t.Fatal(err)
	}
	md := doc.Markdown
	for _, want := range []string{
		"## `app.main`", "| `app.main.browse` | browse |", "### Capability `s.place`",
		"入力: `order`: `s.order`", "業務上の結末 `placed`", "宣言された効果: create（`s.order`）",
		"評価する検証（宣言順）: `s.positive`、`s.ghost`（未解決）",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing %q in:\n%s", want, md)
		}
	}
	if !strings.Contains(md, "Patternは宣言されていません") {
		t.Fatal("Application without Pattern not stated")
	}
}

func TestUnresolvedReferencesAreListedNotHidden(t *testing.T) {
	doc, _ := Render(world())
	got := strings.Join(doc.Unresolved, "\n")
	for _, want := range []string{"s.ghost", "s.gone", "s.nowhere"} {
		if !strings.Contains(got, want) {
			t.Fatalf("unresolved %q missing: %v", want, doc.Unresolved)
		}
	}
	if !strings.Contains(doc.Markdown, "## 未解決の事項") {
		t.Fatal("section missing")
	}
}

func TestStatesLimitsAndIsRepeatable(t *testing.T) {
	a, _ := Render(world())
	b, _ := Render(world())
	if a.Digest != b.Digest || a.Markdown != b.Markdown {
		t.Fatal("not repeatable")
	}
	for _, want := range []string{"編集しても、意味の正本には反映されません", "実際の副作用の観測ではありません", a.StructureRevision} {
		if !strings.Contains(a.Markdown, want) {
			t.Fatalf("missing %q", want)
		}
	}
}

func TestOrderOfApplicationsDoesNotChangeTheBody(t *testing.T) {
	w := world()
	a, _ := Render(w)
	w.Applications[0], w.Applications[1] = w.Applications[1], w.Applications[0]
	b, _ := Render(w)
	body := func(d Document) string { return strings.ReplaceAll(d.Markdown, d.StructureRevision, "") }
	if body(a) != body(b) {
		t.Fatal("body depends on input order")
	}
}

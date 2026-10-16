package accessmatrix

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func world() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{ID: "s",
			Data: []semantic.Data{{ID: "s.order", Symbol: "Order"}, {ID: "s.customer", Symbol: "Customer"}},
			Capabilities: []semantic.Capability{
				{ID: "s.place", Symbol: "place", Effect: &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "s.order"}},
				{ID: "s.amend", Symbol: "amend", Effect: &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "s.order"}},
				{ID: "s.noop", Symbol: "noop"},
			}}},
		Applications: []application.Application{{ID: "app.b"}, {ID: "app.a", Patterns: []application.Pattern{
			{ID: "app.a.browse", Kind: application.PatternBrowse, Data: "s.order"},
			{ID: "app.a.record", Kind: application.PatternRecord, Data: "s.customer"},
			{ID: "app.a.place", Kind: application.PatternCommand, Capability: "s.place"},
			{ID: "app.a.amend", Kind: application.PatternCommand, Capability: "s.amend"},
			{ID: "app.a.noop", Kind: application.PatternCommand, Capability: "s.noop"},
			{ID: "app.a.missing", Kind: application.PatternCommand, Capability: "s.gone"},
			{ID: "app.a.nodata", Kind: application.PatternBrowse},
			{ID: "app.a.ghost", Kind: application.PatternBrowse, Data: "s.ghost"},
		}}},
	}
}

func find(m Matrix, app, data string) *Cell {
	for i := range m.Cells {
		if m.Cells[i].Application == app && m.Cells[i].Data == data {
			return &m.Cells[i]
		}
	}
	return nil
}

func TestCellsCombineReadCreateUpdate(t *testing.T) {
	m, err := Derive(world())
	if err != nil {
		t.Fatal(err)
	}
	c := find(m, "app.a", "s.order")
	if c == nil || strings.Join(c.Access, "") != "CRU" {
		t.Fatalf("cell = %+v", c)
	}
	if len(c.Via) != 3 {
		t.Fatalf("via = %v", c.Via)
	}
	if c := find(m, "app.a", "s.customer"); c == nil || strings.Join(c.Access, "") != "R" {
		t.Fatalf("cell = %+v", c)
	}
}

func TestDeclarationsThatCannotBeReadAreGapsNotGuesses(t *testing.T) {
	m, _ := Derive(world())
	got := map[string]bool{}
	for _, g := range m.Gaps {
		got[g.Code+":"+g.Subject] = true
	}
	for _, want := range []string{
		"capability-not-found:app.a.missing", "capability-declares-no-effect:app.a.noop",
		"pattern-declares-no-data:app.a.nodata", "pattern-data-not-found:app.a.ghost",
	} {
		if !got[want] {
			t.Fatalf("missing gap %s in %v", want, m.Gaps)
		}
	}
}

func TestDeleteIsNeverShownAndTheMatrixSaysSo(t *testing.T) {
	m, _ := Derive(world())
	for _, c := range m.Cells {
		for _, a := range c.Access {
			if a == "D" {
				t.Fatal("delete shown")
			}
		}
	}
	if !strings.Contains(m.Markdown, "削除は現在のモデルで表現できない") || !strings.Contains(m.Markdown, "実際の副作用の観測ではありません") {
		t.Fatal("limits not stated")
	}
}

func TestApplicationWithoutAccessAppearsWithBlankRow(t *testing.T) {
	m, _ := Derive(world())
	if !strings.Contains(m.Markdown, "| `app.b` | — | — |") {
		t.Fatalf("markdown = %s", m.Markdown)
	}
}

func TestDeterministic(t *testing.T) {
	a, _ := Derive(world())
	b, _ := Derive(world())
	if a.Markdown != b.Markdown || a.StructureRevision != b.StructureRevision {
		t.Fatal("not repeatable")
	}
}

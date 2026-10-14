package modelsearch

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

func world() ir.World {
	return ir.World{Semantics: []semantic.Namespace{{ID: "sales", Data: []semantic.Data{
		{ID: "sales.order", Symbol: "Order", Attributes: []semantic.Attribute{
			{ID: "sales.order.no", Symbol: "orderNo"},
			{ID: "sales.order.cust", Symbol: "customerNo"},
		}},
		{ID: "sales.invoice", Symbol: "Invoice", Attributes: []semantic.Attribute{
			{ID: "sales.invoice.no", Symbol: "orderNo"},
		}},
	}}}}
}

func TestSearchIsCaseInsensitiveAndNeverMatchesIDs(t *testing.T) {
	res, err := Search(world(), Query{Text: "ORDER"})
	if err != nil {
		t.Fatal(err)
	}
	// Data "Order" and two "orderNo" attributes; IDs containing "order" are not hits.
	if len(res.Hits) != 3 {
		t.Fatalf("hits = %+v", res.Hits)
	}
	for _, h := range res.Hits {
		if h.Field != "symbol" {
			t.Fatalf("non-symbol hit %+v", h)
		}
	}
	cs, _ := Search(world(), Query{Text: "ORDER", CaseSensitive: true})
	if len(cs.Hits) != 0 {
		t.Fatalf("case-sensitive hits = %+v", cs.Hits)
	}
}

func TestKindFilterAndExact(t *testing.T) {
	res, _ := Search(world(), Query{Text: "orderNo", Exact: true, Kinds: []string{KindAttribute}})
	if len(res.Hits) != 2 {
		t.Fatalf("hits = %+v", res.Hits)
	}
}

func TestProposeDoesNotApplyAndKeepsIDs(t *testing.T) {
	w := world()
	res, _ := Search(w, Query{Text: "orderNo", Exact: true})
	p, err := Propose(w, res, "salesNo", nil)
	if err != nil {
		t.Fatal(err)
	}
	if p.Applied || len(p.Edits) != 2 || p.BaseRevision != res.StructureRevision {
		t.Fatalf("proposal = %+v", p)
	}
	if w.Semantics[0].Data[0].Attributes[0].Symbol != "orderNo" {
		t.Fatal("world was mutated")
	}
	if p.Edits[0].ID != "sales.order.no" {
		t.Fatalf("edit targets %q", p.Edits[0].ID)
	}
}

func TestProposePartialSelection(t *testing.T) {
	w := world()
	res, _ := Search(w, Query{Text: "orderNo", Exact: true})
	p, _ := Propose(w, res, "salesNo", []string{"sales.invoice.no"})
	if len(p.Edits) != 1 || p.Edits[0].ID != "sales.invoice.no" {
		t.Fatalf("edits = %+v", p.Edits)
	}
}

func TestProposeDetectsSymbolCollision(t *testing.T) {
	w := world()
	res, _ := Search(w, Query{Text: "orderNo", Exact: true, Kinds: []string{KindAttribute}})
	p, _ := Propose(w, res, "customerNo", []string{"sales.order.no"})
	if len(p.Diagnostics) != 1 || p.Diagnostics[0].Code != "symbol-collision" {
		t.Fatalf("diagnostics = %+v", p.Diagnostics)
	}
}

func TestProposeRefusesStaleResult(t *testing.T) {
	w := world()
	res, _ := Search(w, Query{Text: "orderNo", Exact: true})
	w.Semantics[0].Data[0].Attributes[0].Symbol = "changed"
	if _, err := Propose(w, res, "x", nil); err == nil {
		t.Fatal("stale result accepted")
	}
}

func TestSubstringReplaceCaseInsensitive(t *testing.T) {
	w := world()
	res, _ := Search(w, Query{Text: "ORDER"})
	p, _ := Propose(w, res, "Sales", nil)
	got := map[string]string{}
	for _, e := range p.Edits {
		got[e.ID] = e.After
	}
	if got["sales.order"] != "Sales" || got["sales.order.no"] != "SalesNo" {
		t.Fatalf("edits = %+v", got)
	}
}

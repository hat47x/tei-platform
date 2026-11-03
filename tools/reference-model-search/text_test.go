package modelsearch

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

func textWorld() ir.World {
	return ir.World{Semantics: []semantic.Namespace{{ID: "sales", Data: []semantic.Data{
		{ID: "sales.order", Symbol: "Order", Description: "受注を表す", Display: "受注画面", Script: "return order;"},
		{ID: "sales.invoice", Symbol: "Invoice", Description: "受注の請求"},
	}}}}
}

func TestProposeTextChangesDescriptionOnlyAndKeepsSymbolsAndIDs(t *testing.T) {
	w := textWorld()
	res, err := Search(w, Query{Text: "受注"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := ProposeText(w, res, "description", "受注済み", []string{"sales.order"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Schema != TextProposalSchema {
		t.Errorf("schema = %q, want %q", p.Schema, TextProposalSchema)
	}
	if p.Applied {
		t.Error("proposal must not be applied")
	}
	if len(p.Edits) != 1 {
		t.Fatalf("edits = %+v", p.Edits)
	}
	edit := p.Edits[0]
	if edit.ID != "sales.order" || edit.Field != "description" {
		t.Errorf("edit = %+v", edit)
	}
	if edit.Before != "受注を表す" || edit.After != "受注済みを表す" {
		t.Errorf("edit text = %+v", edit)
	}
	for _, e := range p.Edits {
		if e.Field == "symbol" || e.Field == "id" {
			t.Errorf("text change touched identity: %+v", e)
		}
	}
}

func TestProposeTextChangesDisplayOnly(t *testing.T) {
	w := textWorld()
	res, _ := Search(w, Query{Text: "受注"})
	p, err := ProposeText(w, res, "display", "受注一覧", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Edits) != 1 || p.Edits[0].Field != "display" || p.Edits[0].ID != "sales.order" {
		t.Fatalf("edits = %+v", p.Edits)
	}
}

func TestProposeTextChangesScriptOnly(t *testing.T) {
	w := textWorld()
	res, _ := Search(w, Query{Text: "order"})
	p, err := ProposeText(w, res, "script", "settle", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Edits) != 1 {
		t.Fatalf("edits = %+v", p.Edits)
	}
	if p.Edits[0].Field != "script" || p.Edits[0].Before != "return order;" || p.Edits[0].After != "return settle;" {
		t.Errorf("edit = %+v", p.Edits[0])
	}
}

func TestProposeTextRejectsSymbolField(t *testing.T) {
	w := textWorld()
	res, _ := Search(w, Query{Text: "order"})
	if _, err := ProposeText(w, res, "symbol", "x", nil); err == nil {
		t.Fatal("symbol must be a separate typed change")
	}
}

func TestProposeTextRefusesStaleResult(t *testing.T) {
	w := textWorld()
	res, _ := Search(w, Query{Text: "受注"})
	w.Semantics[0].Data[0].Symbol = "Changed"
	if _, err := ProposeText(w, res, "description", "x", nil); err == nil {
		t.Fatal("a stale search result must be refused")
	}
}

func TestProposeTextPartialSelection(t *testing.T) {
	w := textWorld()
	res, _ := Search(w, Query{Text: "受注"})
	p, err := ProposeText(w, res, "description", "受注済み", []string{"sales.order"})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Edits) != 1 || p.Edits[0].ID != "sales.order" {
		t.Fatalf("edits = %+v", p.Edits)
	}
}

func TestProposeTextSelectionOutsideFieldIsRejected(t *testing.T) {
	w := textWorld()
	res, _ := Search(w, Query{Text: "order"})
	if _, err := ProposeText(w, res, "description", "x", []string{"sales.order"}); err == nil {
		t.Fatal("selected id without a description hit must be rejected")
	}
}

func TestProposeTextCaseSensitive(t *testing.T) {
	w := ir.World{Semantics: []semantic.Namespace{{ID: "sales", Data: []semantic.Data{
		{ID: "sales.order", Symbol: "Order", Description: "Order order"},
	}}}}
	res, _ := Search(w, Query{Text: "Order", CaseSensitive: true})
	p, err := ProposeText(w, res, "description", "X", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Edits) != 1 || p.Edits[0].After != "X order" {
		t.Fatalf("edits = %+v", p.Edits)
	}

	res, _ = Search(w, Query{Text: "Order"})
	p, err = ProposeText(w, res, "description", "X", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Edits) != 1 || p.Edits[0].After != "X X" {
		t.Fatalf("case-insensitive edits = %+v", p.Edits)
	}
}

func TestProposeTextExactReplacesWholeField(t *testing.T) {
	w := ir.World{Semantics: []semantic.Namespace{{ID: "sales", Data: []semantic.Data{
		{ID: "sales.order", Symbol: "Order", Description: "受注"},
	}}}}
	res, _ := Search(w, Query{Text: "受注", Exact: true})
	p, err := ProposeText(w, res, "description", "受注済み", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Edits) != 1 || p.Edits[0].After != "受注済み" {
		t.Fatalf("edits = %+v", p.Edits)
	}
}

func TestProposeTextEmptyReplacementIsRejected(t *testing.T) {
	w := textWorld()
	res, _ := Search(w, Query{Text: "受注"})
	if _, err := ProposeText(w, res, "description", "  ", nil); err == nil {
		t.Fatal("empty replacement must be rejected")
	}
}

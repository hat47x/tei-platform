package query_test

import (
	"errors"
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/query"
	"tei/internal/semantic/derivationref"
)

func d2Fixture() (query.EmbeddedSQL, query.DerivedSelect, query.CompleteSourceEvidence, []query.DerivedRow) {
	q := query.EmbeddedSQL{
		ID: "inventory.available", Source: "SELECT product_id, stock, reserved FROM inventory",
		ResultColumns: []query.ResultColumnMapping{
			{Column: "product_id", Attribute: "product.id"},
			{Column: "stock", Attribute: "product.stock"},
			{Column: "reserved", Attribute: "product.reserved"},
		},
	}
	p := query.DerivedSelect{
		Rule:    derivationref.Definition{ID: "available-rule", Inputs: []string{"stock", "reserved"}, Operator: "subtract", Pure: true},
		Binding: derivationref.Binding{RuleID: "available-rule", TargetID: "product.available", Scope: "query"},
		Inputs:  map[string]core.CanonicalID{"stock": "product.stock", "reserved": "product.reserved"},
		Order: []query.DerivedOrder{
			{Attribute: "product.available", Descending: true, Kind: "decimal"},
			{Attribute: "product.id", Kind: "text"},
		},
		MaxRows: 100,
	}
	e := query.CompleteSourceEvidence{
		SQLSHA256: query.DigestNativeSQL(q.Source),
		IR: query.DerivedIR{
			Query:    q.ID,
			Coverage: query.AnalysisCoverage{Parse: query.CoverageFull, Lowering: query.CoverageFull},
			Relational: query.RelationalIR{
				Nodes: []query.RelNode{
					{ID: "src", Kind: query.RelSource, Source: "inventory", SourceKind: "table"},
					{ID: "projection", Kind: query.RelProject, Inputs: []string{"src"}, Expressions: []string{"product_id", "stock", "reserved"}},
				},
				Root: "projection",
			},
		},
	}
	record := func(id, stock, reserved string) query.DerivedRow {
		return query.DerivedRow{Values: map[core.CanonicalID]derivationref.Input{
			"product.id":       {State: derivationref.Present, Decimal: id},
			"product.stock":    {State: derivationref.Present, Decimal: stock},
			"product.reserved": {State: derivationref.Present, Decimal: reserved},
		}}
	}
	return q, p, e, []query.DerivedRow{
		record("A", "120", "100"), record("B", "100", "10"),
		record("C", "80", "30"), record("D", "70", "20"), record("E", "40", "70"),
	}
}

func d2IDs(rows []query.DerivedRow) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.Values["product.id"].Decimal)
	}
	return ids
}

func TestD2DerivedFilterSortCountAndPage(t *testing.T) {
	q, p, evidence, rows := d2Fixture()
	if err := query.ValidateDerivedSelect(q, p, evidence, func(core.CanonicalID) bool { return true }); err != nil {
		t.Fatal(err)
	}
	full, err := query.EvaluateDerivedSelect(p, rows, true)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d2IDs(full.Rows), []string{"B", "C", "D", "A", "E"}) || full.Count != 5 {
		t.Fatalf("order/count=%v/%d", d2IDs(full.Rows), full.Count)
	}
	p.Where = &query.DerivedWhere{Op: ">=", Decimal: "50"}
	filtered, err := query.EvaluateDerivedSelect(p, rows, true)
	if err != nil || filtered.Count != 3 || !reflect.DeepEqual(d2IDs(filtered.Rows), []string{"B", "C", "D"}) {
		t.Fatalf("filtered=%v count=%d err=%v", d2IDs(filtered.Rows), filtered.Count, err)
	}
	p.HasLimit, p.Limit = true, 1
	first, err := query.EvaluateDerivedSelect(p, rows, true)
	if err != nil || !reflect.DeepEqual(d2IDs(first.Rows), []string{"B"}) || first.Count != 3 {
		t.Fatalf("first=%v count=%d err=%v", d2IDs(first.Rows), first.Count, err)
	}
	p.Offset, p.Limit = 1, 2
	page, err := query.EvaluateDerivedSelect(p, rows, true)
	if err != nil || !reflect.DeepEqual(d2IDs(page.Rows), []string{"C", "D"}) || page.Count != 3 {
		t.Fatalf("second page=%v count=%d err=%v", d2IDs(page.Rows), page.Count, err)
	}
}

func TestD2PreflightRejectsWrongPhysicalOrderAndUnauthorizedFields(t *testing.T) {
	q, p, evidence, _ := d2Fixture()
	allow := func(core.CanonicalID) bool { return true }
	tests := []struct {
		name   string
		mutate func(*query.EmbeddedSQL, *query.DerivedSelect, *query.CompleteSourceEvidence)
	}{
		{"pre-limit", func(_ *query.EmbeddedSQL, _ *query.DerivedSelect, e *query.CompleteSourceEvidence) {
			e.IR.Relational.Nodes = append(e.IR.Relational.Nodes, query.RelNode{ID: "limited", Kind: query.RelLimit, Inputs: []string{"projection"}})
			e.IR.Relational.Root = "limited"
		}},
		{"pre-filter", func(_ *query.EmbeddedSQL, _ *query.DerivedSelect, e *query.CompleteSourceEvidence) {
			e.IR.Relational.Nodes = append(e.IR.Relational.Nodes, query.RelNode{ID: "filtered", Kind: query.RelFilter})
		}},
		{"pre-aggregate", func(_ *query.EmbeddedSQL, _ *query.DerivedSelect, e *query.CompleteSourceEvidence) {
			e.IR.Relational.Nodes = append(e.IR.Relational.Nodes, query.RelNode{ID: "aggregate", Kind: query.RelAggregate})
		}},
		{"join duplicates", func(_ *query.EmbeddedSQL, _ *query.DerivedSelect, e *query.CompleteSourceEvidence) {
			e.IR.Relational.Nodes = append(e.IR.Relational.Nodes, query.RelNode{ID: "join", Kind: query.RelJoin})
		}},
		{"stale SQL", func(q *query.EmbeddedSQL, _ *query.DerivedSelect, _ *query.CompleteSourceEvidence) {
			q.Source += " LIMIT 1"
		}},
		{"partial lowering", func(_ *query.EmbeddedSQL, _ *query.DerivedSelect, e *query.CompleteSourceEvidence) {
			e.IR.Coverage.Lowering = query.CoveragePartial
		}},
		{"unsupported SQL expression", func(_ *query.EmbeddedSQL, _ *query.DerivedSelect, e *query.CompleteSourceEvidence) {
			e.IR.Relational.Nodes[1].Expressions = []string{"SUM(stock)"}
		}},
		{"unbounded", func(_ *query.EmbeddedSQL, p *query.DerivedSelect, _ *query.CompleteSourceEvidence) {
			p.MaxRows = 0
		}},
		{"unknown input", func(_ *query.EmbeddedSQL, p *query.DerivedSelect, _ *query.CompleteSourceEvidence) {
			p.Inputs["stock"] = "secret.stock"
		}},
		{"missing stable order", func(_ *query.EmbeddedSQL, p *query.DerivedSelect, _ *query.CompleteSourceEvidence) {
			p.Order = nil
			p.HasLimit = true
			p.Limit = 1
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q2, p2, e2, _ := d2Fixture()
			tc.mutate(&q2, &p2, &e2)
			if err := query.ValidateDerivedSelect(q2, p2, e2, allow); !errors.Is(err, query.ErrDerivedUnsupported) {
				t.Fatalf("got %v, expected unsupported", err)
			}
		})
	}
	if err := query.ValidateDerivedSelect(q, p, evidence, nil); !errors.Is(err, query.ErrDerivedUnauthorized) {
		t.Fatalf("nil authority: %v", err)
	}
	if err := query.ValidateDerivedSelect(q, p, evidence, func(id core.CanonicalID) bool {
		return id != "product.stock"
	}); !errors.Is(err, query.ErrDerivedUnauthorized) {
		t.Fatalf("protected raw attribute: %v", err)
	}
	if err := query.ValidateDerivedSelect(q, p, evidence, func(id core.CanonicalID) bool {
		return id != "product.available"
	}); !errors.Is(err, query.ErrDerivedUnauthorized) {
		t.Fatalf("protected derived target: %v", err)
	}
}

func TestD2RejectsIncompleteUnstableAndBadCardinality(t *testing.T) {
	_, p, _, rows := d2Fixture()
	p.HasLimit, p.Limit = true, 1
	if _, err := query.EvaluateDerivedSelect(p, rows[:1], false); !errors.Is(err, query.ErrDerivedIncomplete) {
		t.Fatalf("pre-limited source was accepted: %v", err)
	}
	p.MaxRows = 2
	if _, err := query.EvaluateDerivedSelect(p, rows, true); !errors.Is(err, query.ErrDerivedIncomplete) {
		t.Fatalf("source beyond bound was accepted: %v", err)
	}
	p.MaxRows = 100
	p.Order = p.Order[:1] // C and D tie; no implicit source row order as key
	if _, err := query.EvaluateDerivedSelect(p, rows, true); !errors.Is(err, query.ErrDerivedUnsupported) {
		t.Fatalf("unstable page accepted: %v", err)
	}
	p.Order = append(p.Order, query.DerivedOrder{Attribute: "product.id", Kind: "text"})
	for _, state := range []derivationref.InputState{derivationref.Missing, derivationref.AbsentRelation, derivationref.MultipleRelations} {
		_, _, _, fresh := d2Fixture()
		fresh[0].Values["product.stock"] = derivationref.Input{State: state}
		if _, err := query.EvaluateDerivedSelect(p, fresh, true); !errors.Is(err, query.ErrDerivedUnsupported) {
			t.Fatalf("state %s unexpectedly accepted: %v", state, err)
		}
	}
}

func TestD2NullEmptyAndExactDecimals(t *testing.T) {
	_, p, _, rows := d2Fixture()
	p.Where = &query.DerivedWhere{Op: ">=", Decimal: "50"}
	rows[0].Values["product.stock"] = derivationref.Input{State: derivationref.Null}
	got, err := query.EvaluateDerivedSelect(p, rows, true)
	if err != nil || got.Count != 3 {
		t.Fatalf("NULL should be UNKNOWN and filtered: %#v %v", got, err)
	}
	empty, err := query.EvaluateDerivedSelect(p, nil, true)
	if err != nil || empty.Count != 0 || len(empty.Rows) != 0 {
		t.Fatalf("empty set: %#v %v", empty, err)
	}
	_, p, _, rows = d2Fixture()
	rows[0].Values["product.stock"] = derivationref.Input{State: derivationref.Present, Decimal: "1.10"}
	rows[0].Values["product.reserved"] = derivationref.Input{State: derivationref.Present, Decimal: "0.25"}
	got, err = query.EvaluateDerivedSelect(p, rows[:1], true)
	if err != nil || got.Rows[0].Values["product.available"].Decimal != "0.85" {
		t.Fatalf("exact decimal subtract: %#v %v", got, err)
	}
	p.Rule.Operator = "multiply"
	got, err = query.EvaluateDerivedSelect(p, rows[:1], true)
	if err != nil || got.Rows[0].Values["product.available"].Decimal != "0.275" {
		t.Fatalf("D1 exact multiply reused: %#v %v", got, err)
	}
}

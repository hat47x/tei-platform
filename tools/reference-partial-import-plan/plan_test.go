package partialimport

import (
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func w(types []semantic.Type, data ...semantic.Data) ir.World {
	return ir.World{Semantics: []semantic.Namespace{{ID: "s", Types: types, Data: data}}}
}

var typ = []semantic.Type{{ID: "t.code", Symbol: "Code", Kind: semantic.TypeAlias}}

func order(attrType string) semantic.Data {
	return semantic.Data{ID: "s.order", Symbol: "Order", Attributes: []semantic.Attribute{
		{ID: "s.order.no", Symbol: "no", TypeRef: core.CanonicalID(attrType)},
	}}
}

func codes(d []Diagnostic) map[string]bool {
	m := map[string]bool{}
	for _, x := range d {
		m[x.Code] = true
	}
	return m
}

func TestNewIdenticalAndUpdate(t *testing.T) {
	src := w(typ, order("t.code"))
	p, _ := Analyze(src, w(typ), []string{"s.order"})
	if p.Items[0].Status != New || !p.Importable || p.Applied {
		t.Fatalf("plan = %+v", p)
	}
	p, _ = Analyze(src, w(typ, order("t.code")), []string{"s.order"})
	if p.Items[0].Status != Identical {
		t.Fatalf("plan = %+v", p)
	}
	other := []semantic.Type{{ID: "t.code", Symbol: "Code"}, {ID: "t.id", Symbol: "Id"}}
	p, _ = Analyze(w(other, order("t.id")), w(other, order("t.code")), []string{"s.order"})
	if p.Items[0].Status != Update || p.Items[0].Changes[0] != "attribute-type-changed:s.order.no" {
		t.Fatalf("plan = %+v", p)
	}
}

func TestSameIDIsNotSameMeaningWhenNamesClash(t *testing.T) {
	src := w(typ, semantic.Data{ID: "s.client", Symbol: "Customer"})
	tgt := w(typ, semantic.Data{ID: "s.customer", Symbol: "Customer"})
	p, _ := Analyze(src, tgt, []string{"s.client"})
	if p.Items[0].Status != Conflict || p.Importable || !codes(p.Diagnostics)["symbol-used-by-other-id"] {
		t.Fatalf("plan = %+v", p)
	}
}

func TestMissingDependenciesStayUnresolved(t *testing.T) {
	src := w(typ, semantic.Data{ID: "s.order", Symbol: "Order",
		Attributes: []semantic.Attribute{{ID: "s.order.no", Symbol: "no", TypeRef: "t.code"}},
		Relations:  []semantic.Relation{{ID: "s.order.cust", Target: "s.customer"}}})
	p, _ := Analyze(src, w(nil), []string{"s.order"})
	c := codes(p.Unresolved)
	if p.Importable || !c["type-only-in-source"] || !c["relation-target-missing"] {
		t.Fatalf("plan = %+v", p)
	}
	// Selecting the target of the relation together resolves that dependency.
	src.Semantics[0].Data = append(src.Semantics[0].Data, semantic.Data{ID: "s.customer", Symbol: "Customer"})
	p, _ = Analyze(src, w(typ), []string{"s.order", "s.customer"})
	if !p.Importable || len(p.Unresolved) != 0 {
		t.Fatalf("plan = %+v", p)
	}
}

func TestUpdateThatBreaksTargetDependent(t *testing.T) {
	cust := semantic.Data{ID: "s.customer", Symbol: "Customer", Attributes: []semantic.Attribute{
		{ID: "s.customer.code", Symbol: "code", TypeRef: "t.code"}}}
	newCust := semantic.Data{ID: "s.customer", Symbol: "Customer"} // attribute removed
	line := semantic.Data{ID: "s.line", Symbol: "Line", Attributes: []semantic.Attribute{
		{ID: "s.line.cust", Symbol: "cust", TypeRef: "t.code"}},
		Relations: []semantic.Relation{{ID: "s.line.to_cust", Target: "s.customer",
			Keys: []semantic.RelationKey{{SourceAttribute: "s.line.cust", TargetAttribute: "s.customer.code"}}}}}
	p, _ := Analyze(w(typ, newCust), w(typ, cust, line), []string{"s.customer"})
	if p.Importable || !codes(p.Diagnostics)["dependent-definition-affected"] {
		t.Fatalf("plan = %+v", p)
	}
}

func TestNotInSourceAndEmptySelection(t *testing.T) {
	p, _ := Analyze(w(typ), w(typ), []string{"s.nothing"})
	if p.Importable || !codes(p.Diagnostics)["not-in-source"] {
		t.Fatalf("plan = %+v", p)
	}
	if _, err := Analyze(w(typ), w(typ), nil); err == nil {
		t.Fatal("empty selection accepted")
	}
}

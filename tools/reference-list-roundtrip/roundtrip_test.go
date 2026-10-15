package listroundtrip

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

func world() ir.World {
	return ir.World{Semantics: []semantic.Namespace{{ID: "s", Data: []semantic.Data{
		{ID: "s.order", Symbol: "Order", Attributes: []semantic.Attribute{
			{ID: "s.order.no", Symbol: "no"}, {ID: "s.order.cust", Symbol: "cust"},
		}},
		{ID: "s.invoice", Symbol: "Invoice", Attributes: []semantic.Attribute{
			{ID: "s.invoice.no", Symbol: "no"},
		}},
	}}}}
}

func codes(p Proposal) []string {
	var out []string
	for _, d := range p.Diagnostics {
		out = append(out, d.Code)
	}
	return out
}

func list(t *testing.T, rows ...string) string {
	t.Helper()
	rev, _ := Revision(world())
	var b strings.Builder
	b.WriteString("kind,id,parent,symbol,new_symbol,base_revision\n")
	for _, r := range rows {
		b.WriteString(strings.ReplaceAll(r, "REV", rev) + "\n")
	}
	return b.String()
}

func TestExportThenProposeWithoutEditsIsEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := Export(&buf, world()); err != nil {
		t.Fatal(err)
	}
	p, err := Propose(world(), &buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Edits) != 0 || len(p.Diagnostics) != 0 || p.Applied {
		t.Fatalf("proposal = %+v", p)
	}
}

func TestEditByID(t *testing.T) {
	p, _ := Propose(world(), strings.NewReader(list(t, "attribute,s.order.no,s.order,no,orderNo,REV")))
	if len(p.Edits) != 1 || p.Edits[0].ID != "s.order.no" || p.Edits[0].Resolved != "id" {
		t.Fatalf("proposal = %+v", p)
	}
}

func TestAmbiguousNameIsRefused(t *testing.T) {
	// "no" is an attribute of both Order and Invoice.
	p, _ := Propose(world(), strings.NewReader(list(t, "attribute,,,no,number,")))
	if len(p.Edits) != 0 || len(codes(p)) != 1 || codes(p)[0] != "ambiguous-name" {
		t.Fatalf("proposal = %+v", p)
	}
}

func TestUniqueNameResolves(t *testing.T) {
	p, _ := Propose(world(), strings.NewReader(list(t, "attribute,,,cust,customer,")))
	if len(p.Edits) != 1 || p.Edits[0].ID != "s.order.cust" || p.Edits[0].Resolved != "name" {
		t.Fatalf("proposal = %+v", p)
	}
}

func TestRenamedInSourceIsNotRetargeted(t *testing.T) {
	// The list claims the ID still has the old name.
	p, _ := Propose(world(), strings.NewReader(list(t, "attribute,s.order.cust,s.order,customerCode,customer,")))
	if len(p.Edits) != 0 || codes(p)[0] != "symbol-differs-from-source" {
		t.Fatalf("proposal = %+v", p)
	}
}

func TestStaleListAndUnknownID(t *testing.T) {
	p, _ := Propose(world(), strings.NewReader(list(t,
		"attribute,s.order.no,s.order,no,x,sha256:old",
		"attribute,s.missing,s.order,no,x,")))
	got := codes(p)
	if len(p.Edits) != 0 || got[0] != "stale-list" || got[1] != "unknown-id" {
		t.Fatalf("proposal = %+v", p)
	}
}

func TestDuplicateTargetAndCollision(t *testing.T) {
	p, _ := Propose(world(), strings.NewReader(list(t,
		"attribute,s.order.no,s.order,no,a,",
		"attribute,s.order.no,s.order,no,b,",
		"attribute,s.order.cust,s.order,cust,no,",
	)))
	got := strings.Join(codes(p), ",")
	if !strings.Contains(got, "duplicate-target") {
		t.Fatalf("codes = %s", got)
	}
	// cust -> "no" is fine only if the first "no" was renamed; line 1 renamed it.
	for _, d := range p.Diagnostics {
		if d.Code == "symbol-collision" {
			t.Fatalf("unexpected collision %+v", d)
		}
	}
	p2, _ := Propose(world(), strings.NewReader(list(t, "attribute,s.order.cust,s.order,cust,no,")))
	if len(codes(p2)) != 1 || codes(p2)[0] != "symbol-collision" {
		t.Fatalf("proposal = %+v", p2)
	}
}

func TestMissingColumnAndEmpty(t *testing.T) {
	if _, err := Propose(world(), strings.NewReader("kind,symbol\nx,y\n")); err == nil {
		t.Fatal("missing column accepted")
	}
	if _, err := Propose(world(), strings.NewReader("")); err == nil {
		t.Fatal("empty list accepted")
	}
}

package partialimport

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

func boundarySource() ir.World {
	return ir.World{Semantics: []semantic.Namespace{
		{ID: "own", Types: typ, Data: []semantic.Data{
			{ID: "own.order", Symbol: "Order"},
		}},
		{ID: "shared", Types: typ, Data: []semantic.Data{
			{ID: "shared.customer", Symbol: "Customer"},
		}},
	}}
}

func boundaryTarget() ir.World {
	return ir.World{Semantics: []semantic.Namespace{
		{ID: "own", Types: typ},
	}}
}

func TestReferenceOnlyNamespaceIsNotImported(t *testing.T) {
	p, err := AnalyzeWithOptions(boundarySource(), boundaryTarget(),
		[]string{"own.order", "shared.customer"},
		Options{ReferenceOnlyNamespaces: []string{"shared"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Items) != 1 || p.Items[0].ID != "own.order" || p.Items[0].Status != New {
		t.Fatalf("items = %+v", p.Items)
	}
	if len(p.ReferenceOnly) != 1 || p.ReferenceOnly[0].Subject != "shared.customer" ||
		p.ReferenceOnly[0].Code != "reference-only-namespace" {
		t.Fatalf("referenceOnly = %+v", p.ReferenceOnly)
	}
	if p.Importable {
		t.Error("a reference-only namespace must not make the plan importable")
	}
}

func TestReferenceOnlyNamespaceNeverProducesAnItem(t *testing.T) {
	source := boundarySource()
	target := ir.World{Semantics: []semantic.Namespace{
		{ID: "own", Types: typ},
		{ID: "shared", Types: typ, Data: []semantic.Data{{ID: "shared.customer", Symbol: "Customer"}}},
	}}
	p, err := AnalyzeWithOptions(source, target, []string{"shared.customer"},
		Options{ReferenceOnlyNamespaces: []string{"shared"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Items) != 0 {
		t.Fatalf("reference-only selection produced items: %+v", p.Items)
	}
	if len(p.ReferenceOnly) != 1 {
		t.Fatalf("referenceOnly = %+v", p.ReferenceOnly)
	}
}

func TestReferenceOnlyOtherNamespaceDoesNotBlock(t *testing.T) {
	p, err := AnalyzeWithOptions(boundarySource(), boundaryTarget(),
		[]string{"own.order"},
		Options{ReferenceOnlyNamespaces: []string{"shared"}})
	if err != nil {
		t.Fatal(err)
	}
	if !p.Importable || len(p.ReferenceOnly) != 0 {
		t.Fatalf("plan = %+v", p)
	}
}

func TestDifferentOwnershipBoundaryIsDiagnosed(t *testing.T) {
	p, err := AnalyzeWithOptions(boundarySource(), boundaryTarget(),
		[]string{"own.order"},
		Options{SourceOwner: "team-a", TargetOwner: "team-b"})
	if err != nil {
		t.Fatal(err)
	}
	if !codes(p.Diagnostics)["different-ownership-boundary"] {
		t.Fatalf("diagnostics = %+v", p.Diagnostics)
	}
	if p.Importable {
		t.Error("a different ownership boundary must not be importable")
	}
}

func TestSameOwnershipBoundaryIsAllowed(t *testing.T) {
	p, err := AnalyzeWithOptions(boundarySource(), boundaryTarget(),
		[]string{"own.order"},
		Options{SourceOwner: "team-a", TargetOwner: "team-a"})
	if err != nil {
		t.Fatal(err)
	}
	if codes(p.Diagnostics)["different-ownership-boundary"] {
		t.Fatalf("diagnostics = %+v", p.Diagnostics)
	}
	if !p.Importable {
		t.Fatalf("plan = %+v", p)
	}
}

func TestMissingReferenceOutsideSelectionStaysUnresolved(t *testing.T) {
	source := boundarySource()
	source.Semantics[0].Data[0].Relations = []semantic.Relation{
		{ID: "own.order.customer", Target: "shared.customer"},
	}
	p, err := AnalyzeWithOptions(source, boundaryTarget(), []string{"own.order"},
		Options{ReferenceOnlyNamespaces: []string{"shared"}})
	if err != nil {
		t.Fatal(err)
	}
	if !codes(p.Unresolved)["relation-target-missing"] {
		t.Fatalf("unresolved = %+v", p.Unresolved)
	}
	if len(p.ReferenceOnly) != 0 {
		t.Fatalf("referenceOnly = %+v", p.ReferenceOnly)
	}
	if p.Importable {
		t.Error("a missing relation target must block the plan")
	}
}

package plan_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/plan"
	"tei/internal/query"
)

func TestLogicalMappingIsConservativeAndAllowsFusedFilter(t *testing.T) {
	q := query.DerivedIR{
		Query:   core.MustCanonicalID("app.q"),
		Sources: []query.SourceRef{{ID: "source:0", Name: "public.orders", SourceKind: "table", Alias: "o", Resolution: "schema-resolved"}},
		Relational: query.RelationalIR{Nodes: []query.RelNode{
			{ID: "source:0", Kind: query.RelSource, Source: "public.orders", SourceKind: "table", Alias: "o"},
			{ID: "filter:0", Kind: query.RelFilter, Inputs: []string{"source:0"}},
			{ID: "project:0", Kind: query.RelProject, Inputs: []string{"filter:0"}},
			{ID: "sort:0", Kind: query.RelSort, Inputs: []string{"project:0"}},
		}},
	}
	p := plan.IR{Nodes: []plan.Node{
		{ID: "plan:0", Kind: plan.NodeSort, ProviderKind: "Sort", Children: []string{"plan:0.0"}},
		{ID: "plan:0.0", Kind: plan.NodeScan, ProviderKind: "Seq Scan", PhysicalObject: "public.orders", Alias: "o", ProviderDetails: map[string]any{"Filter": "x > 0"}},
	}}
	got := plan.MapLogical(q, p)
	if len(got) != 3 {
		t.Fatalf("mappings=%#v", got)
	}
	byRel := map[string]plan.LogicalMapping{}
	for _, m := range got {
		byRel[m.RelNodeID] = m
	}
	if byRel["source:0"].Confidence != plan.MappingExact || byRel["source:0"].PlanNodeID != "plan:0.0" {
		t.Fatalf("source=%#v", byRel["source:0"])
	}
	if byRel["filter:0"].Confidence != plan.MappingInferred || byRel["filter:0"].PlanNodeID != "plan:0.0" {
		t.Fatalf("filter=%#v", byRel["filter:0"])
	}
	if byRel["sort:0"].PlanNodeID != "plan:0" {
		t.Fatalf("sort=%#v", byRel["sort:0"])
	}
	if _, ok := byRel["project:0"]; ok {
		t.Fatal("project must stay unmapped when provider has no physical projection node")
	}
}

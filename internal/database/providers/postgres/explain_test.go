package postgres_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/database/plan"
	"tei/internal/database/provider"
	"tei/internal/database/providers/postgres"
	"tei/internal/environment"
	"tei/internal/query"
)

const analyzedExplain = `[
  {"Plan": {
    "Node Type":"Sort","Startup Cost":10,"Total Cost":12,"Plan Rows":10,"Plan Width":32,
    "Actual Startup Time":0.03,"Actual Total Time":0.04,"Actual Rows":8,"Actual Loops":1,
    "Sort Key":["ordered_at DESC"],
    "Plans":[{"Node Type":"Seq Scan","Parent Relationship":"Outer","Schema":"public","Relation Name":"orders","Alias":"o","Startup Cost":0,"Total Cost":9,"Plan Rows":10,"Plan Width":32,"Actual Startup Time":0.01,"Actual Total Time":0.02,"Actual Rows":8,"Actual Loops":1,"Filter":"(ordered_at >= $1)"}]
  },"Planning Time":0.2,"Execution Time":0.07}
]`

func TestNormalizeExplainJSONSeparatesEstimateAndActual(t *testing.T) {
	snap := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{{Schema: "public", Name: "orders"}}}
	got, err := postgres.NormalizeExplainJSON(core.MustCanonicalID("order_entry.recent_orders"), snap.Database, []byte(analyzedExplain), &snap)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != plan.ModeExplainAnalyze || got.Root != "plan:0" || len(got.Nodes) != 2 {
		t.Fatalf("plan=%#v", got)
	}
	scan, ok := got.Node("plan:0.0")
	if !ok || scan.PhysicalObject != "public.orders" || scan.Kind != plan.NodeScan {
		t.Fatalf("scan=%#v", scan)
	}
	if scan.Actual == nil || scan.Actual.Rows == nil || *scan.Actual.Rows != 8 || scan.Estimate.Rows == nil || *scan.Estimate.Rows != 10 {
		t.Fatalf("estimate/actual=%#v", scan)
	}
	ratio, ok := plan.RowEstimateRatio(scan)
	if !ok || ratio != 0.8 {
		t.Fatalf("ratio=%v ok=%v", ratio, ok)
	}
	if scan.ProviderDetails["Filter"] == nil {
		t.Fatalf("provider details lost: %#v", scan.ProviderDetails)
	}
}

func TestNormalizePlainExplainDoesNotInventActuals(t *testing.T) {
	raw := []byte(`[{"Plan":{"Node Type":"Seq Scan","Relation Name":"orders","Startup Cost":0,"Total Cost":10,"Plan Rows":5,"Plan Width":8}}]`)
	got, err := postgres.NormalizeExplainJSON(core.MustCanonicalID("app.q"), core.MustCanonicalID("sales_core"), raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != plan.ModeExplain || len(got.Nodes) != 1 || got.Nodes[0].Actual != nil || got.ExecutionTimeMs != nil {
		t.Fatalf("plan=%#v", got)
	}
}

type fakeExplainReader struct{ req postgres.ExplainRequest }

func (f *fakeExplainReader) ExplainJSON(_ context.Context, req postgres.ExplainRequest) ([]byte, error) {
	f.req = req
	return []byte(analyzedExplain), nil
}

func TestProviderExplainUsesOptionalReaderAndCapabilities(t *testing.T) {
	reader := &fakeExplainReader{}
	p := postgres.Provider{Catalog: fakeCatalog{}, PlanReader: reader}
	inst := environment.DatabaseInstance{ID: core.MustCanonicalID("sales_core.dev"), Definition: core.MustCanonicalID("sales_core"), Provider: environment.ProviderPostgres}
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("order_entry.recent_orders"), Source: "SELECT * FROM orders"}
	got, err := provider.Explain(context.Background(), p, inst, q, nil, provider.ExplainOptions{Analyze: true, Buffers: true})
	if err != nil {
		t.Fatal(err)
	}
	if !reader.req.Analyze || !reader.req.Buffers || reader.req.SQL != q.Source || got.Mode != plan.ModeExplainAnalyze {
		t.Fatalf("request=%#v plan=%#v", reader.req, got)
	}
}

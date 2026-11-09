package driver_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/semantic"
	"tei/internal/semantic/derivationref"
)

type d2Executor func(context.Context, query.EmbeddedSQL, application.DatabaseRequirement, map[string]any, int) (driver.BrowseQueryResult, bool, error)

func (e d2Executor) ExecuteCompleteBrowseQuery(ctx context.Context, q query.EmbeddedSQL, d application.DatabaseRequirement, p map[string]any, n int) (driver.BrowseQueryResult, bool, error) {
	return e(ctx, q, d, p, n)
}

type d2Authority func(context.Context, core.CanonicalID) bool

func (f d2Authority) CanReadAttribute(ctx context.Context, id core.CanonicalID) bool {
	return f(ctx, id)
}

func d2DriverFixture() (driver.Resolver, query.EmbeddedSQL, query.DerivedSelect, query.CompleteSourceEvidence) {
	q := query.EmbeddedSQL{
		ID: "inventory.browse", Application: "inventory", Datasource: "store",
		Source: "SELECT product_id, stock, reserved FROM inventory",
		ResultColumns: []query.ResultColumnMapping{
			{Column: "product_id", Attribute: "inventory.product.id"},
			{Column: "stock", Attribute: "inventory.product.stock"},
			{Column: "reserved", Attribute: "inventory.product.reserved"},
		},
	}
	world := ir.World{
		Applications: []application.Application{{
			ID:        "inventory",
			Databases: []application.DatabaseRequirement{{Alias: "store", TargetDatabase: "inventory_db", Access: database.AccessRead}},
			Patterns:  []application.Pattern{{ID: "inventory.browse", Kind: application.PatternBrowse, Data: "inventory.product", Query: q.ID}},
		}},
		Queries:   []query.EmbeddedSQL{q},
		Semantics: []semantic.Namespace{{ID: "inventory", Data: []semantic.Data{{ID: "inventory.product"}}}},
	}
	p := query.DerivedSelect{
		Rule:    derivationref.Definition{ID: "available", Pure: true, Operator: "subtract", Inputs: []string{"stock", "reserved"}},
		Binding: derivationref.Binding{RuleID: "available", Scope: "query", TargetID: "inventory.product.available"},
		Inputs:  map[string]core.CanonicalID{"stock": "inventory.product.stock", "reserved": "inventory.product.reserved"},
		Where:   &query.DerivedWhere{Op: ">=", Decimal: "50"},
		Order:   []query.DerivedOrder{{Attribute: "inventory.product.available", Kind: "decimal", Descending: true}, {Attribute: "inventory.product.id", Kind: "text"}},
		MaxRows: 20, HasLimit: true, Limit: 2, Offset: 1,
	}
	proof := query.CompleteSourceEvidence{
		SQLSHA256: query.DigestNativeSQL(q.Source),
		IR: query.DerivedIR{
			Query: q.ID, Coverage: query.AnalysisCoverage{Parse: query.CoverageFull, Lowering: query.CoverageFull},
			Relational: query.RelationalIR{Root: "projection", Nodes: []query.RelNode{
				{ID: "src", Kind: query.RelSource, Source: "inventory", SourceKind: "table"},
				{ID: "projection", Kind: query.RelProject, Inputs: []string{"src"}, Expressions: []string{"product_id", "stock", "reserved"}},
			}},
		},
	}
	return driver.NewResolver(world), q, p, proof
}

func TestD2DriverFullMappedBrowseAndSafeWindow(t *testing.T) {
	r, q, p, proof := d2DriverFixture()
	sourceRows := [][]any{
		{"A", "120", "100"}, {"B", "100", "10"}, {"C", "80", "30"}, {"D", "70", "20"}, {"E", "40", "70"},
	}
	invocations := 0
	executor := d2Executor(func(_ context.Context, received query.EmbeddedSQL, ds application.DatabaseRequirement, _ map[string]any, bound int) (driver.BrowseQueryResult, bool, error) {
		invocations++
		if received.Source != q.Source || received.ID != q.ID || ds.Alias != "store" || bound != 20 {
			t.Fatalf("query or datasource was rewritten: %#v %#v bound=%d", received, ds, bound)
		}
		return driver.BrowseQueryResult{Raw: queryresult.RawResult{
			Columns: []string{"reserved", "product_id", "stock"},
			Rows: func() [][]any {
				var v [][]any
				for _, row := range sourceRows {
					v = append(v, []any{row[2], row[0], row[1]})
				}
				return v
			}(),
		}}, true, nil
	})
	allowed := d2Authority(func(context.Context, core.CanonicalID) bool { return true })
	got, err := r.ExecuteDerivedBrowseQuery(context.Background(), "inventory", "inventory.browse", executor, nil, p, proof, allowed)
	if err != nil {
		t.Fatal(err)
	}
	if invocations != 1 || got.Derived.Count != 3 || len(got.Derived.Rows) != 2 {
		t.Fatalf("called=%d result=%#v", invocations, got.Derived)
	}
	ids := []string{}
	for _, row := range got.Derived.Rows {
		ids = append(ids, row.Values["inventory.product.id"].Decimal)
	}
	if !reflect.DeepEqual(ids, []string{"C", "D"}) {
		t.Fatalf("wrong post-derive paging: %v", ids)
	}
	if got.Browse.Result.Mapped.Columns[0].Raw != "product_id" || len(got.Browse.Result.Raw.Rows) != 5 {
		t.Fatal("raw source or explicit mapping was changed")
	}

	// The Host must deny protected raw or derived attributes before dispatch.
	for _, denied := range []core.CanonicalID{"inventory.product.stock", "inventory.product.available"} {
		before := invocations
		authority := d2Authority(func(_ context.Context, id core.CanonicalID) bool { return id != denied })
		if _, err := r.ExecuteDerivedBrowseQuery(context.Background(), "inventory", "inventory.browse", executor, nil, p, proof, authority); !errors.Is(err, query.ErrDerivedUnauthorized) {
			t.Fatalf("denied %s: %v", denied, err)
		}
		if invocations != before {
			t.Fatal("unauthorized query was dispatched")
		}
	}
}

func TestD2DriverRejectsIncompleteFloatsAndUnmappedRaw(t *testing.T) {
	r, _, p, proof := d2DriverFixture()
	allowed := d2Authority(func(context.Context, core.CanonicalID) bool { return true })
	makeExecutor := func(rows [][]any, complete bool) d2Executor {
		return func(context.Context, query.EmbeddedSQL, application.DatabaseRequirement, map[string]any, int) (driver.BrowseQueryResult, bool, error) {
			return driver.BrowseQueryResult{Raw: queryresult.RawResult{
				Columns: []string{"product_id", "stock", "reserved"}, Rows: rows,
			}}, complete, nil
		}
	}
	for _, tc := range []struct {
		name     string
		rows     [][]any
		complete bool
		want     error
	}{
		{"early limit", [][]any{{"A", "120", "100"}}, false, query.ErrDerivedIncomplete},
		{"float precision", [][]any{{"A", 120.0, 100.0}}, true, query.ErrDerivedUnsupported},
		{"missing raw column", [][]any{{"A", "120"}}, true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := r.ExecuteDerivedBrowseQuery(context.Background(), "inventory", "inventory.browse", makeExecutor(tc.rows, tc.complete), nil, p, proof, allowed)
			if err == nil {
				t.Fatal("invalid result was accepted")
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
	// Source analyzer proof that is missing or stale blocks execution entirely.
	invalid := proof
	invalid.SQLSHA256 = "incorrect"
	called := false
	exec := d2Executor(func(context.Context, query.EmbeddedSQL, application.DatabaseRequirement, map[string]any, int) (driver.BrowseQueryResult, bool, error) {
		called = true
		return driver.BrowseQueryResult{}, true, nil
	})
	if _, err := r.ExecuteDerivedBrowseQuery(context.Background(), "inventory", "inventory.browse", exec, nil, p, invalid, allowed); !errors.Is(err, query.ErrDerivedUnsupported) || called {
		t.Fatalf("preflight bypassed: error=%v called=%v", err, called)
	}
}

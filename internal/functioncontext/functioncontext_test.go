package functioncontext_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/environment"
	"tei/internal/functioncontext"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

type pgLikeExecutor struct{ rows [][]any }

func (pgLikeExecutor) DatabaseProvider() environment.DatabaseProvider {
	return environment.ProviderPostgres
}

func (e pgLikeExecutor) ExecuteBrowseQuery(context.Context, query.EmbeddedSQL, application.DatabaseRequirement, map[string]any) (driver.BrowseQueryResult, error) {
	return driver.BrowseQueryResult{Raw: queryresult.RawResult{Columns: []string{"order_id", "customer_id"}, Rows: e.rows}}, nil
}

func world(t *testing.T) ir.World {
	t.Helper()
	loaded, err := workspace.Load(filepath.Join("..", "..", "examples", "xtea-lowcode"))
	if err != nil {
		t.Fatal(err)
	}
	return loaded.World
}

func rowsOf(n int) [][]any {
	rows := make([][]any, n)
	for i := range rows {
		rows[i] = []any{fmt.Sprintf("ORD-%03d", i+1), "C-1"}
	}
	return rows
}

func derive(t *testing.T, w ir.World, n int, requested map[string]any) functioncontext.Context {
	t.Helper()
	exec, err := driver.NewResolver(w).ExecuteBrowseQueryInEnvironment(context.Background(), "development", "order_entry",
		"order_entry.recent_orders", pgLikeExecutor{rows: rowsOf(n)}, requested)
	if err != nil {
		t.Fatal(err)
	}
	got, err := functioncontext.FromBrowseExecution(exec, requested, []string{"status", "status_operator", "page_size", "page_offset"},
		time.Date(2026, 11, 10, 12, 0, 0, 0, time.FixedZone("JST", 9*3600)))
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func request(offset int64) map[string]any {
	return map[string]any{"status": "OPEN", "status_operator": "ne", "page_offset": offset}
}

func TestEntityGrainDefinitionBindingAndScope(t *testing.T) {
	got := derive(t, world(t), 2, request(0))
	d := got.Definition
	if d.ContextID != "order_entry.recent_orders" || d.Subject != "sales.order" || d.Query != "order_entry.recent_orders" ||
		d.Grain != functioncontext.GrainEntity || d.IdentityKey == "" ||
		!reflect.DeepEqual(d.IdentityFields, []core.CanonicalID{"sales.order.order_id"}) || len(d.RelationPath) != 0 {
		t.Fatalf("definition=%+v", d)
	}
	b := got.Binding
	if b.Environment != "development" || b.Instance != "sales_core.dev" || b.Definition != "sales_core" ||
		b.Provider != environment.ProviderPostgres || b.DatasourceAlias != "sales" {
		t.Fatalf("binding=%+v", b)
	}
	wantScope := functioncontext.Scope{
		Filters:     []functioncontext.FilterScope{{Attribute: "sales.order.status", Operator: "ne", Value: "OPEN"}},
		Sorts:       []functioncontext.SortScope{{Attribute: "sales.order.status", Direction: "asc"}, {Attribute: "sales.order.order_id", Direction: "asc"}},
		Page:        &functioncontext.PageScope{Size: 20, Offset: 0},
		OtherInputs: []string{},
	}
	if !reflect.DeepEqual(got.Scope, wantScope) {
		t.Fatalf("scope=%+v", got.Scope)
	}
	if got.Slice.Rows[0].Identity["sales.order.order_id"] != "ORD-001" || got.Slice.RowCount != 2 {
		t.Fatalf("rows=%+v", got.Slice.Rows)
	}
	if got.Slice.ObservedAt.Location() != time.UTC || got.Slice.SourceRevision != "unknown" {
		t.Fatalf("slice=%+v", got.Slice)
	}
}

func TestDeclaredPageContractDoesNotProveCoverage(t *testing.T) {
	for _, tc := range []struct {
		name       string
		rows       int
		offset     int64
		coverage   semantic.Coverage
		state      functioncontext.SliceState
		emptyProof bool
	}{
		{"first page shorter than page-size", 2, 0, semantic.CoverageUnestablished, functioncontext.SliceStatePage, false},
		{"first page exactly page-size", 20, 0, semantic.CoverageUnestablished, functioncontext.SliceStatePage, false},
		{"later page is never complete", 2, 20, semantic.CoverageUnestablished, functioncontext.SliceStatePage, false},
		{"empty first page does not prove subject emptiness", 0, 0, semantic.CoverageUnestablished, functioncontext.SliceStateEmpty, false},
		{"empty later page proves nothing", 0, 20, semantic.CoverageUnestablished, functioncontext.SliceStateEmpty, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := derive(t, world(t), tc.rows, request(tc.offset)).Slice
			if got.Coverage != tc.coverage || got.State != tc.state || got.EmptyJustified != tc.emptyProof ||
				got.CoverageBasis != functioncontext.CoverageBasisDeclaredPageContract {
				t.Fatalf("slice=%+v", got)
			}
		})
	}
}

func TestNoPageDeclarationLeavesCoverageUnestablished(t *testing.T) {
	w := world(t)
	p := &w.Applications[0].Patterns
	for i := range *p {
		if (*p)[i].ID == "order_entry.recent_orders" {
			(*p)[i].PageSize, (*p)[i].PageSizeParameter, (*p)[i].PageOffsetParameter = 0, "", ""
		}
	}
	got := derive(t, w, 0, request(0))
	if got.Slice.Coverage != semantic.CoverageUnestablished || got.Slice.EmptyJustified || got.Scope.Page != nil {
		t.Fatalf("slice=%+v scope=%+v", got.Slice, got.Scope)
	}
}

func TestQueryRowGrainWhenNoCandidateKeyIsCovered(t *testing.T) {
	w := world(t)
	for n := range w.Semantics {
		for i := range w.Semantics[n].Data {
			if w.Semantics[n].Data[i].ID == "sales.order" {
				w.Semantics[n].Data[i].CandidateKeys = nil
			}
		}
	}
	got := derive(t, w, 2, request(0))
	if got.Definition.Grain != functioncontext.GrainQueryRow || got.Slice.Rows[0].Identity != nil || got.Definition.IdentityKey != "" {
		t.Fatalf("definition=%+v rows=%+v", got.Definition, got.Slice.Rows)
	}
}

func TestUndeclaredInputsAreNamedNotValued(t *testing.T) {
	requested := request(0)
	requested["secret_extra"] = "do-not-echo"
	got := derive(t, world(t), 1, requested)
	if !reflect.DeepEqual(got.Scope.OtherInputs, []string{"secret_extra"}) {
		t.Fatalf("otherInputs=%v", got.Scope.OtherInputs)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "do-not-echo") {
		t.Fatalf("undeclared input value leaked: %s", encoded)
	}
}

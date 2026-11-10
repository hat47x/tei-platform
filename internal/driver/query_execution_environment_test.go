package driver_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/validation"
	"tei/internal/workspace"
)

const xteaExampleRoot = "../../examples/xtea-lowcode"

func xteaWorld(t *testing.T) ir.World {
	t.Helper()
	loaded, err := workspace.Load(filepath.FromSlash(xteaExampleRoot))
	if err != nil {
		t.Fatalf("xtea-lowcode Workspaceを読み込めません: %v", err)
	}
	return loaded.World
}

// providerBoundBrowseExecutorは、provider固有のruntime Executorを表すtest double。
// 実DBへは接続しない。実Providerの動作はqueryruntime/postgresのopt-in testが検証する。
type providerBoundBrowseExecutor struct {
	provider environment.DatabaseProvider
	execute  browseQueryExecutorFunc
}

func (e providerBoundBrowseExecutor) ExecuteBrowseQuery(
	ctx context.Context,
	q query.EmbeddedSQL,
	datasource application.DatabaseRequirement,
	parameters map[string]any,
) (driver.BrowseQueryResult, error) {
	return e.execute(ctx, q, datasource, parameters)
}

func (e providerBoundBrowseExecutor) DatabaseProvider() environment.DatabaseProvider {
	return e.provider
}

func TestXTEAExampleRecentOrdersDatasourceIsBoundInEnvironment(t *testing.T) {
	for _, diagnostic := range validation.Validate(xteaWorld(t)) {
		// 未束縛Datasourceの警告自体の検査はvalidation側が所有する。ここでは例に残っていないことだけを見る。
		if strings.HasPrefix(diagnostic.Code, "UNBOUND_") || diagnostic.Severity == validation.Error {
			t.Fatalf("unexpected diagnostic: %s %s", diagnostic.Code, diagnostic.Message)
		}
	}
}

func TestExecuteBrowseQueryInEnvironmentConnectsRecentOrdersThroughDeclaredBinding(t *testing.T) {
	// 独立Expected: SQL原文はtestがfileから直接読み、行はtestが手書きした値と照合する。
	nativeSQL, err := os.ReadFile(filepath.FromSlash(xteaExampleRoot + "/applications/order-entry/queries/recent_orders.sql"))
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	executor := providerBoundBrowseExecutor{
		provider: environment.ProviderPostgres,
		execute: func(_ context.Context, q query.EmbeddedSQL, datasource application.DatabaseRequirement, parameters map[string]any) (driver.BrowseQueryResult, error) {
			calls++
			if q.ID != "order_entry.recent_orders" || q.Source != strings.TrimSpace(string(nativeSQL)) {
				return driver.BrowseQueryResult{}, fmt.Errorf("Query identityまたはSQL原文が正本と一致しません: %s %q", q.ID, q.Source)
			}
			if datasource.Alias != "sales" || datasource.TargetDatabase != "sales_core" {
				return driver.BrowseQueryResult{}, fmt.Errorf("Datasource=%#v", datasource)
			}
			if !reflect.DeepEqual(parameters, map[string]any{"status": "OPEN", "status_operator": "eq", "page_offset": int64(0)}) {
				return driver.BrowseQueryResult{}, fmt.Errorf("parameters=%#v", parameters)
			}
			// 物理列順をQueryのresult宣言と逆にし、明示mappingだけで対応付くことを確かめる。
			return driver.BrowseQueryResult{Raw: queryresult.RawResult{
				Columns: []string{"customer_id", "order_id"},
				Rows:    [][]any{{"C-7", "ORD-1"}, {"C-9", "ORD-2"}},
			}}, nil
		},
	}

	got, err := driver.NewResolver(xteaWorld(t)).ExecuteBrowseQueryInEnvironment(
		context.Background(), "development", "order_entry", "order_entry.recent_orders",
		executor, map[string]any{"status": "OPEN", "status_operator": "eq", "page_offset": int64(0)},
	)
	if err != nil {
		t.Fatalf("Environment経由でbrowse Queryを実行できません: %v", err)
	}
	if calls != 1 {
		t.Fatalf("Executor呼出回数=%d", calls)
	}
	wantInstance := environment.DatabaseInstance{ID: "sales_core.dev", Definition: "sales_core", Provider: environment.ProviderPostgres}
	if got.Environment != "development" || got.Instance != wantInstance {
		t.Fatalf("選択されたEnvironment / instance=%s %+v", got.Environment, got.Instance)
	}
	attributes := make([]core.CanonicalID, len(got.Execution.Result.Mapped.Columns))
	for i, column := range got.Execution.Result.Mapped.Columns {
		attributes[i] = column.Attribute
	}
	if !reflect.DeepEqual(attributes, []core.CanonicalID{"sales.order.order_id", "sales.order.customer_id"}) {
		t.Fatalf("Semantic attributes=%v", attributes)
	}
	if !reflect.DeepEqual(got.Execution.Result.Mapped.Rows, [][]any{{"ORD-1", "C-7"}, {"ORD-2", "C-9"}}) {
		t.Fatalf("mapped rows=%v", got.Execution.Result.Mapped.Rows)
	}
}

func TestExecuteBrowseQueryInEnvironmentRefusesBeforeInvokingExecutor(t *testing.T) {
	for _, tc := range []struct {
		name        string
		environment core.CanonicalID
		provider    environment.DatabaseProvider
		mutate      func(*ir.World)
		want        string
	}{
		{name: "unknown environment", environment: "production", provider: environment.ProviderPostgres, want: "Environment production"},
		{name: "unbound datasource", environment: "development", provider: environment.ProviderPostgres,
			mutate: func(w *ir.World) { w.Environments[0].Bindings = nil }, want: "Database instance Binding"},
		{name: "ambiguous binding", environment: "development", provider: environment.ProviderPostgres,
			mutate: func(w *ir.World) {
				w.Environments[0].Bindings = append(w.Environments[0].Bindings, w.Environments[0].Bindings[0])
			}, want: "複数"},
		{name: "instance defines another database", environment: "development", provider: environment.ProviderPostgres,
			mutate: func(w *ir.World) { w.Environments[0].Instances[0].Definition = "other_db" }, want: "target Database"},
		{name: "query dialect differs from instance provider", environment: "development", provider: environment.ProviderOracle,
			mutate: func(w *ir.World) { w.Environments[0].Instances[0].Provider = environment.ProviderOracle }, want: "dialect"},
		{name: "runtime executor provider differs from instance provider", environment: "development", provider: environment.ProviderOracle,
			want: "runtime Executor provider"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			world := xteaWorld(t)
			if tc.mutate != nil {
				tc.mutate(&world)
			}
			called := false
			executor := providerBoundBrowseExecutor{provider: tc.provider, execute: func(context.Context, query.EmbeddedSQL, application.DatabaseRequirement, map[string]any) (driver.BrowseQueryResult, error) {
				called = true
				return driver.BrowseQueryResult{}, nil
			}}
			_, err := driver.NewResolver(world).ExecuteBrowseQueryInEnvironment(
				context.Background(), tc.environment, "order_entry", "order_entry.recent_orders",
				executor, map[string]any{"status": "OPEN", "status_operator": "eq", "page_offset": int64(0)},
			)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v want substring %q", err, tc.want)
			}
			if called {
				t.Fatal("束縛が成立していないのにExecutorが呼び出されました")
			}
		})
	}
}

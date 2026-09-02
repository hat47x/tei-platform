package driver_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/query"
)

type browseQueryExecutorFunc func(
	context.Context,
	query.EmbeddedSQL,
	application.DatabaseRequirement,
	map[string]any,
) (driver.BrowseQueryResult, error)

func (f browseQueryExecutorFunc) ExecuteBrowseQuery(
	ctx context.Context,
	q query.EmbeddedSQL,
	datasource application.DatabaseRequirement,
	parameters map[string]any,
) (driver.BrowseQueryResult, error) {
	return f(ctx, q, datasource, parameters)
}

func TestDriverExecutesBrowseQueryWithoutReconstructingNativeSQLOrDatasource(t *testing.T) {
	resolver := driver.NewResolver(referenceWorld(t))
	nativeSQL, err := os.ReadFile(referenceRecentOrdersSQL(t))
	if err != nil {
		t.Fatalf("recent_orders.sqlを読み込めません: %v", err)
	}

	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	parameters := map[string]any{"from": from}
	type executorEvidence struct {
		Count int
	}
	const returnedCount = 2

	called := false
	executor := browseQueryExecutorFunc(func(
		_ context.Context,
		q query.EmbeddedSQL,
		datasource application.DatabaseRequirement,
		actualParameters map[string]any,
	) (driver.BrowseQueryResult, error) {
		called = true
		if q.ID != "order_entry.recent_orders" {
			return driver.BrowseQueryResult{}, fmt.Errorf("Query identity=%s", q.ID)
		}
		if q.Application != "order_entry" || q.Datasource != "sales" {
			return driver.BrowseQueryResult{}, fmt.Errorf("Query ownership=%s datasource=%s", q.Application, q.Datasource)
		}
		if q.Source != strings.TrimSpace(string(nativeSQL)) {
			return driver.BrowseQueryResult{}, fmt.Errorf("Loaderによる外側空白の正規化を除き、Query Sourceが正本のrecent_orders.sqlと一致しません")
		}
		if datasource.Alias != "sales" || datasource.TargetDatabase != "sales_core" || string(datasource.Access) != "read-write" {
			return driver.BrowseQueryResult{}, fmt.Errorf("Datasource=%#v", datasource)
		}
		if len(actualParameters) != 1 || actualParameters["from"] != from {
			return driver.BrowseQueryResult{}, fmt.Errorf("parameters=%#v", actualParameters)
		}
		return driver.BrowseQueryResult{
			Raw: queryresult.RawResult{
				Columns: []string{"total_amount", "status", "ordered_at", "customer_id", "order_id"},
				Rows: [][]any{
					{3100.00, "READY", time.Date(2026, 8, 3, 11, 30, 0, 0, time.UTC), int64(12), int64(102)},
					{2500.50, "OPEN", time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC), int64(11), int64(101)},
				},
			},
			Payload: executorEvidence{Count: returnedCount},
		}, nil
	})

	execution, err := resolver.ExecuteBrowseQuery(
		context.Background(),
		"order_entry",
		"order_entry.recent_orders",
		executor,
		parameters,
	)
	if err != nil {
		t.Fatalf("browse Queryを実行できません: %v", err)
	}
	if !called {
		t.Fatal("Query Executorが呼び出されていません")
	}
	if execution.Resolution.Query == nil || execution.Resolution.Query.ID != "order_entry.recent_orders" {
		t.Fatalf("Query resolution=%#v", execution.Resolution.Query)
	}
	if execution.Resolution.Datasource == nil || execution.Resolution.Datasource.TargetDatabase != "sales_core" {
		t.Fatalf("Datasource resolution=%#v", execution.Resolution.Datasource)
	}
	evidence, ok := execution.Result.Payload.(executorEvidence)
	if !ok || evidence.Count != returnedCount {
		t.Fatalf("Query executor evidence=%#v", execution.Result.Payload)
	}

	wantAttributes := []core.CanonicalID{
		"sales.order.order_id",
		"sales.order.customer_id",
		"sales.order.ordered_at",
		"sales.order.status",
		"sales.order.total",
	}
	gotAttributes := make([]core.CanonicalID, len(execution.Result.Mapped.Columns))
	for i, column := range execution.Result.Mapped.Columns {
		gotAttributes[i] = column.Attribute
	}
	if !reflect.DeepEqual(gotAttributes, wantAttributes) {
		t.Fatalf("Semantic attributes=%#v want=%#v", gotAttributes, wantAttributes)
	}
	if got := execution.Result.Mapped.Rows[0]; !reflect.DeepEqual(got, []any{int64(102), int64(12), time.Date(2026, 8, 3, 11, 30, 0, 0, time.UTC), "READY", 3100.00}) {
		t.Fatalf("mapped row=%#v", got)
	}
	if strings.Contains(fmt.Sprintf("%#v", execution), from.Format(time.RFC3339)) {
		t.Fatalf("Driverの実行結果にQuery parameterを保持しています: %#v", execution)
	}
}

func TestBrowseQueryExecutionKeepsExecutionFailureSeparateFromEmptyBusinessData(t *testing.T) {
	resolver := driver.NewResolver(referenceWorld(t))

	t.Run("Executor障害を空結果へ変換しない", func(t *testing.T) {
		executor := browseQueryExecutorFunc(func(
			context.Context,
			query.EmbeddedSQL,
			application.DatabaseRequirement,
			map[string]any,
		) (driver.BrowseQueryResult, error) {
			return driver.BrowseQueryResult{}, errors.New("database unavailable")
		})
		execution, err := resolver.ExecuteBrowseQuery(
			context.Background(),
			"order_entry",
			"order_entry.recent_orders",
			executor,
			nil,
		)
		if err == nil {
			t.Fatalf("Executor障害が正常なQuery結果になっています: %#v", execution)
		}
	})

	t.Run("mapping不整合を正常な空結果へ変換しない", func(t *testing.T) {
		executor := browseQueryExecutorFunc(func(
			context.Context,
			query.EmbeddedSQL,
			application.DatabaseRequirement,
			map[string]any,
		) (driver.BrowseQueryResult, error) {
			return driver.BrowseQueryResult{
				Raw: queryresult.RawResult{
					Columns: []string{"order_id"},
					Rows:    [][]any{{int64(101)}},
				},
			}, nil
		})
		if _, err := resolver.ExecuteBrowseQuery(
			context.Background(),
			"order_entry",
			"order_entry.recent_orders",
			executor,
			nil,
		); err == nil {
			t.Fatal("不完全なraw resultを正常なSemantic resultとして扱っています")
		}
	})

	t.Run("browse以外のPatternをQueryとして実行しない", func(t *testing.T) {
		executor := browseQueryExecutorFunc(func(
			context.Context,
			query.EmbeddedSQL,
			application.DatabaseRequirement,
			map[string]any,
		) (driver.BrowseQueryResult, error) {
			t.Fatal("browse以外のPatternでExecutorが呼び出されました")
			return driver.BrowseQueryResult{}, nil
		})
		if _, err := resolver.ExecuteBrowseQuery(
			context.Background(),
			"order_entry",
			"order_entry.order_record",
			executor,
			nil,
		); err == nil {
			t.Fatal("record Patternをbrowse Queryとして実行しています")
		}
	})
}

func referenceRecentOrdersSQL(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test fileの位置を取得できません")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	return filepath.Join(root, "reference", "workspace", "applications", "order-entry", "queries", "recent_orders.sql")
}

package driver_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"tei/internal/application"
	"tei/internal/driver"
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
	type executorRows struct {
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
		if q.Source != string(nativeSQL) {
			return driver.BrowseQueryResult{}, fmt.Errorf("Loaderが読み込んだnative SQLが正本のrecent_orders.sqlと一致しません")
		}
		if datasource.Alias != "sales" || datasource.TargetDatabase != "sales_core" || string(datasource.Access) != "read-write" {
			return driver.BrowseQueryResult{}, fmt.Errorf("Datasource=%#v", datasource)
		}
		if len(actualParameters) != 1 || actualParameters["from"] != from {
			return driver.BrowseQueryResult{}, fmt.Errorf("parameters=%#v", actualParameters)
		}
		return driver.BrowseQueryResult{Payload: executorRows{Count: returnedCount}}, nil
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
	rows, ok := execution.Result.Payload.(executorRows)
	if !ok || rows.Count != returnedCount {
		t.Fatalf("Query result=%#v", execution.Result.Payload)
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

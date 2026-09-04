package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"

	"tei/internal/application"
	"tei/internal/database/providers/postgres"
	"tei/internal/driver"
	"tei/internal/environment"
	"tei/internal/query"
	"tei/internal/validation"
	"tei/internal/workspace"
)

type postgresBrowseExecutor struct {
	db *sql.DB
}

func (e postgresBrowseExecutor) ExecuteBrowseQuery(
	ctx context.Context,
	q query.EmbeddedSQL,
	datasource application.DatabaseRequirement,
	parameters map[string]any,
) (driver.BrowseQueryResult, error) {
	if q.Dialect != environment.ProviderPostgres {
		return driver.BrowseQueryResult{}, fmt.Errorf("Reference PostgreSQL Executorはdialect %sを扱えません", q.Dialect)
	}
	if datasource.TargetDatabase != "sales_core" {
		return driver.BrowseQueryResult{}, fmt.Errorf("Reference PostgreSQL Executorの対象Databaseはsales_coreですが、%sが指定されています", datasource.TargetDatabase)
	}
	raw, err := postgres.ExecuteRawQuery(ctx, e.db, q.Source, parameters)
	if err != nil {
		return driver.BrowseQueryResult{}, err
	}
	return driver.BrowseQueryResult{Payload: raw}, nil
}

type report struct {
	Application              string   `json:"application"`
	Pattern                  string   `json:"pattern"`
	Query                    string   `json:"query"`
	Datasource               string   `json:"datasource"`
	TargetDatabase           string   `json:"targetDatabase"`
	Dialect                  string   `json:"dialect"`
	Columns                  []string `json:"columns"`
	RowCount                 int      `json:"rowCount"`
	ParameterNames           []string `json:"parameterNames"`
	ParameterValuesPersisted bool     `json:"parameterValuesPersisted"`
	SemanticMappingApplied   bool     `json:"semanticMappingApplied"`
	FirstRawRow              []any    `json:"firstRawRow,omitempty"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dsn := os.Getenv("TEI_TEST_POSTGRES_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "TEI_TEST_POSTGRES_DSN is required")
		os.Exit(2)
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		panic(err)
	}
	if err := prepareOrders(ctx, db); err != nil {
		panic(err)
	}

	loaded, err := workspace.Load(filepath.Join("reference", "workspace"))
	if err != nil {
		panic(fmt.Errorf("Reference Workspaceを読み込めません: %w", err))
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			panic(fmt.Errorf("Reference WorkspaceにValidation Errorがあります: %s: %s", diagnostic.Code, diagnostic.Message))
		}
	}

	resolver := driver.NewResolver(loaded.World)
	executed, err := resolver.ExecuteBrowseQuery(
		ctx,
		"order_entry",
		"order_entry.recent_orders",
		postgresBrowseExecutor{db: db},
		map[string]any{"from": time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
	)
	if err != nil {
		panic(err)
	}
	raw, ok := executed.Result.Payload.(postgres.RawQueryResult)
	if !ok {
		panic(fmt.Errorf("Reference PostgreSQL Executorが想定外のresult型 %T を返しました", executed.Result.Payload))
	}
	wantColumns := []string{"order_id", "customer_id", "ordered_at", "status", "total_amount"}
	if !sameStrings(raw.Columns, wantColumns) {
		panic(fmt.Errorf("recent_ordersのraw columns=%v want=%v", raw.Columns, wantColumns))
	}
	if len(raw.Rows) != 2 {
		panic(fmt.Errorf("recent_ordersのrow数=%d want=2", len(raw.Rows)))
	}
	if !sameStrings(raw.ParameterNames, []string{"from"}) {
		panic(fmt.Errorf("recent_ordersのparameter順=%v want=[from]", raw.ParameterNames))
	}

	out := report{
		Application:              string(executed.Resolution.Application.ID),
		Pattern:                  string(executed.Resolution.Pattern.ID),
		Query:                    string(executed.Resolution.Query.ID),
		Datasource:               string(executed.Resolution.Datasource.Alias),
		TargetDatabase:           string(executed.Resolution.Datasource.TargetDatabase),
		Dialect:                  string(executed.Resolution.Query.Dialect),
		Columns:                  raw.Columns,
		RowCount:                 len(raw.Rows),
		ParameterNames:           raw.ParameterNames,
		ParameterValuesPersisted: false,
		SemanticMappingApplied:   false,
		FirstRawRow:              jsonSafeRow(raw.Rows[0]),
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		panic(err)
	}
}

func prepareOrders(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
DROP TABLE IF EXISTS public.orders;
CREATE TABLE public.orders (
    order_id bigint NOT NULL PRIMARY KEY,
    customer_id bigint NOT NULL,
    ordered_at timestamptz NOT NULL,
    status text NOT NULL,
    total_amount numeric NOT NULL
);
INSERT INTO public.orders(order_id, customer_id, ordered_at, status, total_amount) VALUES
    (1001, 501, '2026-09-01T09:00:00Z', 'OPEN', 1200.50),
    (1002, 502, '2026-08-20T10:30:00Z', 'SUBMITTED', 980.00),
    (1003, 503, '2026-07-15T08:00:00Z', 'CLOSED', 300.00);
`)
	return err
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func jsonSafeRow(row []any) []any {
	out := make([]any, len(row))
	for i, value := range row {
		switch v := value.(type) {
		case []byte:
			out[i] = string(v)
		default:
			out[i] = v
		}
	}
	return out
}

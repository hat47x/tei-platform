package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"tei/internal/driver"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	"tei/internal/validation"
	"tei/internal/workspace"
	"tei/tools/internal/postgresdb"
)

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
	MappedAttributes         []string `json:"mappedAttributes"`
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
	db, err := postgresdb.OpenSQL(dsn)
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

	// 製品側のPostgreSQL Query Executorをそのまま使う。raw resultはExecutorが物理的に観測した
	// 列と値であり、Semantic mappingはDriverがCanonical Queryの明示mappingから適用する。
	resolver := driver.NewResolver(loaded.World)
	executed, err := resolver.ExecuteBrowseQuery(
		ctx,
		"order_entry",
		"order_entry.recent_orders",
		postgresquery.Executor{DB: db, Database: "sales_core"},
		map[string]any{"from": time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
	)
	if err != nil {
		panic(err)
	}
	execution, ok := executed.Result.Payload.(postgresquery.Result)
	if !ok {
		panic(fmt.Errorf("Reference PostgreSQL Executorが想定外のresult型 %T を返しました", executed.Result.Payload))
	}
	raw := executed.Result.Raw
	wantColumns := []string{"order_id", "customer_id", "ordered_at", "status", "total_amount"}
	if !sameStrings(raw.Columns, wantColumns) {
		panic(fmt.Errorf("recent_ordersのraw columns=%v want=%v", raw.Columns, wantColumns))
	}
	if len(raw.Rows) != 2 {
		panic(fmt.Errorf("recent_ordersのrow数=%d want=2", len(raw.Rows)))
	}
	if !sameStrings(execution.ParameterNames, []string{"from"}) {
		panic(fmt.Errorf("recent_ordersのparameter順=%v want=[from]", execution.ParameterNames))
	}
	mapped := executed.Result.Mapped
	if len(mapped.Columns) != len(raw.Columns) {
		panic(fmt.Errorf("recent_ordersのmapping列数=%d want=%d", len(mapped.Columns), len(raw.Columns)))
	}
	mappedAttributes := make([]string, len(mapped.Columns))
	for i, column := range mapped.Columns {
		if column.Raw != raw.Columns[i] {
			panic(fmt.Errorf("recent_ordersのmapping %d列目のraw列=%q want=%q", i, column.Raw, raw.Columns[i]))
		}
		mappedAttributes[i] = string(column.Attribute)
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
		ParameterNames:           execution.ParameterNames,
		ParameterValuesPersisted: false,
		SemanticMappingApplied:   true,
		MappedAttributes:         mappedAttributes,
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

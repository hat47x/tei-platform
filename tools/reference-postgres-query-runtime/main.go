package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	_ "github.com/lib/pq"

	"tei/internal/core"
	"tei/internal/driver"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	"tei/internal/workspace"
)

type evidence struct {
	Query                    core.CanonicalID   `json:"query"`
	Datasource               string             `json:"datasource"`
	TargetDatabase           core.CanonicalID   `json:"targetDatabase"`
	ParameterNames           []string           `json:"parameterNames"`
	ParameterValuesPersisted bool               `json:"parameterValuesPersisted"`
	RawColumns               []string           `json:"rawColumns"`
	RawRows                  int                `json:"rawRows"`
	SemanticAttributes       []core.CanonicalID `json:"semanticAttributes"`
	MappedRows               int                `json:"mappedRows"`
	SemanticMappingApplied   bool               `json:"semanticMappingApplied"`
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
	if err := prepareFixture(ctx, db); err != nil {
		panic(err)
	}

	loaded, err := workspace.Load("reference/workspace")
	if err != nil {
		panic(err)
	}
	resolver := driver.NewResolver(loaded.World)
	executor := postgresquery.Executor{
		DB:       db,
		Database: core.MustCanonicalID("sales_core"),
	}

	executed, err := resolver.ExecuteBrowseQuery(
		ctx,
		core.MustCanonicalID("order_entry"),
		core.MustCanonicalID("order_entry.recent_orders"),
		executor,
		map[string]any{
			"from": time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		},
	)
	if err != nil {
		panic(err)
	}

	result, ok := executed.Result.Payload.(postgresquery.Result)
	if !ok {
		panic(fmt.Errorf("PostgreSQL Executorが想定外のpayloadを返しました: %T", executed.Result.Payload))
	}
	wantRawColumns := []string{"order_id", "customer_id", "ordered_at", "status", "total_amount"}
	if !reflect.DeepEqual(executed.Result.Raw.Columns, wantRawColumns) {
		panic(fmt.Errorf("raw result columns=%#v want=%#v", executed.Result.Raw.Columns, wantRawColumns))
	}
	if len(executed.Result.Raw.Rows) != 2 {
		panic(fmt.Errorf("raw result rows=%d want=2", len(executed.Result.Raw.Rows)))
	}
	if got := rawInt64(executed.Result.Raw.Rows[0][0]); got != 102 {
		panic(fmt.Errorf("first raw order_id=%d want=102", got))
	}
	if got := rawInt64(executed.Result.Raw.Rows[1][0]); got != 101 {
		panic(fmt.Errorf("second raw order_id=%d want=101", got))
	}
	if !reflect.DeepEqual(result.ParameterNames, []string{"from"}) {
		panic(fmt.Errorf("parameterNames=%#v want=[from]", result.ParameterNames))
	}
	if executed.Resolution.Query == nil || executed.Resolution.Datasource == nil {
		panic("browse QueryまたはDatasourceが解決されていません")
	}

	wantSemanticAttributes := []core.CanonicalID{
		"sales.order.order_id",
		"sales.order.customer_id",
		"sales.order.ordered_at",
		"sales.order.status",
		"sales.order.total",
	}
	semanticAttributes := make([]core.CanonicalID, len(executed.Result.Mapped.Columns))
	for i, column := range executed.Result.Mapped.Columns {
		semanticAttributes[i] = column.Attribute
	}
	if !reflect.DeepEqual(semanticAttributes, wantSemanticAttributes) {
		panic(fmt.Errorf("semantic attributes=%#v want=%#v", semanticAttributes, wantSemanticAttributes))
	}
	if len(executed.Result.Mapped.Rows) != 2 {
		panic(fmt.Errorf("mapped result rows=%d want=2", len(executed.Result.Mapped.Rows)))
	}
	if got := rawInt64(executed.Result.Mapped.Rows[0][0]); got != 102 {
		panic(fmt.Errorf("first mapped order_id=%d want=102", got))
	}
	if !reflect.DeepEqual(executed.Result.Mapped.Rows[0][4], executed.Result.Raw.Rows[0][4]) {
		panic(fmt.Errorf("total_amountのraw値がmapping時に変換されています: raw=%#v mapped=%#v", executed.Result.Raw.Rows[0][4], executed.Result.Mapped.Rows[0][4]))
	}

	out := evidence{
		Query:                    executed.Resolution.Query.ID,
		Datasource:               string(executed.Resolution.Datasource.Alias),
		TargetDatabase:           executed.Resolution.Datasource.TargetDatabase,
		ParameterNames:           append([]string(nil), result.ParameterNames...),
		ParameterValuesPersisted: false,
		RawColumns:               append([]string(nil), executed.Result.Raw.Columns...),
		RawRows:                  len(executed.Result.Raw.Rows),
		SemanticAttributes:       semanticAttributes,
		MappedRows:               len(executed.Result.Mapped.Rows),
		SemanticMappingApplied:   true,
	}
	encoded, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(encoded))
}

func prepareFixture(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
DROP TABLE IF EXISTS public.orders CASCADE;
CREATE TABLE public.orders (
    order_id bigint PRIMARY KEY,
    customer_id bigint NOT NULL,
    ordered_at timestamptz NOT NULL,
    status varchar(32) NOT NULL,
    total_amount numeric(12, 2) NOT NULL
);
`); err != nil {
		return fmt.Errorf("recent_orders用fixture tableを作成できません: %w", err)
	}

	rows := []struct {
		orderID    int64
		customerID int64
		orderedAt  time.Time
		status     string
		total      string
	}{
		{100, 10, time.Date(2026, 7, 31, 23, 0, 0, 0, time.UTC), "OLD", "1000.00"},
		{101, 11, time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC), "OPEN", "2500.50"},
		{102, 12, time.Date(2026, 8, 3, 11, 30, 0, 0, time.UTC), "READY", "3100.00"},
	}
	for _, row := range rows {
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO public.orders (order_id, customer_id, ordered_at, status, total_amount) VALUES ($1, $2, $3, $4, $5)`,
			row.orderID,
			row.customerID,
			row.orderedAt,
			row.status,
			row.total,
		); err != nil {
			return fmt.Errorf("recent_orders用fixture rowを追加できません: %w", err)
		}
	}
	return nil
}

func rawInt64(value any) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int32:
		return int64(v)
	case int:
		return int64(v)
	default:
		panic(fmt.Errorf("raw integer valueの型が想定外です: %T (%v)", value, value))
	}
}

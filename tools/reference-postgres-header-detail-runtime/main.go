package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	_ "github.com/lib/pq"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	sqlquery "tei/internal/driver/queryruntime/sql"
	"tei/internal/environment"
	"tei/internal/workspace"
)

type evidence struct {
	Pattern                              core.CanonicalID   `json:"pattern"`
	Relation                             core.CanonicalID   `json:"relation"`
	Environment                          core.CanonicalID   `json:"environment"`
	DatabaseInstance                     core.CanonicalID   `json:"databaseInstance"`
	ExecutorCapability                   core.CanonicalID   `json:"executorCapability"`
	ContractQueryTarget                  string             `json:"contractQueryTarget"`
	ParameterNames                       []string           `json:"parameterNames"`
	ParameterValuesPersisted             bool               `json:"parameterValuesPersisted"`
	Rows                                 int                `json:"rows"`
	ZeroRows                             int                `json:"zeroRows"`
	SemanticAttributes                   []core.CanonicalID `json:"semanticAttributes"`
	RelationStorageColumn                string             `json:"relationStorageColumn"`
	SemanticOrderLineParentAttributeAdded bool              `json:"semanticOrderLineParentAttributeAdded"`
	SemanticMappingApplied               bool               `json:"semanticMappingApplied"`
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

	contractExecutor := sqlquery.Executor{
		Provider:     environment.ProviderPostgres,
		Realizations: loaded.World.ContractQuerySQLRealizations,
		Runner:       postgresquery.Runner{DB: db},
	}
	registry := driver.ContractQueryExecutorMap{
		{
			Environment: core.MustCanonicalID("development"),
			Instance:    core.MustCanonicalID("sales_core.dev"),
			Capability:  core.MustCanonicalID("database.query.postgres"),
		}: contractExecutor,
	}

	source := mappedOrderID(int64(1001))
	executed, err := resolver.ExecuteHeaderDetailContractQueryInEnvironment(
		ctx,
		core.MustCanonicalID("development"),
		core.MustCanonicalID("order_entry"),
		core.MustCanonicalID("order_entry.order_lines"),
		source,
		0,
		registry,
	)
	if err != nil {
		panic(err)
	}
	if len(executed.Execution.Result.Rows) != 2 {
		panic(fmt.Errorf("order_id=1001のOrderLine件数=%d want=2", len(executed.Execution.Result.Rows)))
	}

	wantIDs := []int64{1101, 1102}
	for i, row := range executed.Execution.Result.Rows {
		if got := rawInt64(row.Attributes["sales.order_line.order_line_id"]); got != wantIDs[i] {
			panic(fmt.Errorf("OrderLine[%d].order_line_id=%d want=%d", i, got, wantIDs[i]))
		}
		if _, exists := row.Attributes["sales.order_line.order_id"]; exists {
			panic("物理support column order_idをOrderLineのSemantic attributeとして返しています")
		}
	}

	payload, ok := executed.Execution.Result.Payload.(postgresquery.Result)
	if !ok {
		panic(fmt.Errorf("PostgreSQL Contract Query Executorが想定外のpayloadを返しました: %T", executed.Execution.Result.Payload))
	}
	if len(payload.ParameterNames) != 1 || payload.ParameterNames[0] != "order_id" {
		panic(fmt.Errorf("parameterNames=%#v want=[order_id]", payload.ParameterNames))
	}

	zeroExecuted, err := resolver.ExecuteHeaderDetailContractQueryInEnvironment(
		ctx,
		core.MustCanonicalID("development"),
		core.MustCanonicalID("order_entry"),
		core.MustCanonicalID("order_entry.order_lines"),
		mappedOrderID(int64(9999)),
		0,
		registry,
	)
	if err != nil {
		panic(fmt.Errorf("0件のheader-detail取得をExecution Errorへ変換しています: %w", err))
	}
	if len(zeroExecuted.Execution.Result.Rows) != 0 {
		panic(fmt.Errorf("order_id=9999のOrderLine件数=%d want=0", len(zeroExecuted.Execution.Result.Rows)))
	}

	semanticAttributes := semanticAttributeIDs(executed.Execution.Result.Rows)
	relationStorageColumn, parentAttributeAdded := inspectOrderLineStorage(loaded)

	out := evidence{
		Pattern:                              core.MustCanonicalID("order_entry.order_lines"),
		Relation:                             core.MustCanonicalID("sales.order.lines"),
		Environment:                          executed.Environment.ID,
		DatabaseInstance:                     executed.Executor.Instance.ID,
		ExecutorCapability:                   executed.Executor.Capability.Capability.ID,
		ContractQueryTarget:                  executed.Execution.Resolution.Binding.Target.Identity(),
		ParameterNames:                       append([]string(nil), payload.ParameterNames...),
		ParameterValuesPersisted:             false,
		Rows:                                 len(executed.Execution.Result.Rows),
		ZeroRows:                             len(zeroExecuted.Execution.Result.Rows),
		SemanticAttributes:                   semanticAttributes,
		RelationStorageColumn:                relationStorageColumn,
		SemanticOrderLineParentAttributeAdded: parentAttributeAdded,
		SemanticMappingApplied:               true,
	}
	encoded, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(encoded))
}

func mappedOrderID(orderID int64) queryresult.MappedResult {
	return queryresult.MappedResult{
		Columns: []queryresult.Column{
			{Raw: "order_id", Attribute: core.MustCanonicalID("sales.order.order_id")},
		},
		Rows: [][]any{{orderID}},
	}
}

func prepareFixture(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
DROP TABLE IF EXISTS public.order_lines CASCADE;
CREATE TABLE public.order_lines (
    order_line_id bigint PRIMARY KEY,
    order_id bigint NOT NULL,
    line_number bigint NOT NULL,
    item_code varchar(64) NOT NULL,
    quantity numeric(12, 2) NOT NULL,
    unit_price numeric(12, 2) NOT NULL
);
`); err != nil {
		return fmt.Errorf("header-detail用fixture tableを作成できません: %w", err)
	}

	rows := []struct {
		orderLineID int64
		orderID     int64
		lineNumber  int64
		itemCode    string
		quantity    string
		unitPrice   string
	}{
		{1101, 1001, 1, "SKU-001", "2.00", "1200.00"},
		{1102, 1001, 2, "SKU-002", "1.00", "800.00"},
		{2101, 2000, 1, "SKU-003", "3.00", "500.00"},
	}
	for _, row := range rows {
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO public.order_lines (order_line_id, order_id, line_number, item_code, quantity, unit_price) VALUES ($1, $2, $3, $4, $5, $6)`,
			row.orderLineID,
			row.orderID,
			row.lineNumber,
			row.itemCode,
			row.quantity,
			row.unitPrice,
		); err != nil {
			return fmt.Errorf("header-detail用fixture rowを追加できません: %w", err)
		}
	}
	return nil
}

func semanticAttributeIDs(rows []driver.ContractQueryRow) []core.CanonicalID {
	seen := map[core.CanonicalID]struct{}{}
	for _, row := range rows {
		for attribute := range row.Attributes {
			seen[attribute] = struct{}{}
		}
	}
	out := make([]core.CanonicalID, 0, len(seen))
	for attribute := range seen {
		out = append(out, attribute)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func inspectOrderLineStorage(loaded workspace.Result) (string, bool) {
	for _, mapping := range loaded.World.Persistence {
		if mapping.ID != "sales_core.order_line" {
			continue
		}
		for _, column := range mapping.Columns {
			if column.Attribute == "sales.order_line.order_id" {
				return "", true
			}
		}
		storage, ok := mapping.RelationStorageByRelation("sales.order.lines")
		if !ok || len(storage.Components) != 1 {
			panic(fmt.Errorf("sales.order.linesのRelation Storageが一意に解決できません: %#v", mapping.RelationStorage))
		}
		return storage.Components[0].Column, false
	}
	panic("sales_core.order_line Persistence Mappingがありません")
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

//go:build !tei_repair_only

package main

import (
	"context"
	"fmt"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/environment"
	"tei/internal/realization"
)

// Only order_entry compiles the PostgreSQL order-line Query adapter. Oracle
// customer mapping is shared and remains in reference_contract_query_fixture.go.
// order_entry requires both customer reference and PostgreSQL order-line reads.
func referenceContractQueryExecutors() driver.ContractQueryExecutorMap {
	executors := referenceCustomerContractQueryExecutors()
	executors[driver.ContractQueryExecutorRuntimeKey{
		Environment: core.MustCanonicalID("development"),
		Instance:    core.MustCanonicalID("sales_core.dev"),
		Capability:  core.MustCanonicalID("database.query.postgres"),
	}] = referenceOrderLineContractQueryExecutor{provider: environment.ProviderPostgres}
	return executors
}

type referenceOrderLineContractQueryExecutor struct {
	provider environment.DatabaseProvider
}

func (e referenceOrderLineContractQueryExecutor) DatabaseProvider() environment.DatabaseProvider {
	return e.provider
}

func (e referenceOrderLineContractQueryExecutor) ExecuteContractQuery(
	_ context.Context,
	target realization.ContractQueryTarget,
	queryContract database.QueryContract,
	parameters map[string]any,
) (driver.ContractQueryResult, error) {
	switch target.Identity() {
	case "sales_core/sales_core.order_line/find_by_order":
		if e.provider != environment.ProviderPostgres {
			return driver.ContractQueryResult{}, fmt.Errorf("order-line executor provider=%s", e.provider)
		}
		orderIDValue, ok := parameters["order_id"]
		if !ok {
			return driver.ContractQueryResult{}, fmt.Errorf("order_id parameter is required")
		}
		orderID, ok := orderIDValue.(int64)
		if !ok {
			return driver.ContractQueryResult{}, fmt.Errorf("order_id parameter has unexpected type %T", orderIDValue)
		}
		if queryContract.Name != "find_by_order" {
			return driver.ContractQueryResult{}, fmt.Errorf("unexpected order-line Query %s", queryContract.Name)
		}
		firstQuantity := "2"
		if orderID == 1002 {
			firstQuantity = "0"
		}
		return driver.ContractQueryResult{
			Rows: []driver.ContractQueryRow{
				{Attributes: map[core.CanonicalID]any{
					core.MustCanonicalID("sales.order_line.order_line_id"): int64(5001),
					core.MustCanonicalID("sales.order_line.line_number"):   1,
					core.MustCanonicalID("sales.order_line.item_code"):     "ITEM-A",
					core.MustCanonicalID("sales.order_line.quantity"):      firstQuantity,
					core.MustCanonicalID("sales.order_line.unit_price"):    "4000.00",
				}},
				{Attributes: map[core.CanonicalID]any{
					core.MustCanonicalID("sales.order_line.order_line_id"): int64(5002),
					core.MustCanonicalID("sales.order_line.line_number"):   2,
					core.MustCanonicalID("sales.order_line.item_code"):     "ITEM-B",
					core.MustCanonicalID("sales.order_line.quantity"):      "1",
					core.MustCanonicalID("sales.order_line.unit_price"):    "4500.00",
				}},
			},
			Payload: map[string]any{
				"fixture": "postgres-order-line-read",
				"orderId": orderID,
			},
		}, nil
	default:
		return driver.ContractQueryResult{}, fmt.Errorf("unsupported Contract Query target %s", target.Identity())
	}
}

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

// Shared synthetic Contract Query adapters are kept outside the order-specific
// read scenario. Repair lookup uses the same implementation and contract meaning.
// referenceContractQueryExecutors is the development runtime composition shared by every
// Reference scenario runner, so the Executor-to-instance binding is written once.
// The Executors are synthetic fixtures, not connections to a real Database.
func referenceContractQueryExecutors() driver.ContractQueryExecutorMap {
	return driver.ContractQueryExecutorMap{
		{
			Environment: core.MustCanonicalID("development"),
			Instance:    core.MustCanonicalID("master_data.dev"),
			Capability:  core.MustCanonicalID("database.query.oracle"),
		}: referenceContractQueryExecutor{provider: environment.ProviderOracle},
		{
			Environment: core.MustCanonicalID("development"),
			Instance:    core.MustCanonicalID("sales_core.dev"),
			Capability:  core.MustCanonicalID("database.query.postgres"),
		}: referenceContractQueryExecutor{provider: environment.ProviderPostgres},
	}
}

type referenceContractQueryExecutor struct {
	provider environment.DatabaseProvider
}

func (e referenceContractQueryExecutor) DatabaseProvider() environment.DatabaseProvider {
	return e.provider
}

func (e referenceContractQueryExecutor) ExecuteContractQuery(
	_ context.Context,
	target realization.ContractQueryTarget,
	queryContract database.QueryContract,
	parameters map[string]any,
) (driver.ContractQueryResult, error) {
	switch target.Identity() {
	case "master_data/master_data.customer/find_by_id":
		if e.provider != environment.ProviderOracle {
			return driver.ContractQueryResult{}, fmt.Errorf("customer executor provider=%s", e.provider)
		}
		if queryContract.Name != "find_by_id" {
			return driver.ContractQueryResult{}, fmt.Errorf("unexpected customer Query %s", queryContract.Name)
		}
		id, ok := parameters["id"]
		if !ok {
			return driver.ContractQueryResult{}, fmt.Errorf("customer id parameter is required")
		}
		return driver.ContractQueryResult{
			Rows: []driver.ContractQueryRow{{
				Attributes: map[core.CanonicalID]any{
					core.MustCanonicalID("customer.customer.customer_id"): id,
					core.MustCanonicalID("customer.customer.name"):        "Reference Customer " + fmt.Sprint(id),
				},
			}},
			Payload: map[string]any{"fixture": "oracle-customer-read"},
		}, nil

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

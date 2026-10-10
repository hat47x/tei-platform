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

// These are synthetic Query adapters, not live database connections.
// Compose only capabilities required by the requesting Application; a repair
// lookup must not inherit the order-line PostgreSQL runtime binding.
func referenceCustomerContractQueryExecutors() driver.ContractQueryExecutorMap {
	return driver.ContractQueryExecutorMap{
		{
			Environment: core.MustCanonicalID("development"),
			Instance:    core.MustCanonicalID("master_data.dev"),
			Capability:  core.MustCanonicalID("database.query.oracle"),
		}: referenceContractQueryExecutor{provider: environment.ProviderOracle},
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

	default:
		return driver.ContractQueryResult{}, fmt.Errorf("unsupported Contract Query target %s", target.Identity())
	}
}

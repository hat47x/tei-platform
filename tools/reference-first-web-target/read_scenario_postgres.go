//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"database/sql"
	"fmt"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	sqlquery "tei/internal/driver/queryruntime/sql"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/realization"
)

// readOnlyTxExecutor runs a Contract Query's accepted SQL against PostgreSQL
// inside a server-enforced read-only transaction that lives for exactly one
// execution. A single long-lived transaction would hand every later page the
// snapshot of the first read, which is not what "read the database now" means.
//
// The transaction is opened and rolled back here; nothing in this adapter
// commits, so a read cannot become a write even if a realization were edited.
type readOnlyTxExecutor struct {
	db           *sql.DB
	realizations []realization.ContractQuerySQLRealization
}

var _ driver.ContractQueryExecutor = readOnlyTxExecutor{}
var _ driver.DatabaseProviderBoundContractQueryExecutor = readOnlyTxExecutor{}

func (readOnlyTxExecutor) DatabaseProvider() environment.DatabaseProvider {
	return environment.ProviderPostgres
}

func (e readOnlyTxExecutor) ExecuteContractQuery(
	ctx context.Context,
	target realization.ContractQueryTarget,
	queryContract database.QueryContract,
	parameters map[string]any,
) (driver.ContractQueryResult, error) {
	tx, err := e.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return driver.ContractQueryResult{}, fmt.Errorf("cannot begin a read-only PostgreSQL transaction")
	}
	defer func() { _ = tx.Rollback() }()

	// The inner executor selects the SQL realization from the Workspace by
	// target and provider; this adapter only decides where it runs.
	inner := sqlquery.Executor{
		Provider:     environment.ProviderPostgres,
		Realizations: e.realizations,
		Runner:       postgresquery.Runner{DB: tx},
	}
	return inner.ExecuteContractQuery(ctx, target, queryContract, parameters)
}

// newPostgresReadRunner returns the normal read runner with the header-detail
// order-line read served by PostgreSQL. Browse and the customer reference keep
// their existing executors, so the page still cannot show a read it did not
// make: only the executor for the database-backed target is replaced.
//
// This mirrors how `tei view -environment` builds its registry: the executor is
// chosen by the Workspace's declared binding, and the statement text comes from
// the Workspace's own Contract Query SQL realization.
func newPostgresReadRunner(world ir.World, db *sql.DB) (*referenceReadScenarioRunner, error) {
	if db == nil {
		return nil, fmt.Errorf("a PostgreSQL connection is required for the database-backed read")
	}
	executor := readOnlyTxExecutor{db: db, realizations: world.ContractQuerySQLRealizations}
	added := driver.ContractQueryExecutorMap{}
	for _, binding := range world.ContractQueryExecutorBindings {
		if binding.Executor != core.MustCanonicalID("database.query.postgres") {
			continue
		}
		added[driver.ContractQueryExecutorRuntimeKey{
			Environment: binding.Environment,
			Instance:    binding.Instance,
			Capability:  binding.Executor,
		}] = executor
	}
	if len(added) == 0 {
		return nil, fmt.Errorf("the Workspace declares no PostgreSQL Contract Query executor binding")
	}

	runner := newReferenceReadScenarioRunner(world)
	runner.contracts = overlayContractExecutors(added)
	return runner, nil
}

// overlayContractExecutors adds the database-backed executors to the fixture
// registry.
//
// The database wins on a key it covers. Keeping the fixture there would let a
// page show fixture rows while the host believed it had read the database, which
// is exactly the substitution NOW.md forbids. The fixture remains only for keys
// the database does not cover (the customer reference), and the merge is
// deterministic so which executor serves a read never depends on map order.
func overlayContractExecutors(
	added driver.ContractQueryExecutorMap,
) driver.ContractQueryExecutorMap {
	merged := driver.ContractQueryExecutorMap{}
	for key, executor := range referenceContractQueryExecutors() {
		merged[key] = executor
	}
	for key, executor := range added {
		merged[key] = executor
	}
	return merged
}

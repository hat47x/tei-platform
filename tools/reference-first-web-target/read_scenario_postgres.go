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
	db                 *sql.DB
	realizations       []realization.ContractQuerySQLRealization
	world              ir.World
	coverageResolution driver.ResolvedHeaderDetailContractQuery
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
	tx, err := beginWebReadOnlyQueryTx(ctx, e.db)
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
	result, err := inner.ExecuteContractQuery(ctx, target, queryContract, parameters)
	if err != nil {
		return driver.ContractQueryResult{}, err
	}
	// A proof belongs only to the executed header-detail target and the
	// PostgreSQL snapshot that returned these rows. An unrelated reference
	// query, an authored covers declaration or a later catalog inspection
	// cannot qualify the relation-wide rule.
	if target == e.coverageResolution.Binding.Target {
		qualification := driver.NewResolver(e.world).QualifyHeaderDetailCoverage(
			e.coverageResolution, environment.ProviderPostgres,
		)
		if qualification.Complete {
			if ok, _ := driver.PostgresPhysicalReadAllowsComplete(ctx, tx, e.world, e.coverageResolution); ok {
				result.Payload = webRelationCoverageProof{
					Target: target, Complete: true, ProviderEvidence: result.Payload,
				}
			}
		}
	}
	return result, nil
}

// beginWebReadOnlyQueryTx gives a single Contract Query one stable database
// snapshot for its physical rows and any later catalog visibility check.
// This DOES NOT cover the separate Browse or customer-reference reads, nor
// establish full relation coverage by itself. The caller must roll it back.
func beginWebReadOnlyQueryTx(ctx context.Context, db *sql.DB) (*sql.Tx, error) {
	return db.BeginTx(ctx, &sql.TxOptions{
		ReadOnly:  true,
		Isolation: sql.LevelRepeatableRead,
	})
}

// newPostgresReadRunner returns the normal read runner with both the browse and
// the header-detail order-line read served by PostgreSQL. The customer
// reference keeps its existing executor, so the page still cannot show a read
// it did not make.
//
// Browse is replaced together with the detail read, not left on the fixture.
// The page selects one order from the browse rows and then shows that order's
// detail rows; if browse stayed synthetic, the selected order could be an order
// the database does not hold, or could carry header values the database does not
// agree with, while the detail rows came from the database. Measured at
// 43e6adc01: this builder replaced only the detail executor, and `main` passed
// `wiring.browse` separately, so the two halves were assignable to different
// sources. A database read now refuses rather than falling back.
//
// This mirrors how `tei view -environment` builds its registry: the executor is
// chosen by the Workspace's declared binding, and the statement text comes from
// the Workspace's own Contract Query SQL realization.
func newPostgresReadRunner(world ir.World, db *sql.DB) (*referenceReadScenarioRunner, error) {
	if db == nil {
		return nil, fmt.Errorf("a PostgreSQL connection is required for the database-backed read")
	}
	resolution, err := driver.NewResolver(world).ResolveHeaderDetailContractQuery(
		core.MustCanonicalID("order_entry"), core.MustCanonicalID("order_entry.order_lines"),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve the declared order-line read: %w", err)
	}
	executor := readOnlyTxExecutor{
		db: db, realizations: world.ContractQuerySQLRealizations,
		world: world, coverageResolution: resolution,
	}
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
	runner.browse = postgresBrowseExecutor{db: db}
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

//go:build !tei_repair_only

package main

import (
	"database/sql"
	"errors"

	"tei/internal/core"
	"tei/internal/driver"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	sqlquery "tei/internal/driver/queryruntime/sql"
	"tei/internal/environment"
	"tei/internal/ir"
)

// newPostgresContractQueryReadRunner is the normal read runner with the
// PostgreSQL-bound Contract Queries executed through the same sqlquery adapter
// the product CLI uses.
//
// Only the bindings whose Environment instance is PostgreSQL are replaced. The
// customer reference is bound to another provider and keeps the executor the
// caller composed for it, so this never invents a connection for a database it
// was not given. The reference fixtures therefore stay in place for exactly the
// bindings this runner cannot serve, and an unexpected one still fails rather
// than being silently answered from a fixture.
//
// It does not decide permission, does not infer tables from the SQL, and runs
// inside a server-enforced read-only transaction so an edited realization
// cannot write.
func newPostgresContractQueryReadRunner(world ir.World, db *sql.DB, environmentID core.CanonicalID) (*referenceReadScenarioRunner, error) {
	runner := newReferenceReadScenarioRunner(world)
	runner.browse = postgresBrowseExecutor{db: db}

	executor := sqlquery.Executor{
		Provider:     environment.ProviderPostgres,
		Realizations: world.ContractQuerySQLRealizations,
		Runner:       postgresquery.Runner{DB: db},
	}
	registry, ok := runner.contracts.(driver.ContractQueryExecutorMap)
	if !ok {
		return nil, errRegistryNotReplaceable
	}
	replaced := 0
	for _, binding := range world.ContractQueryExecutorBindings {
		if binding.Environment != environmentID {
			continue
		}
		if !environmentInstanceIsPostgres(world, environmentID, binding.Instance) {
			continue
		}
		registry[driver.ContractQueryExecutorRuntimeKey{
			Environment: binding.Environment,
			Instance:    binding.Instance,
			Capability:  binding.Executor,
		}] = executor
		replaced++
	}
	if replaced == 0 {
		return nil, errNoPostgresBinding
	}
	runner.contracts = registry
	return runner, nil
}

// environmentInstanceIsPostgres reports whether the named instance of the
// named Environment declares PostgreSQL. An instance this runner cannot see
// answers false, so its binding keeps whatever the caller composed.
func environmentInstanceIsPostgres(world ir.World, environmentID, instanceID core.CanonicalID) bool {
	for i := range world.Environments {
		if world.Environments[i].ID != environmentID {
			continue
		}
		instance, ok := world.Environments[i].InstanceByID(instanceID)
		if !ok {
			return false
		}
		return instance.Provider == environment.ProviderPostgres
	}
	return false
}

// errRegistryNotReplaceable is returned when the runner's contract registry is
// not the map this seam knows how to extend. Answering anyway would mean
// guessing which executor serves which binding.
var errRegistryNotReplaceable = errors.New("the read runner's Contract Query registry cannot be replaced")

// errNoPostgresBinding is returned when the Environment declares no PostgreSQL
// instance to execute against. Falling through to the fixtures would make the
// read look real while reading nothing.
var errNoPostgresBinding = errors.New("the Environment declares no PostgreSQL instance for this read")

//go:build !tei_repair_only

package main

import (
	"database/sql"

	"tei/internal/core"
	"tei/internal/driver"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	sqlquery "tei/internal/driver/queryruntime/sql"
	"tei/internal/environment"
	"tei/internal/ir"
)

// newPostgresContractQueryReadRunner is the normal read runner with the
// customer and order-line Contract Queries executed against PostgreSQL through
// the same sqlquery adapter the product CLI uses.
//
// The reference fixtures stay available for the browser-only build variants;
// this is the opt-in seam that makes the Web read and the CLI read the same
// rows. It does not decide permission, does not infer tables from the SQL, and
// runs inside a server-enforced read-only transaction so an edited realization
// cannot write.
func newPostgresContractQueryReadRunner(world ir.World, db *sql.DB, environmentID core.CanonicalID) (*referenceReadScenarioRunner, error) {
	runner := newReferenceReadScenarioRunner(world)
	runner.browse = postgresBrowseExecutor{db: db}

	executor := sqlquery.Executor{
		Provider:     environment.ProviderPostgres,
		Realizations: world.ContractQuerySQLRealizations,
		Runner:       postgresquery.Runner{DB: db},
	}
	registry := driver.ContractQueryExecutorMap{}
	for _, binding := range world.ContractQueryExecutorBindings {
		if binding.Environment != environmentID {
			continue
		}
		instance, ok := instanceOf(world, environmentID, binding.Instance)
		if !ok || instance.Provider != environment.ProviderPostgres {
			continue
		}
		registry[driver.ContractQueryExecutorRuntimeKey{
			Environment: binding.Environment,
			Instance:    binding.Instance,
			Capability:  binding.Executor,
		}] = executor
	}
	runner.contracts = registry
	return runner, nil
}

// instanceOf resolves one Environment instance so only PostgreSQL-bound
// instances get the SQL executor. A non-PostgreSQL instance keeps whatever the
// caller composed for it.
func instanceOf(world ir.World, environmentID, instanceID core.CanonicalID) (instanceDecl, bool) {
	for i := range world.Environments {
		if world.Environments[i].ID != environmentID {
			continue
		}
		if instance, ok := world.Environments[i].InstanceByID(instanceID); ok {
			return instanceDecl{Provider: instance.Provider}, true
		}
	}
	return instanceDecl{}, false
}

// instanceDecl carries the fields this file needs from an Environment
// instance, so the check above does not depend on the rest of its shape.
type instanceDecl struct {
	Provider environment.DatabaseProvider
}

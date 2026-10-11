//go:build !tei_order_only && !tei_repair_only

package main

import (
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
)

// The two Applications may share a Contract Query adapter implementation
// without inheriting each other's runtime executor capabilities. This
// composition test does not prove post-link binary code elimination.
func TestReferenceRuntimeQueryCompositionSelectsOnlyRequiredProviders(t *testing.T) {
	oracle := driver.ContractQueryExecutorRuntimeKey{
		Environment: core.MustCanonicalID("development"),
		Instance:    core.MustCanonicalID("master_data.dev"),
		Capability:  core.MustCanonicalID("database.query.oracle"),
	}
	postgres := driver.ContractQueryExecutorRuntimeKey{
		Environment: core.MustCanonicalID("development"),
		Instance:    core.MustCanonicalID("sales_core.dev"),
		Capability:  core.MustCanonicalID("database.query.postgres"),
	}
	repair := referenceCustomerContractQueryExecutors()
	if len(repair) != 1 {
		t.Fatalf("repair_intake must bind only the Oracle customer reference: %d executors", len(repair))
	}
	if _, ok := repair[oracle]; !ok {
		t.Fatal("repair_intake lost its required Oracle customer query executor")
	}
	if _, ok := repair[postgres]; ok {
		t.Fatal("repair_intake unexpectedly acquired the order_entry PostgreSQL executor")
	}
	order := referenceContractQueryExecutors()
	if len(order) != 2 {
		t.Fatalf("order_entry requires exactly customer+order-line Query executors: got %d", len(order))
	}
	if _, ok := order[oracle]; !ok {
		t.Fatal("order_entry lost Oracle customer query executor")
	}
	if _, ok := order[postgres]; !ok {
		t.Fatal("order_entry lost PostgreSQL order-line executor")
	}
	// Observe the constructed Runner registries too, so refactoring the
	// wiring cannot silently restore the over-broad legacy composition.
	world := loadReferenceWorld(t)
	repairRunner := newReferenceCustomerLookupScenarioRunner(world)
	repairRegistry, ok := repairRunner.contracts.(driver.ContractQueryExecutorMap)
	if !ok || len(repairRegistry) != 1 {
		t.Fatalf("actual repair_intake Runner does not use Oracle-only binding: %T / %+v",
			repairRunner.contracts, repairRunner.contracts)
	}
	if _, ok := repairRegistry[postgres]; ok {
		t.Fatal("actual repair_intake Runner exposes a PostgreSQL binding")
	}
	orderRunner := newReferenceReadScenarioRunner(world)
	orderRegistry, ok := orderRunner.contracts.(driver.ContractQueryExecutorMap)
	if !ok || len(orderRegistry) != 2 {
		t.Fatalf("actual order_entry Runner lost required two-Provider composition: %T / %+v",
			orderRunner.contracts, orderRunner.contracts)
	}
	// Each map is constructed independently, so changing one Application's
	// runtime fixture does not silently grant/revoke another's capabilities.
	delete(order, oracle)
	if _, ok := repair[oracle]; !ok {
		t.Fatal("mutating order_entry's map changed repair_intake's query binding")
	}
}

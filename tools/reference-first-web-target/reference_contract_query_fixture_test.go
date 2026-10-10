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
	// Each map is constructed independently, so changing one Application's
	// runtime fixture does not silently grant/revoke another's capabilities.
	delete(order, oracle)
	if _, ok := repair[oracle]; !ok {
		t.Fatal("mutating order_entry's map changed repair_intake's query binding")
	}
}

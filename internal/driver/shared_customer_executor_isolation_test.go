package driver_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/realization"
)

// This test double deliberately retains only query identities and invocation
// counts. No credential, production authority or physical database is involved.
type sharedCustomerExecutorProbe struct {
	received []int64
}

func (p *sharedCustomerExecutorProbe) ExecuteContractQuery(
	_ context.Context,
	target realization.ContractQueryTarget,
	query database.QueryContract,
	parameters map[string]any,
) (driver.ContractQueryResult, error) {
	if target.Identity() != "master_data/master_data.customer/find_by_id" || query.Name != "find_by_id" {
		return driver.ContractQueryResult{}, fmt.Errorf("unexpected contract query: %s/%s", target.Identity(), query.Name)
	}
	id, ok := parameters["id"].(int64)
	if !ok || len(parameters) != 1 {
		return driver.ContractQueryResult{}, fmt.Errorf("unexpected customer query parameters: %#v", parameters)
	}
	p.received = append(p.received, id)
	return driver.ContractQueryResult{
		Rows: []driver.ContractQueryRow{{Attributes: map[core.CanonicalID]any{
			"customer.customer.customer_id": id,
		}}},
	}, nil
}

// Both authored Applications bind different local aliases to the SAME exact
// Environment/Instance/Capability. Sharing that runtime instance must not
// blend their source parameters or grant a foreign Application's datasource.
func TestSharedReferenceCustomerExecutorKeepsApplicationInputsAndAliasesSeparate(t *testing.T) {
	world := referenceWorld(t)
	executor := &sharedCustomerExecutorProbe{}
	registry := driver.ContractQueryExecutorMap{
		{Environment: "development", Instance: "master_data.dev", Capability: "database.query.oracle"}: executor,
	}
	repairSource := repairIntakeMappedResult()
	repairSource.Rows[0][0] = int64(84)

	cases := []struct {
		application string
		pattern     string
		wantID      int64
	}{
		{
			application: "order_entry",
			pattern:     "order_entry.customer_reference",
			wantID:      42,
		},
		{
			application: "repair_intake",
			pattern:     "repair_intake.customer_reference",
			wantID:      84,
		},
	}

	for _, tc := range cases {
		t.Run(tc.application, func(t *testing.T) {
			source := customerMappedResult()
			if tc.application == "repair_intake" {
				source = repairSource
			}
			result, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
				context.Background(), "development", core.CanonicalID(tc.application),
				core.CanonicalID(tc.pattern), source, 0, registry,
			)
			if err != nil {
				t.Fatal(err)
			}
			if result.Executor.Instance.ID != "master_data.dev" ||
				result.Executor.Capability.Capability.ID != "database.query.oracle" {
				t.Fatalf("shared physical executor was not explicitly selected: %+v", result.Executor)
			}
			if len(result.Execution.Result.Rows) != 1 ||
				result.Execution.Result.Rows[0].Attributes["customer.customer.customer_id"] != tc.wantID {
				t.Fatalf("Application %s leaked another input or result: %+v", tc.application, result.Execution.Result)
			}
		})
	}
	if !slices.Equal(executor.received, []int64{42, 84}) {
		t.Fatalf("same executor did not receive two distinct Application inputs: %v", executor.received)
	}

	// Mutate only the order Application's local alias in the in-memory World.
	// The repair Application remains declared against customer_master.
	modified := false
	for i := range world.Applications {
		if world.Applications[i].ID != "order_entry" {
			continue
		}
		for j := range world.Applications[i].Patterns {
			if world.Applications[i].Patterns[j].ID == "order_entry.customer_reference" {
				world.Applications[i].Patterns[j].Datasource = "customer_master"
				modified = true
			}
		}
	}
	if !modified {
		t.Fatal("source pattern was not found")
	}
	if _, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
		context.Background(), "development", "order_entry",
		"order_entry.customer_reference", customerMappedResult(), 0, registry,
	); err == nil {
		t.Fatal("order Application borrowed repair Application's datasource")
	}
	if !slices.Equal(executor.received, []int64{42, 84}) {
		t.Fatalf("invalid Application reached shared executor: %v", executor.received)
	}

	if _, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
		context.Background(), "development", "repair_intake",
		"repair_intake.customer_reference", repairSource, 0, registry,
	); err != nil {
		t.Fatalf("valid repair Application lost access after another alias failed: %v", err)
	}
	if !slices.Equal(executor.received, []int64{42, 84, 84}) {
		t.Fatalf("shared executor calls were mixed between Applications: %v", executor.received)
	}
}

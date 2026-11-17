//go:build !tei_repair_only

package main

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/environment"
	"tei/internal/realization"
	"tei/internal/workspace"
)

// zeroFirstLineQuantity stands in for a database whose first order line holds
// quantity 0. It composes the real fixture answer and rewrites one value, so
// the rule evaluation sees a failing row through the same read path the server
// uses.
type zeroFirstLineQuantity struct {
	inner driver.ContractQueryExecutor
}

func (z zeroFirstLineQuantity) DatabaseProvider() environment.DatabaseProvider {
	if bound, ok := z.inner.(driver.DatabaseProviderBoundContractQueryExecutor); ok {
		return bound.DatabaseProvider()
	}
	return environment.ProviderPostgres
}

func (z zeroFirstLineQuantity) ExecuteContractQuery(
	ctx context.Context,
	target realization.ContractQueryTarget,
	queryContract database.QueryContract,
	parameters map[string]any,
) (driver.ContractQueryResult, error) {
	held, err := z.inner.ExecuteContractQuery(ctx, target, queryContract, parameters)
	if err != nil {
		return driver.ContractQueryResult{}, err
	}
	for index := range held.Rows {
		if index != 0 {
			continue
		}
		attributes := held.Rows[index].Attributes
		if _, ok := attributes[core.MustCanonicalID("sales.order_line.quantity")]; ok {
			attributes[core.MustCanonicalID("sales.order_line.quantity")] = "0"
		}
	}
	return held, nil
}

// TestReadScenarioReportsAnUnsatisfiedLine checks that a failing value reaches
// the answer surface: the declared positive rule is not-satisfied for the line
// that holds 0, and the reader does not report the other line as failing.
//
// The expected answer is written from the rule and the substituted value. The
// substitution stands in for a database whose first line is 0; it is not a
// product answer, and the second test keeps it out of the server composition.
func TestReadScenarioReportsAnUnsatisfiedLine(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	runner := newReferenceReadScenarioRunner(loaded.World)

	key := driver.ContractQueryExecutorRuntimeKey{
		Environment: core.MustCanonicalID("development"),
		Instance:    core.MustCanonicalID("sales_core.dev"),
		Capability:  core.MustCanonicalID("database.query.postgres"),
	}
	registry, ok := runner.contracts.(driver.ContractQueryExecutorMap)
	if !ok {
		t.Fatalf("contracts=%T", runner.contracts)
	}
	inner, held := registry[key]
	if !held {
		t.Fatal("the order-line executor binding is missing")
	}
	registry[key] = zeroFirstLineQuantity{inner: inner}

	result, err := runner.Run(context.Background(), readScenarioInput{
		From:     "2026-09-18T08:30:00Z",
		RowIndex: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RowAnswers) == 0 {
		t.Fatal("no row answers")
	}

	found := false
	for _, answer := range result.RowAnswers[0].Answers {
		if answer.Rule != "sales.order_line.quantity_positive" {
			continue
		}
		found = true
		if answer.Result != "not-satisfied" {
			t.Fatalf("quantity 0 was not reported as not-satisfied: %+v", answer)
		}
	}
	if !found {
		t.Fatalf("the positive rule was not asked of the first line: %+v", result.RowAnswers[0].Answers)
	}
	for _, answer := range result.RowAnswers[1].Answers {
		if answer.Rule == "sales.order_line.quantity_positive" && answer.Result != "satisfied" {
			t.Fatalf("a line holding 1 was reported as %q: %+v", answer.Result, answer)
		}
	}
}

// TestZeroFirstLineQuantityStaysOutOfTheServerComposition keeps the claim
// honest: the substitution above is a test stand-in, so the composition the
// server builds must not contain it.
func TestZeroFirstLineQuantityStaysOutOfTheServerComposition(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	registry := newReferenceReadScenarioRunner(loaded.World).contracts
	mapped, ok := registry.(driver.ContractQueryExecutorMap)
	if !ok {
		t.Fatalf("contracts=%T", registry)
	}
	for key, executor := range mapped {
		if _, substituted := executor.(zeroFirstLineQuantity); substituted {
			t.Fatalf("the substitution leaked into the server composition at %+v", key)
		}
	}
}

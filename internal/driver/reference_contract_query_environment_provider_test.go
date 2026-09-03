package driver_test

import (
	"context"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/environment"
	"tei/internal/realization"
)

type providerBoundContractQueryExecutor struct {
	provider environment.DatabaseProvider
	called   bool
}

func (e *providerBoundContractQueryExecutor) DatabaseProvider() environment.DatabaseProvider {
	return e.provider
}

func (e *providerBoundContractQueryExecutor) ExecuteContractQuery(
	_ context.Context,
	_ realization.ContractQueryTarget,
	_ database.QueryContract,
	_ map[string]any,
) (driver.ContractQueryResult, error) {
	e.called = true
	return driver.ContractQueryResult{
		Rows: []driver.ContractQueryRow{
			{Attributes: map[core.CanonicalID]any{
				"customer.customer.customer_id": int64(42),
			}},
		},
	}, nil
}

func TestExecuteReferenceContractQueryInEnvironmentRejectsRuntimeExecutorProviderMismatch(t *testing.T) {
	world := contractQueryEnvironmentWorld()
	executor := &providerBoundContractQueryExecutor{provider: environment.ProviderPostgres}

	_, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
		context.Background(),
		"development",
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		driver.ContractQueryExecutorMap{"database.query.oracle": executor},
	)
	if err == nil || !strings.Contains(err.Error(), "runtime Executor provider postgres") || !strings.Contains(err.Error(), "Database instance master_data.devのprovider oracle") {
		t.Fatalf("Capability IDに誤登録された別providerのruntime Executorを実行しています: %v", err)
	}
	if executor.called {
		t.Fatal("provider不一致を検出する前にruntime Executorを呼び出しています")
	}
}

func TestExecuteReferenceContractQueryInEnvironmentAcceptsMatchingRuntimeExecutorProvider(t *testing.T) {
	world := contractQueryEnvironmentWorld()
	executor := &providerBoundContractQueryExecutor{provider: environment.ProviderOracle}

	result, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
		context.Background(),
		"development",
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		driver.ContractQueryExecutorMap{"database.query.oracle": executor},
	)
	if err != nil {
		t.Fatalf("Database instanceと同じproviderのruntime Executorを実行できません: %v", err)
	}
	if !executor.called {
		t.Fatal("provider整合を確認したruntime Executorが呼び出されていません")
	}
	if result.Executor.Instance.Provider != environment.ProviderOracle {
		t.Fatalf("解決済みDatabase instance providerが変わっています: %#v", result.Executor.Instance)
	}
}

func TestExecuteReferenceContractQueryInEnvironmentRejectsGenericRuntimeExecutorProvider(t *testing.T) {
	world := contractQueryEnvironmentWorld()
	executor := &providerBoundContractQueryExecutor{provider: environment.ProviderGeneric}

	_, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
		context.Background(),
		"development",
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		driver.ContractQueryExecutorMap{"database.query.oracle": executor},
	)
	if err == nil || !strings.Contains(err.Error(), "provider \"generic\"は実行に使用できません") {
		t.Fatalf("具体providerを持たないprovider-bound Executorを実行しています: %v", err)
	}
	if executor.called {
		t.Fatal("generic providerを拒否する前にruntime Executorを呼び出しています")
	}
}

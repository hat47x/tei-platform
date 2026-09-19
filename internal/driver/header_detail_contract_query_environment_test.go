package driver_test

import (
	"context"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	sqlquery "tei/internal/driver/queryruntime/sql"
	"tei/internal/environment"
)

func headerDetailExecutorRegistry(executor driver.ContractQueryExecutor) driver.ContractQueryExecutorMap {
	return driver.ContractQueryExecutorMap{
		{
			Environment: "development",
			Instance:    "sales_core.dev",
			Capability:  "database.query.postgres",
		}: executor,
	}
}

func TestExecuteHeaderDetailContractQueryInEnvironmentUsesExplicitPostgreSQLComposition(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	executor := &recordingContractQueryExecutor{
		result: driver.ContractQueryResult{
			Rows: []driver.ContractQueryRow{
				{Attributes: map[core.CanonicalID]any{"sales.order_line.order_line_id": int64(11)}},
				{Attributes: map[core.CanonicalID]any{"sales.order_line.order_line_id": int64(12)}},
			},
		},
	}

	result, err := driver.NewResolver(world).ExecuteHeaderDetailContractQueryInEnvironment(
		context.Background(),
		"development",
		"order_entry",
		"order_entry.order_lines",
		orderHeaderMappedResult(),
		0,
		headerDetailExecutorRegistry(executor),
	)
	if err != nil {
		t.Fatalf("Environment上の明示compositionからheader-detail Contract Queryを実行できません: %v", err)
	}
	if result.Environment.ID != "development" || result.Executor.Instance.ID != "sales_core.dev" {
		t.Fatalf("Environment / Database instance identityが変わっています: %#v", result)
	}
	if result.Executor.Capability.Capability.ID != "database.query.postgres" {
		t.Fatalf("明示Bindingで選んだPostgreSQL executor Capabilityが変わっています: %#v", result.Executor.Capability)
	}
	if executor.target.Identity() != "sales_core/sales_core.order_line/find_by_order" {
		t.Fatalf("runtime Executorへ渡したContract Query targetが変わっています: %s", executor.target.Identity())
	}
	if len(executor.parameters) != 1 || executor.parameters["order_id"] != int64(1001) {
		t.Fatalf("Order source attributeから準備したruntime parameterが変わっています: %#v", executor.parameters)
	}
	if len(result.Execution.Result.Rows) != 2 {
		t.Fatalf("many resultがEnvironment composition後に変わっています: %#v", result.Execution.Result.Rows)
	}
}

func TestExecuteHeaderDetailContractQueryInEnvironmentDoesNotInferExecutorFromPostgresProvider(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	filtered := world.ContractQueryExecutorBindings[:0]
	for _, binding := range world.ContractQueryExecutorBindings {
		if binding.Target.Identity() == "sales_core/sales_core.order_line/find_by_order" {
			continue
		}
		filtered = append(filtered, binding)
	}
	world.ContractQueryExecutorBindings = filtered

	_, err := driver.NewResolver(world).ExecuteHeaderDetailContractQueryInEnvironment(
		context.Background(),
		"development",
		"order_entry",
		"order_entry.order_lines",
		orderHeaderMappedResult(),
		0,
		headerDetailExecutorRegistry(&recordingContractQueryExecutor{}),
	)
	if err == nil || !strings.Contains(err.Error(), "executor Bindingがありません") {
		t.Fatalf("PostgreSQL providerやRegistryからexecutorを暗黙選択しています: %v", err)
	}
}

func TestExecuteHeaderDetailContractQueryInEnvironmentRequiresInstanceAwareRegistryEntry(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)

	_, err := driver.NewResolver(world).ExecuteHeaderDetailContractQueryInEnvironment(
		context.Background(),
		"development",
		"order_entry",
		"order_entry.order_lines",
		orderHeaderMappedResult(),
		0,
		driver.ContractQueryExecutorMap{},
	)
	if err == nil || !strings.Contains(err.Error(), "sales_core.dev") || !strings.Contains(err.Error(), "runtime ExecutorがRegistryにありません") {
		t.Fatalf("未登録のDatabase instance向けruntime Executorを別実装へfallbackしています: %v", err)
	}
}

func TestExecuteHeaderDetailContractQueryInEnvironmentRejectsRuntimeProviderMismatch(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	wrongProvider := sqlquery.Executor{Provider: environment.ProviderOracle}

	_, err := driver.NewResolver(world).ExecuteHeaderDetailContractQueryInEnvironment(
		context.Background(),
		"development",
		"order_entry",
		"order_entry.order_lines",
		orderHeaderMappedResult(),
		0,
		headerDetailExecutorRegistry(wrongProvider),
	)
	if err == nil || !strings.Contains(err.Error(), "provider oracle") || !strings.Contains(err.Error(), "provider postgres") {
		t.Fatalf("Database instanceと異なるproviderのruntime Executorを許可しています: %v", err)
	}
}

func TestExecuteHeaderDetailContractQueryInEnvironmentDoesNotInferDatabaseInstance(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	for envIndex := range world.Environments {
		if world.Environments[envIndex].ID != "development" {
			continue
		}
		bindings := world.Environments[envIndex].Bindings[:0]
		for _, binding := range world.Environments[envIndex].Bindings {
			if binding.Application == "order_entry" && binding.Alias == "sales" {
				continue
			}
			bindings = append(bindings, binding)
		}
		world.Environments[envIndex].Bindings = bindings
	}

	_, err := driver.NewResolver(world).ExecuteHeaderDetailContractQueryInEnvironment(
		context.Background(),
		"development",
		"order_entry",
		"order_entry.order_lines",
		orderHeaderMappedResult(),
		0,
		headerDetailExecutorRegistry(&recordingContractQueryExecutor{}),
	)
	if err == nil || !strings.Contains(err.Error(), "Database instance Bindingがありません") {
		t.Fatalf("Database definitionやproviderからsales_core.devを暗黙選択しています: %v", err)
	}
}

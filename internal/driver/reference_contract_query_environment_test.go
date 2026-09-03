package driver_test

import (
	"context"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/plugin"
	"tei/internal/realization"
)

func contractQueryEnvironmentWorld() (world ir.World) {
	world = referenceParameterWorld()
	world.Environments = []environment.Environment{
		{
			ID: "development",
			Plugins: []environment.ExtensionRequirement{
				{ID: "tei.database.oracle", CompatibleMajor: 1},
			},
			Instances: []environment.DatabaseInstance{
				{ID: "master_data.dev", Definition: "master_data", Provider: environment.ProviderOracle},
			},
			Bindings: []environment.ApplicationDatabaseBinding{
				{Application: "order_entry", Alias: "master", Instance: "master_data.dev"},
			},
		},
	}
	world.Plugins = []plugin.Definition{
		{
			ID:      "tei.database.oracle",
			Version: "1.0.0",
			Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
			Provides: []plugin.Capability{
				{
					Kind:              plugin.CapabilityContractQueryExecutor,
					ID:                "database.query.oracle",
					DatabaseProviders: []environment.DatabaseProvider{environment.ProviderOracle},
				},
			},
		},
	}
	target := realization.ContractQueryTarget{
		Database: "master_data",
		Contract: "master_data.customer",
		Query:    "find_by_id",
	}
	world.ContractQueryExecutorBindings = []realization.ContractQueryExecutorBinding{
		{
			Environment: "development",
			Instance:    "master_data.dev",
			Target:      target,
			Executor:    "database.query.oracle",
		},
	}
	return world
}

func contractQueryExecutorRegistry(
	instanceID core.CanonicalID,
	executor driver.ContractQueryExecutor,
) driver.ContractQueryExecutorMap {
	return driver.ContractQueryExecutorMap{
		{
			Environment: "development",
			Instance:    instanceID,
			Capability:  "database.query.oracle",
		}: executor,
	}
}

func TestExecuteReferenceContractQueryInEnvironmentUsesOnlyExplicitComposition(t *testing.T) {
	world := contractQueryEnvironmentWorld()
	executor := &recordingContractQueryExecutor{
		result: driver.ContractQueryResult{
			Rows: []driver.ContractQueryRow{
				{Attributes: map[core.CanonicalID]any{
					"customer.customer.customer_id": int64(42),
				}},
			},
		},
	}
	registry := contractQueryExecutorRegistry("master_data.dev", executor)

	result, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
		context.Background(),
		"development",
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		registry,
	)
	if err != nil {
		t.Fatalf("Environment上の明示compositionからreference Contract Queryを実行できません: %v", err)
	}
	if result.Environment.ID != "development" {
		t.Fatalf("Environment identityが変わっています: %#v", result.Environment)
	}
	if result.Executor.Instance.ID != "master_data.dev" {
		t.Fatalf("Application datasourceから解決したDatabase instanceが変わっています: %#v", result.Executor.Instance)
	}
	if result.Executor.Capability.Capability.ID != "database.query.oracle" {
		t.Fatalf("明示Bindingで選んだexecutor Capabilityが変わっています: %#v", result.Executor.Capability)
	}
	if executor.target.Identity() != "master_data/master_data.customer/find_by_id" {
		t.Fatalf("runtime Executorへ渡したContract Query targetが変わっています: %s", executor.target.Identity())
	}
	if len(executor.parameters) != 1 || executor.parameters["id"] != int64(42) {
		t.Fatalf("Relation keyから準備したruntime parameterを保持できていません: %#v", executor.parameters)
	}
	if len(result.Execution.Result.Rows) != 1 || result.Execution.Result.Rows[0].Attributes["customer.customer.customer_id"] != int64(42) {
		t.Fatalf("Semantic result rowが変わっています: %#v", result.Execution.Result)
	}
}

func TestExecuteReferenceContractQueryInEnvironmentDoesNotInferDatabaseInstance(t *testing.T) {
	world := contractQueryEnvironmentWorld()
	world.Environments[0].Bindings = nil
	executor := &recordingContractQueryExecutor{}

	_, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
		context.Background(),
		"development",
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		contractQueryExecutorRegistry("master_data.dev", executor),
	)
	if err == nil || !strings.Contains(err.Error(), "Database instance Bindingがありません") {
		t.Fatalf("Database definitionやproviderからinstanceを暗黙選択しています: %v", err)
	}
}

func TestExecuteReferenceContractQueryInEnvironmentDoesNotInferExecutorFromProvider(t *testing.T) {
	world := contractQueryEnvironmentWorld()
	world.ContractQueryExecutorBindings = nil
	executor := &recordingContractQueryExecutor{}

	_, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
		context.Background(),
		"development",
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		contractQueryExecutorRegistry("master_data.dev", executor),
	)
	if err == nil || !strings.Contains(err.Error(), "executor Bindingがありません") {
		t.Fatalf("Oracle providerやRegistryからexecutorを暗黙選択しています: %v", err)
	}
}

func TestExecuteReferenceContractQueryInEnvironmentRequiresRuntimeExecutorRegistration(t *testing.T) {
	world := contractQueryEnvironmentWorld()

	_, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
		context.Background(),
		"development",
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		driver.ContractQueryExecutorMap{},
	)
	if err == nil || !strings.Contains(err.Error(), "master_data.dev") || !strings.Contains(err.Error(), "runtime ExecutorがRegistryにありません") {
		t.Fatalf("未登録のDatabase instance向けruntime Executorを別実装へfallbackしています: %v", err)
	}
}

func TestExecuteReferenceContractQueryInEnvironmentDoesNotFallbackAcrossDatabaseInstances(t *testing.T) {
	world := contractQueryEnvironmentWorld()
	world.Environments[0].Instances = append(
		world.Environments[0].Instances,
		environment.DatabaseInstance{ID: "master_data.shadow", Definition: "master_data", Provider: environment.ProviderOracle},
	)
	wrongExecutor := &recordingContractQueryExecutor{}
	registry := contractQueryExecutorRegistry("master_data.shadow", wrongExecutor)

	_, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
		context.Background(),
		"development",
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		registry,
	)
	if err == nil || !strings.Contains(err.Error(), "master_data.dev") || !strings.Contains(err.Error(), "runtime ExecutorがRegistryにありません") {
		t.Fatalf("同じCapabilityを持つ別Database instanceのruntime Executorへfallbackしています: %v", err)
	}
	if wrongExecutor.target.Valid() {
		t.Fatalf("別Database instance向けruntime Executorを呼び出しています: %#v", wrongExecutor.target)
	}
}

func TestExecuteReferenceContractQueryInEnvironmentRejectsBindingFromAnotherEnvironment(t *testing.T) {
	world := contractQueryEnvironmentWorld()
	world.ContractQueryExecutorBindings[0].Environment = "production"

	_, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
		context.Background(),
		"development",
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		contractQueryExecutorRegistry("master_data.dev", &recordingContractQueryExecutor{}),
	)
	if err == nil || !strings.Contains(err.Error(), "Environment development") || !strings.Contains(err.Error(), "executor Bindingがありません") {
		t.Fatalf("別Environmentのexecutor Bindingを流用しています: %v", err)
	}
}

func TestExecuteReferenceContractQueryInEnvironmentRejectsUnknownEnvironment(t *testing.T) {
	world := contractQueryEnvironmentWorld()

	_, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
		context.Background(),
		"production",
		"order_entry",
		"order_entry.customer_reference",
		customerMappedResult(),
		0,
		contractQueryExecutorRegistry("master_data.dev", &recordingContractQueryExecutor{}),
	)
	if err == nil || !strings.Contains(err.Error(), "Environment productionが見つかりません") {
		t.Fatalf("存在しないEnvironmentを暗黙補完しています: %v", err)
	}
}

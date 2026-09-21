package driver_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/environment"
	"tei/internal/plugin"
	"tei/internal/realization"
)

// ContractQueryExecutorRuntimeKeyは、runtime ExecutorをEnvironmentとDatabase instanceへ
// 明示的に結び付ける。Capability IDだけへ縮退すると接続先を取り違え得る、というのが型の主旨である。
//
// instance成分とbinding解決側のEnvironment判定は既存testが守っていたが、
// **runtime key のEnvironment成分**を固定するtestが無く、落としても既存suiteが通ってしまっていた。
func resolvedContractQueryExecutor(bindingEnvironment, instance, capability string) realization.ResolvedContractQueryExecutor {
	return realization.ResolvedContractQueryExecutor{
		Binding: realization.ContractQueryExecutorBinding{
			Environment: core.CanonicalID(bindingEnvironment),
			Instance:    core.CanonicalID(instance),
			Target: realization.ContractQueryTarget{
				Database: "master_data",
				Contract: "master_data.customer",
				Query:    "find_by_id",
			},
			Executor: core.CanonicalID(capability),
		},
		Instance:   environment.DatabaseInstance{ID: core.CanonicalID(instance), Definition: "master_data", Provider: "oracle"},
		Capability: plugin.ResolvedCapability{Capability: plugin.Capability{ID: core.CanonicalID(capability)}},
	}
}

func TestContractQueryExecutorMapRequiresTheExactEnvironmentInstanceAndCapability(t *testing.T) {
	registered := driver.ContractQueryExecutorRuntimeKey{
		Environment: "development",
		Instance:    "master_data.dev",
		Capability:  "database.query.oracle",
	}
	executor := &recordingContractQueryExecutor{}
	registry := driver.ContractQueryExecutorMap{registered: executor}

	if got, ok := registry.ExecutorForContractQuery(
		resolvedContractQueryExecutor("development", "master_data.dev", "database.query.oracle"),
	); !ok || got != executor {
		t.Fatal("一致するkeyでruntime Executorを取得できません")
	}

	for name, selected := range map[string]realization.ResolvedContractQueryExecutor{
		"different environment": resolvedContractQueryExecutor("production", "master_data.dev", "database.query.oracle"),
		"different instance":    resolvedContractQueryExecutor("development", "master_data.prod", "database.query.oracle"),
		"different capability":  resolvedContractQueryExecutor("development", "master_data.dev", "database.query.postgres"),
	} {
		t.Run(name, func(t *testing.T) {
			if _, ok := registry.ExecutorForContractQuery(selected); ok {
				t.Fatal("一致しない解決結果へruntime Executorを返しています")
			}
		})
	}

	// registryにnilを入れた場合も「見つかった」にしない。
	registry[registered] = nil
	if _, ok := registry.ExecutorForContractQuery(
		resolvedContractQueryExecutor("development", "master_data.dev", "database.query.oracle"),
	); ok {
		t.Fatal("nil executorを解決済みとして返しています")
	}
}

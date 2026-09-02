package realization

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
)

func contractQueryBindingFixture() (environment.Environment, plugin.ResolvedSet, ContractQueryTarget, []ContractQueryExecutorBinding) {
	env := environment.Environment{
		ID: "development",
		Instances: []environment.DatabaseInstance{
			{ID: "master_data.dev", Definition: "master_data", Provider: environment.ProviderOracle},
		},
	}
	resolved := plugin.ResolvedSet{
		Environment: "development",
		Capabilities: []plugin.ResolvedCapability{
			{
				Plugin: plugin.Identity{ID: "tei.database.oracle", Version: "1.0.0"},
				Capability: plugin.Capability{
					Kind:              plugin.CapabilityContractQueryExecutor,
					ID:                "database.query.oracle",
					DatabaseProviders: []environment.DatabaseProvider{environment.ProviderOracle},
				},
			},
		},
	}
	target := ContractQueryTarget{
		Database: "master_data",
		Contract: "master_data.customer",
		Query:    "find_by_id",
	}
	bindings := []ContractQueryExecutorBinding{
		{Instance: "master_data.dev", Target: target, Executor: "database.query.oracle"},
	}
	return env, resolved, target, bindings
}

func TestResolveContractQueryExecutorUsesExplicitBinding(t *testing.T) {
	env, plugins, target, bindings := contractQueryBindingFixture()

	resolved, err := ResolveContractQueryExecutor(env, plugins, bindings, "master_data.dev", target)
	if err != nil {
		t.Fatalf("明示BindingからContract Query executorを解決できません: %v", err)
	}
	if resolved.Instance.ID != "master_data.dev" {
		t.Fatalf("Database instanceが変わっています: %#v", resolved.Instance)
	}
	if resolved.Capability.Capability.ID != "database.query.oracle" {
		t.Fatalf("executor Capabilityが変わっています: %#v", resolved.Capability)
	}
}

func TestResolveContractQueryExecutorDoesNotInferFromProvider(t *testing.T) {
	env, plugins, target, _ := contractQueryBindingFixture()

	_, err := ResolveContractQueryExecutor(env, plugins, nil, "master_data.dev", target)
	if err == nil || !strings.Contains(err.Error(), "Bindingがありません") {
		t.Fatalf("Oracle対応Capabilityの存在だけからexecutorを暗黙選択しています: %v", err)
	}
}

func TestResolveContractQueryExecutorRejectsDuplicateBindings(t *testing.T) {
	env, plugins, target, bindings := contractQueryBindingFixture()
	bindings = append(bindings, bindings[0])

	_, err := ResolveContractQueryExecutor(env, plugins, bindings, "master_data.dev", target)
	if err == nil || !strings.Contains(err.Error(), "Bindingが複数") {
		t.Fatalf("同じinstanceとtargetへの複数Bindingを受け入れています: %v", err)
	}
}

func TestResolveContractQueryExecutorRejectsDifferentDatabaseDefinition(t *testing.T) {
	env, plugins, target, bindings := contractQueryBindingFixture()
	target.Database = "sales_core"
	bindings[0].Target = target

	_, err := ResolveContractQueryExecutor(env, plugins, bindings, "master_data.dev", target)
	if err == nil || !strings.Contains(err.Error(), "master_dataを実現") || !strings.Contains(err.Error(), "sales_coreに属しています") {
		t.Fatalf("Database instanceとContract Query targetのdefinition不一致を拒否していません: %v", err)
	}
}

func TestResolveContractQueryExecutorRejectsUnsupportedProvider(t *testing.T) {
	env, plugins, target, bindings := contractQueryBindingFixture()
	plugins.Capabilities[0].Capability.DatabaseProviders = []environment.DatabaseProvider{environment.ProviderPostgres}

	_, err := ResolveContractQueryExecutor(env, plugins, bindings, "master_data.dev", target)
	if err == nil || !strings.Contains(err.Error(), "provider oracleをサポートしていません") {
		t.Fatalf("executor Capabilityのprovider不一致を拒否していません: %v", err)
	}
}

func TestResolveContractQueryExecutorRequiresContractQueryCapabilityKind(t *testing.T) {
	env, plugins, target, bindings := contractQueryBindingFixture()
	plugins.Capabilities[0].Capability.Kind = plugin.CapabilityDatabaseProvider

	_, err := ResolveContractQueryExecutor(env, plugins, bindings, "master_data.dev", target)
	if err == nil || !strings.Contains(err.Error(), "一意に取得できません") {
		t.Fatalf("database-provider CapabilityをContract Query executorとして流用しています: %v", err)
	}
}

func TestResolveContractQueryExecutorRejectsIncompleteTarget(t *testing.T) {
	env, plugins, _, bindings := contractQueryBindingFixture()
	target := ContractQueryTarget{Database: core.CanonicalID("master_data")}

	_, err := ResolveContractQueryExecutor(env, plugins, bindings, "master_data.dev", target)
	if err == nil || !strings.Contains(err.Error(), "target identityが不完全") {
		t.Fatalf("不完全なContract Query targetを受け入れています: %v", err)
	}
}

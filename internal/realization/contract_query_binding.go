package realization

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
)

// ContractQueryExecutorBindingは、Environment上のDatabase instanceと
// Database-ownedなContract Query targetを、一つのexecutor Capabilityへ明示的に結び付ける。
//
// provider一致だけでexecutorを自動選択しない。どのinstanceで、どのContract Queryを、
// どのCapabilityへ渡すかをauthoring側の明示情報として保持する。
type ContractQueryExecutorBinding struct {
	Instance core.CanonicalID    `json:"instance"`
	Target   ContractQueryTarget `json:"target"`
	Executor core.CanonicalID    `json:"executor"`
}

// ResolvedContractQueryExecutorは、実行時に利用するDatabase instanceと
// Plugin Capabilityが明示Bindingから一意に解決できた結果を表す。
type ResolvedContractQueryExecutor struct {
	Binding    ContractQueryExecutorBinding `json:"binding"`
	Instance   environment.DatabaseInstance `json:"instance"`
	Capability plugin.ResolvedCapability    `json:"capability"`
}

// ResolveContractQueryExecutorは、指定されたinstanceとtargetに完全一致するBindingを一つだけ選ぶ。
//
// providerが一致するCapabilityを探索して暗黙に選択することはない。Bindingがない場合や複数ある場合は
// 未解決として返し、Database definitionとtarget Databaseの不一致、Capability kindやproviderの不一致も拒否する。
func ResolveContractQueryExecutor(
	env environment.Environment,
	plugins plugin.ResolvedSet,
	bindings []ContractQueryExecutorBinding,
	instanceID core.CanonicalID,
	target ContractQueryTarget,
) (ResolvedContractQueryExecutor, error) {
	if instanceID == "" {
		return ResolvedContractQueryExecutor{}, fmt.Errorf("Contract Query executorを解決するDatabase instanceが指定されていません")
	}
	if !target.Valid() {
		return ResolvedContractQueryExecutor{}, fmt.Errorf("Contract Query executorを解決するtarget identityが不完全です")
	}

	instance, ok := env.InstanceByID(instanceID)
	if !ok {
		return ResolvedContractQueryExecutor{}, fmt.Errorf("Database instance %sはEnvironment %sに存在しません", instanceID, env.ID)
	}
	if instance.Definition != target.Database {
		return ResolvedContractQueryExecutor{}, fmt.Errorf(
			"Database instance %sは%sを実現しますが、Contract Query targetは%sに属しています",
			instanceID,
			instance.Definition,
			target.Database,
		)
	}

	var matched *ContractQueryExecutorBinding
	for i := range bindings {
		binding := bindings[i]
		if binding.Instance != instanceID || binding.Target != target {
			continue
		}
		if matched != nil {
			return ResolvedContractQueryExecutor{}, fmt.Errorf(
				"Database instance %sのContract Query %sにexecutor Bindingが複数あります",
				instanceID,
				target.Identity(),
			)
		}
		copy := binding
		matched = &copy
	}
	if matched == nil {
		return ResolvedContractQueryExecutor{}, fmt.Errorf(
			"Database instance %sのContract Query %sにexecutor Bindingがありません",
			instanceID,
			target.Identity(),
		)
	}
	if matched.Executor == "" {
		return ResolvedContractQueryExecutor{}, fmt.Errorf(
			"Database instance %sのContract Query %sに指定されたexecutor Capabilityが空です",
			instanceID,
			target.Identity(),
		)
	}

	capability, ok := plugins.Capability(plugin.CapabilityContractQueryExecutor, matched.Executor)
	if !ok {
		return ResolvedContractQueryExecutor{}, fmt.Errorf(
			"Contract Query executor Capability %sをEnvironment %sの解決済みPluginから一意に取得できません",
			matched.Executor,
			env.ID,
		)
	}
	if !capability.Capability.SupportsDatabaseProvider(instance.Provider) {
		return ResolvedContractQueryExecutor{}, fmt.Errorf(
			"Contract Query executor Capability %sはDatabase instance %sのprovider %sをサポートしていません",
			matched.Executor,
			instanceID,
			instance.Provider,
		)
	}

	return ResolvedContractQueryExecutor{
		Binding:    *matched,
		Instance:   instance,
		Capability: capability,
	}, nil
}

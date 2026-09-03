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
// provider一致だけでexecutorを自動選択しない。どのEnvironmentの、どのinstanceで、
// どのContract Queryを、どのCapabilityへ渡すかをauthoring側の明示情報として保持する。
type ContractQueryExecutorBinding struct {
	Environment core.CanonicalID    `json:"environment"`
	Instance    core.CanonicalID    `json:"instance"`
	Target      ContractQueryTarget `json:"target"`
	Executor    core.CanonicalID    `json:"executor"`
}

// Identityは、Bindingの選択対象を一意に示す安定identityを返す。
// executor Capability自体は選択結果なのでidentityへ含めず、同じEnvironment / instance / targetへ
// 複数executorを宣言した場合は同一Bindingの競合として検出できるようにする。
func (b ContractQueryExecutorBinding) Identity() string {
	return string(b.Environment) + "/" + string(b.Instance) + "/" + b.Target.Identity()
}

// ResolvedContractQueryExecutorは、実行時に利用するDatabase instanceと
// Plugin Capabilityが明示Bindingから一意に解決できた結果を表す。
type ResolvedContractQueryExecutor struct {
	Binding    ContractQueryExecutorBinding `json:"binding"`
	Instance   environment.DatabaseInstance `json:"instance"`
	Capability plugin.ResolvedCapability    `json:"capability"`
}

// ResolveContractQueryExecutorは、指定されたEnvironment、instance、targetに完全一致するBindingを一つだけ選ぶ。
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
	if env.ID == "" {
		return ResolvedContractQueryExecutor{}, fmt.Errorf("Contract Query executorを解決するEnvironmentが指定されていません")
	}
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
		if binding.Environment != env.ID || binding.Instance != instanceID || binding.Target != target {
			continue
		}
		if matched != nil {
			return ResolvedContractQueryExecutor{}, fmt.Errorf(
				"Environment %sのDatabase instance %sにあるContract Query %sへexecutor Bindingが複数あります",
				env.ID,
				instanceID,
				target.Identity(),
			)
		}
		copy := binding
		matched = &copy
	}
	if matched == nil {
		return ResolvedContractQueryExecutor{}, fmt.Errorf(
			"Environment %sのDatabase instance %sにあるContract Query %sへexecutor Bindingがありません",
			env.ID,
			instanceID,
			target.Identity(),
		)
	}
	if matched.Executor == "" {
		return ResolvedContractQueryExecutor{}, fmt.Errorf(
			"Environment %sのDatabase instance %sにあるContract Query %sへ指定されたexecutor Capabilityが空です",
			env.ID,
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

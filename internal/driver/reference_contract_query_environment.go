package driver

import (
	"context"
	"fmt"

	"tei/internal/core"
	"tei/internal/driver/queryresult"
	"tei/internal/environment"
	"tei/internal/plugin"
	"tei/internal/realization"
)

// ContractQueryExecutorRegistryは、明示的に解決されたexecutor Capabilityを
// 実行可能なContractQueryExecutorへ接続するruntime registryである。
//
// provider名やPlugin名から実装を推測せず、Capability IDだけを明示keyとして利用する。
type ContractQueryExecutorRegistry interface {
	ExecutorByCapability(core.CanonicalID) (ContractQueryExecutor, bool)
}

// ContractQueryExecutorMapは、単純なruntime compositionやConformanceで利用できるregistry実装である。
type ContractQueryExecutorMap map[core.CanonicalID]ContractQueryExecutor

func (m ContractQueryExecutorMap) ExecutorByCapability(id core.CanonicalID) (ContractQueryExecutor, bool) {
	executor, ok := m[id]
	return executor, ok && executor != nil
}

// ReferenceContractQueryEnvironmentExecutionは、Environment固有のruntime compositionを経て
// 実行されたreference Contract Queryの結果を表す。
//
// runtime parameter値は保持せず、どのEnvironment / Database instance / executor Capabilityを
// 明示的に選択したかと、通常のContract Query実行結果だけを残す。
type ReferenceContractQueryEnvironmentExecution struct {
	Environment environment.Environment                    `json:"environment"`
	Executor    realization.ResolvedContractQueryExecutor `json:"executor"`
	Execution   ReferenceContractQueryExecution            `json:"execution"`
}

// ExecuteReferenceContractQueryInEnvironmentは、reference Patternのlogical datasourceを
// Environment上のDatabase instanceへ明示Bindingし、そのinstanceとContract Query targetに対する
// executor Bindingを解決してから実行する。
//
// Environment、Application datasource、executor Capabilityのいずれも暗黙選択しない。
func (r Resolver) ExecuteReferenceContractQueryInEnvironment(
	ctx context.Context,
	environmentID, applicationID, patternID core.CanonicalID,
	source queryresult.MappedResult,
	rowIndex int,
	registry ContractQueryExecutorRegistry,
) (ReferenceContractQueryEnvironmentExecution, error) {
	if registry == nil {
		return ReferenceContractQueryEnvironmentExecution{}, fmt.Errorf("reference Pattern %sを実行するContract Query Executor Registryが指定されていません", patternID)
	}

	prepared, err := r.PrepareReferenceContractQuery(applicationID, patternID, source, rowIndex)
	if err != nil {
		return ReferenceContractQueryEnvironmentExecution{}, err
	}

	env, err := r.environmentByID(environmentID)
	if err != nil {
		return ReferenceContractQueryEnvironmentExecution{}, err
	}
	instanceID, err := applicationDatabaseInstance(env, applicationID, prepared.Resolution.Datasource.Alias)
	if err != nil {
		return ReferenceContractQueryEnvironmentExecution{}, fmt.Errorf(
			"reference Pattern %sのDatasource %sをEnvironment %sへ接続できません: %w",
			patternID,
			prepared.Resolution.Datasource.Alias,
			env.ID,
			err,
		)
	}

	resolvedPlugins, err := (plugin.Catalog{Plugins: r.world.Plugins, Bundles: r.world.Bundles}).ResolveEnvironment(env)
	if err != nil {
		return ReferenceContractQueryEnvironmentExecution{}, fmt.Errorf("Environment %sのPluginを解決できません: %w", env.ID, err)
	}

	target := realization.ContractQueryTarget{
		Database: prepared.Resolution.Database.ID,
		Contract: prepared.Resolution.Contract.ID,
		Query:    prepared.Resolution.Invocation.Query,
	}
	selected, err := realization.ResolveContractQueryExecutor(
		env,
		resolvedPlugins,
		r.world.ContractQueryExecutorBindings,
		instanceID,
		target,
	)
	if err != nil {
		return ReferenceContractQueryEnvironmentExecution{}, fmt.Errorf("reference Pattern %sのContract Query executorを解決できません: %w", patternID, err)
	}

	executorID := selected.Capability.Capability.ID
	executor, ok := registry.ExecutorByCapability(executorID)
	if !ok {
		return ReferenceContractQueryEnvironmentExecution{}, fmt.Errorf(
			"Contract Query executor Capability %sに対応するruntime ExecutorがRegistryにありません",
			executorID,
		)
	}

	execution, err := r.ExecuteReferenceContractQuery(ctx, applicationID, patternID, source, rowIndex, executor)
	if err != nil {
		return ReferenceContractQueryEnvironmentExecution{}, err
	}

	return ReferenceContractQueryEnvironmentExecution{
		Environment: env,
		Executor:    selected,
		Execution:   execution,
	}, nil
}

func (r Resolver) environmentByID(id core.CanonicalID) (environment.Environment, error) {
	if id == "" {
		return environment.Environment{}, fmt.Errorf("Environmentが指定されていません")
	}

	var found environment.Environment
	count := 0
	for _, candidate := range r.world.Environments {
		if candidate.ID != id {
			continue
		}
		found = candidate
		count++
	}
	if count == 0 {
		return environment.Environment{}, fmt.Errorf("Environment %sが見つかりません", id)
	}
	if count > 1 {
		return environment.Environment{}, fmt.Errorf("Environment %sが複数定義されています: %d件", id, count)
	}
	return found, nil
}

func applicationDatabaseInstance(
	env environment.Environment,
	applicationID core.CanonicalID,
	alias core.LocalName,
) (core.CanonicalID, error) {
	var instanceID core.CanonicalID
	count := 0
	for _, binding := range env.Bindings {
		if binding.Application != applicationID || binding.Alias != alias {
			continue
		}
		instanceID = binding.Instance
		count++
	}
	if count == 0 {
		return "", fmt.Errorf("Application %sのDatasource %sにDatabase instance Bindingがありません", applicationID, alias)
	}
	if count > 1 {
		return "", fmt.Errorf("Application %sのDatasource %sにDatabase instance Bindingが複数あります: %d件", applicationID, alias, count)
	}
	if _, ok := env.InstanceByID(instanceID); !ok {
		return "", fmt.Errorf("Binding先のDatabase instance %sがEnvironment %sに存在しません", instanceID, env.ID)
	}
	return instanceID, nil
}

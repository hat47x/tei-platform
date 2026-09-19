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

// HeaderDetailContractQueryEnvironmentExecutionは、Environment固有のruntime compositionを経て
// 実行されたheader-detail Contract Queryの結果を表す。
//
// runtime parameter値は保持せず、どのEnvironment / Database instance / executor Capabilityを
// 明示的に選択したかと、通常のContract Query実行結果だけを残す。
type HeaderDetailContractQueryEnvironmentExecution struct {
	Environment environment.Environment                   `json:"environment"`
	Executor    realization.ResolvedContractQueryExecutor `json:"executor"`
	Execution   HeaderDetailContractQueryExecution        `json:"execution"`
}

// ExecuteHeaderDetailContractQueryInEnvironmentは、header-detail traversalのlogical Datasourceを
// Environment上のDatabase instanceへ明示Bindingし、そのinstanceとContract Query targetに対する
// executor Bindingを解決してから実行する。
//
// Environment、Application datasource、executor Capability、runtime Database接続のいずれも暗黙選択しない。
func (r Resolver) ExecuteHeaderDetailContractQueryInEnvironment(
	ctx context.Context,
	environmentID, applicationID, patternID core.CanonicalID,
	source queryresult.MappedResult,
	rowIndex int,
	registry ContractQueryExecutorRegistry,
) (HeaderDetailContractQueryEnvironmentExecution, error) {
	if registry == nil {
		return HeaderDetailContractQueryEnvironmentExecution{}, fmt.Errorf("header-detail Pattern %sを実行するContract Query Executor Registryが指定されていません", patternID)
	}

	prepared, err := r.PrepareHeaderDetailContractQuery(applicationID, patternID, source, rowIndex)
	if err != nil {
		return HeaderDetailContractQueryEnvironmentExecution{}, err
	}

	env, err := r.environmentByID(environmentID)
	if err != nil {
		return HeaderDetailContractQueryEnvironmentExecution{}, err
	}
	instanceID, err := applicationDatabaseInstance(env, applicationID, prepared.Resolution.Datasource.Alias)
	if err != nil {
		return HeaderDetailContractQueryEnvironmentExecution{}, fmt.Errorf(
			"header-detail Pattern %sのDatasource %sをEnvironment %sへ接続できません: %w",
			patternID,
			prepared.Resolution.Datasource.Alias,
			env.ID,
			err,
		)
	}

	resolvedPlugins, err := (plugin.Catalog{Plugins: r.world.Plugins, Bundles: r.world.Bundles}).ResolveEnvironment(env)
	if err != nil {
		return HeaderDetailContractQueryEnvironmentExecution{}, fmt.Errorf("Environment %sのPluginを解決できません: %w", env.ID, err)
	}

	target := prepared.Resolution.Binding.Target
	selected, err := realization.ResolveContractQueryExecutor(
		env,
		resolvedPlugins,
		r.world.ContractQueryExecutorBindings,
		instanceID,
		target,
	)
	if err != nil {
		return HeaderDetailContractQueryEnvironmentExecution{}, fmt.Errorf("header-detail Pattern %sのContract Query executorを解決できません: %w", patternID, err)
	}

	executorID := selected.Capability.Capability.ID
	executor, ok := registry.ExecutorForContractQuery(selected)
	if !ok {
		return HeaderDetailContractQueryEnvironmentExecution{}, fmt.Errorf(
			"Environment %sのDatabase instance %s / Contract Query executor Capability %sに対応するruntime ExecutorがRegistryにありません",
			selected.Binding.Environment,
			selected.Instance.ID,
			executorID,
		)
	}
	if providerBound, ok := executor.(DatabaseProviderBoundContractQueryExecutor); ok {
		runtimeProvider := providerBound.DatabaseProvider()
		if !runtimeProvider.Known() || runtimeProvider == environment.ProviderGeneric {
			return HeaderDetailContractQueryEnvironmentExecution{}, fmt.Errorf(
				"Contract Query executor Capability %sに登録されたruntime ExecutorのDatabase provider %qは実行に使用できません",
				executorID,
				runtimeProvider,
			)
		}
		if runtimeProvider != selected.Instance.Provider {
			return HeaderDetailContractQueryEnvironmentExecution{}, fmt.Errorf(
				"Contract Query executor Capability %sのruntime Executor provider %sとDatabase instance %sのprovider %sが一致しません",
				executorID,
				runtimeProvider,
				selected.Instance.ID,
				selected.Instance.Provider,
			)
		}
	}

	execution, err := r.ExecuteHeaderDetailContractQuery(ctx, applicationID, patternID, source, rowIndex, executor)
	if err != nil {
		return HeaderDetailContractQueryEnvironmentExecution{}, err
	}

	return HeaderDetailContractQueryEnvironmentExecution{
		Environment: env,
		Executor:    selected,
		Execution:   execution,
	}, nil
}

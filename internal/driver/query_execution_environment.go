package driver

import (
	"context"
	"fmt"

	"tei/internal/core"
	"tei/internal/environment"
)

// DatabaseProviderBoundBrowseQueryExecutorは、特定のDatabase providerへ結び付いた
// BrowseQueryExecutorがruntime compositionへ自身のproviderを明示するための任意interfaceである。
//
// Fixture等のDatabase providerを持たないExecutorへ実装を強制しない。一方、SQL Executorのように
// provider固有である実装はこのinterfaceを実装し、Environment上のDatabase instance providerと
// 実行前に照合できるようにする。Contract Query側のDatabaseProviderBoundContractQueryExecutorと同じ規則である。
type DatabaseProviderBoundBrowseQueryExecutor interface {
	BrowseQueryExecutor
	DatabaseProvider() environment.DatabaseProvider
}

// BrowseQueryEnvironmentExecutionは、Environment上のDatabase instanceを明示的に選択してから
// 実行されたbrowse Queryの結果を表す。
//
// runtime parameter値や接続情報は保持せず、どのEnvironment / Database instance / providerを
// 選択したかと、通常のbrowse Query実行結果だけを残す。
type BrowseQueryEnvironmentExecution struct {
	Environment core.CanonicalID             `json:"environment"`
	Instance    environment.DatabaseInstance `json:"instance"`
	Execution   BrowseQueryExecution         `json:"execution"`
}

// ResolveBrowseQueryInstanceは、browse Patternが宣言するlogical DatasourceをEnvironment上の
// Database instanceへ解決し、Database定義・provider・Query dialectの整合を確認する。
// 接続先の選択だけを行い、DBへは接続しない。ExecuteBrowseQueryInEnvironmentと、実接続を
// 用意する呼出側(CLI等)が同じ解決結果を使えるよう公開する。
func (r Resolver) ResolveBrowseQueryInstance(
	environmentID, applicationID, patternID core.CanonicalID,
) (environment.Environment, environment.DatabaseInstance, PatternResolution, error) {
	resolved, err := r.ResolvePattern(applicationID, patternID)
	if err != nil {
		return environment.Environment{}, environment.DatabaseInstance{}, PatternResolution{}, err
	}
	if resolved.Query == nil || resolved.Datasource == nil {
		return environment.Environment{}, environment.DatabaseInstance{}, PatternResolution{}, fmt.Errorf("browse Pattern %sのQueryまたはDatasourceが解決されていません", patternID)
	}

	env, err := r.environmentByID(environmentID)
	if err != nil {
		return environment.Environment{}, environment.DatabaseInstance{}, PatternResolution{}, err
	}
	instanceID, err := applicationDatabaseInstance(env, applicationID, resolved.Datasource.Alias)
	if err != nil {
		return environment.Environment{}, environment.DatabaseInstance{}, PatternResolution{}, fmt.Errorf(
			"browse Pattern %sのDatasource %sをEnvironment %sへ接続できません: %w",
			patternID,
			resolved.Datasource.Alias,
			env.ID,
			err,
		)
	}
	instance, _ := env.InstanceByID(instanceID)
	if instance.Definition != resolved.Datasource.TargetDatabase {
		return environment.Environment{}, environment.DatabaseInstance{}, PatternResolution{}, fmt.Errorf(
			"Environment %sのDatabase instance %sの定義 %sとDatasource %sのtarget Database %sが一致しません",
			env.ID,
			instance.ID,
			instance.Definition,
			resolved.Datasource.Alias,
			resolved.Datasource.TargetDatabase,
		)
	}
	if !instance.Provider.Known() {
		return environment.Environment{}, environment.DatabaseInstance{}, PatternResolution{}, fmt.Errorf(
			"Environment %sのDatabase instance %sのprovider %qは実行に使用できません",
			env.ID,
			instance.ID,
			instance.Provider,
		)
	}
	if dialect := resolved.Query.Dialect; dialect != environment.ProviderGeneric && dialect != instance.Provider {
		return environment.Environment{}, environment.DatabaseInstance{}, PatternResolution{}, fmt.Errorf(
			"Query %sのdialect %sとEnvironment %sのDatabase instance %sのprovider %sが一致しません",
			resolved.Query.ID,
			dialect,
			env.ID,
			instance.ID,
			instance.Provider,
		)
	}
	return env, instance, resolved, nil
}

// ExecuteBrowseQueryInEnvironmentは、browse PatternのQueryが宣言するlogical Datasourceを
// Environment上のDatabase instanceへ明示Bindingで接続し、そのinstanceのDatabase定義とproviderが
// Datasource / Query dialect / runtime Executorと一致することを確認してからExecuteBrowseQueryへ委譲する。
//
// ExecuteBrowseQueryはEnvironmentを参照しないため、それだけではDatasource aliasが実際にどの
// Database instance / providerへ束縛されたかを実行経路上で確かめられない。この入口は、
// Environment、Binding、Database instance、providerのいずれも暗黙選択せず、
// 不一致や未束縛はExecutorを呼び出す前のerrorとして返す。正常な空結果へは変換しない。
func (r Resolver) ExecuteBrowseQueryInEnvironment(
	ctx context.Context,
	environmentID, applicationID, patternID core.CanonicalID,
	executor BrowseQueryExecutor,
	parameters map[string]any,
) (BrowseQueryEnvironmentExecution, error) {
	env, instance, _, err := r.ResolveBrowseQueryInstance(environmentID, applicationID, patternID)
	if err != nil {
		return BrowseQueryEnvironmentExecution{}, err
	}
	if providerBound, ok := executor.(DatabaseProviderBoundBrowseQueryExecutor); ok {
		runtimeProvider := providerBound.DatabaseProvider()
		if !runtimeProvider.Known() || runtimeProvider == environment.ProviderGeneric {
			return BrowseQueryEnvironmentExecution{}, fmt.Errorf(
				"browse Pattern %sのruntime ExecutorのDatabase provider %qは実行に使用できません",
				patternID,
				runtimeProvider,
			)
		}
		if runtimeProvider != instance.Provider {
			return BrowseQueryEnvironmentExecution{}, fmt.Errorf(
				"browse Pattern %sのruntime Executor provider %sとDatabase instance %sのprovider %sが一致しません",
				patternID,
				runtimeProvider,
				instance.ID,
				instance.Provider,
			)
		}
	}

	execution, err := r.ExecuteBrowseQuery(ctx, applicationID, patternID, executor, parameters)
	if err != nil {
		return BrowseQueryEnvironmentExecution{}, err
	}
	return BrowseQueryEnvironmentExecution{
		Environment: env.ID,
		Instance:    instance,
		Execution:   execution,
	}, nil
}

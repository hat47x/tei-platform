package driver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/query"
)

// BrowseQueryExecutorRuntimeKeyは、runtime Executorを一つのEnvironmentとDatabase instanceへ
// 明示的に結び付けるkeyである。Executorの選択にDatabase定義名・SQL本文・alias文字列を使わない。
type BrowseQueryExecutorRuntimeKey struct {
	Environment core.CanonicalID
	Instance    core.CanonicalID
}

// BrowseQueryExecutorRegistryは、解決済みのEnvironmentとDatabase instanceに対応する
// runtime Executorを返す。対応が無いときはExecutorを返さず、別instanceのExecutorで代替しない。
type BrowseQueryExecutorRegistry interface {
	ExecutorForBrowseQuery(environment, instance core.CanonicalID) (BrowseQueryExecutor, bool)
}

// BrowseQueryExecutorMapは、単純なcompositionと試験で使うregistry実装である。
type BrowseQueryExecutorMap map[BrowseQueryExecutorRuntimeKey]BrowseQueryExecutor

func (m BrowseQueryExecutorMap) ExecutorForBrowseQuery(environment, instance core.CanonicalID) (BrowseQueryExecutor, bool) {
	executor, ok := m[BrowseQueryExecutorRuntimeKey{Environment: environment, Instance: instance}]
	return executor, ok && executor != nil
}

// DatabaseProviderBoundBrowseQueryExecutorは、Executor自身がどのProviderの実装かを申告できるときに
// 実装する。申告がある場合に限り、Database instanceのproviderと一致することを実行前に確認する。
type DatabaseProviderBoundBrowseQueryExecutor interface {
	DatabaseProvider() environment.DatabaseProvider
}

// BrowseQueryEnvironmentExecutionは、Environment上の束縛を経て実行したbrowse Queryの結果と、
// その束縛の宣言側の事実を保持する。
//
// Instanceは宣言である。実行時の接続がそのInstanceであることはhostの申告であり、ここでは検証しない。
// QuerySourceSHA256は、Executorが実際に受け取ったQuery.Sourceのbytesのdigestである。
// Source Revision、実引数の値、接続の観測値、被覆は保持しない。
type BrowseQueryEnvironmentExecution struct {
	Environment       core.CanonicalID
	Instance          environment.DatabaseInstance
	QuerySourceSHA256 string
	Execution         BrowseQueryExecution
}

// ResolveBrowseQueryInstanceは、browse PatternのDatasourceをEnvironment上の唯一のDatabase instanceへ
// 解決し、そのDefinitionとProviderがDatasource・Queryの宣言と一致することを確かめる。
// Executorは呼ばない。
func (r Resolver) ResolveBrowseQueryInstance(
	environmentID, applicationID, patternID core.CanonicalID,
) (environment.DatabaseInstance, error) {
	resolved, err := r.ResolvePattern(applicationID, patternID)
	if err != nil {
		return environment.DatabaseInstance{}, err
	}
	return r.resolveBrowseInstance(environmentID, resolved)
}

func (r Resolver) resolveBrowseInstance(environmentID core.CanonicalID, resolved PatternResolution) (environment.DatabaseInstance, error) {
	if resolved.Query == nil || resolved.Datasource == nil {
		return environment.DatabaseInstance{}, fmt.Errorf("browse Pattern %sのQueryまたはDatasourceが解決されていません", resolved.Pattern.ID)
	}
	env, err := r.environmentByID(environmentID)
	if err != nil {
		return environment.DatabaseInstance{}, err
	}
	instanceID, err := applicationDatabaseInstance(env, resolved.Application.ID, resolved.Datasource.Alias)
	if err != nil {
		return environment.DatabaseInstance{}, fmt.Errorf(
			"browse Pattern %sのDatasource %sをEnvironment %sへ接続できません: %w",
			resolved.Pattern.ID, resolved.Datasource.Alias, env.ID, err,
		)
	}
	instance, _ := env.InstanceByID(instanceID)
	if instance.Definition != resolved.Datasource.TargetDatabase {
		return environment.DatabaseInstance{}, fmt.Errorf(
			"Database instance %sのDefinition %sがDatasource %sのtarget Database %sと一致しません",
			instance.ID, instance.Definition, resolved.Datasource.Alias, resolved.Datasource.TargetDatabase,
		)
	}
	if instance.Provider != resolved.Query.Dialect {
		return environment.DatabaseInstance{}, fmt.Errorf(
			"Database instance %sのprovider %sとQuery %sのdialect %sが一致しません",
			instance.ID, instance.Provider, resolved.Query.ID, resolved.Query.Dialect,
		)
	}
	return instance, nil
}

// ExecuteBrowseQueryInEnvironmentは、browse PatternのDatasourceをEnvironmentのDatabase instanceへ
// 明示的に束縛し、そのinstance用にregistryへ登録されたExecutorだけで既存のExecuteBrowseQueryを実行する。
//
// Environment・束縛・Definition・Provider・Executorのいずれかが不足または不一致なら、
// Executorを一度も呼ばずにerrorを返す。実行先をSQL本文やalias文字列から推測しない。
func (r Resolver) ExecuteBrowseQueryInEnvironment(
	ctx context.Context,
	environmentID, applicationID, patternID core.CanonicalID,
	registry BrowseQueryExecutorRegistry,
	parameters map[string]any,
) (BrowseQueryEnvironmentExecution, error) {
	if registry == nil {
		return BrowseQueryEnvironmentExecution{}, fmt.Errorf("browse Pattern %sを実行するExecutor Registryが指定されていません", patternID)
	}
	resolved, err := r.ResolvePattern(applicationID, patternID)
	if err != nil {
		return BrowseQueryEnvironmentExecution{}, err
	}
	instance, err := r.resolveBrowseInstance(environmentID, resolved)
	if err != nil {
		return BrowseQueryEnvironmentExecution{}, err
	}
	executor, ok := registry.ExecutorForBrowseQuery(environmentID, instance.ID)
	if !ok {
		return BrowseQueryEnvironmentExecution{}, fmt.Errorf(
			"Environment %sのDatabase instance %sに対応するruntime ExecutorがRegistryにありません",
			environmentID, instance.ID,
		)
	}
	if bound, ok := executor.(DatabaseProviderBoundBrowseQueryExecutor); ok {
		if runtime := bound.DatabaseProvider(); runtime != instance.Provider {
			return BrowseQueryEnvironmentExecution{}, fmt.Errorf(
				"runtime Executorのprovider %qとDatabase instance %sのprovider %sが一致しません",
				runtime, instance.ID, instance.Provider,
			)
		}
	}

	recorder := &sourceRecordingBrowseExecutor{inner: executor}
	execution, err := r.ExecuteBrowseQuery(ctx, applicationID, patternID, recorder, parameters)
	if err != nil {
		return BrowseQueryEnvironmentExecution{}, err
	}
	return BrowseQueryEnvironmentExecution{
		Environment:       environmentID,
		Instance:          instance,
		QuerySourceSHA256: recorder.digest,
		Execution:         execution,
	}, nil
}

// sourceRecordingBrowseExecutorは、Executorが実際に受け取ったQuery.Sourceのdigestを記録する。
type sourceRecordingBrowseExecutor struct {
	inner  BrowseQueryExecutor
	digest string
}

func (e *sourceRecordingBrowseExecutor) ExecuteBrowseQuery(
	ctx context.Context,
	q query.EmbeddedSQL,
	datasource application.DatabaseRequirement,
	parameters map[string]any,
) (BrowseQueryResult, error) {
	sum := sha256.Sum256([]byte(q.Source))
	e.digest = hex.EncodeToString(sum[:])
	return e.inner.ExecuteBrowseQuery(ctx, q, datasource, parameters)
}

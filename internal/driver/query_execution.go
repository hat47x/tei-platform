package driver

import (
	"context"
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver/queryresult"
	"tei/internal/query"
)

// BrowseQueryResultは、Executorが観測したraw resultと、Canonical Queryの明示mappingを
// 適用した結果を分けて保持する。
//
// PayloadにはExecutor固有の非secret Evidenceを保持できる。実行時parameter値はここへ保存しない。
type BrowseQueryResult struct {
	Raw     queryresult.RawResult
	Mapped  queryresult.MappedResult
	Payload any
}

// BrowseQueryExecutorは、Canonical QueryとApplicationが明示したDatasourceを実行環境へ接続する。
//
// DriverはQuery.Sourceを解析して物理Databaseやtableを推測せず、解決済みのQueryとDatasourceを
// そのままExecutorへ渡す。接続情報やSQL placeholderの処理方法はExecutor側の責務とする。
// Executorは物理的に観測したraw resultを返し、Semantic mappingはDriverがCanonical Queryから適用する。
type BrowseQueryExecutor interface {
	ExecuteBrowseQuery(
		context.Context,
		query.EmbeddedSQL,
		application.DatabaseRequirement,
		map[string]any,
	) (BrowseQueryResult, error)
}

// BrowseQueryExecutionは、どのPattern / Query / Datasourceを解決して実行したかと、
// Executorが返した結果をまとめる。実行時parameter自体は保持しない。
type BrowseQueryExecution struct {
	Resolution PatternResolution
	Result     BrowseQueryResult
}

// ExecuteBrowseQueryは、browse PatternをCanonical IDから解決し、
// Queryの正本とlogical Datasourceを変更せずExecutorへ渡す。
//
// Executorが返したraw resultにはQuery authoringの明示mappingだけを適用する。
// native SQL、列名の類似、Persistence MappingからSemantic attributeを推測しない。
// Executorの障害やmapping不整合は正常な空結果へ変換せず、Query実行を完了できなかったerrorとして返す。
func (r Resolver) ExecuteBrowseQuery(
	ctx context.Context,
	applicationID, patternID core.CanonicalID,
	executor BrowseQueryExecutor,
	parameters map[string]any,
) (BrowseQueryExecution, error) {
	resolved, err := r.ResolvePattern(applicationID, patternID)
	if err != nil {
		return BrowseQueryExecution{}, err
	}
	if resolved.Pattern.Kind != application.PatternBrowse {
		return BrowseQueryExecution{}, fmt.Errorf("Pattern %sはbrowseではありません: %s", resolved.Pattern.ID, resolved.Pattern.Kind)
	}
	if resolved.Query == nil {
		return BrowseQueryExecution{}, fmt.Errorf("browse Pattern %sのQueryが解決されていません", resolved.Pattern.ID)
	}
	if resolved.Datasource == nil {
		return BrowseQueryExecution{}, fmt.Errorf("browse Pattern %sのDatasourceが解決されていません", resolved.Pattern.ID)
	}
	if executor == nil {
		return BrowseQueryExecution{}, fmt.Errorf("browse Pattern %sを実行するQuery Executorが指定されていません", resolved.Pattern.ID)
	}

	result, err := executor.ExecuteBrowseQuery(ctx, *resolved.Query, *resolved.Datasource, parameters)
	if err != nil {
		return BrowseQueryExecution{}, fmt.Errorf("browse Pattern %sのQuery %sをDatasource %sで実行できませんでした: %w", resolved.Pattern.ID, resolved.Query.ID, resolved.Datasource.Alias, err)
	}
	mapped, err := queryresult.Map(*resolved.Query, result.Raw)
	if err != nil {
		return BrowseQueryExecution{}, fmt.Errorf("browse Pattern %sのQuery %s resultをSemantic attributeへ対応付けられません: %w", resolved.Pattern.ID, resolved.Query.ID, err)
	}
	result.Mapped = mapped
	return BrowseQueryExecution{Resolution: resolved, Result: result}, nil
}

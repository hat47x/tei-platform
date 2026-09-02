package driver

import (
	"context"
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/query"
)

// BrowseQueryResultは、Query Executorが返した結果をDriverが解釈せず保持する最小境界である。
//
// 現時点ではnative SQLの列をSemantic Dataのattributeへ対応付けるCanonicalな規則がないため、
// Driver側で独自のrow mappingを作らない。Payloadの具体型はExecutor側が所有し、
// 対応付けの正本が成立した段階で、必要な型境界を上位の意味とともに追加する。
type BrowseQueryResult struct {
	Payload any
}

// BrowseQueryExecutorは、Canonical QueryとApplicationが明示したDatasourceを実行環境へ接続する。
//
// DriverはQuery.Sourceを解析して物理Databaseやtableを推測せず、解決済みのQueryとDatasourceを
// そのままExecutorへ渡す。接続情報やSQL placeholderの処理方法はExecutor側の責務とする。
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
// native SQLをDriverへ複製したり、SQL本文から物理Database objectを推測したりしない。
// Executorの障害は業務上の空結果へ変換せず、Query実行を完了できなかったerrorとして返す。
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
	return BrowseQueryExecution{Resolution: resolved, Result: result}, nil
}

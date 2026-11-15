package driver

import (
	"context"
	"fmt"
	"sort"
	"time"

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
// Executorは物理的に観測したRaw resultを返し、Semantic mappingはDriverがCanonical Queryから適用する。
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

	// A resolved Query is not an invocation permission: reject missing, undeclared or
	// unsupported typed inputs before any provider can be called.
	if err := validateBrowseQueryParameters(*resolved.Query, parameters); err != nil {
		return BrowseQueryExecution{}, err
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

// validateBrowseQueryParameters checks only the runtime representations for which
// this Browse path has an explicit mapping. An input's domain type declaration is
// not authority to infer a Go value or coerce a string to a timestamp.
//
// In particular, std.datetime is a time.Time supplied by the caller. NULL,
// absent values and undeclared names are not silently interpreted as defaults.
// The current Query/Input model does not declare optional or nullable inputs.
// Other semantic types stay unsupported until their own type-specific contract
// has an independently verified runtime binding. No parameter value is logged.
func validateBrowseQueryParameters(q query.EmbeddedSQL, parameters map[string]any) error {
	// The unqualified legacy synthetic Browse route remains compatible with
	// historical Source fixtures. The actual Environment-bound Provider path
	// rejects undeclared input values separately; only a declared Query gets
	// typed-input qualification here.
	if len(q.Inputs) == 0 {
		return nil
	}
	declared := make(map[string]core.CanonicalID, len(q.Inputs))
	for _, in := range q.Inputs {
		name := string(in.Name)
		if name == "" || in.Type == "" {
			return fmt.Errorf("Query %s has an incomplete input declaration", q.ID)
		}
		if _, exists := declared[name]; exists {
			return fmt.Errorf("Query %s declares input %s more than once", q.ID, name)
		}
		declared[name] = in.Type
	}

	// Sort map keys so diagnostics are stable across Go map iterations.
	keys := make([]string, 0, len(parameters))
	for name := range parameters {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		if _, ok := declared[name]; !ok {
			return fmt.Errorf("Query %s received undeclared input %s", q.ID, name)
		}
	}

	for _, in := range q.Inputs {
		name := string(in.Name)
		value, present := parameters[name]
		if !present {
			return fmt.Errorf("Query %s requires input %s", q.ID, name)
		}
		if value == nil {
			return fmt.Errorf("Query %s input %s is NULL without a nullable declaration", q.ID, name)
		}
		switch in.Type {
		case "std.datetime":
			if _, ok := value.(time.Time); !ok {
				return fmt.Errorf("Query %s input %s requires std.datetime (time.Time)", q.ID, name)
			}
		default:
			return fmt.Errorf("Query %s input %s has unsupported runtime binding for type %s", q.ID, name, in.Type)
		}
	}
	return nil
}

// RelationQueryResultは、Executorが観測したraw resultと、Canonical Relation Queryの明示mappingを
// 適用した結果を分けて保持する。
type RelationQueryResult struct {
	Raw     queryresult.RawResult
	Mapped  queryresult.MappedResult
	Payload any
}

// RelationQueryExecutorは、解決済みのRelation Queryを実行環境へ接続する。
type RelationQueryExecutor interface {
	ExecuteRelationQuery(
		context.Context,
		query.ResolvedRelationQuery,
		application.DatabaseRequirement,
		map[string]any,
	) (RelationQueryResult, error)
}

// RelationQueryExecutionは、解決したRelation QueryとExecutorが返した結果をまとめる。
type RelationQueryExecution struct {
	Resolution query.ResolvedRelationQuery
	Result     RelationQueryResult
}

// ExecuteRelationQueryは、Relation QueryをCanonical IDから解決し、
// SQLを生成してExecutorで実行する。
func (r Resolver) ExecuteRelationQuery(
	ctx context.Context,
	applicationID, queryID core.CanonicalID,
	executor RelationQueryExecutor,
	parameters map[string]any,
) (RelationQueryExecution, error) {
	resolved, err := r.ResolveRelationQuery(applicationID, queryID)
	if err != nil {
		return RelationQueryExecution{}, err
	}
	app, err := r.applicationByID(resolved.Query.Application)
	if err != nil {
		return RelationQueryExecution{}, fmt.Errorf("Relation Query %sのApplication %sを解決できません: %w", queryID, resolved.Query.Application, err)
	}
	datasource, err := readableDatasource(app, core.LocalName(resolved.Query.Datasource))
	if err != nil {
		return RelationQueryExecution{}, fmt.Errorf("Relation Query %sのDatasource %sを解決できません: %w", queryID, resolved.Query.Datasource, err)
	}
	if executor == nil {
		return RelationQueryExecution{}, fmt.Errorf("Relation Query %sを実行するExecutorが指定されていません", queryID)
	}

	queryResult, err := executor.ExecuteRelationQuery(ctx, resolved, datasource, parameters)
	if err != nil {
		return RelationQueryExecution{}, fmt.Errorf("Relation Query %sをDatasource %sで実行できませんでした: %w", queryID, datasource.Alias, err)
	}
	mapped, err := queryresult.MapRelationQuery(resolved, queryResult.Raw)
	if err != nil {
		return RelationQueryExecution{}, fmt.Errorf("Relation Query %sのresultをSemantic attributeへ対応付けられません: %w", queryID, err)
	}
	queryResult.Mapped = mapped
	return RelationQueryExecution{Resolution: resolved, Result: queryResult}, nil
}

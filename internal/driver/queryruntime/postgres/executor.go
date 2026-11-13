package postgresquery

import (
	"context"
	"database/sql"
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database/persistence"
	postgresprovider "tei/internal/database/providers/postgres"
	postgressqlgen "tei/internal/database/sqlgen/postgres"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/environment"
	"tei/internal/query"
)

// Resultは、PostgreSQL固有の実行Evidenceとしてnamed parameterの束縛順だけを保持する。
// ParameterNamesには実行時の値を含めない。raw resultとSemantic mappingはDriver共通境界が所有する。
type Result struct {
	ParameterNames []string `json:"parameterNames,omitempty"`
}

// Queryer accepts either a connection pool or an explicitly owned transaction.
// The host chooses transaction isolation and lifetime; the executor never commits.
type Queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// Executorは、解決済みのPostgreSQL Queryを一つの明示的なDatabase接続へ渡す。
//
// DBとDatabaseの対応はcomposition側が注入する。SQL本文やDatasource aliasから接続先を推測しない。
type Executor struct {
	DB       Queryer
	Database core.CanonicalID
	Mappings []persistence.Mapping
}

var _ driver.BrowseQueryExecutor = Executor{}

// ExecuteBrowseQueryは、PostgreSQL固有のparameter bindingを行った後、native SQLを実DBへ渡す。
//
// 戻り値は物理的に観測したraw resultのままとし、Semantic mappingはDriverがCanonical Queryの
// 明示mappingから適用する。列名の類似や命名規則からSemantic attributeを推測しない。
func (e Executor) ExecuteBrowseQuery(
	ctx context.Context,
	q query.EmbeddedSQL,
	datasource application.DatabaseRequirement,
	parameters map[string]any,
) (driver.BrowseQueryResult, error) {
	if e.DB == nil {
		return driver.BrowseQueryResult{}, fmt.Errorf("PostgreSQL Query ExecutorにDatabase接続が指定されていません")
	}
	if e.Database == "" {
		return driver.BrowseQueryResult{}, fmt.Errorf("PostgreSQL Query Executorが表すDatabase identityが指定されていません")
	}
	if q.Dialect != environment.ProviderPostgres {
		return driver.BrowseQueryResult{}, fmt.Errorf("Query %sのdialect %sはPostgreSQL Executorでは実行できません", q.ID, q.Dialect)
	}
	if datasource.TargetDatabase != e.Database {
		return driver.BrowseQueryResult{}, fmt.Errorf(
			"Datasource %sのtarget Database %sとPostgreSQL ExecutorのDatabase %sが一致しません",
			datasource.Alias,
			datasource.TargetDatabase,
			e.Database,
		)
	}

	bound, err := postgresprovider.BindNamedParameters(q.Source, parameters)
	if err != nil {
		return driver.BrowseQueryResult{}, fmt.Errorf("Query %sのparameterをPostgreSQLへ束縛できません: %w", q.ID, err)
	}

	rows, err := e.DB.QueryContext(ctx, bound.SQL, bound.Args...)
	if err != nil {
		return driver.BrowseQueryResult{}, fmt.Errorf("Query %sをPostgreSQLで実行できません: %w", q.ID, err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return driver.BrowseQueryResult{}, fmt.Errorf("Query %sのresult列を取得できません: %w", q.ID, err)
	}

	rawRows := make([][]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for i := range values {
			destinations[i] = &values[i]
		}
		if err := rows.Scan(destinations...); err != nil {
			return driver.BrowseQueryResult{}, fmt.Errorf("Query %sのresult rowを読み取れません: %w", q.ID, err)
		}
		for i, value := range values {
			values[i] = stabilizeRawValue(value)
		}
		rawRows = append(rawRows, values)
	}
	if err := rows.Err(); err != nil {
		return driver.BrowseQueryResult{}, fmt.Errorf("Query %sのresult row取得中にerrorが発生しました: %w", q.ID, err)
	}

	return driver.BrowseQueryResult{
		Raw: queryresult.RawResult{
			Columns: append([]string(nil), columns...),
			Rows:    rawRows,
		},
		Payload: Result{ParameterNames: append([]string(nil), bound.ParameterNames...)},
	}, nil
}

func stabilizeRawValue(value any) any {
	bytes, ok := value.([]byte)
	if !ok {
		return value
	}
	return append([]byte(nil), bytes...)
}

var _ driver.RelationQueryExecutor = Executor{}

// ExecuteRelationQueryは、PostgreSQLでRelation Queryを実行する。
//
// 解決済みのRelation QueryをSQLに変換し、PostgreSQLで実行してraw resultを返す。
// Semantic mappingはDriverが別途適用する。
func (e Executor) ExecuteRelationQuery(
	ctx context.Context,
	resolved query.ResolvedRelationQuery,
	datasource application.DatabaseRequirement,
	parameters map[string]any,
) (driver.RelationQueryResult, error) {
	if e.DB == nil {
		return driver.RelationQueryResult{}, fmt.Errorf("PostgreSQL Relation Query ExecutorにDatabase接続が指定されていません")
	}
	if e.Database == "" {
		return driver.RelationQueryResult{}, fmt.Errorf("PostgreSQL Relation Query Executorが表すDatabase identityが指定されていません")
	}
	if datasource.TargetDatabase != e.Database {
		return driver.RelationQueryResult{}, fmt.Errorf(
			"Datasource %sのtarget Database %sとPostgreSQL ExecutorのDatabase %sが一致しません",
			datasource.Alias,
			datasource.TargetDatabase,
			e.Database,
		)
	}

	builder := driver.RelationQuerySQL{
		Dialect:    postgressqlgen.NewDialect(),
		Resolved:   resolved,
		Parameters: parameters,
		Mappings:   e.Mappings,
	}
	sql, err := builder.BuildSelect()
	if err != nil {
		return driver.RelationQueryResult{}, fmt.Errorf("Relation Query %sのSQL生成に失敗しました: %w", resolved.Query.ID, err)
	}
	resolved.SQL = sql.Statement

	switch resolved.Query.FetchMode {
	case query.RelationFetchModeEager, "":
		return e.executeResolvedRelationQuery(ctx, sql.Statement, sql.Args...)
	case query.RelationFetchModeBatch:
		if resolved.Query.BatchSize <= 0 {
			return driver.RelationQueryResult{}, fmt.Errorf("Relation Query %sのbatch sizeが未指定です", resolved.Query.ID)
		}
		return e.executeBatchedRelationQuery(ctx, resolved, sql.Statement, sql.Args...)
	case query.RelationFetchModeLazy:
		return driver.RelationQueryResult{}, fmt.Errorf("Relation Query %sのlazy fetchingは未実装です", resolved.Query.ID)
	default:
		return driver.RelationQueryResult{}, fmt.Errorf("Relation Query %sのfetch mode %sは未対応です", resolved.Query.ID, resolved.Query.FetchMode)
	}
}

func (e Executor) executeBatchedRelationQuery(ctx context.Context, resolved query.ResolvedRelationQuery, sql string, args ...any) (driver.RelationQueryResult, error) {
	batchSize := resolved.Query.BatchSize
	offset := 0
	var allRows [][]any
	allColumns := []string{}

	for {
		batchSQL := fmt.Sprintf("%s LIMIT %d OFFSET %d", sql, batchSize, offset)
		batchResult, err := e.executeResolvedRelationQuery(ctx, batchSQL, args...)
		if err != nil {
			return driver.RelationQueryResult{}, err
		}
		if len(batchResult.Raw.Rows) == 0 {
			break
		}
		if allColumns == nil {
			allColumns = batchResult.Raw.Columns
		}
		allRows = append(allRows, batchResult.Raw.Rows...)
		offset += batchSize
		if len(batchResult.Raw.Rows) < batchSize {
			break
		}
	}

	return driver.RelationQueryResult{
		Raw: queryresult.RawResult{
			Columns: allColumns,
			Rows:    allRows,
		},
		Payload: Result{ParameterNames: nil},
	}, nil
}

func (e Executor) executeResolvedRelationQuery(ctx context.Context, sql string, args ...any) (driver.RelationQueryResult, error) {
	rows, err := e.DB.QueryContext(ctx, sql, args...)
	if err != nil {
		return driver.RelationQueryResult{}, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return driver.RelationQueryResult{}, err
	}

	rawRows := make([][]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for i := range values {
			destinations[i] = &values[i]
		}
		if err := rows.Scan(destinations...); err != nil {
			return driver.RelationQueryResult{}, err
		}
		for i, value := range values {
			values[i] = stabilizeRawValue(value)
		}
		rawRows = append(rawRows, values)
	}
	if err := rows.Err(); err != nil {
		return driver.RelationQueryResult{}, err
	}

	return driver.RelationQueryResult{
		Raw: queryresult.RawResult{
			Columns: append([]string(nil), columns...),
			Rows:    rawRows,
		},
		Payload: Result{ParameterNames: nil},
	}, nil
}

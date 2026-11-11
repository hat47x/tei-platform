package sqlitequery

import (
	"context"
	"database/sql"
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database/persistence"
	sqlitedialect "tei/internal/database/sqlgen/sqlite"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/query"
)

// Resultは、SQLite固有の実行Evidenceとしてnamed parameterの束縛順だけを保持する。
// ParameterNamesには実行時の値を含めない。raw resultとSemantic mappingはDriver共通境界が所有する。
type Result struct {
	ParameterNames []string `json:"parameterNames,omitempty"`
}

// Executorは、解決済みのRelation QueryをSQLiteで実行する。
type Executor struct {
	DB       Queryer
	Database core.CanonicalID
	Mappings []persistence.Mapping
}

// Queryer accepts either a connection pool or an explicitly owned transaction.
type Queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

var _ driver.RelationQueryExecutor = Executor{}

// ExecuteRelationQueryは、SQLiteでRelation Queryを実行する。
func (e Executor) ExecuteRelationQuery(
	ctx context.Context,
	resolved query.ResolvedRelationQuery,
	datasource application.DatabaseRequirement,
	parameters map[string]any,
) (driver.RelationQueryResult, error) {
	if e.DB == nil {
		return driver.RelationQueryResult{}, fmt.Errorf("SQLite Relation Query ExecutorにDatabase接続が指定されていません")
	}
	if e.Database == "" {
		return driver.RelationQueryResult{}, fmt.Errorf("SQLite Relation Query Executorが表すDatabase identityが指定されていません")
	}
	if datasource.TargetDatabase != e.Database {
		return driver.RelationQueryResult{}, fmt.Errorf(
			"Datasource %sのtarget Database %sとSQLite ExecutorのDatabase %sが一致しません",
			datasource.Alias,
			datasource.TargetDatabase,
			e.Database,
		)
	}

	builder := driver.RelationQuerySQL{
		Dialect:    sqlitedialect.NewDialect(),
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
		Payload: nil,
	}, nil
}

func stabilizeRawValue(value any) any {
	bytes, ok := value.([]byte)
	if !ok {
		return value
	}
	return append([]byte(nil), bytes...)
}

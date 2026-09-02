package postgresquery

import (
	"context"
	"database/sql"
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	postgresprovider "tei/internal/database/providers/postgres"
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

// Executorは、解決済みのPostgreSQL Queryを一つの明示的なDatabase接続へ渡す。
//
// DBとDatabaseの対応はcomposition側が注入する。SQL本文やDatasource aliasから接続先を推測しない。
type Executor struct {
	DB       *sql.DB
	Database core.CanonicalID
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

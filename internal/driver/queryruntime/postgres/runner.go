package postgresquery

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	postgresprovider "tei/internal/database/providers/postgres"
	"tei/internal/driver/queryresult"
	sqlquery "tei/internal/driver/queryruntime/sql"
)

// Runnerは、既に確定しているnative PostgreSQL SQLとphysical parameterを、
// 注入済みのdatabase/sql接続へ渡す最小実行境界である。
//
// Contract inputとphysical parameterの対応は上位のSQL realizationが所有する。
// Runnerは:name形式のphysical parameterをPostgreSQLのpositional placeholderへ変換するだけで、
// Semantic attribute、Contract input、table / column、接続先Databaseを推測しない。
type Runner struct {
	DB *sql.DB
}

var _ sqlquery.Runner = Runner{}

// ExecuteSQLは、physical parameterをPostgreSQLの$1, $2, ...へ明示束縛してnative SQLを実行し、
// 観測した列名とrow値をraw resultとして返す。
//
// 0 rowは正常結果として返す。Query実行、parameter束縛、row走査の失敗だけをerrorとして返し、
// runtime parameter値はEvidenceへ保持しない。
func (r Runner) ExecuteSQL(
	ctx context.Context,
	source string,
	parameters map[string]any,
) (sqlquery.RawExecution, error) {
	if r.DB == nil {
		return sqlquery.RawExecution{}, fmt.Errorf("PostgreSQL Contract Query RunnerにDatabase接続が指定されていません")
	}
	if strings.TrimSpace(source) == "" {
		return sqlquery.RawExecution{}, fmt.Errorf("PostgreSQL Contract Query Runnerへ渡すnative SQLが空です")
	}

	bound, err := postgresprovider.BindNamedParameters(source, parameters)
	if err != nil {
		return sqlquery.RawExecution{}, fmt.Errorf("PostgreSQL Contract Queryのphysical parameterを束縛できません: %w", err)
	}
	rows, err := r.DB.QueryContext(ctx, bound.SQL, bound.Args...)
	if err != nil {
		return sqlquery.RawExecution{}, fmt.Errorf("PostgreSQL Contract Queryのnative SQLを実行できません: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return sqlquery.RawExecution{}, fmt.Errorf("PostgreSQL Contract Queryのraw result列を取得できません: %w", err)
	}

	rawRows := make([][]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for i := range values {
			destinations[i] = &values[i]
		}
		if err := rows.Scan(destinations...); err != nil {
			return sqlquery.RawExecution{}, fmt.Errorf("PostgreSQL Contract Queryのraw result rowを読み取れません: %w", err)
		}
		for i, value := range values {
			values[i] = stabilizeRawValue(value)
		}
		rawRows = append(rawRows, values)
	}
	if err := rows.Err(); err != nil {
		return sqlquery.RawExecution{}, fmt.Errorf("PostgreSQL Contract Queryのraw result取得中にerrorが発生しました: %w", err)
	}

	return sqlquery.RawExecution{
		Result: queryresult.RawResult{
			Columns: append([]string(nil), columns...),
			Rows:    rawRows,
		},
		Payload: Result{ParameterNames: append([]string(nil), bound.ParameterNames...)},
	}, nil
}

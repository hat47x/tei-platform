package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

// RawQueryResultは、PostgreSQLから観測した物理列名とrowを、
// Semantic Dataへ意味付けする前の状態で保持する。
//
// ParameterNamesは実行時の値を含まず、Providerがどのnamed parameterを
// どの順序でplaceholderへ束縛したかだけを示す。
type RawQueryResult struct {
	Columns        []string
	Rows           [][]any
	ParameterNames []string
}

// ExecuteRawQueryは、native SQLのnamed parameterをPostgreSQL用placeholderへ束縛し、
// 実DatabaseでQueryを実行するProvider側の最小境界である。
//
// この関数は物理列名からSemantic attributeを推測しない。また、実行時parameterの値を
// RawQueryResultへ保存しない。result mappingは、実際のraw resultを観測した後で別途決める。
func ExecuteRawQuery(
	ctx context.Context,
	db *sql.DB,
	source string,
	parameters map[string]any,
) (RawQueryResult, error) {
	if db == nil {
		return RawQueryResult{}, fmt.Errorf("PostgreSQL Queryを実行するDatabase接続が指定されていません")
	}
	bound, err := BindNamedParameters(source, parameters)
	if err != nil {
		return RawQueryResult{}, fmt.Errorf("PostgreSQL Query parameterを束縛できません: %w", err)
	}

	rows, err := db.QueryContext(ctx, bound.SQL, bound.Args...)
	if err != nil {
		return RawQueryResult{}, fmt.Errorf("PostgreSQL Queryを実行できません: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return RawQueryResult{}, fmt.Errorf("PostgreSQL Queryの列情報を取得できません: %w", err)
	}

	result := RawQueryResult{
		Columns:        append([]string(nil), columns...),
		ParameterNames: append([]string(nil), bound.ParameterNames...),
	}
	for rows.Next() {
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for i := range values {
			destinations[i] = &values[i]
		}
		if err := rows.Scan(destinations...); err != nil {
			return RawQueryResult{}, fmt.Errorf("PostgreSQL Queryのrowを読み取れません: %w", err)
		}
		for i, value := range values {
			if raw, ok := value.([]byte); ok {
				values[i] = append([]byte(nil), raw...)
			}
		}
		result.Rows = append(result.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return RawQueryResult{}, fmt.Errorf("PostgreSQL Queryのrow走査を完了できません: %w", err)
	}
	return result, nil
}

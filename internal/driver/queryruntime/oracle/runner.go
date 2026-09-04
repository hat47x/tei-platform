package oraclequery

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"tei/internal/driver/queryresult"
	sqlquery "tei/internal/driver/queryruntime/sql"
)

// Resultは、Oracle固有の実行Evidenceとして、実際にdatabase/sqlへ渡した
// named parameterの名前と、Databaseから返されたraw result列名だけを保持する。
// runtime parameter値やraw row値は保存しない。
type Result struct {
	ParameterNames   []string `json:"parameterNames,omitempty"`
	RawResultColumns []string `json:"rawResultColumns,omitempty"`
}

// Runnerは、既に確定しているnative Oracle SQLとphysical parameterを、
// 注入済みのdatabase/sql接続へ渡すだけの最小実行境界である。
//
// table / view / column、Contract input、Semantic attribute、接続先Databaseを
// SQL本文やparameter名から推測しない。接続先の選択はinstance-awareなruntime Registry、
// Semantic mappingは上位のContract Query SQL realizationがそれぞれ所有する。
type Runner struct {
	DB *sql.DB
}

var _ sqlquery.Runner = Runner{}

// ExecuteSQLは、physical parameterをnamed argumentとしてdatabase/sqlへ渡し、
// 観測した列名とrow値をraw resultとして返す。
//
// parameter名は実行の再現性とEvidenceの安定性のため辞書順に並べるが、名前自体は変更しない。
// 0 rowは正常結果として返し、Query実行やrow走査の失敗だけをerrorとして返す。
func (r Runner) ExecuteSQL(
	ctx context.Context,
	source string,
	parameters map[string]any,
) (sqlquery.RawExecution, error) {
	if r.DB == nil {
		return sqlquery.RawExecution{}, fmt.Errorf("Oracle Contract Query RunnerにDatabase接続が指定されていません")
	}
	if strings.TrimSpace(source) == "" {
		return sqlquery.RawExecution{}, fmt.Errorf("Oracle Contract Query Runnerへ渡すnative SQLが空です")
	}

	parameterNames := make([]string, 0, len(parameters))
	for name := range parameters {
		if name == "" {
			return sqlquery.RawExecution{}, fmt.Errorf("Oracle Contract Query Runnerへ名前のないphysical parameterが渡されています")
		}
		parameterNames = append(parameterNames, name)
	}
	sort.Strings(parameterNames)

	args := make([]any, 0, len(parameterNames))
	for _, name := range parameterNames {
		args = append(args, sql.Named(name, parameters[name]))
	}

	rows, err := r.DB.QueryContext(ctx, source, args...)
	if err != nil {
		return sqlquery.RawExecution{}, fmt.Errorf("Oracle Contract Queryのnative SQLを実行できません: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return sqlquery.RawExecution{}, fmt.Errorf("Oracle Contract Queryのraw result列を取得できません: %w", err)
	}

	rawRows := make([][]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for i := range values {
			destinations[i] = &values[i]
		}
		if err := rows.Scan(destinations...); err != nil {
			return sqlquery.RawExecution{}, fmt.Errorf("Oracle Contract Queryのraw result rowを読み取れません: %w", err)
		}
		for i, value := range values {
			values[i] = stabilizeRawValue(value)
		}
		rawRows = append(rawRows, values)
	}
	if err := rows.Err(); err != nil {
		return sqlquery.RawExecution{}, fmt.Errorf("Oracle Contract Queryのraw result取得中にerrorが発生しました: %w", err)
	}

	return sqlquery.RawExecution{
		Result: queryresult.RawResult{
			Columns: append([]string(nil), columns...),
			Rows:    rawRows,
		},
		Payload: Result{
			ParameterNames:   append([]string(nil), parameterNames...),
			RawResultColumns: append([]string(nil), columns...),
		},
	}, nil
}

func stabilizeRawValue(value any) any {
	bytes, ok := value.([]byte)
	if !ok {
		return value
	}
	return append([]byte(nil), bytes...)
}

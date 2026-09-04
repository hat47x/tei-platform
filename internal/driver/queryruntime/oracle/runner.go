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

// RawResultColumnMetadataは、database/sql driverがactual read時に提供できたraw result列metadataだけを保持する。
//
// 取得できない情報をfalseや0へ補完せず、nilのまま「未観測 / driver未提供」として残す。
// ScanTypeはGo側のscan表現でありexternal Oracleのphysical typeそのものではないためEvidenceへ保存しない。
type RawResultColumnMetadata struct {
	Name             string  `json:"name"`
	DatabaseTypeName *string `json:"databaseTypeName"`
	Nullable         *bool   `json:"nullable"`
	Length           *int64  `json:"length"`
	Precision        *int64  `json:"precision"`
	Scale            *int64  `json:"scale"`
}

// Resultは、Oracle固有の実行Evidenceとして、実際にdatabase/sqlへ渡した
// named parameterの名前と、Databaseから観測したraw result列情報だけを保持する。
// runtime parameter値やraw row値は保存しない。
type Result struct {
	ParameterNames          []string                  `json:"parameterNames,omitempty"`
	RawResultColumns        []string                  `json:"rawResultColumns,omitempty"`
	RawResultColumnMetadata []RawResultColumnMetadata `json:"rawResultColumnMetadata,omitempty"`
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
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return sqlquery.RawExecution{}, fmt.Errorf("Oracle Contract Queryのraw result列metadataを取得できません: %w", err)
	}
	columnMetadata, err := observedColumnMetadata(columns, columnTypes)
	if err != nil {
		return sqlquery.RawExecution{}, err
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
			ParameterNames:          append([]string(nil), parameterNames...),
			RawResultColumns:        append([]string(nil), columns...),
			RawResultColumnMetadata: columnMetadata,
		},
	}, nil
}

func observedColumnMetadata(columns []string, columnTypes []*sql.ColumnType) ([]RawResultColumnMetadata, error) {
	if len(columnTypes) != len(columns) {
		return nil, fmt.Errorf("Oracle Contract Queryのraw result列metadata件数=%d、列件数=%dで一致しません", len(columnTypes), len(columns))
	}

	result := make([]RawResultColumnMetadata, len(columnTypes))
	for i, columnType := range columnTypes {
		if columnType == nil {
			return nil, fmt.Errorf("Oracle Contract Queryのraw result列metadata[%d]がnilです", i)
		}
		if columnType.Name() != columns[i] {
			return nil, fmt.Errorf("Oracle Contract Queryのraw result列metadata[%d]の名前=%q、列名=%qで一致しません", i, columnType.Name(), columns[i])
		}

		metadata := RawResultColumnMetadata{Name: columns[i]}
		if databaseTypeName := columnType.DatabaseTypeName(); databaseTypeName != "" {
			metadata.DatabaseTypeName = ptr(databaseTypeName)
		}
		if nullable, ok := columnType.Nullable(); ok {
			metadata.Nullable = ptr(nullable)
		}
		if length, ok := columnType.Length(); ok {
			metadata.Length = ptr(length)
		}
		if precision, scale, ok := columnType.DecimalSize(); ok {
			metadata.Precision = ptr(precision)
			metadata.Scale = ptr(scale)
		}
		result[i] = metadata
	}
	return result, nil
}

func ptr[T any](value T) *T {
	return &value
}

func stabilizeRawValue(value any) any {
	bytes, ok := value.([]byte)
	if !ok {
		return value
	}
	return append([]byte(nil), bytes...)
}

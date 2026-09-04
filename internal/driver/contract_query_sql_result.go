package driver

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/driver/queryresult"
	"tei/internal/realization"
)

// MapContractQuerySQLResultは、SQL realizationが観測したraw resultへ明示result mappingを適用し、
// ContractQueryExecutor境界で扱うSemantic rowへ変換する。
//
// raw列名の類似、SELECT順序、Persistence MappingからSemantic attributeを推測しない。
// runtime値は型変換せず、そのままSemantic attributeへ対応付ける。
func MapContractQuerySQLResult(
	realizationDef realization.ContractQuerySQLRealization,
	raw queryresult.RawResult,
) (ContractQueryResult, error) {
	subject := "Contract Query SQL realization " + realizationDef.Identity()
	mapped, err := queryresult.MapColumns(subject, realizationDef.ResultColumns, raw)
	if err != nil {
		return ContractQueryResult{}, err
	}

	rows := make([]ContractQueryRow, len(mapped.Rows))
	for rowIndex, values := range mapped.Rows {
		if len(values) != len(mapped.Columns) {
			return ContractQueryResult{}, fmt.Errorf(
				"%sのmapped row %dは列数が一致しません: columns=%d values=%d",
				subject,
				rowIndex,
				len(mapped.Columns),
				len(values),
			)
		}
		attributes := make(map[core.CanonicalID]any, len(mapped.Columns))
		for columnIndex, column := range mapped.Columns {
			attributes[column.Attribute] = values[columnIndex]
		}
		rows[rowIndex] = ContractQueryRow{Attributes: attributes}
	}
	return ContractQueryResult{Rows: rows}, nil
}

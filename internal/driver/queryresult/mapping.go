package queryresult

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/query"
)

// Columnは、native Queryが返した列名と、その列へ明示的に対応付けられた
// Semantic attributeを一組として保持する。
type Column struct {
	Raw       string           `json:"raw"`
	Attribute core.CanonicalID `json:"attribute"`
}

// MappedResultは、Query authoringで明示されたresult mappingを適用した結果である。
//
// Valuesはruntimeが受け取った値をそのまま保持し、この境界ではSemantic型への変換や
// Presentation向けの整形を行わない。Columnsと各Rowの値は同じ順序で対応する。
type MappedResult struct {
	Columns []Column `json:"columns"`
	Rows    [][]any  `json:"rows"`
}

// Mapは、raw resultの列名をQuery authoringの明示mappingへ対応付ける。
//
// 列名の類似、SELECT順序、Persistence Mappingから意味を推測しない。
// mappingにないraw列や、raw resultに存在しないmapping列があれば、意味を欠いたまま
// 実行を続けずerrorを返す。
func Map(q query.EmbeddedSQL, rawColumns []string, rawRows [][]any) (MappedResult, error) {
	if len(q.ResultColumns) == 0 {
		return MappedResult{}, fmt.Errorf("Query %sにresult mappingが定義されていません", q.ID)
	}

	rawIndex := make(map[string]int, len(rawColumns))
	for i, column := range rawColumns {
		if column == "" {
			return MappedResult{}, fmt.Errorf("Query %sのraw resultに名前のない列があります", q.ID)
		}
		if _, exists := rawIndex[column]; exists {
			return MappedResult{}, fmt.Errorf("Query %sのraw resultで列名 %qが重複しています", q.ID, column)
		}
		rawIndex[column] = i
	}

	mappedColumns := make([]Column, len(q.ResultColumns))
	usedRawColumns := make(map[string]struct{}, len(q.ResultColumns))
	for i, mapping := range q.ResultColumns {
		rawPosition, exists := rawIndex[mapping.Column]
		if !exists {
			return MappedResult{}, fmt.Errorf(
				"Query %sのresult mappingが要求するraw列 %qが実行結果にありません",
				q.ID,
				mapping.Column,
			)
		}
		_ = rawPosition
		usedRawColumns[mapping.Column] = struct{}{}
		mappedColumns[i] = Column{Raw: mapping.Column, Attribute: mapping.Attribute}
	}
	for _, column := range rawColumns {
		if _, mapped := usedRawColumns[column]; !mapped {
			return MappedResult{}, fmt.Errorf(
				"Query %sのraw result列 %qにSemantic attributeのmappingがありません",
				q.ID,
				column,
			)
		}
	}

	mappedRows := make([][]any, len(rawRows))
	for rowIndex, rawRow := range rawRows {
		if len(rawRow) != len(rawColumns) {
			return MappedResult{}, fmt.Errorf(
				"Query %sのraw result row %dは列数が一致しません: columns=%d values=%d",
				q.ID,
				rowIndex,
				len(rawColumns),
				len(rawRow),
			)
		}
		mappedRow := make([]any, len(q.ResultColumns))
		for i, mapping := range q.ResultColumns {
			mappedRow[i] = rawRow[rawIndex[mapping.Column]]
		}
		mappedRows[rowIndex] = mappedRow
	}

	return MappedResult{Columns: mappedColumns, Rows: mappedRows}, nil
}

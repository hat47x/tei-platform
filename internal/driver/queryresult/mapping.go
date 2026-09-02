package queryresult

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/query"
)

// RawResultは、Query Executorが物理的に観測した列名と値を、
// Semantic上の意味へ対応付ける前の状態で保持する。
type RawResult struct {
	Columns []string `json:"columns"`
	Rows    [][]any  `json:"rows"`
}

// Columnは、native Queryが返した列名と、その列に明示的に対応付けられた
// Semantic attributeを一組として保持する。
type Column struct {
	Raw       string           `json:"raw"`
	Attribute core.CanonicalID `json:"attribute"`
}

// MappedResultは、Query authoringで明示されたresult mappingを適用した結果である。
//
// Rowsにはruntimeで観測した値をそのまま保持し、この境界ではSemantic型への変換や
// Presentation向けの整形を行わない。Columnsと各Rowの値は同じ順序で対応する。
type MappedResult struct {
	Columns []Column `json:"columns"`
	Rows    [][]any  `json:"rows"`
}

// Mapは、raw resultの列名をQuery authoringの明示mappingへ対応付ける。
//
// 列名の類似、SELECT順序、Persistence Mappingから意味を推測しない。
// 明示mappingとraw resultの列集合が一致しない場合や、どちらかに曖昧な重複がある場合は、
// 意味を曖昧にしたまま処理を続けずerrorを返す。
func Map(q query.EmbeddedSQL, raw RawResult) (MappedResult, error) {
	if len(q.ResultColumns) == 0 {
		return MappedResult{}, fmt.Errorf("Query %sにresult mappingが定義されていません", q.ID)
	}

	rawIndex := make(map[string]int, len(raw.Columns))
	for i, column := range raw.Columns {
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
	usedAttributes := make(map[core.CanonicalID]struct{}, len(q.ResultColumns))
	for i, mapping := range q.ResultColumns {
		if mapping.Column == "" {
			return MappedResult{}, fmt.Errorf("Query %sのresult mappingに名前のないraw列があります", q.ID)
		}
		if mapping.Attribute == "" {
			return MappedResult{}, fmt.Errorf("Query %sのresult column %qにSemantic attributeが指定されていません", q.ID, mapping.Column)
		}
		if _, exists := usedRawColumns[mapping.Column]; exists {
			return MappedResult{}, fmt.Errorf("Query %sのresult mappingでraw列 %qが重複しています", q.ID, mapping.Column)
		}
		if _, exists := usedAttributes[mapping.Attribute]; exists {
			return MappedResult{}, fmt.Errorf("Query %sのresult mappingでSemantic attribute %sが重複しています", q.ID, mapping.Attribute)
		}
		if _, exists := rawIndex[mapping.Column]; !exists {
			return MappedResult{}, fmt.Errorf(
				"Query %sのresult mappingが要求するraw列 %qが実行結果にありません",
				q.ID,
				mapping.Column,
			)
		}
		usedRawColumns[mapping.Column] = struct{}{}
		usedAttributes[mapping.Attribute] = struct{}{}
		mappedColumns[i] = Column{Raw: mapping.Column, Attribute: mapping.Attribute}
	}
	for _, column := range raw.Columns {
		if _, mapped := usedRawColumns[column]; !mapped {
			return MappedResult{}, fmt.Errorf(
				"Query %sのraw result列 %qにSemantic attributeのmappingがありません",
				q.ID,
				column,
			)
		}
	}

	mappedRows := make([][]any, len(raw.Rows))
	for rowIndex, rawRow := range raw.Rows {
		if len(rawRow) != len(raw.Columns) {
			return MappedResult{}, fmt.Errorf(
				"Query %sのraw result row %dは列数が一致しません: columns=%d values=%d",
				q.ID,
				rowIndex,
				len(raw.Columns),
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

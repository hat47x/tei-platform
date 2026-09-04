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

// Mapは、Application Queryのraw resultへ明示result mappingを適用する。
func Map(q query.EmbeddedSQL, raw RawResult) (MappedResult, error) {
	return MapColumns("Query "+string(q.ID), q.ResultColumns, raw)
}

// MapColumnsは、raw resultの列名を明示されたSemantic attribute mappingへ対応付ける。
//
// Application QueryとDatabase Contract Queryのどちらでも、列名の類似、SELECT順序、
// Persistence Mappingから意味を推測しない。明示mappingとraw resultの列集合が一致しない場合や、
// どちらかに曖昧な重複がある場合は、意味を曖昧にしたまま処理を続けずerrorを返す。
// subjectは診断用の安定した論理identityだけを渡し、file pathやprovider固有接続情報は含めない。
func MapColumns(subject string, mappings []query.ResultColumnMapping, raw RawResult) (MappedResult, error) {
	if len(mappings) == 0 {
		return MappedResult{}, fmt.Errorf("%sにresult mappingが定義されていません", subject)
	}

	rawIndex := make(map[string]int, len(raw.Columns))
	for i, column := range raw.Columns {
		if column == "" {
			return MappedResult{}, fmt.Errorf("%sのraw resultに名前のない列があります", subject)
		}
		if _, exists := rawIndex[column]; exists {
			return MappedResult{}, fmt.Errorf("%sのraw resultで列名 %qが重複しています", subject, column)
		}
		rawIndex[column] = i
	}

	mappedColumns := make([]Column, len(mappings))
	usedRawColumns := make(map[string]struct{}, len(mappings))
	usedAttributes := make(map[core.CanonicalID]struct{}, len(mappings))
	for i, mapping := range mappings {
		if mapping.Column == "" {
			return MappedResult{}, fmt.Errorf("%sのresult mappingに名前のないraw列があります", subject)
		}
		if mapping.Attribute == "" {
			return MappedResult{}, fmt.Errorf("%sのresult column %qにSemantic attributeが指定されていません", subject, mapping.Column)
		}
		if _, exists := usedRawColumns[mapping.Column]; exists {
			return MappedResult{}, fmt.Errorf("%sのresult mappingでraw列 %qが重複しています", subject, mapping.Column)
		}
		if _, exists := usedAttributes[mapping.Attribute]; exists {
			return MappedResult{}, fmt.Errorf("%sのresult mappingでSemantic attribute %sが重複しています", subject, mapping.Attribute)
		}
		if _, exists := rawIndex[mapping.Column]; !exists {
			return MappedResult{}, fmt.Errorf(
				"%sのresult mappingが要求するraw列 %qが実行結果にありません",
				subject,
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
				"%sのraw result列 %qにSemantic attributeのmappingがありません",
				subject,
				column,
			)
		}
	}

	mappedRows := make([][]any, len(raw.Rows))
	for rowIndex, rawRow := range raw.Rows {
		if len(rawRow) != len(raw.Columns) {
			return MappedResult{}, fmt.Errorf(
				"%sのraw result row %dは列数が一致しません: columns=%d values=%d",
				subject,
				rowIndex,
				len(raw.Columns),
				len(rawRow),
			)
		}
		mappedRow := make([]any, len(mappings))
		for i, mapping := range mappings {
			mappedRow[i] = rawRow[rawIndex[mapping.Column]]
		}
		mappedRows[rowIndex] = mappedRow
	}

	return MappedResult{Columns: mappedColumns, Rows: mappedRows}, nil
}

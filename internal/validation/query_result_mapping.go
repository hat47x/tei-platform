package validation

import (
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/subject"
)

func validateQueryResultMappings(w ir.World) []Diagnostic {
	attributes := map[core.CanonicalID]struct{}{}
	for _, namespace := range w.Semantics {
		for _, data := range namespace.Data {
			for _, attribute := range data.Attributes {
				attributes[attribute.ID] = struct{}{}
			}
		}
	}

	var out []Diagnostic
	for _, q := range w.Queries {
		if len(q.ResultColumns) == 0 {
			continue
		}
		qSubject := subject.Subject{Kind: "query", ID: string(q.ID)}
		seenColumns := map[string]struct{}{}
		seenAttributes := map[core.CanonicalID]struct{}{}
		for _, mapping := range q.ResultColumns {
			if mapping.Column == "" {
				out = append(out, errFor(qSubject, "QUERY_RESULT_COLUMN_REQUIRED", "Query %sのresult mappingにraw列名がありません", q.ID))
			} else {
				if _, duplicate := seenColumns[mapping.Column]; duplicate {
					out = append(out, errFor(qSubject, "DUPLICATE_QUERY_RESULT_COLUMN", "Query %sのresult mappingでraw列 %qが重複しています", q.ID, mapping.Column))
				}
				seenColumns[mapping.Column] = struct{}{}
			}

			if mapping.Attribute == "" {
				out = append(out, errFor(qSubject, "QUERY_RESULT_ATTRIBUTE_REQUIRED", "Query %sのresult mappingにSemantic attributeがありません", q.ID))
				continue
			}
			if _, duplicate := seenAttributes[mapping.Attribute]; duplicate {
				out = append(out, errFor(qSubject, "DUPLICATE_QUERY_RESULT_ATTRIBUTE", "Query %sのresult mappingでSemantic attribute %sが重複しています", q.ID, mapping.Attribute))
			}
			seenAttributes[mapping.Attribute] = struct{}{}
			if _, exists := attributes[mapping.Attribute]; !exists {
				out = append(out, errFor(qSubject, "UNKNOWN_QUERY_RESULT_ATTRIBUTE", "Query %sのresult mappingが存在しないSemantic attribute %sを参照しています", q.ID, mapping.Attribute))
			}
		}
	}
	return out
}

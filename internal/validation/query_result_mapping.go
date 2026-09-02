package validation

import (
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/subject"
)

// validateQueryResultMappingsは、Query authorが明示したresult column mappingの
// Semantic参照がWorkspace全体で解決できることを確認する。
//
// raw column labelとSemantic attributeの対応そのものはQueryが所有する。
// ここでは列名やPersistence Mappingから意味を推測せず、明示されたCanonical IDの存在だけを検査する。
func validateQueryResultMappings(w ir.World) []Diagnostic {
	semanticAttributes := map[core.CanonicalID]struct{}{}
	for _, namespace := range w.Semantics {
		for _, data := range namespace.Data {
			for _, attribute := range data.Attributes {
				semanticAttributes[attribute.ID] = struct{}{}
			}
		}
	}

	var out []Diagnostic
	for _, q := range w.Queries {
		querySubject := subject.Subject{Kind: "query", ID: string(q.ID)}
		for _, mapping := range q.ResultColumns {
			if _, ok := semanticAttributes[mapping.Attribute]; ok {
				continue
			}
			out = append(out, errFor(
				querySubject,
				"UNKNOWN_QUERY_RESULT_ATTRIBUTE",
				"Query %s のresult column %qは、存在しないSemantic attribute %sを参照しています",
				q.ID,
				mapping.Column,
				mapping.Attribute,
			))
		}
	}
	return out
}

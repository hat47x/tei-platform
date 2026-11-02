package validation

import (
	"strings"

	"tei/internal/ir"
	"tei/internal/subject"
)

// validateQueryInputsは、Queryが受け付けるパラメータの名前と型を検査する。
func validateQueryInputs(w ir.World) []Diagnostic {
	var out []Diagnostic

	types := map[string]bool{}
	for _, namespace := range w.Semantics {
		for _, typ := range namespace.Types {
			types[string(typ.ID)] = true
		}
	}
	for _, query := range w.Queries {
		querySubject := subject.Subject{Kind: "query", ID: string(query.ID)}
		seen := map[string]bool{}
		for _, input := range query.Inputs {
			name := string(input.Name)
			if seen[name] {
				out = append(out, errFor(querySubject, "QUERY_INPUT_DUPLICATE", "query %s declares input %s more than once", query.ID, name))
			}
			seen[name] = true
			if !types[string(input.Type)] && !strings.HasPrefix(string(input.Type), "std.") {
				out = append(out, errFor(querySubject, "UNKNOWN_QUERY_INPUT_TYPE", "query %s input %s refers to unknown type %s", query.ID, name, input.Type))
			}
		}
	}
	return out
}

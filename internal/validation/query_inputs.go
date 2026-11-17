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
			// Only a NAMED placeholder is detected: this check looks for the
			// input's own name after a prefix. A positional placeholder such as
			// $1 is valid PostgreSQL and the driver does bind it, so saying the
			// input is "not referenced" would be false. The message therefore
			// says what was not found and which forms are recognised, so an
			// author who wrote $1 knows what to change and why.
			if query.Source != "" && !strings.Contains(query.Source, ":"+name) && !strings.Contains(query.Source, "$"+name) && !strings.Contains(query.Source, "@"+name) {
				out = append(out, warnd("QUERY_INPUT_NOT_REFERENCED",
					"query %s input %s is not referenced by name in its SQL source; write it as :%s, $%s or @%s (a positional placeholder such as $1 is not matched)",
					query.ID, name, name, name, name))
			}
		}
	}
	return out
}

package workspace

import (
	"tei/internal/sourceindex"
	"tei/internal/validation"
)

// ProjectedDiagnostic keeps the semantic validation result intact and adds the
// authoring-side source references that can be resolved for its Subject.
// Validation itself does not depend on Source Index.
type ProjectedDiagnostic struct {
	Diagnostic validation.Diagnostic   `json:"diagnostic"`
	Sources    []sourceindex.Reference `json:"sources,omitempty"`
}

// ProjectDiagnostics resolves path-independent diagnostic Subjects through a
// Source Index. Definition references are authoritative for diagnostics and are
// preferred when present. A missing source is not an error: derived or
// synthetic subjects may legitimately have no physical source.
func ProjectDiagnostics(diagnostics []validation.Diagnostic, index sourceindex.Index) []ProjectedDiagnostic {
	out := make([]ProjectedDiagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		projected := ProjectedDiagnostic{Diagnostic: diagnostic}
		if diagnostic.Subject == nil {
			out = append(out, projected)
			continue
		}

		refs := index.ReferencesFor(*diagnostic.Subject)
		definitions := make([]sourceindex.Reference, 0, len(refs))
		for _, ref := range refs {
			if ref.Role == sourceindex.RoleDefinition {
				definitions = append(definitions, ref)
			}
		}
		if len(definitions) > 0 {
			projected.Sources = definitions
		} else {
			projected.Sources = refs
		}
		out = append(out, projected)
	}
	return out
}

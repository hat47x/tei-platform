package workspace

import (
	"sort"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/syntax"
	"tei/internal/validation"
)

// ProjectDiagnosticsWithLoadedDuplicates adds declaration candidates only when
// all local semantic files can be inspected from the bytes retained by Load.
// It does not alter the Source Index or identify a particular rejected source.
// Imported definitions and the implicit prelude cannot be located here.
func ProjectDiagnosticsWithLoadedDuplicates(diagnostics []validation.Diagnostic, index sourceindex.Index, loaded Result) []ProjectedDiagnostic {
	out := ProjectDiagnostics(diagnostics, index)
	if len(loaded.Imports) != 0 || loaded.sourceSnapshot == nil {
		return out
	}
	var needed bool
	for _, item := range out {
		if item.Diagnostic.Code == "DUPLICATE_SEMANTIC_ELEMENT" && item.Diagnostic.Subject != nil {
			needed = true
			break
		}
	}
	if !needed {
		return out
	}
	byID := make(map[string][]sourceindex.Reference)
	for _, path := range loaded.semanticSourceFiles {
		content, ok := loaded.sourceSnapshot[path]
		if !ok {
			return out
		}
		parsed, err := syntax.ParseSemanticFileAnchored(string(content))
		if err != nil {
			return out
		}
		for _, anchor := range parsed.Anchors {
			switch anchor.Kind {
			case "semantic-type", "semantic-data", "semantic-attribute", "semantic-relation", "semantic-capability":
			default:
				continue
			}
			rng, err := sourceindex.RangeFromOffsets(content, anchor.Anchor.Start.Offset, anchor.Anchor.End.Offset)
			if err != nil {
				return out
			}
			byID[anchor.ID] = append(byID[anchor.ID], sourceindex.Reference{
				Subject:      sourceindex.Subject{Kind: anchor.Kind, ID: anchor.ID},
				ArtifactPath: path, Range: rng, Role: sourceindex.RoleDefinition,
			})
		}
	}
	for i := range out {
		d := out[i].Diagnostic
		if d.Code != "DUPLICATE_SEMANTIC_ELEMENT" || d.Subject == nil || strings.HasPrefix(d.Subject.ID, "std.") {
			continue
		}
		refs := byID[d.Subject.ID]
		// A single local source may conflict with the built-in prelude; do not
		// claim that this is a complete set of declarations.
		if len(refs) < 2 {
			continue
		}
		sort.Slice(refs, func(i, j int) bool {
			if refs[i].ArtifactPath != refs[j].ArtifactPath {
				return refs[i].ArtifactPath < refs[j].ArtifactPath
			}
			return refs[i].Range.Start.Offset < refs[j].Range.Start.Offset
		})
		out[i].Sources = refs
	}
	return out
}

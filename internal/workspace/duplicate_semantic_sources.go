package workspace

import (
	"sort"

	"tei/internal/sourceindex"
	"tei/internal/stdlib"
	"tei/internal/syntax"
	"tei/internal/validation"
)

// ProjectDiagnosticsWithLoadedDuplicates adds declaration candidates when the
// local semantic files parsed by Load can be inspected from retained bytes.
// It does not alter the Source Index or identify a particular rejected source.
//
// The standard semantic prelude is an in-process library value with no source
// file. A duplicate against it therefore reports the local declarations and
// names "standard" as an unresolved origin instead of inventing a position.
//
// Imported Workspaces cannot reach DUPLICATE_SEMANTIC_ELEMENT with a local
// declaration: Load rejects an import whose namespace is already owned before
// validation runs. An unrelated import no longer suppresses local candidates.
func ProjectDiagnosticsWithLoadedDuplicates(diagnostics []validation.Diagnostic, index sourceindex.Index, loaded Result) []ProjectedDiagnostic {
	out := ProjectDiagnostics(diagnostics, index)
	if loaded.sourceSnapshot == nil {
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
		if d.Code != "DUPLICATE_SEMANTIC_ELEMENT" || d.Subject == nil {
			continue
		}
		refs := byID[d.Subject.ID]
		var unresolved []string
		if standardSemanticSubject(d.Subject.ID) {
			unresolved = append(unresolved, "standard")
		}
		if len(refs) < 2 && len(unresolved) == 0 {
			// The diagnostic proves that another declaration exists, but its
			// position is not in the retained snapshot. Do not present the one
			// known declaration as the complete set.
			if len(refs) == 0 {
				continue
			}
			unresolved = append(unresolved, "unknown")
		}
		sort.Slice(refs, func(i, j int) bool {
			if refs[i].ArtifactPath != refs[j].ArtifactPath {
				return refs[i].ArtifactPath < refs[j].ArtifactPath
			}
			return refs[i].Range.Start.Offset < refs[j].Range.Start.Offset
		})
		out[i].Sources = refs
		if len(unresolved) > 0 {
			sort.Strings(unresolved)
			out[i].UnresolvedSourceOrigins = unresolved
		}
	}
	return out
}

// standardSemanticSubject reports whether the implicit prelude declares the
// subject. Only the prelude's own identities count; a local std.* name that is
// not in the prelude is not treated as a standard declaration.
func standardSemanticSubject(id string) bool {
	for _, typ := range stdlib.SemanticNamespace().Types {
		if string(typ.ID) == id {
			return true
		}
	}
	return false
}

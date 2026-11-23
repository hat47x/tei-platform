// Package fromworkspace converts the two Workspace diagnostics that carry
// source positions into diag.Record. It is a separate package so that diag,
// and every generator that imports it, does not depend on the Workspace
// loader.
//
// Offsets and columns are taken over unchanged: both sides count UTF-8 bytes,
// and both number lines and columns from 1.
package fromworkspace

import (
	"tei/internal/codegen/diag"
	"tei/internal/core"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// SourceDiagnostic converts a source-direct finding, such as a parse error.
func SourceDiagnostic(d workspace.SourceDiagnostic) diag.Record {
	r := diag.Record{
		Code:     d.Code,
		Severity: string(d.Severity),
		Message:  d.Message,
	}
	place(&r, d.Source.ArtifactPath, d.Source.Range)
	return r
}

// ProjectedDiagnostic converts a semantic finding and the sources resolved for
// its Subject. The Subject's ID becomes Element. The first source gives File,
// Range, Line and Column; each further source becomes a Related entry whose
// Role is the source reference's role.
//
// UnresolvedSourceOrigins has no place in a Record and is not carried over.
// It names origins whose position is unknown; a converted record then simply
// has fewer places than the duplicate has declarations.
func ProjectedDiagnostic(d workspace.ProjectedDiagnostic) diag.Record {
	r := diag.Record{
		Code:     d.Diagnostic.Code,
		Severity: string(d.Diagnostic.Severity),
		Message:  d.Diagnostic.Message,
	}
	if d.Diagnostic.Subject != nil {
		r.Element = core.CanonicalID(d.Diagnostic.Subject.ID)
	}
	for i, ref := range d.Sources {
		if i == 0 {
			place(&r, ref.ArtifactPath, ref.Range)
			continue
		}
		r.Related = append(r.Related, diag.Related{
			File:    ref.ArtifactPath,
			Range:   &diag.Range{Start: ref.Range.Start.Offset, End: ref.Range.End.Offset},
			Element: core.CanonicalID(ref.Subject.ID),
			Role:    string(ref.Role),
		})
	}
	return r
}

func place(r *diag.Record, path string, span sourceindex.Range) {
	r.File = path
	r.Range = &diag.Range{Start: span.Start.Offset, End: span.End.Offset}
	r.Line = span.Start.Line
	r.Column = span.Start.Column
}

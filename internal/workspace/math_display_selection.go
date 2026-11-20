package workspace

import (
	"fmt"
	"strings"

	"tei/internal/sourceindex"
)

// ExplicitMathPreviewRequest names one range chosen by the caller in a
// product-selected *.tei.md Source. It is NOT automatic Markdown recognition.
// SourceFingerprint must come from the exact Load-time Source artifact.
type ExplicitMathPreviewRequest struct {
	Path              string
	SourceFingerprint string
	StartOffset       int
	EndOffset         int
}

// ExplicitMathPreview is a disposable, read-only display input. Its occurrence
// is local to one document revision and is never a semantic identifier,
// accepted expression, writer target, or execution grant.
type ExplicitMathPreview struct {
	Path              string
	SourceFingerprint string
	Occurrence        string
	SourceRange       sourceindex.Range
	TeXRange          sourceindex.Range
	Delimiter         string
	Display           bool
	TeX               string
}

// SelectExplicitMathPreview extracts exactly one explicitly selected TeX
// fragment from bytes actually consumed by the selected Markdown Reader at
// Load time. No filesystem read, auto-detection, renderer, or meaning change
// occurs. A caller wanting to show the current source must separately check
// freshness and reselect against the current Load; this historical snapshot
// alone provides no current editor/write qualification.
//
// K0 accepts only paired \(...\), \[...\], and $$...$$ delimiters. Single
// dollars are intentionally unsupported because they can denote currency.
// This is not a general Markdown math tokenizer; selecting code/quoted text
// explicitly grants only a read-only preview, never definition authority.
func SelectExplicitMathPreview(root string, loaded Result, request ExplicitMathPreviewRequest) (ExplicitMathPreview, error) {
	var empty ExplicitMathPreview
	realRoot, err := canonicalWorkspaceRoot(root)
	if err != nil {
		return empty, err
	}
	if loaded.sourceSnapshot == nil || loaded.sourceSnapshotRoot != realRoot {
		return empty, fmt.Errorf("math preview requires a Load snapshot of the same Workspace")
	}
	path, err := sourceindex.NormalizeWorkspacePath(request.Path)
	if err != nil || path != request.Path || !strings.HasSuffix(path, ".tei.md") {
		return empty, fmt.Errorf("math preview requires a normalized selected *.tei.md path")
	}
	selected := false
	for _, entry := range loaded.AuthoringSources {
		if entry.Path == path {
			selected = true
			break
		}
	}
	if !selected {
		return empty, fmt.Errorf("math preview source %q was not read by a selected authoring language", path)
	}
	source, ok := loaded.sourceSnapshot[path]
	if !ok {
		return empty, fmt.Errorf("math preview source %q has no parsed Load-time bytes", path)
	}

	// Use the existing Source Index to verify the identity of those bytes;
	// do not introduce a second Source revision authority.
	index, err := BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		return empty, fmt.Errorf("math preview Source Index: %w", err)
	}
	identity := sourceindex.Fingerprint(source)
	found := false
	for _, artifact := range index.Artifacts {
		if artifact.Path == path && artifact.Fingerprint == identity {
			found = true
			break
		}
	}
	if !found || request.SourceFingerprint == "" || request.SourceFingerprint != identity {
		return empty, fmt.Errorf("math preview source fingerprint is missing, stale, or unbound")
	}
	if request.StartOffset < 0 || request.EndOffset > len(source) || request.StartOffset >= request.EndOffset {
		return empty, fmt.Errorf("math preview has an invalid source byte range")
	}
	selection := string(source[request.StartOffset:request.EndOffset])
	var delimiter string
	var display bool
	var opening, closing string
	switch {
	case strings.HasPrefix(selection, "\\(") && strings.HasSuffix(selection, "\\)"):
		delimiter, opening, closing = "tex-inline-parens", "\\(", "\\)"
	case strings.HasPrefix(selection, "\\[") && strings.HasSuffix(selection, "\\]"):
		delimiter, display, opening, closing = "tex-display-brackets", true, "\\[", "\\]"
	case strings.HasPrefix(selection, "$$") && strings.HasSuffix(selection, "$$"):
		delimiter, display, opening, closing = "tex-display-dollars", true, "$$", "$$"
	default:
		return empty, fmt.Errorf("math preview delimiter is unsupported; select one complete supported TeX expression")
	}
	if len(selection) < len(opening)+len(closing) {
		return empty, fmt.Errorf("math preview has no expression body")
	}
	body := selection[len(opening) : len(selection)-len(closing)]
	if strings.TrimSpace(body) == "" || strings.Contains(body, closing) {
		return empty, fmt.Errorf("math preview has an empty or ambiguous expression body")
	}
	whole, err := sourceindex.RangeFromOffsets(source, request.StartOffset, request.EndOffset)
	if err != nil {
		return empty, err
	}
	tex, err := sourceindex.RangeFromOffsets(source, request.StartOffset+len(opening), request.EndOffset-len(closing))
	if err != nil {
		return empty, err
	}
	return ExplicitMathPreview{
		Path:              path,
		SourceFingerprint: identity,
		Occurrence:        fmt.Sprintf("%s@%s:%d-%d", path, identity, request.StartOffset, request.EndOffset),
		SourceRange:       whole,
		TeXRange:          tex,
		Delimiter:         delimiter,
		Display:           display,
		TeX:               body,
	}, nil
}

package workspace

import (
	"errors"
	"fmt"
	"path/filepath"

	"tei/internal/sourceindex"
	"tei/internal/syntax"
	"tei/internal/validation"
)

// SourceError preserves a syntax error together with the physical source file
// that was being loaded. It does not invent a Canonical / Semantic Subject.
type SourceError struct {
	Path    string
	Range   sourceindex.Range
	Message string
	Err     error
}

func (e *SourceError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.Path, e.Range.Start.Line, e.Range.Start.Column, e.Message)
}

func (e *SourceError) Unwrap() error { return e.Err }

// DirectSource identifies the physical source location of a diagnostic that is
// produced before a Canonical subject can exist, such as a parser error.
type DirectSource struct {
	ArtifactPath string            `json:"artifactPath"`
	Range        sourceindex.Range `json:"range"`
}

// SourceDiagnostic is used for source-direct findings. Semantic validation
// findings use validation.Diagnostic + Subject and are projected separately.
type SourceDiagnostic struct {
	Code     string              `json:"code"`
	Severity validation.Severity `json:"severity"`
	Message  string              `json:"message"`
	Source   DirectSource        `json:"source"`
}

func wrapSourceError(path string, err error) error {
	start, end, ok := syntax.ErrorRange(err)
	if !ok {
		return fmt.Errorf("%s: %w", path, err)
	}
	message, _ := syntax.ErrorMessage(err)
	return &SourceError{
		Path: path,
		Range: sourceindex.Range{
			Start: sourceindex.Position{Offset: start.Offset, Line: start.Line, Column: start.Column},
			End:   sourceindex.Position{Offset: end.Offset, Line: end.Line, Column: end.Column},
		},
		Message: message,
		Err:     err,
	}
}

// SourceDiagnosticFromError converts a loader error into a source-direct
// diagnostic when the error carries syntax position information. The source
// path is normalized relative to the Workspace root when possible.
func SourceDiagnosticFromError(root string, err error) (SourceDiagnostic, bool) {
	var sourceErr *SourceError
	if !errors.As(err, &sourceErr) {
		return SourceDiagnostic{}, false
	}
	p := sourceErr.Path
	if root != "" {
		if rel, relErr := filepath.Rel(root, p); relErr == nil {
			p = rel
		}
	}
	p = filepath.ToSlash(p)
	return SourceDiagnostic{
		Code:     "PARSE_ERROR",
		Severity: validation.Error,
		Message:  sourceErr.Message,
		Source: DirectSource{
			ArtifactPath: p,
			Range:        sourceErr.Range,
		},
	}, true
}

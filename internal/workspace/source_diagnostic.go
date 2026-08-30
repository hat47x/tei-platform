package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/sourceindex"
	"tei/internal/syntax"
	"tei/internal/validation"
)

// SourceDiagnostic represents an authoring error that exists before a
// path-independent semantic Subject can be constructed. It therefore points
// directly to a physical source range instead of inventing a Canonical target.
type SourceDiagnostic struct {
	Code         string              `json:"code"`
	Severity     validation.Severity `json:"severity"`
	Message      string              `json:"message"`
	ArtifactPath string              `json:"artifactPath"`
	Range        sourceindex.Range   `json:"range"`
}

// DiagnosticFromLoadError projects a parser failure returned by Load into a
// source-direct diagnostic. Non-parser and non-source errors are left untouched
// so callers can continue treating them as operational failures.
func DiagnosticFromLoadError(root string, err error) (SourceDiagnostic, bool) {
	var sourceErr *SourceError
	if !errors.As(err, &sourceErr) {
		return SourceDiagnostic{}, false
	}
	var parseErr *syntax.ParseError
	if !errors.As(err, &parseErr) {
		return SourceDiagnostic{}, false
	}

	rel, relErr := filepath.Rel(root, sourceErr.Path)
	if relErr != nil {
		return SourceDiagnostic{}, false
	}
	rel = filepath.ToSlash(rel)
	rel, relErr = sourceindex.NormalizeWorkspacePath(rel)
	if relErr != nil {
		return SourceDiagnostic{}, false
	}
	content, readErr := os.ReadFile(sourceErr.Path)
	if readErr != nil {
		return SourceDiagnostic{}, false
	}

	end := parseErr.End.Offset
	if end < parseErr.Pos.Offset || end > len(content) {
		end = parseErr.Pos.Offset
	}
	rng, rangeErr := sourceindex.RangeFromOffsets(content, parseErr.Pos.Offset, end)
	if rangeErr != nil {
		return SourceDiagnostic{}, false
	}

	return SourceDiagnostic{
		Code:         "PARSE_ERROR",
		Severity:     validation.Error,
		Message:      parseErr.Msg,
		ArtifactPath: rel,
		Range:        rng,
	}, true
}

func (d SourceDiagnostic) Location() string {
	return fmt.Sprintf("%s:%d:%d", d.ArtifactPath, d.Range.Start.Line, d.Range.Start.Column)
}

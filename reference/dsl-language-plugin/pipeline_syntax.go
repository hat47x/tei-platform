package languagereference

import (
	"fmt"
	"strings"
	"unicode"
)

// PipelineSpan is a half-open byte range in the original source.
// It is only a reference parser offset, not TEI's common source-index span.
type PipelineSpan struct {
	Start int
	End   int
}

// PipelineToken is a syntax candidate; it does not carry semantic acceptance.
type PipelineToken struct {
	ID   string
	Span PipelineSpan
}

// PipelineSyntax stores source identities without deciding operator versions.
type PipelineSyntax struct {
	Input     PipelineToken
	Operators []PipelineToken
}

// ParsePipelineSyntax performs only source-level parsing. The source spans
// survive lowering and can later be projected into the shared diagnostics ABI.
func ParsePipelineSyntax(source string) (PipelineSyntax, error) {
	parts := strings.Split(source, "|>")
	if len(parts) < 2 {
		return PipelineSyntax{}, fmt.Errorf("pipeline requires at least one operator")
	}
	result := PipelineSyntax{Operators: make([]PipelineToken, 0, len(parts)-1)}
	offset := 0
	for i, raw := range parts {
		trimmedLeft := strings.TrimLeftFunc(raw, unicode.IsSpace)
		token := strings.TrimSpace(raw)
		if token == "" || strings.IndexFunc(token, unicode.IsSpace) >= 0 {
			if i == 0 {
				return PipelineSyntax{}, fmt.Errorf("invalid pipeline input type")
			}
			return PipelineSyntax{}, fmt.Errorf("step %d: invalid operator name", i-1)
		}
		start := offset + len(raw) - len(trimmedLeft)
		candidate := PipelineToken{ID: token, Span: PipelineSpan{Start: start, End: start + len(token)}}
		if i == 0 {
			result.Input = candidate
		} else {
			result.Operators = append(result.Operators, candidate)
		}
		offset += len(raw) + len("|>")
	}
	return result, nil
}

package languagereference

import (
	"fmt"
	"strings"

	"tei/internal/core"
)

// ExpandInvocationSyntax is a reference language's parsed, named expand call.
// Its spans use UTF-8 byte offsets in the original source, not rune indexes.
// This is not a new shared parser ABI or an accepted canonical meaning.
type ExpandInvocationSyntax struct {
	RelationID   core.CanonicalID
	Policy       RelationMatchPolicy
	CallSpan     PipelineSpan
	RelationSpan PipelineSpan
	PolicySpans  map[string]PipelineSpan
}

// ParseExpandInvocation parses only the fixed reference form:
// std.op.expand(relation: x.y, unmatched: reject, null_key: no-match)
// Argument order and whitespace are presentation details; all three named
// arguments are required. Grammar expansion is owned by a Language Plugin.
func ParseExpandInvocation(source string) (ExpandInvocationSyntax, error) {
	full := PipelineSpan{Start: 0, End: len(source)}
	trimmed := strings.TrimSpace(source)
	start := strings.Index(source, trimmed)
	if trimmed == "" {
		return ExpandInvocationSyntax{}, &PipelineDiagnostic{Span: full, Step: 0, Cause: fmt.Errorf("empty expand invocation")}
	}
	open := strings.IndexByte(trimmed, '(')
	if open < 0 || !strings.HasSuffix(trimmed, ")") ||
		strings.TrimSpace(trimmed[:open]) != "std.op.expand" {
		return ExpandInvocationSyntax{}, &PipelineDiagnostic{Span: full, Step: 0, Cause: fmt.Errorf("expected std.op.expand(...)")}
	}
	body := trimmed[open+1 : len(trimmed)-1]
	bodyStart := start + open + 1
	result := ExpandInvocationSyntax{
		CallSpan:    PipelineSpan{Start: start, End: start + len(trimmed)},
		PolicySpans: make(map[string]PipelineSpan, 3),
	}
	seen := make(map[string]bool, 3)
	cursor := 0
	for _, segment := range strings.Split(body, ",") {
		piece := strings.TrimSpace(segment)
		pieceStart := bodyStart + cursor + strings.Index(segment, piece)
		cursor += len(segment) + 1
		span := PipelineSpan{Start: pieceStart, End: pieceStart + len(piece)}
		colon := strings.IndexByte(piece, ':')
		if piece == "" || colon <= 0 || strings.Count(piece, ":") != 1 {
			return ExpandInvocationSyntax{}, &PipelineDiagnostic{Span: span, Step: 0, Cause: fmt.Errorf("expected named expand argument")}
		}
		name := strings.TrimSpace(piece[:colon])
		valueRaw := piece[colon+1:]
		value := strings.TrimSpace(valueRaw)
		valueStart := pieceStart + colon + 1 + strings.Index(valueRaw, value)
		valueSpan := PipelineSpan{Start: valueStart, End: valueStart + len(value)}
		if value == "" {
			return ExpandInvocationSyntax{}, &PipelineDiagnostic{Span: valueSpan, Step: 0, Cause: fmt.Errorf("empty value for %q", name)}
		}
		if seen[name] {
			return ExpandInvocationSyntax{}, &PipelineDiagnostic{Span: span, Step: 0, Cause: fmt.Errorf("duplicate expand argument %q", name)}
		}
		seen[name] = true
		switch name {
		case "relation":
			result.RelationID = core.CanonicalID(value)
			result.RelationSpan = valueSpan
		case "unmatched":
			result.Policy.Unmatched = UnmatchedPolicy(value)
			result.PolicySpans[name] = valueSpan
		case "null_key":
			result.Policy.NullKey = NullKeyPolicy(value)
			result.PolicySpans[name] = valueSpan
		default:
			return ExpandInvocationSyntax{}, &PipelineDiagnostic{Span: span, Step: 0, Cause: fmt.Errorf("unknown expand argument %q", name)}
		}
	}
	for _, name := range []string{"relation", "unmatched", "null_key"} {
		if !seen[name] {
			return ExpandInvocationSyntax{}, &PipelineDiagnostic{Span: result.CallSpan, Step: 0, Cause: fmt.Errorf("missing required expand argument %q", name)}
		}
	}
	if err := result.Policy.Validate(); err != nil {
		span := result.CallSpan
		if result.Policy.Unmatched != UnmatchedReject && result.Policy.Unmatched != UnmatchedDiscard &&
			result.Policy.Unmatched != UnmatchedRetainAbsent {
			span = result.PolicySpans["unmatched"]
		} else {
			span = result.PolicySpans["null_key"]
		}
		return ExpandInvocationSyntax{}, &PipelineDiagnostic{Span: span, Step: 0, Cause: err}
	}
	return result, nil
}

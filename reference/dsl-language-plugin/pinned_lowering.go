package languagereference

import "fmt"

// ResolvedPipelineStep carries the original source offset alongside a
// checked contract. It remains an unaccepted reference candidate.
type ResolvedPipelineStep struct {
	Span      PipelineSpan
	InputType string
	Contract  OperatorContract
}

// LowerPinnedPipeline accepts syntax and exact pins from outside. It does not
// obtain pins from a catalog and never elevates a candidate to acceptance.
func LowerPinnedPipeline(syntax PipelineSyntax, pins []OperatorContract, catalog []OperatorContract) ([]ResolvedPipelineStep, error) {
	if syntax.Input.ID == "" || len(syntax.Operators) == 0 {
		return nil, ErrOperatorInvalid
	}
	if len(pins) != len(syntax.Operators) {
		index := len(pins)
		if index >= len(syntax.Operators) {
			index = len(syntax.Operators) - 1
		}
		return nil, &PipelineDiagnostic{
			Span: syntax.Operators[index].Span, Step: index,
			Cause: fmt.Errorf("pipeline has %d operators but %d accepted pins", len(syntax.Operators), len(pins)),
		}
	}
	current := syntax.Input.ID
	result := make([]ResolvedPipelineStep, 0, len(pins))
	for i, token := range syntax.Operators {
		pin := pins[i]
		if token.ID == "" || pin.ID != token.ID {
			return nil, &PipelineDiagnostic{
				Span: token.Span, Step: i,
				Cause: fmt.Errorf("%w: pinned operator %q does not match source %q", ErrOperatorInvalid, pin.ID, token.ID),
			}
		}
		resolved, err := ResolveOperator(pin, catalog)
		if err != nil {
			return nil, &PipelineDiagnostic{Span: token.Span, Step: i, Cause: err}
		}
		if resolved.InputType != current {
			return nil, &PipelineDiagnostic{
				Span: token.Span, Step: i,
				Cause: fmt.Errorf("%w: expected %s, got %s", ErrOperatorInvalid, current, resolved.InputType),
			}
		}
		result = append(result, ResolvedPipelineStep{Span: token.Span, InputType: current, Contract: resolved})
		current = resolved.OutputType
	}
	return result, nil
}

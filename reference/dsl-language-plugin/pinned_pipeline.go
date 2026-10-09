package languagereference

import "fmt"

// ParsePinnedPipeline validates already accepted operator pins against a
// syntax-only candidate, without silently choosing a registered version.
func ParsePinnedPipeline(source string, pins []OperatorContract, catalog []OperatorContract) (string, []PinnedCall, error) {
	syntax, err := ParsePipelineSyntax(source)
	if err != nil {
		return "", nil, err
	}
	if len(pins) != len(syntax.Operators) {
		return "", nil, fmt.Errorf("pipeline has %d operators but %d accepted pins", len(syntax.Operators), len(pins))
	}
	current := syntax.Input.ID
	calls := make([]PinnedCall, 0, len(pins))
	for i, token := range syntax.Operators {
		pin := pins[i]
		if pin.ID != token.ID {
			return "", nil, fmt.Errorf("step %d: pinned operator %q does not match source %q", i, pin.ID, token.ID)
		}
		resolved, err := ResolveOperator(pin, catalog)
		if err != nil {
			return "", nil, fmt.Errorf("step %d: %w", i, err)
		}
		if resolved.InputType != current {
			return "", nil, fmt.Errorf("step %d: %w: expected %s, got %s", i, ErrOperatorInvalid, current, resolved.InputType)
		}
		calls = append(calls, PinnedCall{Contract: resolved, InputType: current})
		current = resolved.OutputType
	}
	return syntax.Input.ID, calls, nil
}

package languagereference

import (
	"fmt"
	"strings"
	"unicode"
)

// ParsePinnedPipeline separates syntax from meaning. Pins must be supplied by
// the caller from an already accepted semantic set. The parser cannot choose
// an operator version or fingerprint by name alone.
func ParsePinnedPipeline(source string, pins []OperatorContract, catalog []OperatorContract) (string, []PinnedCall, error) {
	parts := strings.Split(source, "|>")
	if len(parts) < 2 {
		return "", nil, fmt.Errorf("pipeline requires at least one operator")
	}
	input := strings.TrimSpace(parts[0])
	if input == "" || strings.IndexFunc(input, unicode.IsSpace) >= 0 {
		return "", nil, fmt.Errorf("invalid pipeline input type")
	}
	if len(pins) != len(parts)-1 {
		return "", nil, fmt.Errorf("pipeline has %d operators but %d accepted pins", len(parts)-1, len(pins))
	}
	current := input
	calls := make([]PinnedCall, 0, len(pins))
	for i, raw := range parts[1:] {
		id := strings.TrimSpace(raw)
		if id == "" || strings.IndexFunc(id, unicode.IsSpace) >= 0 {
			return "", nil, fmt.Errorf("step %d: invalid operator name", i)
		}
		pin := pins[i]
		if pin.ID != id {
			return "", nil, fmt.Errorf("step %d: pinned operator %q does not match source %q", i, pin.ID, id)
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
	return input, calls, nil
}

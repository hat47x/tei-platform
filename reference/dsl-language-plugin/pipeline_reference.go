package languagereference

import (
	"fmt"
	"strings"
	"unicode"
)

// ParsePipeline is a deliberately small, opt-in syntax adapter. It lowers a
// source line to operator calls without executing them or granting acceptance.
// Grammar: <input-type> |> <operator-id> |> <operator-id> ...
// Type resolution is delegated to the pinned operator contracts.
func ParsePipeline(source string, catalog []Operator) (string, []Call, error) {
	parts := strings.Split(source, "|>")
	if len(parts) < 2 {
		return "", nil, fmt.Errorf("pipeline requires at least one operator")
	}
	input := strings.TrimSpace(parts[0])
	if input == "" || strings.IndexFunc(input, unicode.IsSpace) >= 0 {
		return "", nil, fmt.Errorf("invalid pipeline input type")
	}
	current := input
	calls := make([]Call, 0, len(parts)-1)
	for i, raw := range parts[1:] {
		id := strings.TrimSpace(raw)
		if id == "" || strings.IndexFunc(id, unicode.IsSpace) >= 0 {
			return "", nil, fmt.Errorf("step %d: invalid operator name", i)
		}
		call := Call{OperatorID: id, InputType: current}
		op, err := ResolveCall(call, catalog)
		if err != nil {
			return "", nil, fmt.Errorf("step %d: %w", i, err)
		}
		calls = append(calls, call)
		current = op.OutputType
	}
	return input, calls, nil
}

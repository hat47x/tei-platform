package languagereference

// ParsePinnedPipeline is a compatibility-facing adapter for the E2 reference.
// Callers needing source spans should use ParsePipelineSyntax followed by
// LowerPinnedPipeline; no reparse or unpinned resolution is required.
func ParsePinnedPipeline(source string, pins []OperatorContract, catalog []OperatorContract) (string, []PinnedCall, error) {
	syntax, err := ParsePipelineSyntax(source)
	if err != nil {
		return "", nil, err
	}
	steps, err := LowerPinnedPipeline(syntax, pins, catalog)
	if err != nil {
		return "", nil, err
	}
	calls := make([]PinnedCall, 0, len(steps))
	for _, step := range steps {
		calls = append(calls, PinnedCall{Contract: step.Contract, InputType: step.InputType})
	}
	return syntax.Input.ID, calls, nil
}

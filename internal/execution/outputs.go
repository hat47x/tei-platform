package execution

import (
	"context"

	"tei/internal/core"
	"tei/internal/semantic"
)

// RowReader reads the rows a plan declares as output. It only reads: the output
// is a projection, so nothing here may change the data.
type RowReader interface {
	ReadRows(ctx context.Context, data core.CanonicalID) (int, error)
}

// OutputResult is one declared output after the run.
type OutputResult struct {
	Data string `json:"data"`
	Rows int    `json:"rows"`
	Err  string `json:"err,omitempty"`
}

// ProjectOutputs reads every output the plan declares. A failed read is recorded
// in the result rather than stopping the other outputs, because one unreadable
// projection must not hide the rest.
func ProjectOutputs(ctx context.Context, plan semantic.RunPlan, reader RowReader) []OutputResult {
	results := make([]OutputResult, 0, len(plan.Outputs))
	for _, output := range plan.Outputs {
		result := OutputResult{Data: string(output)}
		if reader == nil {
			result.Err = "no reader is configured"
			results = append(results, result)
			continue
		}
		rows, err := reader.ReadRows(ctx, output)
		if err != nil {
			result.Err = err.Error()
			results = append(results, result)
			continue
		}
		result.Rows = rows
		results = append(results, result)
	}
	return results
}

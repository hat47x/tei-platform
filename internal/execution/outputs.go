package execution

import (
	"context"
	"fmt"
	"strings"

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

// RenderOutputs renders the projection as a report body: one row per declared
// output with the number of rows read, or the reason the read failed. The report
// is a projection of the run, so it says nothing about the data itself.
func RenderOutputs(plan semantic.RunPlan, results []OutputResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## 出力: %s\n\n", plan.ID)
	if len(results) == 0 {
		b.WriteString("出力の宣言はありません。\n")
		return b.String()
	}
	b.WriteString("| 出力 | 行数 | 備考 |\n| --- | ---: | --- |\n")
	for _, result := range results {
		if result.Err != "" {
			fmt.Fprintf(&b, "| %s | - | %s |\n", result.Data, result.Err)
			continue
		}
		fmt.Fprintf(&b, "| %s | %d | |\n", result.Data, result.Rows)
	}
	return b.String()
}

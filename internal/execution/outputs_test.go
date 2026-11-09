package execution_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/execution"
	"tei/internal/semantic"
)

type fakeReader struct {
	rows map[string]int
	err  error
}

func (r fakeReader) ReadRows(_ context.Context, data core.CanonicalID) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	return r.rows[string(data)], nil
}

func outputPlan() semantic.RunPlan {
	return semantic.RunPlan{ID: "sales.monthly_close", Outputs: []core.CanonicalID{"sales.close_statement", "sales.check_list"}}
}

func TestProjectOutputsReadsEveryDeclaredOutput(t *testing.T) {
	results := execution.ProjectOutputs(context.Background(), outputPlan(), fakeReader{rows: map[string]int{"sales.close_statement": 12, "sales.check_list": 3}})
	if len(results) != 2 {
		t.Fatalf("every output must be read: %+v", results)
	}
	if results[0].Rows != 12 || results[0].Err != "" {
		t.Fatalf("unexpected first result: %+v", results[0])
	}
	if results[1].Rows != 3 {
		t.Fatalf("unexpected second result: %+v", results[1])
	}
}

func TestProjectOutputsRecordsAFailedReadAndContinues(t *testing.T) {
	results := execution.ProjectOutputs(context.Background(), outputPlan(), fakeReader{err: errors.New("read failed")})
	if len(results) != 2 {
		t.Fatalf("one failed read must not hide the other outputs: %+v", results)
	}
	for _, result := range results {
		if result.Err != "read failed" {
			t.Fatalf("the failure must be recorded: %+v", result)
		}
	}
}

func TestProjectOutputsWithoutAReaderSaysSo(t *testing.T) {
	results := execution.ProjectOutputs(context.Background(), outputPlan(), nil)
	if len(results) != 2 || results[0].Err == "" {
		t.Fatalf("a missing reader must be reported: %+v", results)
	}
}

func TestProjectOutputsWithoutOutputsIsEmpty(t *testing.T) {
	if results := execution.ProjectOutputs(context.Background(), semantic.RunPlan{}, nil); len(results) != 0 {
		t.Fatalf("a plan without outputs has nothing to project: %+v", results)
	}
}
func TestRenderOutputsListsEveryOutput(t *testing.T) {
	plan := outputPlan()
	report := execution.RenderOutputs(plan, execution.ProjectOutputs(context.Background(), plan, fakeReader{rows: map[string]int{"sales.close_statement": 12, "sales.check_list": 0}}))
	wants := []string{
		"## 出力: sales.monthly_close",
		"| sales.close_statement | 12 | |",
		"| sales.check_list | 0 | |",
	}
	for _, want := range wants {
		if !strings.Contains(report, want) {
			t.Fatalf("report lost %q:\n%s", want, report)
		}
	}
}

func TestRenderOutputsShowsAFailedRead(t *testing.T) {
	report := execution.RenderOutputs(outputPlan(), []execution.OutputResult{{Data: "sales.close_statement", Err: "read failed"}})
	if !strings.Contains(report, "| sales.close_statement | - | read failed |") {
		t.Fatalf("a failed read must be visible in the report:\n%s", report)
	}
}

func TestRenderOutputsSaysWhenNothingIsDeclared(t *testing.T) {
	report := execution.RenderOutputs(semantic.RunPlan{ID: "sales.empty"}, nil)
	if !strings.Contains(report, "出力の宣言はありません") {
		t.Fatalf("an empty projection must say so:\n%s", report)
	}
}

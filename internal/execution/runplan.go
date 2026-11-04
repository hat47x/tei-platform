package execution

import (
	"context"

	"tei/internal/core"
	"tei/internal/runlog"
	"tei/internal/semantic"
)

// Journal is what the plan runner needs to record and read runs.
type Journal interface {
	Append(runlog.Record) error
	Last(string) (runlog.Record, bool, error)
}

// StepResult is the outcome of one operation in the plan.
type StepResult struct {
	Operation string `json:"operation"`
	Status    string `json:"status"`
	Detail    string `json:"detail,omitempty"`
}

// StepFunc performs one operation. A failure is reported in the result, not as
// an error, so the plan can decide how to record it.
type StepFunc func(ctx context.Context, operation core.CanonicalID) StepResult

// RunPlan identifies the run, applies the start/resume/refuse rules, and runs the
// operations in the declared order. The refusal is returned as a decision, not an
// error. Resuming is not implemented yet: the progress of the steps is not
// recorded, so resuming would silently repeat work.
func RunPlan(ctx context.Context, plan semantic.RunPlan, keyValues []string, journal Journal, step StepFunc) (runlog.Decision, []StepResult, error) {
	runKey := runlog.RunKey(string(plan.ID), keyValues)
	last, found, err := journal.Last(runKey)
	if err != nil {
		return runlog.Decision{}, nil, err
	}
	decision := runlog.Decide(plan.Resume, last, found)
	if decision.Action == runlog.Refuse {
		return decision, nil, nil
	}
	var done []string
	if decision.Action == runlog.Resume {
		done = append([]string(nil), last.Steps...)
	}

	record := runlog.Record{
		RunKey:     runKey,
		PlanID:     string(plan.ID),
		Key:        append([]string(nil), keyValues...),
		Attempt:    decision.Attempt,
		FenceEpoch: decision.FenceEpoch,
		Status:     runlog.Running,
		Steps:      append([]string(nil), done...),
	}
	if err := journal.Append(record); err != nil {
		return decision, nil, err
	}

	var results []StepResult
	for _, operation := range plan.Operations {
		if containsStep(done, string(operation)) {
			results = append(results, StepResult{Operation: string(operation), Status: "resolved", Detail: "already finished in an earlier attempt"})
			continue
		}
		result := step(ctx, operation)
		results = append(results, result)
		if result.Status != "resolved" {
			record.Status = runlog.Failed
			if err := journal.Append(record); err != nil {
				return decision, results, err
			}
			return decision, results, nil
		}
		record.Steps = append(record.Steps, string(operation))
		record.Status = runlog.Running
		if err := journal.Append(record); err != nil {
			return decision, results, err
		}
	}
	record.Status = runlog.Completed
	if err := journal.Append(record); err != nil {
		return decision, results, err
	}
	return decision, results, nil
}

// containsStep reports whether an operation already finished in this run.
func containsStep(steps []string, operation string) bool {
	for _, step := range steps {
		if step == operation {
			return true
		}
	}
	return false
}

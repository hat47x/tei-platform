package execution

import (
	"context"
	"fmt"

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
	switch decision.Action {
	case runlog.Refuse:
		return decision, nil, nil
	case runlog.Resume:
		return decision, nil, fmt.Errorf("run plan %s resumes from a checkpoint, but the progress of the steps is not recorded yet", plan.ID)
	}

	record := runlog.Record{
		RunKey:     runKey,
		PlanID:     string(plan.ID),
		Key:        append([]string(nil), keyValues...),
		Attempt:    decision.Attempt,
		FenceEpoch: decision.FenceEpoch,
		Status:     runlog.Running,
	}
	if err := journal.Append(record); err != nil {
		return decision, nil, err
	}

	var results []StepResult
	for _, operation := range plan.Operations {
		result := step(ctx, operation)
		results = append(results, result)
		if result.Status != "resolved" {
			record.Status = runlog.Failed
			if err := journal.Append(record); err != nil {
				return decision, results, err
			}
			return decision, results, nil
		}
	}
	record.Status = runlog.Completed
	if err := journal.Append(record); err != nil {
		return decision, results, err
	}
	return decision, results, nil
}

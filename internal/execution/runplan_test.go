package execution_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/execution"
	"tei/internal/runlog"
	"tei/internal/semantic"
)

type runJournal struct {
	records []runlog.Record
}

func (j *runJournal) Append(record runlog.Record) error {
	j.records = append(j.records, record)
	return nil
}

func (j *runJournal) Last(runKey string) (runlog.Record, bool, error) {
	for i := len(j.records) - 1; i >= 0; i-- {
		if j.records[i].RunKey == runKey {
			return j.records[i], true, nil
		}
	}
	return runlog.Record{}, false, nil
}

func closePlan() semantic.RunPlan {
	return semantic.RunPlan{
		ID:         "sales.monthly_close",
		Operations: []core.CanonicalID{"sales.decide", "sales.import", "sales.close"},
		Key:        []core.CanonicalID{"sales.period.month"},
		Resume:     "none",
	}
}

func TestRunPlanRunsTheOperationsInOrderAndCompletes(t *testing.T) {
	journal := &runJournal{}
	var order []string
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		order = append(order, string(operation))
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}

	decision, results, err := execution.RunPlan(context.Background(), closePlan(), []string{"2026-10"}, journal, step)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if decision.Action != runlog.Start {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	if len(results) != 3 || len(order) != 3 || order[0] != "sales.decide" {
		t.Fatalf("the operations must run in order: %+v", order)
	}
	if len(journal.records) != 2 || journal.records[1].Status != runlog.Completed {
		t.Fatalf("the run must be recorded running then completed: %+v", journal.records)
	}
}

func TestRunPlanStopsAtTheFirstFailureAndRecordsIt(t *testing.T) {
	journal := &runJournal{}
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		if operation == "sales.import" {
			return execution.StepResult{Operation: string(operation), Status: "unresolved", Detail: "missing input"}
		}
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}

	_, results, err := execution.RunPlan(context.Background(), closePlan(), []string{"2026-10"}, journal, step)
	if err != nil {
		t.Fatalf("a failed step is recorded, not returned: %v", err)
	}
	if len(results) != 2 || results[1].Status != "unresolved" {
		t.Fatalf("the plan must stop at the failure: %+v", results)
	}
	if len(journal.records) != 2 || journal.records[1].Status != runlog.Failed {
		t.Fatalf("the failed run must be recorded: %+v", journal.records)
	}
}

func TestRunPlanRefusesACompletedRun(t *testing.T) {
	journal := &runJournal{}
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	if _, _, err := execution.RunPlan(context.Background(), closePlan(), []string{"2026-10"}, journal, step); err != nil {
		t.Fatalf("first run: %v", err)
	}
	decision, results, err := execution.RunPlan(context.Background(), closePlan(), []string{"2026-10"}, journal, step)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if decision.Action != runlog.Refuse || len(results) != 0 {
		t.Fatalf("a completed run must be refused without running: %+v %+v", decision, results)
	}
}

func TestRunPlanRefusesToPretendItResumed(t *testing.T) {
	journal := &runJournal{}
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	plan := closePlan()
	plan.Resume = "checkpoint"
	// Record an unfinished attempt so the rules choose resume.
	journal.records = append(journal.records, runlog.Record{
		RunKey:     runlog.RunKey(string(plan.ID), []string{"2026-10"}),
		PlanID:     string(plan.ID),
		Attempt:    1,
		FenceEpoch: 1,
		Status:     runlog.Running,
	})
	if _, _, err := execution.RunPlan(context.Background(), plan, []string{"2026-10"}, journal, step); err == nil {
		t.Fatal("resuming without recorded progress must be reported, not faked")
	}
}

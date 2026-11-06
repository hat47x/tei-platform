package execution_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/effectlog"
	"tei/internal/execution"
	"tei/internal/operations"
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

	decision, results, err := execution.RunPlan(context.Background(), closePlan(), []string{"2026-10"}, journal, step, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if decision.Action != runlog.Start {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	if len(results) != 3 || len(order) != 3 || order[0] != "sales.decide" {
		t.Fatalf("the operations must run in order: %+v", order)
	}
	last := journal.records[len(journal.records)-1]
	if last.Status != runlog.Completed || len(last.Steps) != 3 {
		t.Fatalf("the run must end completed with every step recorded: %+v", last)
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

	_, results, err := execution.RunPlan(context.Background(), closePlan(), []string{"2026-10"}, journal, step, nil)
	if err != nil {
		t.Fatalf("a failed step is recorded, not returned: %v", err)
	}
	if len(results) != 2 || results[1].Status != "unresolved" {
		t.Fatalf("the plan must stop at the failure: %+v", results)
	}
	last := journal.records[len(journal.records)-1]
	if last.Status != runlog.Failed || len(last.Steps) != 1 || last.Steps[0] != "sales.decide" {
		t.Fatalf("the failed run must record the steps that finished: %+v", last)
	}
}

func TestRunPlanRefusesACompletedRun(t *testing.T) {
	journal := &runJournal{}
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	if _, _, err := execution.RunPlan(context.Background(), closePlan(), []string{"2026-10"}, journal, step, nil); err != nil {
		t.Fatalf("first run: %v", err)
	}
	decision, results, err := execution.RunPlan(context.Background(), closePlan(), []string{"2026-10"}, journal, step, nil)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if decision.Action != runlog.Refuse || len(results) != 0 {
		t.Fatalf("a completed run must be refused without running: %+v %+v", decision, results)
	}
}

func TestRunPlanResumesAfterTheRecordedSteps(t *testing.T) {
	journal := &runJournal{}
	plan := closePlan()
	plan.Resume = "checkpoint"
	key := []string{"2026-10"}
	journal.records = append(journal.records, runlog.Record{
		RunKey:     runlog.RunKey(string(plan.ID), key),
		PlanID:     string(plan.ID),
		Key:        key,
		Attempt:    1,
		FenceEpoch: 1,
		Status:     runlog.Running,
		Steps:      []string{"sales.decide"},
	})

	var ran []string
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		ran = append(ran, string(operation))
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	decision, results, err := execution.RunPlan(context.Background(), plan, key, journal, step, nil)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if decision.Action != runlog.Resume || decision.Attempt != 2 || decision.FenceEpoch != 2 {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	if len(ran) != 2 || ran[0] != "sales.import" {
		t.Fatalf("the finished step must not run again: %+v", ran)
	}
	if len(results) != 3 || results[0].Status != "resolved" {
		t.Fatalf("the earlier step must be reported as already finished: %+v", results)
	}
	last := journal.records[len(journal.records)-1]
	if last.Status != runlog.Completed || len(last.Steps) != 3 {
		t.Fatalf("the resumed run must complete with every step: %+v", last)
	}
}
func unattendedPlan() semantic.RunPlan {
	plan := closePlan()
	plan.Autonomy = "unattended"
	return plan
}

func TestUnattendedRunStopsWhenNoPreconditionCheckIsConfigured(t *testing.T) {
	journal := &runJournal{}
	called := false
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		called = true
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	_, results, err := execution.RunPlan(context.Background(), unattendedPlan(), []string{"2026-10"}, journal, step, nil)
	if err != nil {
		t.Fatalf("a blocked run is not an error: %v", err)
	}
	if called {
		t.Fatal("an unattended run without a precondition check must not execute")
	}
	if len(results) != 1 || results[0].Status != "blocked" {
		t.Fatalf("the blocked step must be reported: %+v", results)
	}
	last := journal.records[len(journal.records)-1]
	if last.Status != runlog.Blocked {
		t.Fatalf("the run must be recorded blocked: %+v", last)
	}
}

func TestUnattendedRunStopsWhenAPreconditionIsNotMet(t *testing.T) {
	journal := &runJournal{}
	called := false
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		called = true
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	preconditions := func(operation core.CanonicalID) string {
		if operation == "sales.import" {
			return "the period is closed"
		}
		return ""
	}
	_, results, err := execution.RunPlan(context.Background(), unattendedPlan(), []string{"2026-10"}, journal, step, preconditions)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !called {
		t.Fatal("the first step may run before the blocked one")
	}
	if len(results) != 2 || results[1].Status != "blocked" || results[1].Detail != "the period is closed" {
		t.Fatalf("the blocked step must name the reason: %+v", results)
	}
}

func TestUnattendedRunExecutesWhenEveryPreconditionIsMet(t *testing.T) {
	journal := &runJournal{}
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	preconditions := func(core.CanonicalID) string { return "" }
	_, results, err := execution.RunPlan(context.Background(), unattendedPlan(), []string{"2026-10"}, journal, step, preconditions)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("every step must run: %+v", results)
	}
	if journal.records[len(journal.records)-1].Status != runlog.Completed {
		t.Fatalf("the run must complete: %+v", journal.records)
	}
}

func TestAttendedRunKeepsRunningWithoutAPreconditionCheck(t *testing.T) {
	journal := &runJournal{}
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	_, results, err := execution.RunPlan(context.Background(), closePlan(), []string{"2026-10"}, journal, step, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("an attended run is not blocked by the absence of a check: %+v", results)
	}
}
func periodSurface() operations.OperationSurface {
	return operations.OperationSurface{Operations: []operations.Operation{{
		ID:            "app.create.sales.close",
		Kind:          "create",
		Target:        "sales.period",
		Binding:       "sales.close",
		Period:        "sales.month",
		PeriodRejects: []string{"create"},
	}}}
}

func TestPeriodPreconditionsStopAnUnattendedRun(t *testing.T) {
	journal := &runJournal{}
	called := false
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		called = true
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	preconditions := execution.PeriodPreconditions(periodSurface(), map[string]operations.PeriodState{"sales.month": operations.PeriodClosed})

	plan := semantic.RunPlan{
		ID:         "sales.monthly_close",
		Operations: []core.CanonicalID{"sales.close"},
		Resume:     "none",
		Autonomy:   "unattended",
	}
	_, results, err := execution.RunPlan(context.Background(), plan, []string{"2026-10"}, journal, step, preconditions)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if called {
		t.Fatal("a capability refused by its period must not run unattended")
	}
	if len(results) != 1 || results[0].Status != "blocked" {
		t.Fatalf("the refusal must block the run: %+v", results)
	}
	if journal.records[len(journal.records)-1].Status != runlog.Blocked {
		t.Fatalf("the run must be recorded blocked: %+v", journal.records)
	}
}

func TestPeriodPreconditionsLetAnOpenPeriodRun(t *testing.T) {
	journal := &runJournal{}
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	preconditions := execution.PeriodPreconditions(periodSurface(), map[string]operations.PeriodState{"sales.month": operations.PeriodOpen})
	plan := semantic.RunPlan{ID: "sales.monthly_close", Operations: []core.CanonicalID{"sales.close"}, Autonomy: "unattended"}
	if _, results, err := execution.RunPlan(context.Background(), plan, []string{"2026-10"}, journal, step, preconditions); err != nil || len(results) != 1 || results[0].Status != "resolved" {
		t.Fatalf("an open period must run: results=%+v err=%v", results, err)
	}
}

func TestPeriodPreconditionsIgnoreCapabilitiesWithoutAPeriod(t *testing.T) {
	preconditions := execution.PeriodPreconditions(periodSurface(), map[string]operations.PeriodState{"sales.month": operations.PeriodClosed})
	if reason := preconditions(core.MustCanonicalID("sales.unbound")); reason != "" {
		t.Fatalf("a capability without a period is not guarded: %q", reason)
	}
}

type countingObserver struct{ calls int }

func (o *countingObserver) ObserveEffect(context.Context, effectlog.Record) (bool, string, error) {
	o.calls++
	return true, "", nil
}

func lateEffectRules() []semantic.EffectRule {
	return []semantic.EffectRule{
		{ID: "sales.order.stock", OnCapability: "sales.close", Stage: "after-commit", EffectKind: "update", EffectData: "sales.period", ViaCapability: "sales.close"},
		{ID: "sales.order.audit", OnCapability: "sales.decide", Stage: "in-transaction", EffectKind: "create", EffectData: "sales.period"},
	}
}

func TestEffectPreconditionsStopAnUnobservableEffect(t *testing.T) {
	preconditions := execution.EffectPreconditions(lateEffectRules(), nil)
	if reason := preconditions(core.MustCanonicalID("sales.close")); reason == "" {
		t.Fatal("an effect outside the transaction without an observer must block an unattended run")
	}
	if reason := preconditions(core.MustCanonicalID("sales.decide")); reason != "" {
		t.Fatalf("an effect inside the transaction needs no observer: %q", reason)
	}
	if reason := preconditions(core.MustCanonicalID("sales.unbound")); reason != "" {
		t.Fatalf("a capability without effects is not blocked: %q", reason)
	}
}

func TestEffectPreconditionsAllowAnObservableEffect(t *testing.T) {
	preconditions := execution.EffectPreconditions(lateEffectRules(), &countingObserver{})
	if reason := preconditions(core.MustCanonicalID("sales.close")); reason != "" {
		t.Fatalf("an observable effect must not block: %q", reason)
	}
}

func TestEffectPreconditionsBlockAnUnattendedRun(t *testing.T) {
	journal := &runJournal{}
	called := false
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		called = true
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	plan := semantic.RunPlan{ID: "sales.monthly_close", Operations: []core.CanonicalID{"sales.close"}, Autonomy: "unattended"}
	_, results, err := execution.RunPlan(context.Background(), plan, []string{"2026-10"}, journal, step, execution.EffectPreconditions(lateEffectRules(), nil))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if called {
		t.Fatal("an unobservable effect must stop the run before it executes")
	}
	if len(results) != 1 || results[0].Status != "blocked" {
		t.Fatalf("the effect must block the run: %+v", results)
	}
}
func TestCombinePreconditionsStopsOnTheFirstUnmetCheck(t *testing.T) {
	period := execution.PeriodPreconditions(periodSurface(), map[string]operations.PeriodState{"sales.month": operations.PeriodOpen})
	effects := execution.EffectPreconditions(lateEffectRules(), nil)

	combined := execution.CombinePreconditions(period, effects)
	if reason := combined(core.MustCanonicalID("sales.close")); reason == "" {
		t.Fatal("the unobservable effect must stop the run even when the period is open")
	}

	withObserver := execution.CombinePreconditions(period, execution.EffectPreconditions(lateEffectRules(), &countingObserver{}))
	if reason := withObserver(core.MustCanonicalID("sales.close")); reason != "" {
		t.Fatalf("both checks pass, so the run may continue: %q", reason)
	}
}

func TestCombinePreconditionsWithoutChecksIsNil(t *testing.T) {
	if execution.CombinePreconditions(nil, nil) != nil {
		t.Fatal("no checks means nothing was verified, so the result must stay nil")
	}
}

func TestCombinedChecksBlockAnUnattendedRun(t *testing.T) {
	journal := &runJournal{}
	called := false
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		called = true
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	plan := semantic.RunPlan{ID: "sales.monthly_close", Operations: []core.CanonicalID{"sales.close"}, Autonomy: "unattended"}
	preconditions := execution.CombinePreconditions(
		execution.PeriodPreconditions(periodSurface(), map[string]operations.PeriodState{"sales.month": operations.PeriodOpen}),
		execution.EffectPreconditions(lateEffectRules(), nil),
	)
	_, results, err := execution.RunPlan(context.Background(), plan, []string{"2026-10"}, journal, step, preconditions)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if called || len(results) != 1 || results[0].Status != "blocked" {
		t.Fatalf("the combined check must block the run: called=%v results=%+v", called, results)
	}
}

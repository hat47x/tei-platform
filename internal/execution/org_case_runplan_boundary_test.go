package execution_test

import (
	"context"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/effectlog"
	"tei/internal/execution"
	"tei/internal/runlog"
	"tei/internal/semantic"
)

// This is a direct characterization of the production RunPlan and
// PreconditionFunc. No DB, real CaseRecord, current Host Grant, or actual
// Provider effect is present. In particular, a StepFunc returning "resolved"
// is NOT an effect receipt, even if RunPlan records "completed".
type orgCaseJournal struct{ records []runlog.Record }

func (j *orgCaseJournal) Append(record runlog.Record) error {
	record.Steps = append([]string(nil), record.Steps...)
	j.records = append(j.records, record)
	return nil
}
func (j *orgCaseJournal) Last(key string) (runlog.Record, bool, error) {
	for i := len(j.records) - 1; i >= 0; i-- {
		if j.records[i].RunKey == key {
			return j.records[i], true, nil
		}
	}
	return runlog.Record{}, false, nil
}
func (j *orgCaseJournal) lastStatus(t *testing.T) runlog.Status {
	t.Helper()
	if len(j.records) == 0 {
		t.Fatal("no journal record")
	}
	return j.records[len(j.records)-1].Status
}
func orgCasePlan(operations ...core.CanonicalID) semantic.RunPlan {
	return semantic.RunPlan{ID: "organization.case_transfer", Operations: operations, Autonomy: "unattended", Resume: "none"}
}
func orgCaseResolvedStep(calls *[]string) execution.StepFunc {
	return func(_ context.Context, op core.CanonicalID) execution.StepResult {
		*calls = append(*calls, string(op))
		return execution.StepResult{Operation: string(op), Status: "resolved"}
	}
}

// OC-RP-01: the product runner cannot infer missing *required* checks
// from one supplied check. The first execution below is a deliberate
// unsafe-caller negative, not an authorized transfer. The second shows
// the stop behavior when the owner/Host check is explicitly supplied.
func TestOrgCaseRunPlanDoesNotInferMissingHostGrantCheck(t *testing.T) {
	plan := orgCasePlan("organization.case.transfer")
	journal := &orgCaseJournal{}
	var calls []string
	periodOnly := execution.CombinePreconditions(func(core.CanonicalID) string { return "" }, nil)
	_, results, err := execution.RunPlan(context.Background(), plan, []string{"CASE-C1"}, journal, orgCaseResolvedStep(&calls), periodOnly)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || len(results) != 1 || results[0].Status != "resolved" ||
		journal.lastStatus(t) != runlog.Completed {
		t.Fatalf("partial coverage still ran a toy Step; calls=%v results=%+v journal=%+v", calls, results, journal.records)
	}
	// This completion does not imply that the missing Host check was done.
	// A different run key and a real caller-supplied required check now stop
	// before the step could run.
	denyHost := func(core.CanonicalID) string { return "no current Host Grant for Case C1" }
	guards := execution.CombinePreconditions(periodOnly, denyHost)
	blockedJournal := &orgCaseJournal{}
	var blockedCalls []string
	_, blocked, err := execution.RunPlan(context.Background(), plan, []string{"CASE-C2"}, blockedJournal, orgCaseResolvedStep(&blockedCalls), guards)
	if err != nil {
		t.Fatal(err)
	}
	if len(blockedCalls) != 0 || len(blocked) != 1 || blocked[0].Status != "blocked" ||
		!strings.Contains(blocked[0].Detail, "Host Grant") || blockedJournal.lastStatus(t) != runlog.Blocked {
		t.Fatalf("explicit Host check should block toy step: calls=%v results=%+v", blockedCalls, blocked)
	}
}

type orgCaseObserver struct{ calls int }

func (o *orgCaseObserver) ObserveEffect(context.Context, effectlog.Record) (bool, string, error) {
	o.calls++
	return false, "not observed", nil
}

// OC-RP-02: EffectPreconditions verifies that an observer was supplied.
// It does not invoke ObserveEffect, and a RunPlan "completed" status is not
// an observed after-commit effect. RunEffect/actual Provider requires its own
// evidence and is intentionally not invoked in this characterization.
func TestOrgCaseRunPlanObserverPresenceDoesNotProveAfterCommitEffect(t *testing.T) {
	operation := core.MustCanonicalID("organization.case.transfer")
	rule := semantic.EffectRule{
		ID:            "organization.case_transfer_notification",
		OnCapability:  operation,
		Stage:         "after-commit",
		EffectKind:    "update",
		EffectData:    "organization.case_record",
		ViaCapability: operation,
	}
	observer := &orgCaseObserver{}
	guards := execution.CombinePreconditions(execution.EffectPreconditions([]semantic.EffectRule{rule}, observer))
	var calls []string
	journal := &orgCaseJournal{}
	_, results, err := execution.RunPlan(context.Background(), orgCasePlan(operation), []string{"CASE-C3"}, journal, orgCaseResolvedStep(&calls), guards)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || len(results) != 1 || results[0].Status != "resolved" ||
		observer.calls != 0 || journal.lastStatus(t) != runlog.Completed {
		t.Fatalf("presence-only observer contract: calls=%v observed=%d results=%+v", calls, observer.calls, results)
	}
	// The journal has no external receipt or independent read.
}

// OC-RP-03: a later blocked step does not roll back a previously
// "resolved" synthetic step. This is not a test of real DB commits or rollback.
func TestOrgCaseRunPlanBlockedLaterStepRetainsEarlierToySideEffect(t *testing.T) {
	first := core.MustCanonicalID("organization.case.transfer")
	second := core.MustCanonicalID("organization.case.notify")
	plan := orgCasePlan(first, second)
	guards := execution.CombinePreconditions(func(op core.CanonicalID) string {
		if op == second {
			return "policy/source revision no longer qualified"
		}
		return ""
	})
	var toyEffects int
	step := func(_ context.Context, op core.CanonicalID) execution.StepResult {
		toyEffects++
		return execution.StepResult{Operation: string(op), Status: "resolved"}
	}
	journal := &orgCaseJournal{}
	_, results, err := execution.RunPlan(context.Background(), plan, []string{"CASE-C4"}, journal, step, guards)
	if err != nil {
		t.Fatal(err)
	}
	if toyEffects != 1 || len(results) != 2 || results[0].Status != "resolved" ||
		results[1].Status != "blocked" || journal.lastStatus(t) != runlog.Blocked {
		t.Fatalf("a blocked later step cannot erase an earlier result: effects=%d results=%+v", toyEffects, results)
	}
	last := journal.records[len(journal.records)-1]
	if len(last.Steps) != 1 || last.Steps[0] != string(first) {
		t.Fatalf("earlier resolved step must remain in journal: %+v", last)
	}
}

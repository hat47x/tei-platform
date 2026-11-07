package execution_test

import (
	"context"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/execution"
	"tei/internal/runlog"
)

// takingOverJournal lets a test take the run away from the attempt that is
// running, the way a resumed attempt does when it bumps the fence epoch.
type takingOverJournal struct {
	runJournal
	takeOver func(runKey string)
}

func (j *takingOverJournal) Append(record runlog.Record) error {
	takeOver := j.takeOver
	j.takeOver = nil
	if err := j.runJournal.Append(record); err != nil {
		return err
	}
	// The takeover happens after this attempt's record, the way a resume from
	// another process appends its own record on top of the stale one.
	if takeOver != nil {
		takeOver(record.RunKey)
	}
	return nil
}

// TestRunPlanStopsWhenAnotherAttemptTakesTheRunOver pins the fence: after a
// resume bumps the attempt and the fence epoch, the older attempt must not keep
// writing. Without this check a superseded attempt's late "completed" would
// overwrite the newer attempt's state, and the newer attempt's work would be
// reported as unfinished even though it finished.
func TestRunPlanStopsWhenAnotherAttemptTakesTheRunOver(t *testing.T) {
	journal := &takingOverJournal{}
	plan := closePlan()
	runKey := runlog.RunKey(string(plan.ID), []string{"2026-10"})
	journal.takeOver = func(key string) {
		// A resume from another process: the second attempt now owns the run.
		_ = journal.runJournal.Append(runlog.Record{
			RunKey:     key,
			PlanID:     string(plan.ID),
			Attempt:    2,
			FenceEpoch: 2,
			Status:     runlog.Running,
		})
	}

	var ran []string
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		ran = append(ran, string(operation))
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	_, _, err := execution.RunPlan(context.Background(), plan, []string{"2026-10"}, journal, step, nil)
	if err == nil {
		t.Fatal("an attempt that lost the run must not finish quietly")
	}
	if !strings.Contains(err.Error(), "no longer owns run") {
		t.Fatalf("the error must say the attempt lost the run: %v", err)
	}
	last, found, lastErr := journal.runJournal.Last(runKey)
	if lastErr != nil || !found {
		t.Fatalf("the journal must still hold the newer attempt: %+v err=%v", last, lastErr)
	}
	if last.Attempt != 2 || last.Status != runlog.Running {
		t.Fatalf("the older attempt must not overwrite the newer one: %+v", last)
	}
}

// TestRunPlanKeepsWritingWhileItOwnsTheRun is the other half: the check must not
// refuse an attempt that still owns the run, or every normal run would fail.
func TestRunPlanKeepsWritingWhileItOwnsTheRun(t *testing.T) {
	journal := &runJournal{}
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	_, results, err := execution.RunPlan(context.Background(), closePlan(), []string{"2026-10"}, journal, step, nil)
	if err != nil {
		t.Fatalf("an owning attempt must be able to write: %v", err)
	}
	if len(results) != 3 || journal.records[len(journal.records)-1].Status != runlog.Completed {
		t.Fatalf("the run must complete: %+v", journal.records)
	}
}

// TestRunPlanResumeTakesOverFromAStaleAttempt pins that the resume path still
// works: the second attempt owns the run and completes it.
func TestRunPlanResumeTakesOverFromAStaleAttempt(t *testing.T) {
	journal := &runJournal{}
	plan := closePlan()
	plan.Resume = "checkpoint"
	key := []string{"2026-10"}
	runKey := runlog.RunKey(string(plan.ID), key)
	if err := journal.Append(runlog.Record{
		RunKey: runKey, PlanID: string(plan.ID), Key: key,
		Attempt: 1, FenceEpoch: 1, Status: runlog.Running,
	}); err != nil {
		t.Fatal(err)
	}
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	decision, _, err := execution.RunPlan(context.Background(), plan, key, journal, step, nil)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if decision.Action != runlog.Resume || decision.Attempt != 2 {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	last, _, err := journal.Last(runKey)
	if err != nil || last.Attempt != 2 || last.Status != runlog.Completed {
		t.Fatalf("the resumed attempt must own and complete the run: %+v err=%v", last, err)
	}
}

// TestRunPlanStopsWhenTheJournalCannotBeRead keeps the failure mode honest: a
// journal that cannot be read is an error, not a lost fence and not a decision.
func TestRunPlanStopsWhenTheJournalCannotBeRead(t *testing.T) {
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	if _, _, err := execution.RunPlan(context.Background(), closePlan(), []string{"2026-10"}, unreadableJournal{}, step, nil); err == nil {
		t.Fatal("an unreadable journal must be reported")
	}
}

// unreadableJournal cannot be read, so nothing about the run is known.
type unreadableJournal struct{}

func (unreadableJournal) Append(runlog.Record) error { return nil }

func (unreadableJournal) Last(string) (runlog.Record, bool, error) {
	return runlog.Record{}, false, errUnreadableJournal{}
}

type errUnreadableJournal struct{}

func (errUnreadableJournal) Error() string { return "journal is unreadable" }

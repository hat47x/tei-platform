package main

import (
	"strings"
	"testing"
)

func TestIndependentFakeExecutionsMayProduceIdenticalResults(t *testing.T) {
	plan := referenceBuildPlan()
	builder := &fakeSuccessfulBuilder{}
	first, err := builder.Run(plan, 1, []byte("identical-artifact"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := builder.Run(plan, 2, []byte("identical-artifact"))
	if err != nil {
		t.Fatal(err)
	}
	if builder.runs != 2 {
		t.Fatalf("fake builder was invoked %d times, want 2", builder.runs)
	}
	if first.ResultFingerprint != second.ResultFingerprint {
		t.Fatal("identical output bytes unexpectedly changed result identity")
	}
	if first.ExecutionOccurrenceID == second.ExecutionOccurrenceID {
		t.Fatal("two actual fake-builder calls used the same occurrence identity")
	}
	if first.BuildPlanFingerprint != second.BuildPlanFingerprint {
		t.Fatal("execution occurrence changed exact BuildPlan identity")
	}
	history, err := newAttemptHistory(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := history.AppendSuccess(plan, first); err != nil {
		t.Fatal(err)
	}
	if err := history.AppendSuccess(plan, second); err != nil {
		t.Fatal(err)
	}
	if err := history.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(history.SuccessfulAttempts) != 2 {
		t.Fatal("two executions collapsed into one result")
	}
}

func TestCopiedResultOrOccurrenceIsNotAnotherExecution(t *testing.T) {
	plan := referenceBuildPlan()
	builder := &fakeSuccessfulBuilder{}
	first, err := builder.Run(plan, 1, []byte("artifact"))
	if err != nil {
		t.Fatal(err)
	}
	history, err := newAttemptHistory(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := history.AppendSuccess(plan, first); err != nil {
		t.Fatal(err)
	}
	copied := first
	copied.AttemptOrdinal = 2
	if err := history.AppendSuccess(plan, copied); err == nil {
		t.Fatal("replaying one result/occurrence was counted as a second execution")
	}
	history.SuccessfulAttempts = append(history.SuccessfulAttempts, copied)
	if err := history.Validate(); err == nil {
		t.Fatal("history accepted a copied execution occurrence")
	}
	if builder.runs != 1 {
		t.Fatal("test created a second fake execution unexpectedly")
	}
}

func TestSuccessfulAttemptCannotReuseFailureOrdinal(t *testing.T) {
	plan := referenceBuildPlan()
	exitCode := 2
	failure, _, err := newFailedAttempt(plan, 1, fakeFailure{
		Phase: attemptPhaseCompile, FailureClass: "compile-error",
		ExitCode: &exitCode, RawDiagnosticHint: "failed",
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	history, err := newAttemptHistory(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := history.AppendFailure(plan, failure); err != nil {
		t.Fatal(err)
	}
	builder := &fakeSuccessfulBuilder{}
	success, err := builder.Run(plan, 1, []byte("artifact"))
	if err != nil {
		t.Fatal(err)
	}
	if err := history.AppendSuccess(plan, success); err == nil {
		t.Fatal("same ordinal counted for both failed and successful outcomes")
	}
	history.SuccessfulAttempts = append(history.SuccessfulAttempts, success)
	if err := history.Validate(); err == nil {
		t.Fatal("history accepted a reused execution ordinal")
	}
}

func TestFailureAfterSuccessRequiresNewOrdinal(t *testing.T) {
	plan := referenceBuildPlan()
	history, err := newAttemptHistory(plan)
	if err != nil {
		t.Fatal(err)
	}
	builder := &fakeSuccessfulBuilder{}
	success, err := builder.Run(plan, 2, []byte("artifact"))
	if err != nil {
		t.Fatal(err)
	}
	if err := history.AppendSuccess(plan, success); err != nil {
		t.Fatal(err)
	}
	exitCode := 2
	failure, _, err := newFailedAttempt(plan, 1, fakeFailure{
		Phase: attemptPhaseLink, FailureClass: "link-error",
		ExitCode: &exitCode, RawDiagnosticHint: "later failure",
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := history.AppendFailure(plan, failure); err == nil {
		t.Fatal("history accepted an out-of-order failure after a success")
	}
}

func TestSuccessfulAttemptMustHaveOccurrenceAndOutcome(t *testing.T) {
	plan := referenceBuildPlan()
	builder := &fakeSuccessfulBuilder{}
	attempt, err := builder.Run(plan, 1, []byte("artifact"))
	if err != nil {
		t.Fatal(err)
	}
	withoutOccurrence := attempt
	withoutOccurrence.ExecutionOccurrenceID = ""
	if err := withoutOccurrence.ValidateAgainst(plan); err == nil {
		t.Fatal("content fingerprint alone was accepted as an execution")
	}
	unknownOutcome := attempt
	unknownOutcome.Outcome = "cancelled"
	if err := unknownOutcome.ValidateAgainst(plan); err == nil {
		t.Fatal("cancelled attempt was accepted as successful")
	}
	changedResult := attempt
	changedResult.ResultFingerprint = strings.Repeat("?", 64)
	if err := changedResult.ValidateAgainst(plan); err == nil {
		t.Fatal("invalid artifact fingerprint was accepted")
	}
}

func TestFailedHistoryRejectsUnfinalizedEvidence(t *testing.T) {
	plan := referenceBuildPlan()
	exitCode := 2
	failure, _, err := newFailedAttempt(plan, 1, fakeFailure{
		Phase: attemptPhaseCompile, FailureClass: "compile-error",
		ExitCode: &exitCode, RawDiagnosticHint: "failed",
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	history, err := newAttemptHistory(plan)
	if err != nil {
		t.Fatal(err)
	}
	failure.EvidenceFingerprint = ""
	if err := history.AppendFailure(plan, failure); err == nil {
		t.Fatal("unfinalized failure was inserted into attempt history")
	}
}

package execution_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/effectlog"
	"tei/internal/execution"
	"tei/internal/failure"
	"tei/internal/semantic"
)

// capabilityInvoker answers per capability, so one compensation can succeed
// while another fails. A single error field cannot express "this one worked,
// that one did not", which is exactly the state under test.
type capabilityInvoker struct {
	failFor map[string]error
	called  []string
}

func (i *capabilityInvoker) InvokeCapability(_ context.Context, _ core.CanonicalID, capability core.CanonicalID, _ driver.BindingAdapterResolver, _ any) (driver.CapabilityInvocation, error) {
	i.called = append(i.called, string(capability))
	if err, ok := i.failFor[string(capability)]; ok {
		return driver.CapabilityInvocation{}, err
	}
	return driver.CapabilityInvocation{}, nil
}

// twoCompensations names two compensations for the same capability and the same
// data, so CompensationsFor returns both and the run has something to split on.
func twoCompensations() []semantic.EffectRule {
	return []semantic.EffectRule{
		{ID: "sales.order.undo", OnCapability: "sales.order.submit", Stage: "compensation", EffectKind: "update", EffectData: "sales.stock", ViaCapability: "sales.stock.restore", OnFailure: "compensate"},
		{ID: "sales.order.price", OnCapability: "sales.order.submit", Stage: "compensation", EffectKind: "update", EffectData: "sales.stock", ViaCapability: "sales.price.restore", OnFailure: "compensate"},
	}
}

func failedRule() semantic.EffectRule {
	rule := lateRule()
	rule.OnFailure = "compensate"
	return rule
}

// A compensation run that applied everything is not a failure at all.
func TestCompensateAllAppliedIsNotAFailure(t *testing.T) {
	records, err := execution.Compensate(context.Background(), &capabilityInvoker{}, nil, &memoryJournal{}, "app", twoCompensations(), failedRule(), nil, nil)
	if err != nil {
		t.Fatalf("every compensation applied, so there is no failure: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("both compensations must run: %+v", records)
	}
	for _, record := range records {
		if !record.Succeeded {
			t.Fatalf("record reported as not applied: %+v", record)
		}
	}
}

// A compensation that could not be performed is a record, not an error - that
// contract already existed. With a single compensation there is nothing to split,
// so the run stays nil and the caller reads the record.
func TestCompensateNoneAppliedIsNotPartial(t *testing.T) {
	invoker := &capabilityInvoker{failFor: map[string]error{
		"sales.stock.restore": errors.New("undo failed"),
		"sales.price.restore": errors.New("undo failed"),
	}}
	records, err := execution.Compensate(context.Background(), invoker, nil, &memoryJournal{}, "app", twoCompensations(), failedRule(), nil, nil)
	if err != nil {
		t.Fatalf("a compensation that could not be performed is recorded, not raised: %v", err)
	}
	if failure.IsPartialSuccess(err) {
		t.Fatalf("nothing applied, so this is not a partial success: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("all compensations are attempted even when they fail: %+v", records)
	}
	for _, record := range records {
		if record.Succeeded {
			t.Fatalf("a failed compensation was reported as applied: %+v", record)
		}
	}
}

// A compensation run where some applied and some did not is a partial success,
// and the error must say so: a caller told nothing would either resend
// everything - duplicating what landed - or discard everything - losing it.
func TestCompensatePartiallyAppliedIsPartialSuccess(t *testing.T) {
	invoker := &capabilityInvoker{failFor: map[string]error{
		"sales.price.restore": errors.New("undo failed"),
	}}
	journal := &memoryJournal{}
	_, err := execution.Compensate(context.Background(), invoker, nil, journal, "app", twoCompensations(), failedRule(), nil, nil)
	if err == nil {
		t.Fatal("a run where one compensation failed must report a failure")
	}
	if !failure.IsPartialSuccess(err) {
		t.Fatalf("some compensations applied, so this is a partial success: %v", err)
	}

	unified, ok := failure.AsFailure(err)
	if !ok {
		t.Fatalf("the failure must satisfy the unified contract: %T", err)
	}
	if unified.Kind() != failure.PartialSuccess {
		t.Errorf("Kind() = %v, want %v", unified.Kind(), failure.PartialSuccess)
	}
	// The caller must find out what was applied before doing anything, so the
	// retry is a lookup rather than a blind resend.
	if unified.Retryable() != failure.LookupFirst {
		t.Errorf("Retryable() = %v, want %v", unified.Retryable(), failure.LookupFirst)
	}
	if !unified.HasEffectUnknown() {
		t.Error("HasEffectUnknown() = false for a partial success")
	}
	if unified.FailureCode() != "COMPENSATION_PARTIAL" {
		t.Errorf("FailureCode() = %q, want COMPENSATION_PARTIAL", unified.FailureCode())
	}
	// The message must say how much landed, because "partial" alone is not
	// actionable.
	if got := unified.FailureMessage(); got != "1 of 2 compensations were applied" {
		t.Errorf("FailureMessage() = %q", got)
	}
	// The compensations' own failures stay reachable through the records, since
	// RunEffect reports them there rather than as an error.
	if unified.FailureCause() != nil {
		t.Errorf("FailureCause() = %v, want nil: the compensation failures are records, not a wrapped error", unified.FailureCause())
	}

	// Both attempts are journaled, so a reader can tell which one landed
	// without re-deriving it from the error.
	if len(journal.records) != 2 {
		t.Fatalf("both attempts must be journaled: %+v", journal.records)
	}
	applied := 0
	for _, record := range journal.records {
		if record.Succeeded {
			applied++
		}
	}
	if applied != 1 {
		t.Fatalf("journal shows %d applied, want 1", applied)
	}
}

// partialJournal fails only for one rule, so one compensation lands in the
// journal and the other's record cannot be written.
type partialJournal struct {
	records []effectlog.Record
	failFor string
}

func (j *partialJournal) Append(record effectlog.Record) error {
	if record.RuleID == j.failFor {
		return errJournalUnavailable
	}
	j.records = append(j.records, record)
	return nil
}

func (j *partialJournal) Records() ([]effectlog.Record, error) { return j.records, nil }

// A journal failure next to a compensation that did land is a partial success:
// the record is incomplete, but something was applied, and the caller must not
// be told the whole run is lost.
func TestCompensateJournalErrorBesideAnAppliedCompensationIsPartial(t *testing.T) {
	journal := &partialJournal{failFor: "sales.order.price"}
	invoker := &capabilityInvoker{failFor: map[string]error{
		"sales.stock.restore": errors.New("undo failed"),
	}}
	records, err := execution.Compensate(context.Background(), invoker, nil, journal, "app", twoCompensations(), failedRule(), nil, nil)
	if err == nil {
		t.Fatal("a journal failure must be reported")
	}
	if !failure.IsPartialSuccess(err) {
		t.Fatalf("a compensation applied before the journal failed: %v", err)
	}
	// The journal error is the cause, so it stays reachable.
	if !errors.Is(err, errJournalUnavailable) {
		t.Error("the journal error is no longer reachable")
	}
	// The records still say what was attempted, so the caller is not left with
	// only the error.
	if len(records) != 2 {
		t.Fatalf("both attempts must be reported: %+v", records)
	}
	applied := 0
	for _, record := range records {
		if record.Succeeded {
			applied++
		}
	}
	if applied != 1 {
		t.Fatalf("records show %d applied, want 1", applied)
	}
}

var errJournalUnavailable = errors.New("journal unavailable")

// PartialError must satisfy the contract on its own, not only through Compensate.
func TestPartialErrorSatisfiesTheUnifiedContract(t *testing.T) {
	cause := errors.New("second half failed")
	err := &failure.PartialError{Code: "BATCH_PARTIAL", Message: "1 of 3 items were written", Cause: cause}

	if err.Error() == "" {
		t.Fatal("Error() is empty")
	}
	if err.FailureCode() != "BATCH_PARTIAL" {
		t.Errorf("FailureCode() = %q", err.FailureCode())
	}
	if err.FailureMessage() != "1 of 3 items were written" {
		t.Errorf("FailureMessage() = %q", err.FailureMessage())
	}
	if err.FailureCause() != cause {
		t.Error("FailureCause() does not return the cause")
	}
	if err.Kind() != failure.PartialSuccess {
		t.Errorf("Kind() = %v, want %v", err.Kind(), failure.PartialSuccess)
	}
	if err.Retryable() != failure.LookupFirst {
		t.Errorf("Retryable() = %v, want %v", err.Retryable(), failure.LookupFirst)
	}
	if !err.HasEffectUnknown() {
		t.Error("HasEffectUnknown() = false")
	}
	if !errors.Is(err, cause) {
		t.Error("the cause is no longer reachable through errors.Is")
	}
	// A nil cause must not panic and must not print "()".
	bare := &failure.PartialError{Code: "BATCH_PARTIAL", Message: "1 of 3 items were written"}
	if got := bare.Error(); got != "BATCH_PARTIAL: 1 of 3 items were written" {
		t.Errorf("Error() = %q", got)
	}
	if bare.FailureCause() != nil {
		t.Error("FailureCause() is not nil")
	}
}

var _ = effectlog.Record{}

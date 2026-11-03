package execution_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/effectlog"
	"tei/internal/execution"
	"tei/internal/semantic"
)

func TestEffectsOutsideTransactionSelectsLateStages(t *testing.T) {
	rules := []semantic.EffectRule{
		{ID: "s.order.audit", OnCapability: "s.order.submit", Stage: "in-transaction", EffectKind: "create", EffectData: "s.order"},
		{ID: "s.order.stock", OnCapability: "s.order.submit", Stage: "after-commit", EffectKind: "update", EffectData: "s.stock", ViaCapability: "s.stock.decrement"},
		{ID: "s.order.undo", OnCapability: "s.order.submit", Stage: "compensation", EffectKind: "update", EffectData: "s.stock", ViaCapability: "s.stock.restore"},
		{ID: "s.other.stock", OnCapability: "s.other.submit", Stage: "after-commit", EffectKind: "update", EffectData: "s.stock", ViaCapability: "s.stock.decrement"},
	}

	requirements := execution.EffectsOutsideTransaction(rules, core.MustCanonicalID("s.order.submit"))
	if len(requirements) != 2 {
		t.Fatalf("only the late stages of the bound capability need observation: %+v", requirements)
	}
	if requirements[0].RuleID != "s.order.stock" || requirements[0].Stage != "after-commit" {
		t.Fatalf("unexpected first requirement: %+v", requirements[0])
	}
	if requirements[1].RuleID != "s.order.undo" || requirements[1].Stage != "compensation" {
		t.Fatalf("unexpected second requirement: %+v", requirements[1])
	}
}

type effectInvoker struct {
	called bool
	err    error
}

func (i *effectInvoker) InvokeCapability(context.Context, core.CanonicalID, core.CanonicalID, driver.BindingAdapterResolver, any) (driver.CapabilityInvocation, error) {
	i.called = true
	return driver.CapabilityInvocation{}, i.err
}

type effectObserver struct {
	observed bool
	detail   string
	err      error
}

func (o effectObserver) ObserveEffect(context.Context, effectlog.Record) (bool, string, error) {
	return o.observed, o.detail, o.err
}

type memoryJournal struct {
	records []effectlog.Record
	err     error
}

func (j *memoryJournal) Append(record effectlog.Record) error {
	if j.err != nil {
		return j.err
	}
	j.records = append(j.records, record)
	return nil
}

func (j *memoryJournal) Records() ([]effectlog.Record, error) { return j.records, nil }

func lateRule() semantic.EffectRule {
	return semantic.EffectRule{
		ID:            "sales.order.stock",
		OnCapability:  "sales.order.submit",
		Stage:         "after-commit",
		EffectKind:    "update",
		EffectData:    "sales.stock",
		ViaCapability: "sales.stock.decrement",
	}
}

func TestRunEffectRecordsTheAttemptAndTheObservation(t *testing.T) {
	invoker := &effectInvoker{}
	journal := &memoryJournal{}
	record, err := execution.RunEffect(context.Background(), invoker, effectObserver{observed: true, detail: "visible"}, journal, "app", lateRule(), nil, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !invoker.called || !record.Succeeded || !record.Observed {
		t.Fatalf("unexpected record: %+v (called=%v)", record, invoker.called)
	}
	if len(journal.records) != 1 || journal.records[0].RuleID != "sales.order.stock" {
		t.Fatalf("the attempt must be journaled: %+v", journal.records)
	}
}

func TestRunEffectDoesNotConfuseSuccessWithObservation(t *testing.T) {
	journal := &memoryJournal{}
	record, err := execution.RunEffect(context.Background(), &effectInvoker{}, effectObserver{observed: false, detail: "not visible yet"}, journal, "app", lateRule(), nil, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !record.Succeeded {
		t.Fatalf("the invocation succeeded: %+v", record)
	}
	if record.Observed {
		t.Fatalf("a success code must not become an observation: %+v", record)
	}
}

func TestRunEffectRecordsAFailedInvocation(t *testing.T) {
	journal := &memoryJournal{}
	record, err := execution.RunEffect(context.Background(), &effectInvoker{err: errors.New("capability failed")}, nil, journal, "app", lateRule(), nil, nil)
	if err != nil {
		t.Fatalf("a failed effect is a record, not an error: %v", err)
	}
	if record.Succeeded || record.Detail != "capability failed" {
		t.Fatalf("unexpected record: %+v", record)
	}
	if len(journal.records) != 1 {
		t.Fatalf("the failed attempt must be journaled: %+v", journal.records)
	}
}

func TestRunEffectReportsAJournalError(t *testing.T) {
	journal := &memoryJournal{err: errors.New("disk full")}
	if _, err := execution.RunEffect(context.Background(), &effectInvoker{}, nil, journal, "app", lateRule(), nil, nil); err == nil {
		t.Fatal("an unrecorded effect must be reported")
	}
}

func TestRunEffectRejectsAnInTransactionRule(t *testing.T) {
	rule := lateRule()
	rule.Stage = "in-transaction"
	if _, err := execution.RunEffect(context.Background(), &effectInvoker{}, nil, &memoryJournal{}, "app", rule, nil, nil); err == nil {
		t.Fatal("an effect the transaction covers must not be run as a late effect")
	}
}
func compensationRules() []semantic.EffectRule {
	return []semantic.EffectRule{
		{ID: "sales.order.undo", OnCapability: "sales.order.submit", Stage: "compensation", EffectKind: "update", EffectData: "sales.stock", ViaCapability: "sales.stock.restore", OnFailure: "compensate"},
		{ID: "sales.order.other", OnCapability: "sales.order.submit", Stage: "compensation", EffectKind: "update", EffectData: "sales.price", ViaCapability: "sales.price.restore", OnFailure: "compensate"},
		{ID: "sales.order.late", OnCapability: "sales.order.submit", Stage: "after-commit", EffectKind: "update", EffectData: "sales.stock", ViaCapability: "sales.stock.decrement"},
		{ID: "other.order.undo", OnCapability: "sales.other.submit", Stage: "compensation", EffectKind: "update", EffectData: "sales.stock", ViaCapability: "sales.stock.restore"},
	}
}

func TestCompensationsForMatchesCapabilityAndData(t *testing.T) {
	failed := lateRule()
	compensations := execution.CompensationsFor(compensationRules(), failed)
	if len(compensations) != 1 {
		t.Fatalf("only the compensation for the same capability and data answers: %+v", compensations)
	}
	if compensations[0].ID != "sales.order.undo" {
		t.Fatalf("unexpected compensation: %+v", compensations[0])
	}
}

func TestNeedsCompensationFollowsTheRule(t *testing.T) {
	if !execution.NeedsCompensation(semantic.EffectRule{OnFailure: "compensate"}) {
		t.Fatal("a rule that asks for compensation must start one")
	}
	if execution.NeedsCompensation(semantic.EffectRule{OnFailure: "reject"}) {
		t.Fatal("a rule that rejects must not start a compensation")
	}
}
func TestCompensateRunsTheMatchingCompensation(t *testing.T) {
	journal := &memoryJournal{}
	failed := lateRule()
	failed.OnFailure = "compensate"
	records, err := execution.Compensate(context.Background(), &effectInvoker{}, nil, journal, "app", compensationRules(), failed, nil, nil)
	if err != nil {
		t.Fatalf("compensate: %v", err)
	}
	if len(records) != 1 || records[0].RuleID != "sales.order.undo" || !records[0].Succeeded {
		t.Fatalf("unexpected compensation records: %+v", records)
	}
	if len(journal.records) != 1 {
		t.Fatalf("the compensation must be journaled: %+v", journal.records)
	}
}

func TestCompensateDoesNothingForARejectingRule(t *testing.T) {
	failed := lateRule()
	failed.OnFailure = "reject"
	records, err := execution.Compensate(context.Background(), &effectInvoker{}, nil, &memoryJournal{}, "app", compensationRules(), failed, nil, nil)
	if err != nil || len(records) != 0 {
		t.Fatalf("a rejecting rule must not start a compensation: records=%+v err=%v", records, err)
	}
}

func TestCompensateRecordsItsOwnFailure(t *testing.T) {
	journal := &memoryJournal{}
	failed := lateRule()
	failed.OnFailure = "compensate"
	records, err := execution.Compensate(context.Background(), &effectInvoker{err: errors.New("undo failed")}, nil, journal, "app", compensationRules(), failed, nil, nil)
	if err != nil {
		t.Fatalf("a failed compensation is a record: %v", err)
	}
	if len(records) != 1 || records[0].Succeeded || records[0].Detail != "undo failed" {
		t.Fatalf("unexpected compensation record: %+v", records)
	}
}

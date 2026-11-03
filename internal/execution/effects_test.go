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

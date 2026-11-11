package execution

import (
	"context"
	"fmt"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/effectlog"
	"tei/internal/failure"
	"tei/internal/semantic"
)

// EffectRequirement names an effect that the invocation result cannot witness.
// An effect outside the transaction may still fail after the capability has
// returned success, so the caller has to observe it independently.
type EffectRequirement struct {
	RuleID string `json:"ruleId"`
	Stage  string `json:"stage"`
}

// EffectsOutsideTransaction lists the rules bound to a capability that run at a
// stage where the transaction no longer protects them. Comparing these with the
// invocation outcome would conflate a success code with an observation, so they
// are reported separately.
func EffectsOutsideTransaction(rules []semantic.EffectRule, capabilityID core.CanonicalID) []EffectRequirement {
	var out []EffectRequirement
	for _, rule := range rules {
		if rule.OnCapability != capabilityID {
			continue
		}
		if rule.Stage != "after-commit" && rule.Stage != "compensation" {
			continue
		}
		out = append(out, EffectRequirement{RuleID: string(rule.ID), Stage: rule.Stage})
	}
	return out
}

// EffectObserver reads an effect back through a path independent from the
// invocation. It reports whether the effect is visible and any detail worth
// recording. An error means the observation could not be made.
type EffectObserver interface {
	ObserveEffect(ctx context.Context, then effectlog.Record) (bool, string, error)
}

// RunEffect performs an effect that the invocation result cannot witness: it
// invokes the capability the rule names, records the attempt, and marks the
// record observed only when an independent read confirms it. A failed
// invocation is recorded, not returned as an error; an error from the journal or
// the observer is returned because the record would otherwise be incomplete.
func RunEffect(ctx context.Context, invoker Invoker, observer EffectObserver, journal effectlog.Journal, applicationID core.CanonicalID, rule semantic.EffectRule, adapters driver.BindingAdapterResolver, input any) (effectlog.Record, error) {
	record := effectlog.Record{
		RuleID:     string(rule.ID),
		Stage:      rule.Stage,
		Capability: string(rule.ViaCapability),
	}
	if rule.ViaCapability == "" {
		return record, fmt.Errorf("effect rule %s runs at %s, so it must name the capability that performs it", rule.ID, rule.Stage)
	}
	if rule.Stage != "after-commit" && rule.Stage != "compensation" {
		return record, fmt.Errorf("effect rule %s runs at %s, which the transaction already covers", rule.ID, rule.Stage)
	}

	if _, err := invoker.InvokeCapability(ctx, applicationID, rule.ViaCapability, adapters, input); err != nil {
		record.Detail = err.Error()
		if journalErr := journal.Append(record); journalErr != nil {
			return record, journalErr
		}
		return record, nil
	}
	record.Succeeded = true

	if observer != nil {
		observed, detail, err := observer.ObserveEffect(ctx, record)
		if err != nil {
			return record, fmt.Errorf("observe effect %s: %w", rule.ID, err)
		}
		record.Observed = observed
		record.Detail = detail
	}
	if err := journal.Append(record); err != nil {
		return record, err
	}
	return record, nil
}

// CompensationsFor returns the compensation rules that answer a failed effect:
// the same capability that owns the failed effect, and the same data the failed
// effect changed. A compensation for other data would undo work that is still
// wanted, so it is not returned.
func CompensationsFor(rules []semantic.EffectRule, failed effectRule) []semantic.EffectRule {
	var out []semantic.EffectRule
	for _, rule := range rules {
		if rule.Stage != "compensation" || rule.ViaCapability == "" {
			continue
		}
		if rule.OnCapability != failed.OnCapability || rule.EffectData != failed.EffectData {
			continue
		}
		out = append(out, rule)
	}
	return out
}

type effectRule = semantic.EffectRule

// NeedsCompensation reports whether a failed effect asks for compensation. A
// failure that only rejects does not start one.
func NeedsCompensation(rule semantic.EffectRule) bool {
	return rule.OnFailure == "compensate"
}

// Compensate runs the compensations that answer a failed effect, in the order the
// rules were declared. Each attempt is journaled by RunEffect, so a compensation
// that itself fails leaves a record instead of disappearing. All compensations
// are attempted even when one fails.
//
// The error reports what the run left behind, classified by the unified failure
// contract:
//   - nil when every compensation was applied, and also when none was: a
//     compensation that could not be performed is recorded, not raised, so the
//     caller reads the records rather than an error;
//   - a plain error when the journal or the observer failed, because the record
//     is incomplete and there is nothing a caller could look up;
//   - a failure.PartialError when some compensations were applied and some were
//     not. Returning nothing here would invite the caller to either resend
//     everything - duplicating the compensations that succeeded - or discard
//     everything - losing them. Neither is acceptable, so the state gets its own
//     kind.
func Compensate(ctx context.Context, invoker Invoker, observer EffectObserver, journal effectlog.Journal, applicationID core.CanonicalID, rules []semantic.EffectRule, failed semantic.EffectRule, adapters driver.BindingAdapterResolver, input any) ([]effectlog.Record, error) {
	if !NeedsCompensation(failed) {
		return nil, nil
	}
	var (
		records  []effectlog.Record
		firstErr error
	)
	for _, rule := range CompensationsFor(rules, failed) {
		record, err := RunEffect(ctx, invoker, observer, journal, applicationID, rule, adapters, input)
		records = append(records, record)
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("compensate %s: %w", rule.ID, err)
		}
	}
	return records, compensationFailure(records, firstErr)
}

// compensationFailure decides what a compensation run left behind.
//
// It reads the records rather than counting errors, because RunEffect reports a
// compensation whose invocation failed as a record with Succeeded=false and no
// error at all: counting errors would miss exactly the compensations that did
// not land, and would report a half-applied run as clean.
func compensationFailure(records []effectlog.Record, cause error) error {
	if len(records) < 2 {
		// One compensation cannot be half-applied. A single failure is a record
		// the caller reads, and a single journal error is the plain cause.
		return cause
	}
	applied := 0
	for _, record := range records {
		if record.Succeeded {
			applied++
		}
	}
	if applied == 0 || applied == len(records) {
		return cause
	}
	return &failure.PartialError{
		Code:    "COMPENSATION_PARTIAL",
		Message: fmt.Sprintf("%d of %d compensations were applied", applied, len(records)),
		Cause:   cause,
	}
}

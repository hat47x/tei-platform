package hook

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"tei/internal/application"
	"tei/internal/command"
	"tei/internal/core"
	"tei/internal/failure"
)

func TestWrapFailurePreservesUnknownCommandEffect(t *testing.T) {
	cause := fmt.Errorf("provider acknowledgement lost: %w",
		command.NewUnknownExecutionError("COMMAND_COMMIT_FAILED", "commit result unavailable", context.DeadlineExceeded))
	for _, policy := range []string{"reject", "warn"} {
		t.Run(policy, func(t *testing.T) {
			err := wrapFailure(cause, &application.Hook{Phase: "before-update", Failure: policy})
			var warning *Warning
			if errors.As(err, &warning) {
				t.Fatal("an unknown effect was treated as a warning and may be continued")
			}
			f, ok := failure.AsFailure(err)
			if !ok {
				t.Fatalf("no shared failure in wrapped hook error: %v", err)
			}
			if f.Kind() != failure.TechnicalFailure {
				t.Fatalf("Kind=%s, want technical_failure", f.Kind())
			}
			if f.Retryable() != failure.LookupFirst {
				t.Fatalf("Retryable=%s, want lookup_first", f.Retryable())
			}
			if !f.HasEffectUnknown() {
				t.Fatal("the hook wrapper erased the called Command's effectUnknown")
			}
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("the original cause was lost")
			}
		})
	}
}

func TestHookBusinessCodeDoesNotErasePartialEffect(t *testing.T) {
	partial := &failure.PartialError{Code: "PROVIDER_PARTIAL", Message: "some rows may be committed"}
	err := &Error{Code: "HOOK_CALL_REFUSED", Message: "outer hook refusal", Cause: partial}
	f, ok := failure.AsFailure(err)
	if !ok {
		t.Fatal("outer hook is not a shared Failure")
	}
	if f.Kind() != failure.PartialSuccess {
		t.Fatalf("Kind=%s, want partial_success", f.Kind())
	}
	if f.Retryable() != failure.LookupFirst || !f.HasEffectUnknown() {
		t.Fatalf("wrapped partial success lost its safety contract: retry=%s unknown=%v",
			f.Retryable(), f.HasEffectUnknown())
	}
}

func TestHookCleanRefusalAndWarningRemainUnchanged(t *testing.T) {
	clean := &Error{Code: "HOOK_CALL_REFUSED", Message: "no call made"}
	if clean.Kind() != failure.BusinessRejection ||
		clean.Retryable() != failure.NotRetryable ||
		clean.HasEffectUnknown() {
		t.Fatalf("clean refusal was reclassified: %+v", clean)
	}
	err := wrapFailure(errors.New("local read-only check failed"),
		&application.Hook{Phase: "before-read", Failure: "warn"})
	var warning *Warning
	if !errors.As(err, &warning) {
		t.Fatalf("ordinary warning was changed: %T %v", err, err)
	}
}

func TestRunSequenceStopsAfterUnknownEffectEvenWithWarnPolicy(t *testing.T) {
	runner := Runner{
		World: hookWorld("warn"),
		Implementation: fakeImplementation{
			err: command.NewUnknownExecutionError("COMMAND_COMMIT_FAILED", "ack lost", nil),
		},
	}
	report, err := runner.RunSequence(context.Background(), invocation())
	if err == nil {
		t.Fatal("RunSequence continued after an unknown effect")
	}
	if report.Refused != "before-update" {
		t.Fatalf("Refused=%q, want before-update", report.Refused)
	}
	if len(report.Warned) != 0 || len(report.Executed) != 0 {
		t.Fatalf("unknown effect was recorded as a successful/warned phase: %+v", report)
	}
	f, ok := failure.AsFailure(err)
	if !ok || !f.HasEffectUnknown() || f.Retryable() != failure.LookupFirst {
		t.Fatalf("sequence lost effect uncertainty: error=%v", err)
	}
}

func TestRunReachableDoesNotWarnOnUnknownEffect(t *testing.T) {
	world := hookWorld("warn")
	world.Applications[0].Patterns[0].Interaction.Hooks[0].Uses = nil
	runner := Runner{
		World: world,
		Implementation: fakeImplementation{
			err: command.NewUnknownExecutionError("COMMAND_COMMIT_FAILED", "ack lost", nil),
		},
	}
	report, err := runner.RunReachable(context.Background(), "order_entry",
		"order_entry.order_record", invocation().Reads, ReferenceReach{})
	if err == nil {
		t.Fatal("RunReachable continued after an unknown effect")
	}
	if report == nil || report.Refused != "before-update" {
		t.Fatalf("unknown-effect phase was not refused: %+v", report)
	}
	if len(report.Warned) != 0 {
		t.Fatalf("unknown-effect phase became a warning: %+v", report)
	}
	f, ok := failure.AsFailure(err)
	if !ok || !f.HasEffectUnknown() || f.Retryable() != failure.LookupFirst {
		t.Fatalf("RunReachable lost effect uncertainty: %v", err)
	}
}

func TestRunRowsKeepsUnknownEffectAcrossBothWrappers(t *testing.T) {
	world := hookWorld("reject")
	h := &world.Applications[0].Patterns[0].Interaction.Hooks[0]
	h.Scope = "row"
	h.Uses = nil
	runner := Runner{
		World: world,
		Implementation: fakeImplementation{
			err: command.NewUnknownExecutionError("COMMAND_COMMIT_FAILED", "ack lost", nil),
		},
	}
	err := runner.RunRows(context.Background(), invocation(),
		[]map[core.CanonicalID]string{{"sales.order.status": "open"}})
	if err == nil {
		t.Fatal("row execution did not refuse an uncertain effect")
	}
	f, ok := failure.AsFailure(err)
	if !ok || !f.HasEffectUnknown() || f.Retryable() != failure.LookupFirst {
		t.Fatalf("row wrapper erased the called Command's uncertain effect: %v", err)
	}
}

package hook

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
)

// sequenceWorld declares a record pattern with the given hooks, so the test can
// see which phases a sequence reaches and in what order.
func sequenceWorld(hooks ...application.Hook) ir.World {
	return ir.World{
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID:          "order_entry.order_record",
				Kind:        application.PatternRecord,
				Data:        "sales.order",
				Interaction: &application.Interaction{Hooks: hooks},
			}},
		}},
	}
}

// recordingSequence records the phases the implementation was asked to run.
type recordingSequence struct {
	phases []string
	// fail maps a phase to the error it returns.
	fail map[string]error
}

func (s *recordingSequence) Run(_ context.Context, invocation Invocation) error {
	s.phases = append(s.phases, invocation.Phase)
	return s.fail[invocation.Phase]
}

func sequenceInvocation() Invocation {
	return Invocation{Application: "order_entry", Pattern: "order_entry.order_record"}
}

// TestRunSequenceUsesThePhaseOrder checks that phases run in the declared order,
// not in declaration order: an update hook declared before a read hook still runs
// after it.
func TestRunSequenceUsesThePhaseOrder(t *testing.T) {
	world := sequenceWorld(
		application.Hook{Phase: "before-update"},
		application.Hook{Phase: "before-read"},
		application.Hook{Phase: "after-update"},
	)
	implementation := &recordingSequence{fail: map[string]error{}}
	runner := Runner{World: world, Implementation: implementation}
	result, err := runner.RunSequence(context.Background(), sequenceInvocation())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	want := []string{"before-read", "before-update", "after-update"}
	if !reflect.DeepEqual(implementation.phases, want) {
		t.Fatalf("phases=%v want=%v", implementation.phases, want)
	}
	if !reflect.DeepEqual(result.Executed, want) {
		t.Fatalf("executed=%v", result.Executed)
	}
}

// TestRunSequenceSkipsUndeclaredPhases checks that a phase the pattern does not
// declare is not reported as executed.
func TestRunSequenceSkipsUndeclaredPhases(t *testing.T) {
	world := sequenceWorld(application.Hook{Phase: "after-create"})
	implementation := &recordingSequence{fail: map[string]error{}}
	runner := Runner{World: world, Implementation: implementation}
	result, err := runner.RunSequence(context.Background(), sequenceInvocation())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !reflect.DeepEqual(result.Executed, []string{"after-create"}) {
		t.Fatalf("executed=%v", result.Executed)
	}
	if len(implementation.phases) != 1 {
		t.Fatalf("phases=%v", implementation.phases)
	}
}

// TestRunSequenceStopsAtARefusal checks that a business refusal stops the
// sequence and names the phase that refused: later hooks must not run after a
// condition failed.
func TestRunSequenceStopsAtARefusal(t *testing.T) {
	world := sequenceWorld(
		application.Hook{Phase: "before-read"},
		application.Hook{Phase: "before-update"},
		application.Hook{Phase: "after-update"},
	)
	refusal := &NotSatisfied{Reason: "quantity must be positive", Outcome: "value-rejected"}
	implementation := &recordingSequence{fail: map[string]error{"before-update": refusal}}
	runner := Runner{World: world, Implementation: implementation}
	result, err := runner.RunSequence(context.Background(), sequenceInvocation())
	if !errors.Is(err, refusal) {
		t.Fatalf("err=%v", err)
	}
	if result.Refused != "before-update" {
		t.Fatalf("refused=%s", result.Refused)
	}
	if !reflect.DeepEqual(result.Executed, []string{"before-read"}) {
		t.Fatalf("executed=%v", result.Executed)
	}
	if len(implementation.phases) != 2 {
		t.Fatalf("phases=%v", implementation.phases)
	}
}

// TestRunSequenceContinuesAfterAWarn checks that an on-failure warn hook records
// the warning and lets the later phases run.
func TestRunSequenceContinuesAfterAWarn(t *testing.T) {
	world := sequenceWorld(
		application.Hook{Phase: "before-read"},
		application.Hook{Phase: "before-update", Failure: "warn"},
		application.Hook{Phase: "after-update"},
	)
	implementation := &recordingSequence{fail: map[string]error{"before-update": errors.New("provider unavailable")}}
	runner := Runner{World: world, Implementation: implementation}
	result, err := runner.RunSequence(context.Background(), sequenceInvocation())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !reflect.DeepEqual(result.Executed, []string{"before-read", "after-update"}) {
		t.Fatalf("executed=%v", result.Executed)
	}
	if !reflect.DeepEqual(result.Warned, []string{"before-update"}) {
		t.Fatalf("warned=%v", result.Warned)
	}
	if len(implementation.phases) != 3 {
		t.Fatalf("phases=%v", implementation.phases)
	}
}

// TestRunSequenceRefusesWithoutADeclaration checks that a pattern with no hook is
// refused instead of reported as an empty success.
func TestRunSequenceRefusesWithoutADeclaration(t *testing.T) {
	implementation := &recordingSequence{fail: map[string]error{}}
	runner := Runner{World: sequenceWorld(), Implementation: implementation}
	if _, err := runner.RunSequence(context.Background(), sequenceInvocation()); err == nil {
		t.Fatal("a pattern with no hook was accepted")
	}
	if len(implementation.phases) != 0 {
		t.Fatalf("phases=%v", implementation.phases)
	}
}

// TestRunSequenceRefusesUnsupportedPhase checks that a declaration with a phase
// the order does not name is not silently skipped.
func TestRunSequenceRefusesUnsupportedPhase(t *testing.T) {
	world := sequenceWorld(application.Hook{Phase: "during-update"})
	implementation := &recordingSequence{fail: map[string]error{}}
	runner := Runner{World: world, Implementation: implementation}
	if _, err := runner.RunSequence(context.Background(), sequenceInvocation()); err == nil {
		t.Fatal("an unsupported phase was accepted")
	}
	if len(implementation.phases) != 0 {
		t.Fatalf("phases=%v", implementation.phases)
	}
}

// TestRunSequenceRefusesNonRecordPattern checks that the sequence uses the same
// pattern boundary as Run.
func TestRunSequenceRefusesNonRecordPattern(t *testing.T) {
	world := ir.World{
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID:   "order_entry.recent_orders",
				Kind: application.PatternBrowse,
			}},
		}},
	}
	implementation := &recordingSequence{fail: map[string]error{}}
	runner := Runner{World: world, Implementation: implementation}
	_, err := runner.RunSequence(context.Background(), Invocation{Application: "order_entry", Pattern: "order_entry.recent_orders"})
	var hookError *Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_PATTERN_NOT_RECORD" {
		t.Fatalf("err=%v", err)
	}
}

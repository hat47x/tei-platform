package standardbusiness_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/command"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
)

// RQ-C2の「確認方法」を一箇所で固定する試験です。
//
// 要求文は次の3つを求めています。
//
//  1. 行レベルの検証、導出項目、参照入力の補助を、フック宣言と規則Pluginで表し、
//     同じ入力に同じ結果を返すこと。
//  2. 宣言していない能力を呼ぶフックが、検証時と実行時の両方で拒否されること。
//  3. フックの失敗が、拒否・不明・未実行のいずれかとして観測できること。
//
// 規則Pluginの供給路は未実装なので、ここでは参照実装が持つ宣言・拒否・観測の面を
// 確かめます。Plugin供給そのものを確かめたとは書きません。

func rqHookPattern(hooks ...application.Hook) *application.Pattern {
	return &application.Pattern{
		ID:          "order_entry.order_record",
		Kind:        application.PatternRecord,
		Data:        "sales.order",
		Interaction: &application.Interaction{Hooks: hooks},
	}
}

// TestRQC2HookDeclarationCarriesTheContract checks 確認方法1: a hook declaration
// carries the trigger, the reads, the capabilities it uses, what it produces, and
// the failure name, so the declaration — not the implementation — says what may
// happen.
func TestRQC2HookDeclarationCarriesTheContract(t *testing.T) {
	declaration := application.Hook{
		Phase:    "before-update",
		Reads:    []core.CanonicalID{"sales.order.amount"},
		Uses:     []core.CanonicalID{"sales.check_limit"},
		Produces: []core.CanonicalID{"sales.order.limit_ok"},
		Failure:  hook.OutcomeValueRejected,
	}
	if declaration.Phase == "" || len(declaration.Reads) == 0 || len(declaration.Uses) == 0 || len(declaration.Produces) == 0 {
		t.Fatalf("the declaration lost part of the contract: %+v", declaration)
	}
	// The failure name is a word from the declared vocabulary, not a free string.
	if err := hook.ValidateOutcome(declaration.Failure); err != nil {
		t.Fatalf("the declared failure was refused: %v", err)
	}
	if err := hook.ValidateOutcome("valueRejected"); err == nil {
		t.Fatal("a misspelt failure name was accepted")
	}
}

// TestRQC2SameInputGivesTheSameReads checks 確認方法1 for the reads: the same
// declaration and the same observed values produce the same bound reads, and a read
// outside the declaration is refused.
func TestRQC2SameInputGivesTheSameReads(t *testing.T) {
	declaration := &application.Hook{
		Phase: "before-update",
		Reads: []core.CanonicalID{"sales.order.amount"},
	}
	reads := map[core.CanonicalID]string{"sales.order.amount": "1200"}
	first, err := hook.ReadsFor(declaration, reads)
	if err != nil {
		t.Fatal(err)
	}
	second, err := hook.ReadsFor(declaration, reads)
	if err != nil {
		t.Fatal(err)
	}
	if first["sales.order.amount"] != second["sales.order.amount"] {
		t.Fatalf("the same input gave %q then %q", first["sales.order.amount"], second["sales.order.amount"])
	}
	// An observed value the declaration does not name is refused.
	if _, err := hook.ReadsFor(declaration, map[core.CanonicalID]string{
		"sales.order.amount": "1200",
		"sales.order.secret": "x",
	}); err == nil {
		t.Fatal("an undeclared read was accepted")
	}
	// A declared read that was not observed is refused, not passed as empty.
	if _, err := hook.ReadsFor(declaration, map[core.CanonicalID]string{}); err == nil {
		t.Fatal("an unobserved read was accepted")
	}
}

// TestRQC2UndeclaredCallIsRefusedAtExecution checks 確認方法2 for the execution side:
// the caller a hook receives permits only the capabilities the declaration lists, so
// validation and execution apply the same rule.
func TestRQC2UndeclaredCallIsRefusedAtExecution(t *testing.T) {
	recording := &rqCaller{}
	declaration := &application.Hook{
		Phase: "before-update",
		Uses:  []core.CanonicalID{"sales.check_limit"},
	}
	caller := hook.Declared{Inner: recording, Hook: declaration}

	if _, err := caller.Call(context.Background(), "sales.check_limit", map[string]string{"amount": "1200"}); err != nil {
		t.Fatalf("the declared capability was refused: %v", err)
	}
	if len(recording.calls) != 1 {
		t.Fatalf("the declared call did not reach the boundary: %v", recording.calls)
	}
	// A capability the declaration does not name never reaches the boundary.
	if _, err := caller.Call(context.Background(), "sales.delete_all", nil); err == nil {
		t.Fatal("an undeclared capability was accepted")
	}
	if len(recording.calls) != 1 {
		t.Fatalf("an undeclared call reached the boundary: %v", recording.calls)
	}
}

type rqCaller struct {
	calls []core.CanonicalID
}

func (c *rqCaller) Call(_ context.Context, capability core.CanonicalID, _ map[string]string) (any, error) {
	c.calls = append(c.calls, capability)
	return "ok", nil
}

// TestRQC2RunSequenceStopsAtTheRefusedPhase checks 確認方法3: a phase that was not
// reached is observed as such, a business refusal stops the sequence, and the record
// says which phases ran and which one refused.
func TestRQC2RunSequenceStopsAtTheRefusedPhase(t *testing.T) {
	pattern := rqHookPattern(
		application.Hook{Phase: "before-read"},
		application.Hook{Phase: "before-update"},
		application.Hook{Phase: "after-update"},
	)
	refusal := &hook.NotSatisfied{Reason: "the amount exceeds the limit", Outcome: hook.OutcomeValueRejected}
	implementation := &rqImplementation{fail: map[string]error{"before-update": refusal}}
	runner := hook.Runner{
		World:          ir.World{Applications: []application.Application{{ID: "order_entry", Patterns: []application.Pattern{*pattern}}}},
		Implementation: implementation,
	}
	result, err := runner.RunSequence(context.Background(), hook.Invocation{Application: "order_entry", Pattern: "order_entry.order_record"})
	if err == nil {
		t.Fatal("the sequence reported success after a refusal")
	}
	if !errors.Is(err, error(refusal)) && err.Error() != refusal.Error() {
		t.Fatalf("the returned error is not the refusal: %v", err)
	}
	if len(result.Executed) != 1 || result.Executed[0] != "before-read" {
		t.Fatalf("executed=%v", result.Executed)
	}
	if result.Refused != "before-update" {
		t.Fatalf("refused=%q", result.Refused)
	}
	// The phase after the refusal never ran, which is what "未実行" observes.
	if len(implementation.ran) != 2 {
		t.Fatalf("a phase after the refusal ran: %v", implementation.ran)
	}
}

// TestRQC2WarnContinuesAndIsRecorded checks that an `on-failure warn` hook is
// recorded as warned and the sequence continues, so a soft failure is neither a
// refusal nor silent success.
func TestRQC2WarnContinuesAndIsRecorded(t *testing.T) {
	pattern := rqHookPattern(
		application.Hook{Phase: "before-read", Failure: "warn"},
		application.Hook{Phase: "before-update"},
	)
	implementation := &rqImplementation{fail: map[string]error{"before-read": errors.New("soft failure")}}
	runner := hook.Runner{
		World:          ir.World{Applications: []application.Application{{ID: "order_entry", Patterns: []application.Pattern{*pattern}}}},
		Implementation: implementation,
	}
	result, err := runner.RunSequence(context.Background(), hook.Invocation{Application: "order_entry", Pattern: "order_entry.order_record"})
	if err != nil {
		t.Fatalf("a warn failure stopped the sequence: %v", err)
	}
	if len(result.Warned) != 1 || result.Warned[0] != "before-read" {
		t.Fatalf("warned=%v", result.Warned)
	}
	if len(result.Executed) != 1 || result.Executed[0] != "before-update" {
		t.Fatalf("executed=%v", result.Executed)
	}
	if result.Refused != "" {
		t.Fatalf("refused=%q", result.Refused)
	}
}

type rqImplementation struct {
	ran  []string
	fail map[string]error
}

func (i *rqImplementation) Run(_ context.Context, invocation hook.Invocation) error {
	i.ran = append(i.ran, invocation.Phase)
	return i.fail[invocation.Phase]
}

// TestRQC2NotSatisfiedIsNotAnExecutionFault checks that a business refusal is
// distinguishable from an execution fault: `OutcomeOf` reports the refusal's name
// and refuses to read an execution error as a business outcome.
func TestRQC2NotSatisfiedIsNotAnExecutionFault(t *testing.T) {
	refusal := &hook.NotSatisfied{Reason: "the amount exceeds the limit", Outcome: hook.OutcomeValueRejected}
	var execution *command.ExecutionError
	if errors.As(refusal, &execution) {
		t.Fatal("a business refusal was reported as an execution fault")
	}
	outcome, ok := hook.OutcomeOf(refusal)
	if !ok || outcome != hook.OutcomeValueRejected {
		t.Fatalf("outcome=%q ok=%v", outcome, ok)
	}
	// A reason-only refusal is still a refusal, and has no Command outcome.
	if _, ok := hook.OutcomeOf(&hook.NotSatisfied{Reason: "the amount exceeds the limit"}); ok {
		t.Fatal("a reason-only refusal was given a Command outcome")
	}
	// An execution error is not a business outcome.
	if _, ok := hook.OutcomeOf(&hook.Error{Code: "HOOK_IMPLEMENTATION_FAILED", Message: "boom"}); ok {
		t.Fatal("an execution fault was read as a business outcome")
	}
}

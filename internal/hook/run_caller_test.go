package hook

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/core"
)

// callingImplementation stands in for a rule plugin that needs to call a
// capability. It records what it was allowed to call, so the test can tell
// whether the declaration, not the implementation, decided the reach.
type callingImplementation struct {
	calls    []core.CanonicalID
	requests []core.CanonicalID
}

func (c *callingImplementation) Run(context.Context, Invocation) error {
	panic("a calling implementation must be run through RunWithCaller")
}

func (c *callingImplementation) RunWithCaller(ctx context.Context, _ Invocation, caller Caller) error {
	for _, capability := range c.requests {
		if _, err := caller.Call(ctx, capability, nil); err != nil {
			return err
		}
		c.calls = append(c.calls, capability)
	}
	return nil
}

// countingCaller records the capabilities that reached the boundary.
type countingCaller struct {
	called []core.CanonicalID
	err    error
}

func (c *countingCaller) Call(_ context.Context, capability core.CanonicalID, _ map[string]string) (any, error) {
	if c.err != nil {
		return nil, c.err
	}
	c.called = append(c.called, capability)
	return nil, nil
}

// hookErrorCodes walks the error chain and lists every hook.Error code, so a
// test can see the outer rejection and its cause separately: errors.As alone
// would stop at the first match and hide the refused call underneath it.
func hookErrorCodes(err error) []string {
	var codes []string
	for err != nil {
		var hookError *Error
		if errors.As(err, &hookError) {
			codes = append(codes, hookError.Code)
		}
		err = errors.Unwrap(err)
	}
	return codes
}

func hasCode(codes []string, want string) bool {
	for _, code := range codes {
		if code == want {
			return true
		}
	}
	return false
}

// TestRunnerHandsTheCallingImplementationADeclaredCaller checks that a hook
// which calls capabilities reaches the boundary its declaration permits.
func TestRunnerHandsTheCallingImplementationADeclaredCaller(t *testing.T) {
	inner := &countingCaller{}
	implementation := &callingImplementation{requests: []core.CanonicalID{"sales.submit"}}
	runner := Runner{World: hookWorld("reject"), Implementation: implementation, Caller: inner}
	if err := runner.Run(context.Background(), invocation()); err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(inner.called) != 1 || inner.called[0] != "sales.submit" {
		t.Fatalf("inner calls=%v", inner.called)
	}
}

// TestRunnerRefusesACallTheHookDidNotDeclare checks that the Runner's caller
// refuses an undeclared capability before it reaches the boundary, even though
// the implementation asked for it.
func TestRunnerRefusesACallTheHookDidNotDeclare(t *testing.T) {
	inner := &countingCaller{}
	implementation := &callingImplementation{requests: []core.CanonicalID{"sales.delete"}}
	runner := Runner{World: hookWorld("reject"), Implementation: implementation, Caller: inner}
	err := runner.Run(context.Background(), invocation())
	if err == nil {
		t.Fatal("an undeclared capability was allowed to run")
	}
	codes := hookErrorCodes(err)
	if !hasCode(codes, "HOOK_REJECTED") || !hasCode(codes, "HOOK_UNDECLARED_CALL") {
		t.Fatalf("codes=%v err=%v", codes, err)
	}
	if len(inner.called) != 0 {
		t.Fatalf("undeclared capability reached the boundary: %v", inner.called)
	}
	if implementation.calls != nil {
		t.Fatalf("implementation recorded a refused call: %v", implementation.calls)
	}
}

// TestRunnerReportsAMissingBoundary checks that a declared capability with no
// configured boundary is a boundary failure, not a silent success.
func TestRunnerReportsAMissingBoundary(t *testing.T) {
	implementation := &callingImplementation{requests: []core.CanonicalID{"sales.submit"}}
	runner := Runner{World: hookWorld("reject"), Implementation: implementation}
	err := runner.Run(context.Background(), invocation())
	if err == nil {
		t.Fatal("a declared capability ran without a boundary")
	}
	if codes := hookErrorCodes(err); !hasCode(codes, "HOOK_IMPLEMENTATION_UNAVAILABLE") {
		t.Fatalf("codes=%v err=%v", codes, err)
	}
}

// failingImplementation fails with the given error, so the failure kinds can be
// compared in one place.
type failingImplementation struct {
	err error
}

func (f failingImplementation) Run(context.Context, Invocation) error { return f.err }

// TestRunnerReturnsNotSatisfiedAsItself checks that a business refusal is not
// wrapped as an execution fault: the caller must be able to tell the two apart.
func TestRunnerReturnsNotSatisfiedAsItself(t *testing.T) {
	refusal := &NotSatisfied{Reason: "quantity must be positive", Outcome: "value-rejected"}
	runner := Runner{World: hookWorld("reject"), Implementation: failingImplementation{err: refusal}}
	err := runner.Run(context.Background(), invocation())
	if !errors.Is(err, refusal) {
		t.Fatalf("err=%v", err)
	}
	var notSatisfied *NotSatisfied
	if !errors.As(err, &notSatisfied) || notSatisfied.Outcome != "value-rejected" {
		t.Fatalf("err=%v", err)
	}
	var hookError *Error
	if errors.As(err, &hookError) {
		t.Fatalf("a business refusal was reported as an execution error: %v", hookError)
	}
}

// TestRunnerKeepsNotSatisfiedUnderWarn checks that on-failure warn does not
// turn a business refusal into a warning: the hook said no, it did not fail.
func TestRunnerKeepsNotSatisfiedUnderWarn(t *testing.T) {
	refusal := &NotSatisfied{Reason: "quantity must be positive"}
	runner := Runner{World: hookWorld("warn"), Implementation: failingImplementation{err: refusal}}
	err := runner.Run(context.Background(), invocation())
	var warning *Warning
	if errors.As(err, &warning) {
		t.Fatalf("a business refusal was reported as a warning: %v", warning)
	}
	if !errors.Is(err, refusal) {
		t.Fatalf("err=%v", err)
	}
}

// TestRunnerWrapsAnExecutionFailure checks that the kinds stay distinct: an
// ordinary failure is still an execution failure.
func TestRunnerWrapsAnExecutionFailure(t *testing.T) {
	runner := Runner{World: hookWorld("reject"), Implementation: failingImplementation{err: errors.New("provider unavailable")}}
	err := runner.Run(context.Background(), invocation())
	if isNotSatisfiedError(err) {
		t.Fatalf("an execution failure was reported as a business refusal: %v", err)
	}
	var hookError *Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_REJECTED" {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(err.Error(), "provider unavailable") {
		t.Fatalf("cause was lost: %v", err)
	}
}

// isNotSatisfiedError is the exported-free check the test uses to assert a
// failure is not a business refusal.
func isNotSatisfiedError(err error) bool {
	var notSatisfied *NotSatisfied
	return errors.As(err, &notSatisfied)
}

// TestRunnerStillRunsAPlainImplementation checks that an implementation which
// does not call capabilities keeps working through Run, so the new contract is
// opt-in.
func TestRunnerStillRunsAPlainImplementation(t *testing.T) {
	runner := Runner{World: hookWorld("reject"), Implementation: fakeImplementation{}}
	if err := runner.Run(context.Background(), invocation()); err != nil {
		t.Fatalf("err=%v", err)
	}
}

// TestRunnerWarnsOnACallingFailure checks that on-failure warn also applies to
// a calling implementation's refusal, so the failure keeps its kind.
func TestRunnerWarnsOnACallingFailure(t *testing.T) {
	inner := &countingCaller{}
	implementation := &callingImplementation{requests: []core.CanonicalID{"sales.delete"}}
	runner := Runner{World: hookWorld("warn"), Implementation: implementation, Caller: inner}
	err := runner.Run(context.Background(), invocation())
	var warning *Warning
	if !errors.As(err, &warning) {
		t.Fatalf("err=%v", err)
	}
	if codes := hookErrorCodes(err); !hasCode(codes, "HOOK_UNDECLARED_CALL") {
		t.Fatalf("codes=%v err=%v", codes, err)
	}
	if len(inner.called) != 0 {
		t.Fatalf("refused capability reached the boundary: %v", inner.called)
	}
}

// TestRunnerKeepsCallerFailureDistinctFromRefusal checks that a boundary failure
// the hook reports as a business refusal keeps the caller's cause available.
func TestRunnerKeepsCallerFailureDistinctFromRefusal(t *testing.T) {
	inner := &countingCaller{err: errors.New("boundary refused")}
	implementation := &callingImplementation{requests: []core.CanonicalID{"sales.submit"}}
	runner := Runner{World: hookWorld("reject"), Implementation: implementation, Caller: inner}
	err := runner.Run(context.Background(), invocation())
	if err == nil {
		t.Fatal("a boundary failure was ignored")
	}
	if isNotSatisfiedError(err) {
		t.Fatalf("a boundary failure was reported as a business refusal: %v", err)
	}
	var hookError *Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_REJECTED" {
		t.Fatalf("err=%v", err)
	}
	if !errors.Is(err, inner.err) {
		t.Fatalf("the boundary cause was lost: %v", err)
	}
}

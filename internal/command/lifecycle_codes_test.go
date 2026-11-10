package command

import (
	"context"
	"testing"
)

// A lifecycle that refuses to record the attempt stops the command before the
// executor runs, and the refusal has its own code so a caller can tell "the
// dispatcher did not start this attempt" from "the attempt failed" (#4032).
func TestLifecycleRefusalCodeIsProduced(t *testing.T) {
	lifecycle := NewLifecycle()
	if _, err := lifecycle.Dispatch("cmd-1", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	// The same owner/generation cannot dispatch twice: the first attempt is
	// still open, so the second dispatch is refused by the lifecycle.
	invoker := LifecycleInvoker{
		Invoker:   Invoker{World: commandWorld(), Executor: fakeExecutor{}, Authorizer: ReferenceAuthorizer{}},
		Lifecycle: lifecycle,
	}
	_, err := invoker.Invoke(context.Background(), request(), DispatchOptions{
		CommandID: "cmd-1", Owner: "owner-a", Generation: 1,
	})
	if code := executionCode(t, err); code != "COMMAND_LIFECYCLE_REFUSED" {
		t.Fatalf("code=%s (%v)", code, err)
	}
}

package runtime_test

import (
	"context"
	"testing"

	pruntime "tei/internal/plugin/runtime"
)

func TestInvocationRecoveryUnknownSeparatesSupervisorAndRuntimeAuthority(t *testing.T) {
	ctx := context.Background()
	store := pruntime.NewMemoryInvocationHandleStore()
	h := recoveryHandle()
	h.RuntimeFencingToken = 1
	if _, err := store.Save(ctx, h, 0); err != nil {
		t.Fatal(err)
	}
	rt := pruntime.NewReferenceReattachableRuntime()
	rt.Put(h.RuntimeHandle, pruntime.RuntimeUnknown, 1)

	unresolved, err := pruntime.ReconcileDurableInvocation(ctx, store, rt, recoveryPlan(), h.InvocationID, "host-b", 2)
	if err != nil {
		t.Fatal(err)
	}
	if unresolved.Outcome != pruntime.RecoveryWaitingObserve {
		t.Fatalf("outcome=%s", unresolved.Outcome)
	}
	if unresolved.Handle.SupervisorOwnerID != "host-b" || unresolved.Handle.FencingToken != 2 {
		t.Fatalf("supervisor authority=%#v", unresolved.Handle)
	}
	if unresolved.Handle.RuntimeFencingToken != 1 {
		t.Fatalf("unknown observation advanced runtime authority: %#v", unresolved.Handle)
	}
	if len(unresolved.Findings) < 2 {
		t.Fatalf("findings=%#v", unresolved.Findings)
	}

	// The same supervisor generation may retry observation. Once the runtime is
	// live again, authority-bearing reattach advances only the runtime-confirmed
	// generation and fences the stale runtime token.
	rt.Put(h.RuntimeHandle, pruntime.RuntimeRunning, 1)
	reattached, err := pruntime.ReconcileDurableInvocation(ctx, store, rt, recoveryPlan(), h.InvocationID, "host-b", 2)
	if err != nil {
		t.Fatal(err)
	}
	if reattached.Outcome != pruntime.RecoveryReattached || reattached.Handle.FencingToken != 2 || reattached.Handle.RuntimeFencingToken != 2 {
		t.Fatalf("reattached=%#v", reattached)
	}
	if _, err := rt.ReattachInvocation(ctx, h.RuntimeHandle, 1); err == nil {
		t.Fatal("expected stale runtime generation rejection after authority convergence")
	}
}

func TestDurableInvocationHandleRejectsRuntimeAuthorityAheadOfSupervisor(t *testing.T) {
	h := recoveryHandle()
	h.FencingToken = 1
	h.RuntimeFencingToken = 2
	if err := h.Validate(); err == nil {
		t.Fatal("expected runtime fencing token ahead of supervisor to be rejected")
	}
}

func TestDurableInvocationHandleDoesNotInferMissingRuntimeFencingToken(t *testing.T) {
	h := recoveryHandle()
	h.RuntimeFencingToken = 0
	if err := h.Validate(); err != nil {
		t.Fatal(err)
	}
	if h.RuntimeFencingToken != 0 {
		t.Fatalf("runtime fencing token was inferred: %#v", h)
	}
}

func TestInvocationRecoveryTerminalObservationDoesNotAdvanceRuntimeAuthority(t *testing.T) {
	ctx := context.Background()
	for _, lifecycle := range []pruntime.RuntimeLifecycle{pruntime.RuntimeSucceeded, pruntime.RuntimeFailed, pruntime.RuntimeMissing} {
		t.Run(string(lifecycle), func(t *testing.T) {
			store := pruntime.NewMemoryInvocationHandleStore()
			h := recoveryHandle()
			h.RuntimeFencingToken = 1
			if _, err := store.Save(ctx, h, 0); err != nil {
				t.Fatal(err)
			}
			rt := pruntime.NewReferenceReattachableRuntime()
			if lifecycle != pruntime.RuntimeMissing {
				rt.Put(h.RuntimeHandle, lifecycle, 1)
			}

			res, err := pruntime.ReconcileDurableInvocation(ctx, store, rt, recoveryPlan(), h.InvocationID, "host-b", 2)
			if err != nil {
				t.Fatal(err)
			}
			if res.Handle.FencingToken != 2 || res.Handle.RuntimeFencingToken != 1 {
				t.Fatalf("non-authority-bearing observation advanced runtime authority: %#v", res.Handle)
			}
		})
	}
}

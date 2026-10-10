package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

const (
	testAccessSlot = "tei.reference.context-slot.authorization-access/v0"
	testTraceSlot  = "tei.reference.context-slot.trace/v0"
)

func accessSlotContracts() []ContextSlotContract {
	return []ContextSlotContract{
		{
			ID:                           testAccessSlot,
			Version:                      "0.1.0",
			ForbidDirectResultReflection: true,
		},
		{
			ID:      testTraceSlot,
			Version: "0.1.0",
		},
	}
}

func accessRequirement() []ContextSlotRequirement {
	return []ContextSlotRequirement{{
		ContractID:      testAccessSlot,
		ContractVersion: "0.1.0",
	}}
}

func accessHost(value string) HostExecutionContext {
	return HostExecutionContext{
		Bindings: []ContextSlotBinding{{
			ContractID:        testAccessSlot,
			ContractVersion:   "0.1.0",
			BindingReceiptRef: "auth-receipt:current",
			Value:             []byte(value),
		}},
		ReusePartitionRef: "visibility-partition:current",
	}
}

func TestExecutionContextIsSeparateFromSemanticRequest(t *testing.T) {
	request := []byte(`{"query":"item-42","grantedAccessRefs":["attacker"]}`)
	host := accessHost("trusted-access")

	var observed []byte
	execution, err := invokeWithExecutionContext(
		context.Background(),
		accessSlotContracts(),
		accessRequirement(),
		host,
		request,
		func(_ context.Context, slots ContextSlotView, got []byte) ([]byte, error) {
			value, ok := slots.Lookup(testAccessSlot)
			if !ok {
				t.Fatal("declared access slot was not visible")
			}
			observed = value
			if string(got) != string(request) {
				t.Fatal("semantic request changed while passing context gate")
			}
			return []byte(`{"status":"ok"}`), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if string(observed) != "trusted-access" {
		t.Fatalf("semantic request overrode Host binding: %q", observed)
	}
	if execution.Receipt.ReusePartitionRef != "visibility-partition:current" {
		t.Fatalf("Host reuse partition was lost: %+v", execution.Receipt)
	}
}

func TestMissingRequiredSlotFailsBeforeInvocation(t *testing.T) {
	invoked := false
	_, err := invokeWithExecutionContext(
		context.Background(),
		accessSlotContracts(),
		accessRequirement(),
		HostExecutionContext{ReusePartitionRef: "partition:none"},
		[]byte("request"),
		func(context.Context, ContextSlotView, []byte) ([]byte, error) {
			invoked = true
			return []byte("result"), nil
		},
	)
	if err == nil || !strings.Contains(err.Error(), "is missing") {
		t.Fatalf("missing context slot was accepted: %v", err)
	}
	if invoked {
		t.Fatal("downstream invoker ran despite missing context")
	}
}

func TestWrongSlotVersionFailsClosed(t *testing.T) {
	host := accessHost("trusted-access")
	host.Bindings[0].ContractVersion = "0.2.0"

	_, err := invokeWithExecutionContext(
		context.Background(),
		accessSlotContracts(),
		accessRequirement(),
		host,
		[]byte("request"),
		func(context.Context, ContextSlotView, []byte) ([]byte, error) {
			return []byte("result"), nil
		},
	)
	if err == nil || !strings.Contains(err.Error(), "version mismatch") {
		t.Fatalf("wrong context slot version was accepted: %v", err)
	}
}

func TestUndeclaredHostSlotIsNotVisibleToProvider(t *testing.T) {
	host := accessHost("trusted-access")
	host.Bindings = append(host.Bindings, ContextSlotBinding{
		ContractID:        testTraceSlot,
		ContractVersion:   "0.1.0",
		BindingReceiptRef: "trace-receipt:internal",
		Value:             []byte("internal-trace"),
	})

	_, err := invokeWithExecutionContext(
		context.Background(),
		accessSlotContracts(),
		accessRequirement(),
		host,
		[]byte("request"),
		func(_ context.Context, slots ContextSlotView, _ []byte) ([]byte, error) {
			if _, ok := slots.Lookup(testTraceSlot); ok {
				t.Fatal("undeclared Host slot was visible to provider")
			}
			return []byte("result"), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestControlContextHidesAmbientValuesButPreservesDeadline(t *testing.T) {
	type privateKey struct{}
	parent := context.WithValue(context.Background(), privateKey{}, "ambient-secret")
	deadline := time.Now().Add(time.Minute)
	parent, cancel := context.WithDeadline(parent, deadline)
	defer cancel()

	_, err := invokeWithExecutionContext(
		parent,
		accessSlotContracts(),
		accessRequirement(),
		accessHost("trusted-access"),
		[]byte("request"),
		func(ctx context.Context, _ ContextSlotView, _ []byte) ([]byte, error) {
			if got := ctx.Value(privateKey{}); got != nil {
				t.Fatalf("ambient context value leaked to provider: %v", got)
			}
			gotDeadline, ok := ctx.Deadline()
			if !ok || !gotDeadline.Equal(deadline) {
				t.Fatalf("control deadline was not preserved: got=%v ok=%v", gotDeadline, ok)
			}
			return []byte("result"), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestControlContextPreservesCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := invokeWithExecutionContext(
		parent,
		accessSlotContracts(),
		accessRequirement(),
		accessHost("trusted-access"),
		[]byte("request"),
		func(ctx context.Context, _ ContextSlotView, _ []byte) ([]byte, error) {
			if ctx.Err() != context.Canceled {
				t.Fatalf("control cancellation was not preserved: %v", ctx.Err())
			}
			return []byte("result"), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBindingReceiptOmitsRawExecutionValue(t *testing.T) {
	execution, err := invokeWithExecutionContext(
		context.Background(),
		accessSlotContracts(),
		accessRequirement(),
		accessHost("secret-access-value"),
		[]byte("request"),
		func(context.Context, ContextSlotView, []byte) ([]byte, error) {
			return []byte(`{"status":"ok"}`), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(execution.Receipt)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "secret-access-value") {
		t.Fatalf("receipt exposed raw execution context: %s", encoded)
	}
	if !strings.Contains(string(encoded), "auth-receipt:current") {
		t.Fatalf("receipt lost Host binding reference: %s", encoded)
	}
}

func TestDirectConfidentialReflectionIsRejected(t *testing.T) {
	_, err := invokeWithExecutionContext(
		context.Background(),
		accessSlotContracts(),
		accessRequirement(),
		accessHost("secret-access-value"),
		[]byte("request"),
		func(_ context.Context, slots ContextSlotView, _ []byte) ([]byte, error) {
			value, _ := slots.Lookup(testAccessSlot)
			return append([]byte("echo:"), value...), nil
		},
	)
	if err == nil || !strings.Contains(err.Error(), "directly reflected") {
		t.Fatalf("direct execution-context reflection was accepted: %v", err)
	}
}

func TestDirectReflectionGuardDoesNotClaimGeneralNonDisclosure(t *testing.T) {
	execution, err := invokeWithExecutionContext(
		context.Background(),
		accessSlotContracts(),
		accessRequirement(),
		accessHost("secret-access-value"),
		[]byte("request"),
		func(context.Context, ContextSlotView, []byte) ([]byte, error) {
			return []byte("derived-value"), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, limit := range execution.Receipt.AuthorityLimits {
		if limit == "direct-reflection-check-does-not-prove-general-non-disclosure" {
			found = true
		}
	}
	if !found {
		t.Fatal("direct-reflection guard overstated its confidentiality guarantee")
	}
}

func TestExecutionContextRequiresHostReusePartition(t *testing.T) {
	host := accessHost("trusted-access")
	host.ReusePartitionRef = ""

	_, err := invokeWithExecutionContext(
		context.Background(),
		accessSlotContracts(),
		accessRequirement(),
		host,
		[]byte("request"),
		func(context.Context, ContextSlotView, []byte) ([]byte, error) {
			return []byte("result"), nil
		},
	)
	if err == nil || !strings.Contains(err.Error(), "reuse partition") {
		t.Fatalf("execution context without reuse partition was accepted: %v", err)
	}
}

func TestSameSemanticRequestKeepsDifferentHostReusePartitions(t *testing.T) {
	run := func(partition string) InvocationContextExecution {
		host := accessHost("trusted-access")
		host.ReusePartitionRef = partition
		execution, err := invokeWithExecutionContext(
			context.Background(),
			accessSlotContracts(),
			accessRequirement(),
			host,
			[]byte("same-request"),
			func(context.Context, ContextSlotView, []byte) ([]byte, error) {
				return []byte("same-result"), nil
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		return execution
	}

	first := run("partition:a")
	second := run("partition:b")
	if first.Receipt.SemanticRequestDigest != second.Receipt.SemanticRequestDigest {
		t.Fatal("same semantic request changed identity")
	}
	if first.Receipt.ReusePartitionRef == second.Receipt.ReusePartitionRef {
		t.Fatal("different Host reuse partitions were collapsed")
	}
}

func TestNoRequiredExecutionContextNeedsNoReusePartition(t *testing.T) {
	execution, err := invokeWithExecutionContext(
		context.Background(),
		nil,
		nil,
		HostExecutionContext{},
		[]byte("request"),
		func(_ context.Context, slots ContextSlotView, _ []byte) ([]byte, error) {
			if _, ok := slots.Lookup(testAccessSlot); ok {
				t.Fatal("unexpected slot was visible")
			}
			return []byte("result"), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if execution.Receipt.ReusePartitionRef != "" {
		t.Fatalf("context-free invocation gained reuse partition: %+v", execution.Receipt)
	}
}

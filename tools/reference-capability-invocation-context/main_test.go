package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

const (
	testAccessSlot    = "tei.reference.context-slot.authorization-access/v0"
	testPrincipalSlot = "tei.reference.context-slot.verified-principal/v0"
	testTraceSlot     = "tei.reference.context-slot.trace/v0"
)

func accessSlotContracts() []ContextSlotContract {
	return []ContextSlotContract{
		{
			ID:                           testAccessSlot,
			Version:                      "0.1.0",
			ForbidDirectResultReflection: true,
		},
		{
			ID:                           testPrincipalSlot,
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

func TestVerifiedPrincipalSecondConsumerUsesSameContextGate(t *testing.T) {
	requirements := []ContextSlotRequirement{{
		ContractID:      testPrincipalSlot,
		ContractVersion: "0.1.0",
	}}
	host := HostExecutionContext{
		Bindings: []ContextSlotBinding{{
			ContractID:        testPrincipalSlot,
			ContractVersion:   "0.1.0",
			BindingReceiptRef: "principal-receipt:current",
			Value:             []byte("principal:alice"),
		}},
		ReusePartitionRef: "principal-partition:alice",
	}
	request := []byte(`{"action":"order.read","resource":"order:42"}`)

	_, err := invokeWithExecutionContext(
		context.Background(),
		accessSlotContracts(),
		requirements,
		host,
		request,
		func(_ context.Context, slots ContextSlotView, got []byte) ([]byte, error) {
			principal, ok := slots.Lookup(testPrincipalSlot)
			if !ok || string(principal) != "principal:alice" {
				t.Fatalf("verified Principal slot was not delivered: %q ok=%v", principal, ok)
			}
			if string(got) != string(request) {
				t.Fatal("Web semantic request changed while passing context gate")
			}
			return []byte(`{"status":"authorized-path-reached"}`), nil
		},
	)
	if err != nil {
		t.Fatal(err)
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

func TestMalformedUndeclaredHostSlotDoesNotAffectInvocation(t *testing.T) {
	host := accessHost("trusted-access")
	host.Bindings = append(host.Bindings, ContextSlotBinding{
		ContractID: testTraceSlot,
		// The undeclared slot is intentionally malformed. It must not affect
		// an invocation that does not require it.
	})

	_, err := invokeWithExecutionContext(
		context.Background(),
		accessSlotContracts(),
		accessRequirement(),
		host,
		[]byte("request"),
		func(_ context.Context, slots ContextSlotView, _ []byte) ([]byte, error) {
			if _, ok := slots.Lookup(testTraceSlot); ok {
				t.Fatal("malformed undeclared Host slot was visible to provider")
			}
			return []byte("result"), nil
		},
	)
	if err != nil {
		t.Fatalf("unrelated Host slot affected invocation: %v", err)
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
	defer cancel()
	invoked := false

	_, err := invokeWithExecutionContext(
		parent,
		accessSlotContracts(),
		accessRequirement(),
		accessHost("trusted-access"),
		[]byte("request"),
		func(ctx context.Context, _ ContextSlotView, _ []byte) ([]byte, error) {
			invoked = true
			if ctx.Err() != nil {
				t.Fatalf("control context was canceled before dispatch: %v", ctx.Err())
			}
			cancel()
			<-ctx.Done()
			if ctx.Err() != context.Canceled {
				t.Fatalf("control cancellation was not preserved: %v", ctx.Err())
			}
			return []byte("result"), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !invoked {
		t.Fatal("downstream was not invoked with an active control context")
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

func TestDirectConfidentialErrorReflectionIsRejected(t *testing.T) {
	_, err := invokeWithExecutionContext(
		context.Background(),
		accessSlotContracts(),
		accessRequirement(),
		accessHost("secret-access-value"),
		[]byte("request"),
		func(_ context.Context, slots ContextSlotView, _ []byte) ([]byte, error) {
			value, _ := slots.Lookup(testAccessSlot)
			return nil, fmt.Errorf("provider failed for %s", value)
		},
	)
	if err == nil || !strings.Contains(err.Error(), "directly reflected into downstream error") {
		t.Fatalf("direct execution-context error reflection was accepted: %v", err)
	}
	if strings.Contains(err.Error(), "secret-access-value") {
		t.Fatalf("reflection rejection re-exposed confidential value: %v", err)
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

func TestContextGateDoesNotClaimDownstreamEffectSafety(t *testing.T) {
	execution, err := invokeWithExecutionContext(
		context.Background(),
		accessSlotContracts(),
		accessRequirement(),
		accessHost("trusted-access"),
		[]byte("request"),
		func(context.Context, ContextSlotView, []byte) ([]byte, error) {
			return []byte("result"), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, limit := range execution.Receipt.AuthorityLimits {
		if limit == "context-gate-does-not-prove-downstream-invocation-is-read-only" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("context gate overstated downstream invocation-effect safety")
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

func TestSemanticRequestDigestUsesDispatchSnapshot(t *testing.T) {
	request := []byte("original-request")
	sum := sha256.Sum256(request)
	expected := "sha256:" + hex.EncodeToString(sum[:])

	execution, err := invokeWithExecutionContext(
		context.Background(),
		accessSlotContracts(),
		accessRequirement(),
		accessHost("trusted-access"),
		request,
		func(_ context.Context, _ ContextSlotView, got []byte) ([]byte, error) {
			if string(got) != "original-request" {
				t.Fatalf("downstream did not receive the original request: %q", got)
			}
			// Mutation is intentionally sequenced during the callback, not
			// concurrent with snapshot creation.
			request[0] = 'X'
			return []byte("result"), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if execution.Receipt.SemanticRequestDigest != expected {
		t.Fatalf("request receipt changed after caller mutation: got=%s want=%s",
			execution.Receipt.SemanticRequestDigest, expected)
	}
}

func TestAlreadyCanceledControlContextSkipsDownstream(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	invoked := false
	_, err := invokeWithExecutionContext(
		parent,
		accessSlotContracts(),
		accessRequirement(),
		accessHost("trusted-access"),
		[]byte("request"),
		func(context.Context, ContextSlotView, []byte) ([]byte, error) {
			invoked = true
			return []byte("result"), nil
		},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled context was not rejected before dispatch: %v", err)
	}
	if invoked {
		t.Fatal("downstream invoker ran after context cancellation")
	}
}

func TestContextSlotLookupReturnsIndependentBytes(t *testing.T) {
	_, err := invokeWithExecutionContext(
		context.Background(),
		accessSlotContracts(),
		accessRequirement(),
		accessHost("trusted-access"),
		[]byte("request"),
		func(_ context.Context, slots ContextSlotView, _ []byte) ([]byte, error) {
			first, ok := slots.Lookup(testAccessSlot)
			if !ok {
				t.Fatal("required slot missing")
			}
			first[0] = 'X'
			second, ok := slots.Lookup(testAccessSlot)
			if !ok || string(second) != "trusted-access" {
				t.Fatalf("slot value mutated through lookup: %q ok=%v", second, ok)
			}
			return []byte("result"), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestInvalidTrustedContextDeclarationFailsBeforeDispatch(t *testing.T) {
	tests := []struct {
		name         string
		contracts    []ContextSlotContract
		requirements []ContextSlotRequirement
		host         HostExecutionContext
		errorText    string
	}{
		{
			name:         "duplicate required declaration",
			contracts:    accessSlotContracts(),
			requirements: append(accessRequirement(), accessRequirement()...),
			host:         accessHost("trusted-access"),
			errorText:    "duplicated",
		},
		{
			name:         "required contract absent",
			contracts:    nil,
			requirements: accessRequirement(),
			host:         accessHost("trusted-access"),
			errorText:    "has no contract",
		},
		{
			name:         "required contract version not accepted",
			contracts:    accessSlotContracts(),
			requirements: []ContextSlotRequirement{{ContractID: testAccessSlot, ContractVersion: "9.9.9"}},
			host:         accessHost("trusted-access"),
			errorText:    "has no exact contract",
		},
		{
			name:         "duplicate host binding",
			contracts:    accessSlotContracts(),
			requirements: accessRequirement(),
			host: HostExecutionContext{
				Bindings:          append(accessHost("trusted-access").Bindings, accessHost("trusted-access").Bindings...),
				ReusePartitionRef: "partition:dup",
			},
			errorText: "bound more than once",
		},
		{
			name:         "empty required host value",
			contracts:    accessSlotContracts(),
			requirements: accessRequirement(),
			host: HostExecutionContext{
				Bindings: []ContextSlotBinding{{
					ContractID: testAccessSlot, ContractVersion: "0.1.0",
					BindingReceiptRef: "auth-receipt:empty",
				}},
				ReusePartitionRef: "partition:empty",
			},
			errorText: "requires version, receipt, and value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			_, err := invokeWithExecutionContext(
				context.Background(), tt.contracts, tt.requirements, tt.host,
				[]byte("request"),
				func(context.Context, ContextSlotView, []byte) ([]byte, error) {
					called = true
					return []byte("result"), nil
				},
			)
			if err == nil || !strings.Contains(err.Error(), tt.errorText) {
				t.Fatalf("invalid trusted context declaration was accepted: %v", err)
			}
			if called {
				t.Fatal("downstream ran despite invalid trusted context declaration")
			}
		})
	}
}

func TestHostBindingIsSnapshotBeforeDispatch(t *testing.T) {
	host := accessHost("trusted-access")
	_, err := invokeWithExecutionContext(
		context.Background(),
		accessSlotContracts(),
		accessRequirement(),
		host,
		[]byte("request"),
		func(_ context.Context, slots ContextSlotView, _ []byte) ([]byte, error) {
			host.Bindings[0].Value[0] = 'X'
			value, ok := slots.Lookup(testAccessSlot)
			if !ok || string(value) != "trusted-access" {
				t.Fatalf("Host buffer mutation changed the bound slot: %q ok=%v", value, ok)
			}
			return []byte("result"), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDownstreamRequestMutationDoesNotChangeReceiptIdentity(t *testing.T) {
	request := []byte("original-request")
	sum := sha256.Sum256(request)
	expected := "sha256:" + hex.EncodeToString(sum[:])
	execution, err := invokeWithExecutionContext(
		context.Background(), accessSlotContracts(), accessRequirement(),
		accessHost("trusted-access"), request,
		func(_ context.Context, _ ContextSlotView, received []byte) ([]byte, error) {
			received[0] = 'X'
			return []byte("result"), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if execution.Receipt.SemanticRequestDigest != expected || string(request) != "original-request" {
		t.Fatalf("downstream mutation changed request identity or caller buffer: digest=%s request=%q",
			execution.Receipt.SemanticRequestDigest, request)
	}
}

func TestConfidentialDownstreamErrorDropsTypedCause(t *testing.T) {
	_, err := invokeWithExecutionContext(
		context.Background(), accessSlotContracts(), accessRequirement(),
		accessHost("top-secret-value"), []byte("request"),
		func(_ context.Context, slots ContextSlotView, _ []byte) ([]byte, error) {
			value, _ := slots.Lookup(testAccessSlot)
			return nil, &hiddenCredentialError{credential: string(value)}
		},
	)
	if err == nil {
		t.Fatal("downstream error disappeared")
	}
	var secret *hiddenCredentialError
	if errors.As(err, &secret) {
		t.Fatal("typed downstream error leaked confidential context through errors.As")
	}
	if strings.Contains(fmt.Sprintf("%+v", err), "top-secret-value") {
		t.Fatal("confidential context leaked through error formatting")
	}
	if !strings.Contains(err.Error(), "confidential error detail withheld") {
		t.Fatalf("opaque failure classification was lost: %v", err)
	}
}

type hiddenCredentialError struct{ credential string }

func (e *hiddenCredentialError) Error() string { return "temporary provider failure" }

func TestNonConfidentialDownstreamErrorPreservesType(t *testing.T) {
	original := &hiddenCredentialError{credential: "diagnostic-only"}
	_, err := invokeWithExecutionContext(
		context.Background(), accessSlotContracts(),
		[]ContextSlotRequirement{{ContractID: testTraceSlot, ContractVersion: "0.1.0"}},
		HostExecutionContext{
			Bindings:          []ContextSlotBinding{{ContractID: testTraceSlot, ContractVersion: "0.1.0", BindingReceiptRef: "trace:1", Value: []byte("trace")}},
			ReusePartitionRef: "trace-partition:1",
		},
		[]byte("request"),
		func(context.Context, ContextSlotView, []byte) ([]byte, error) { return nil, original },
	)
	if err != original {
		t.Fatalf("nonconfidential downstream error changed identity: %v", err)
	}
}

func TestConfidentialDownstreamErrorPreservesTrustedControlCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := invokeWithExecutionContext(
		parent, accessSlotContracts(), accessRequirement(),
		accessHost("secret-access"), []byte("request"),
		func(ctx context.Context, _ ContextSlotView, _ []byte) ([]byte, error) {
			cancel()
			<-ctx.Done()
			return nil, fmt.Errorf("provider canceled: %w", ctx.Err())
		},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("trusted control cancellation was erased: %v", err)
	}
}

func TestConfidentialDownstreamErrorPreservesTrustedControlDeadline(t *testing.T) {
	parent, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	_, err := invokeWithExecutionContext(
		parent, accessSlotContracts(), accessRequirement(),
		accessHost("secret-access"), []byte("request"),
		func(ctx context.Context, slots ContextSlotView, _ []byte) ([]byte, error) {
			<-ctx.Done()
			secret, _ := slots.Lookup(testAccessSlot)
			return nil, &hiddenCredentialError{credential: string(secret)}
		},
	)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("trusted control deadline was erased: %v", err)
	}
	var secret *hiddenCredentialError
	if errors.As(err, &secret) {
		t.Fatal("typed downstream error escaped during deadline cancellation")
	}
}

func TestConfidentialDownstreamCannotSpoofControlCancellation(t *testing.T) {
	_, err := invokeWithExecutionContext(
		context.Background(), accessSlotContracts(), accessRequirement(),
		accessHost("secret-access"), []byte("request"),
		func(context.Context, ContextSlotView, []byte) ([]byte, error) {
			return nil, fmt.Errorf("forged cancel: %w", context.Canceled)
		},
	)
	if err == nil || errors.Is(err, context.Canceled) {
		t.Fatalf("untrusted downstream error forged trusted cancellation: %v", err)
	}
	if !strings.Contains(err.Error(), "confidential error detail withheld") {
		t.Fatalf("expected opaque error rather than downstream detail: %v", err)
	}
}

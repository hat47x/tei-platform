package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pruntime "tei/internal/plugin/runtime"
)

func transportDispatchPlanForTest(
	t *testing.T,
	dispatch GenericDispatch,
) TransportDispatchPlan {
	t.Helper()
	plan, err := buildTransportDispatchPlan(genericDispatchIdentity(dispatch))
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func newPlannedHTTPTransportServer(
	t *testing.T,
	plan TransportDispatchPlan,
) *httptest.Server {
	t.Helper()
	handler, err := NewPlannedHTTPTransportHandler(
		plan,
		echoHTTPTransportImplementation,
	)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func TestPlannedProcessTransportRejectsDispatchDriftBeforeLauncher(t *testing.T) {
	dispatch := genericDispatch([]byte("payload"))
	plan := transportDispatchPlanForTest(t, dispatch)
	dispatch.Target = "tampered-target"

	launcher := &countingProcessLauncher{}
	_, err := invokePlannedOverProcessTransport(
		context.Background(),
		launcher,
		pruntimeProcessCommandForPlanTest(),
		plan,
		transportCarrier(),
		"authorized-process-drift",
		dispatch,
	)
	if err == nil || !strings.Contains(err.Error(), "differs from transport dispatch plan") {
		t.Fatalf("dispatch drift was accepted: %v", err)
	}
	if launcher.calls != 0 {
		t.Fatalf("dispatch drift reached process transport: calls=%d", launcher.calls)
	}
}

func TestPlannedHTTPTransportRejectsDispatchDriftBeforeNetwork(t *testing.T) {
	dispatch := genericDispatch([]byte("payload"))
	plan := transportDispatchPlanForTest(t, dispatch)
	dispatch.Provider.ImplementationID = "tampered.implementation"

	roundTripper := &countingRoundTripper{}
	client := &http.Client{Transport: roundTripper}
	_, err := invokePlannedOverHTTPTransport(
		context.Background(),
		client,
		"http://example.invalid/capability",
		plan,
		"authorized-http-drift",
		dispatch,
	)
	if err == nil || !strings.Contains(err.Error(), "differs from transport dispatch plan") {
		t.Fatalf("dispatch drift was accepted: %v", err)
	}
	if roundTripper.calls != 0 {
		t.Fatalf("dispatch drift reached HTTP transport: calls=%d", roundTripper.calls)
	}
}

func TestPlannedProcessAndHTTPReceiptsBindSameDispatchPlan(t *testing.T) {
	dispatch := genericDispatch([]byte{0x00, 0xfe, 0xff, 'P', 'L', 'A', 'N'})
	plan := transportDispatchPlanForTest(t, dispatch)

	processLauncher, processCommand := helperLauncher("echo")
	processExecution, err := invokePlannedOverProcessTransport(
		context.Background(),
		processLauncher,
		processCommand,
		plan,
		transportCarrier(),
		"authorized-cross-process",
		dispatch,
	)
	if err != nil {
		t.Fatal(err)
	}

	server := newPlannedHTTPTransportServer(t, plan)
	httpExecution, err := invokePlannedOverHTTPTransport(
		context.Background(),
		server.Client(),
		server.URL,
		plan,
		"authorized-cross-http",
		dispatch,
	)
	if err != nil {
		t.Fatal(err)
	}

	if processExecution.Receipt.DispatchPlanFingerprint != plan.PlanFingerprint ||
		httpExecution.Receipt.DispatchPlanFingerprint != plan.PlanFingerprint {
		t.Fatalf(
			"dispatch plan identity was not preserved: process=%q http=%q plan=%q",
			processExecution.Receipt.DispatchPlanFingerprint,
			httpExecution.Receipt.DispatchPlanFingerprint,
			plan.PlanFingerprint,
		)
	}
	if processExecution.Receipt.InvocationIdentity != plan.InvocationIdentity ||
		httpExecution.Receipt.InvocationIdentity != plan.InvocationIdentity {
		t.Fatalf(
			"receipt identity differs from plan: process=%+v http=%+v plan=%+v",
			processExecution.Receipt.InvocationIdentity,
			httpExecution.Receipt.InvocationIdentity,
			plan.InvocationIdentity,
		)
	}
	if processExecution.Receipt.Transport == httpExecution.Receipt.Transport {
		t.Fatalf(
			"physical transport identity collapsed: process=%q http=%q",
			processExecution.Receipt.Transport,
			httpExecution.Receipt.Transport,
		)
	}
}

func TestTransportDispatchPlanRejectsIdentityTamperWithReusedFingerprint(t *testing.T) {
	dispatch := genericDispatch([]byte("payload"))
	plan := transportDispatchPlanForTest(t, dispatch)
	plan.InvocationIdentity.RequestShapeFingerprint =
		"sha256:" + strings.Repeat("a", 64)

	if err := validateTransportDispatchPlan(plan); err == nil ||
		!strings.Contains(err.Error(), "fingerprint is stale") {
		t.Fatalf("tampered dispatch plan was accepted: %v", err)
	}
}

func TestPlannedTransportReceiptDoesNotPromoteDispatchPlanToAuthority(t *testing.T) {
	dispatch := genericDispatch([]byte("payload"))
	plan := transportDispatchPlanForTest(t, dispatch)
	server := newPlannedHTTPTransportServer(t, plan)

	execution, err := invokePlannedOverHTTPTransport(
		context.Background(),
		server.Client(),
		server.URL,
		plan,
		"authorized-plan-authority-limit",
		dispatch,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"dispatch-plan-is-not-capability-conformance",
		"dispatch-plan-is-not-runtime-authority",
		"dispatch-plan-is-not-canonical-acceptance",
	} {
		found := false
		for _, got := range execution.Receipt.AuthorityLimits {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("dispatch plan authority limit %q missing: %+v", want, execution.Receipt.AuthorityLimits)
		}
	}
}

func pruntimeProcessCommandForPlanTest() pruntime.ProcessCommand {
	return pruntime.ProcessCommand{Executable: "must-not-run"}
}

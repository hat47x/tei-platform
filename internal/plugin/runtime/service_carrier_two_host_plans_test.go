package runtime_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	pruntime "tei/internal/plugin/runtime"
)

// Two Host-owned EnvironmentPlans can bind the same logical Service to one
// Provider and test ProcessLauncher, while independently admitting that route.
// The labels stand for two Application configurations; InvokeSelectedServiceCarrier
// itself does NOT authenticate an Application identity or choose its Host plan.
func TestSelectedServiceSharedCarrierHonorsIndependentHostPlans(t *testing.T) {
	planA, selected, child, verifier := serviceCarrierFixture(t)
	host, acceptance := serviceCarrierHost(selected, child, verifier)
	planB := planA
	// Avoid accidental aliasing of the per-Host binding declarations.
	planA.ServiceBindings = append([]pruntime.ServiceCarrierBinding(nil), planA.ServiceBindings...)
	planB.ServiceBindings = append([]pruntime.ServiceCarrierBinding(nil), planB.ServiceBindings...)

	// Application B has not been granted a route by its Host-selected plan,
	// although the Provider is installed and Application A has a valid route.
	planB.ServiceBindings = nil
	planB.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(planB)
	payload := json.RawMessage(`{"value":"synthetic-customer-reference"}`)
	invoke := func(plan pruntime.EnvironmentPlan, requestID string) error {
		_, err := host.InvokeSelectedServiceCarrier(
			context.Background(), plan, selected, requestID, payload,
		)
		return err
	}
	if err := invoke(planB, "B-without-binding"); err == nil || !strings.Contains(err.Error(), "no Host binding") {
		t.Fatalf("Plan B borrowed another Application's Host binding: %v", err)
	}
	if child.calls != 0 || verifier.calls != 0 || acceptance.calls != 0 {
		t.Fatal("unbound Plan B reached Host acceptance, verification or process")
	}

	if err := invoke(planA, "A-bound"); err != nil {
		t.Fatalf("bound Plan A could not use its Service: %v", err)
	}
	if child.calls != 1 {
		t.Fatalf("Plan A did not reach the shared process: %d", child.calls)
	}

	// A separate Host decision can bind the same Service to B without
	// duplicating the logical Provider or registered ProcessLauncher.
	planB.ServiceBindings = append(planB.ServiceBindings, planA.ServiceBindings...)
	planB.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(planB)
	if err := invoke(planB, "B-bound"); err != nil {
		t.Fatalf("Plan B could not use the shared Service after binding: %v", err)
	}
	if child.calls != 2 || verifier.calls != 2 || acceptance.calls != 2 {
		t.Fatalf("two Host-selected Plans did not share the same admitted runtime: process=%d verifier=%d accepted=%d",
			child.calls, verifier.calls, acceptance.calls)
	}

	// Withdrawing A's Host binding must not invalidate B's independent
	// binding, even though both use the same Provider and launcher test object.
	planA.ServiceBindings = nil
	planA.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(planA)
	if err := invoke(planA, "A-withdrawn"); err == nil || !strings.Contains(err.Error(), "no Host binding") {
		t.Fatalf("withdrawn Plan A remained callable: %v", err)
	}
	if err := invoke(planB, "B-after-A-withdrawn"); err != nil {
		t.Fatalf("Plan B lost a separately admitted Service: %v", err)
	}
	if child.calls != 3 || verifier.calls != 3 || acceptance.calls != 3 {
		t.Fatalf("Host binding withdrawal leaked into the shared process: process=%d verifier=%d accepted=%d",
			child.calls, verifier.calls, acceptance.calls)
	}

	// Service/Provider acceptance is a DIFFERENT Host-level decision:
	// revoking it must reject both Plans, not only the first consumer.
	planA.ServiceBindings = append([]pruntime.ServiceCarrierBinding(nil), planB.ServiceBindings...)
	planA.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(planA)
	acceptance.denied = true
	for _, tc := range []struct {
		name string
		plan pruntime.EnvironmentPlan
	}{
		{"A-shared-provider-revoked", planA},
		{"B-shared-provider-revoked", planB},
	} {
		if err := invoke(tc.plan, tc.name); !errors.Is(err, pruntime.ErrServiceAcceptanceDenied) {
			t.Fatalf("%s bypassed current Service acceptance: %v", tc.name, err)
		}
	}
	if child.calls != 3 || verifier.calls != 3 || acceptance.calls != 5 {
		t.Fatalf("revoked shared Service was executed or current acceptance was not rechecked: process=%d verifier=%d accepted=%d",
			child.calls, verifier.calls, acceptance.calls)
	}
}

package driver_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/core"
	"tei/internal/operations"
	"tei/internal/realization"
	"tei/internal/semantic/capabilityinvoke"
)

// TestActionSurfaceToBoundRequestWrongTarget is a test-only characterization.
// It carries an ACTUAL derived Surface binding into the existing Driver, then
// introduces a malicious/broken adapter whose effect subject differs from the
// caller's selected subject.  This does NOT assert that the current Driver
// already guarantees business-target equality: that acceptance rule belongs to
// the caller's contract and the real Provider/Host (issue #2598/#1174).
func TestActionSurfaceToBoundRequestWrongTarget(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine fixture location")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	fixture := filepath.Join(root, "reference", "standard-crud", "workspace")

	surface, err := operations.SurfaceWorkspace(fixture, "ledger")
	if err != nil {
		t.Fatalf("derive actual surface: %v", err)
	}
	var binding string
	for _, operation := range surface.Operations {
		if operation.ID == "ledger.update.sales.order" {
			if operation.Kind != "update" || operation.Target != "sales.order" {
				t.Fatalf("derived operation changed target: %+v", operation)
			}
			if operation.Concurrency != "expected-state" {
				t.Fatalf("expected-state qualifier lost: %+v", operation)
			}
			binding = operation.Binding
		}
	}
	if binding != "sales.order.cancel" {
		t.Fatalf("derived operation does not resolve to the bound capability: %q", binding)
	}

	resolver, err := standardCRDWorld(t)
	if err != nil {
		t.Fatal(err)
	}

	// Synthetic witness with an explicit, independently fixed subject.
	// Actual Host/DB and concurrency/authorization were NOT exercised.
	selected := "O-1"
	wrongEffectTarget := "O-2"
	store := map[string]string{}
	receivedInput := ""
	execution, err := resolver.InvokeCapability(
		context.Background(), "ledger", core.CanonicalID(binding),
		bindingAdapterResolverFunc(func(_ context.Context, resolved realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
			if resolved.Identity() != "ledger/sales.order.cancel" {
				t.Fatalf("wrong actual bound realization: %s", resolved.Identity())
			}
			return capabilityAdapterFunc(func(_ context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
				if capability != core.CanonicalID(binding) {
					t.Fatalf("wrong capability passed to adapter: %q", capability)
				}
				receivedInput, _ = input.(map[string]any)["orderId"].(string)
				// Deliberately wrong. The fake adapter commits to the OTHER
				// subject even though its response claims success.
				store[wrongEffectTarget] = "cancelled"
				return capabilityinvoke.BusinessResult{
					Capability: core.CanonicalID(binding),
					Mode:       capabilityinvoke.ResultOutcome,
					Outcome:    "cancelled",
					Payload: map[string]any{
						"orderId": wrongEffectTarget,
						"status":  "cancelled",
					},
				}, nil
			}), nil
		}),
		map[string]any{"orderId": selected, "expectedStatus": "new"},
	)
	if err != nil {
		t.Fatalf("unexpected Driver technical error: %v", err)
	}

	// The Driver correctly resolves/invokes this Capability, but that alone
	// is NOT sufficient evidence that the selected subject was affected.
	if receivedInput != selected {
		t.Fatalf("input target changed before adapter: %q", receivedInput)
	}
	if execution.Result.Outcome != "cancelled" {
		t.Fatalf("synthetic adapter outcome unexpectedly changed: %+v", execution.Result)
	}
	returnedTarget, _ := execution.Result.Payload.(map[string]any)["orderId"].(string)
	if returnedTarget != wrongEffectTarget {
		t.Fatalf("fixture must reveal mismatched response target: %q", returnedTarget)
	}
	if _, changed := store[selected]; changed {
		t.Fatalf("incorrectly attributed effect to selected subject: %+v", store)
	}
	if store[wrongEffectTarget] != "cancelled" {
		t.Fatalf("negative fixture did not affect wrong target: %+v", store)
	}

	// Required business Claim is FALSE under the independently fixed selected
	// subject. Do not promote a successful adapter invocation or outcome to
	// BusinessVerified for the originally selected order.
	businessVerified := returnedTarget == selected && store[selected] == "cancelled"
	if businessVerified {
		t.Fatal("wrong-subject execution must never count as BusinessVerified")
	}
}

package driver_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/operations"
	"tei/internal/realization"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/workspace"
)

// This test connects the derived operation surface to the bound write path for
// #2564: the surface declares that an update detects concurrency by comparing
// the expected state, and the bound capability returns a conflict as a business
// result. It does not add a generic write executor; the capability still does
// the comparison.
func TestDerivedUpdateConcurrencyDrivesTheBoundConflictOutcome(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	workspaceRoot := filepath.Join(root, "reference", "standard-crud", "workspace")
	loaded, err := workspace.Load(workspaceRoot)
	if err != nil {
		t.Fatalf("load standard-crud: %v", err)
	}

	surface, err := operations.SurfaceWorkspace(workspaceRoot, "ledger")
	if err != nil {
		t.Fatalf("derive surface: %v", err)
	}
	var update operations.Operation
	found := false
	for _, operation := range surface.Operations {
		if operation.ID == "ledger.update.sales.order" {
			update = operation
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("derived update operation is missing: %+v", surface.Operations)
	}
	if update.Concurrency != "expected-state" {
		t.Fatalf("update concurrency=%q, want expected-state", update.Concurrency)
	}
	if update.Binding == "" {
		t.Fatalf("update has no execution binding, so the conflict outcome cannot be exercised: %+v", update)
	}

	resolver := driver.NewResolver(loaded.World)
	store := map[string]string{"status": "cancelled"}
	resolve := bindingAdapterResolverFunc(func(_ context.Context, _ realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
		return capabilityAdapterFunc(func(_ context.Context, _ core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
			values, _ := input.(map[string]any)
			expected, _ := values["expectedStatus"].(string)
			if expected != store["status"] {
				return capabilityinvoke.BusinessResult{
					Capability: core.CanonicalID(update.Binding),
					Mode:       capabilityinvoke.ResultOutcome,
					Outcome:    "conflict",
					Payload:    map[string]any{"orderId": values["orderId"], "expectedStatus": expected, "actualStatus": store["status"]},
				}, nil
			}
			store["status"] = "cancelled"
			return capabilityinvoke.BusinessResult{
				Capability: core.CanonicalID(update.Binding),
				Mode:       capabilityinvoke.ResultOutcome,
				Outcome:    "cancelled",
				Payload:    map[string]any{"orderId": values["orderId"], "status": store["status"]},
			}, nil
		}), nil
	})

	conflict, err := resolver.InvokeCapability(
		context.Background(), "ledger", core.CanonicalID(update.Binding), resolve,
		map[string]any{"orderId": "O-1", "expectedStatus": "new"},
	)
	if err != nil {
		t.Fatalf("a stale expected state must not be an execution error: %v", err)
	}
	if conflict.Result.Mode != capabilityinvoke.ResultOutcome || conflict.Result.Outcome != "conflict" {
		t.Fatalf("stale expected state was not a business conflict: %+v", conflict.Result)
	}
	if store["status"] != "cancelled" {
		t.Fatalf("rejected update changed the store: %v", store)
	}

	applied, err := resolver.InvokeCapability(
		context.Background(), "ledger", core.CanonicalID(update.Binding), resolve,
		map[string]any{"orderId": "O-1", "expectedStatus": "cancelled"},
	)
	if err != nil {
		t.Fatalf("matching expected state failed: %v", err)
	}
	if applied.Result.Outcome != "cancelled" {
		t.Fatalf("matching expected state was not applied: %+v", applied.Result)
	}
}

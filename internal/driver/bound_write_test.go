package driver_test

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/realization"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/workspace"
)

// standardCRDWorld loads the negative fixture, whose ledger application binds
// sales.order.cancel (effect update sales.order) to a custom Go realization.
func standardCRDWorld(t *testing.T) (driver.Resolver, error) {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test fileの位置を取得できません")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	loaded, err := workspace.Load(filepath.Join(root, "reference", "standard-crud", "workspace"))
	if err != nil {
		t.Fatalf("standard-crudを読み込めません: %v", err)
	}
	return driver.NewResolver(loaded.World), nil
}

func TestInvokeBoundOperationResolvesTheDeclaredRealization(t *testing.T) {
	resolver, err := standardCRDWorld(t)
	if err != nil {
		t.Fatal(err)
	}

	var seen realization.CustomGoBinding
	execution, err := resolver.InvokeCapability(
		context.Background(),
		"ledger",
		"sales.order.cancel",
		bindingAdapterResolverFunc(func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
			seen = binding
			return capabilityAdapterFunc(func(_ context.Context, capability core.CanonicalID, _ any) (capabilityinvoke.BusinessResult, error) {
				return capabilityinvoke.BusinessResult{
					Capability: "sales.order.cancel",
					Mode:       capabilityinvoke.ResultOutcome,
					Outcome:    "cancelled",
					Payload:    map[string]any{"orderId": "O-1", "status": "cancelled"},
				}, nil
			}), nil
		}),
		map[string]any{"orderId": "O-1"},
	)
	if err != nil {
		t.Fatalf("束縛ありの操作を実行できません: %v", err)
	}
	if seen.Identity() != "ledger/sales.order.cancel" {
		t.Fatalf("束縛の解決先が違います: %s", seen.Identity())
	}
	if execution.Result.Mode != capabilityinvoke.ResultOutcome || execution.Result.Outcome != "cancelled" || execution.Result.Payload == nil {
		t.Fatalf("業務結果がそのまま返っていません: %+v", execution.Result)
	}
}

func TestInvokeBoundOperationSurfacesResolutionFailureAsError(t *testing.T) {
	resolver, err := standardCRDWorld(t)
	if err != nil {
		t.Fatal(err)
	}

	_, err = resolver.InvokeCapability(
		context.Background(),
		"ledger",
		"sales.order.cancel",
		bindingAdapterResolverFunc(func(_ context.Context, _ realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
			return nil, errors.New("adapter not available")
		}),
		map[string]any{"orderId": "O-1"},
	)
	if err == nil {
		t.Fatal("アダプタを解決できない場合は実行エラーでなければならない")
	}
	if !strings.Contains(err.Error(), "adapter not available") {
		t.Fatalf("原因が保持されていません: %v", err)
	}
}

// TestInvokeBoundOperationReportsRejectionAsBusinessResult fixes the H1 shape:
// an expected-state mismatch is a business outcome, not an execution error.
func TestInvokeBoundOperationReportsRejectionAsBusinessResult(t *testing.T) {
	resolver, err := standardCRDWorld(t)
	if err != nil {
		t.Fatal(err)
	}

	// Synthetic store: the adapter writes here, so the test can observe that a
	// rejected update left the row unchanged.
	// Another writer already cancelled the order; the caller still expects "new".
	store := map[string]string{"status": "cancelled"}

	execution, err := resolver.InvokeCapability(
		context.Background(),
		"ledger",
		"sales.order.cancel",
		bindingAdapterResolverFunc(func(_ context.Context, _ realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
			return capabilityAdapterFunc(func(_ context.Context, _ core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
				expected, _ := input.(map[string]any)["expectedStatus"].(string)
				if expected != store["status"] {
					return capabilityinvoke.BusinessResult{
						Capability: "sales.order.cancel",
						Mode:       capabilityinvoke.ResultOutcome,
						Outcome:    "conflict",
						Payload:    map[string]any{"orderId": "O-1", "expectedStatus": expected, "actualStatus": store["status"]},
					}, nil
				}
				store["status"] = "cancelled"
				return capabilityinvoke.BusinessResult{
					Capability: "sales.order.cancel",
					Mode:       capabilityinvoke.ResultOutcome,
					Outcome:    "cancelled",
					Payload:    map[string]any{"orderId": "O-1", "status": store["status"]},
				}, nil
			}), nil
		}),
		map[string]any{"orderId": "O-1", "expectedStatus": "new"},
	)
	if err != nil {
		t.Fatalf("拒否は実行エラーではなく業務結果で返らなければならない: %v", err)
	}
	if execution.Result.Outcome != "conflict" || execution.Result.Payload == nil {
		t.Fatalf("拒否の業務結果が失われています: %+v", execution.Result)
	}
	// Independent observation of the synthetic store: the rejected update did
	// not change the row.
	if store["status"] != "cancelled" {
		t.Fatalf("拒否された更新が行を変えています: %s", store["status"])
	}
}

// TestInvokeBoundCreateOperationWritesAndIsObservable fixes the create side of
// the same path: the bound capability runs and the synthetic store changes.
func TestInvokeBoundCreateOperationWritesAndIsObservable(t *testing.T) {
	resolver, err := standardCRDWorld(t)
	if err != nil {
		t.Fatal(err)
	}

	store := map[string]string{}
	execution, err := resolver.InvokeCapability(
		context.Background(),
		"ledger",
		"sales.order.register",
		bindingAdapterResolverFunc(func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
			if binding.Identity() != "ledger/sales.order.register" {
				return nil, errors.New("unexpected binding: " + binding.Identity())
			}
			return capabilityAdapterFunc(func(_ context.Context, _ core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
				orderID, _ := input.(map[string]any)["orderId"].(string)
				if _, exists := store[orderID]; exists {
					return capabilityinvoke.BusinessResult{
						Capability: "sales.order.register",
						Mode:       capabilityinvoke.ResultOutcome,
						Outcome:    "duplicate",
						Payload:    map[string]any{"orderId": orderID},
					}, nil
				}
				store[orderID] = "registered"
				return capabilityinvoke.BusinessResult{
					Capability: "sales.order.register",
					Mode:       capabilityinvoke.ResultOutcome,
					Outcome:    "created",
					Payload:    map[string]any{"orderId": orderID},
				}, nil
			}), nil
		}),
		map[string]any{"orderId": "O-2"},
	)
	if err != nil {
		t.Fatalf("束縛ありの登録を実行できません: %v", err)
	}
	if execution.Result.Outcome != "created" {
		t.Fatalf("登録の業務結果=%+v", execution.Result)
	}
	if store["O-2"] != "registered" {
		t.Fatalf("独立した観測で登録が確認できません: %+v", store)
	}
}

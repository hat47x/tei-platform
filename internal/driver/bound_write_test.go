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
					Mode:       capabilityinvoke.ResultOutput,
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
	if execution.Result.Mode != capabilityinvoke.ResultOutput || execution.Result.Payload == nil {
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

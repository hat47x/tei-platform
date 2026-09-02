package capabilityinvoke_test

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/workspace"
)

type adapterFunc func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error)

func (f adapterFunc) Invoke(ctx context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	return f(ctx, capability, input)
}

func TestReferenceAuthenticationReturnsNamedBusinessOutcomes(t *testing.T) {
	capability := referenceCapability(t, "security.authenticate")
	adapter := adapterFunc(func(_ context.Context, id core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
		switch input {
		case "correct":
			return capabilityinvoke.BusinessResult{
				Capability: id,
				Mode:       capabilityinvoke.ResultOutcome,
				Outcome:    "authenticated",
				Payload:    map[string]string{"subjectId": "subject-001"},
			}, nil
		case "mismatch":
			return capabilityinvoke.BusinessResult{
				Capability: id,
				Mode:       capabilityinvoke.ResultOutcome,
				Outcome:    "denied",
				Payload:    map[string]string{"reason": "credential_mismatch"},
			}, nil
		case "locked":
			return capabilityinvoke.BusinessResult{
				Capability: id,
				Mode:       capabilityinvoke.ResultOutcome,
				Outcome:    "denied",
				Payload:    map[string]string{"reason": "account_locked"},
			}, nil
		default:
			return capabilityinvoke.BusinessResult{}, errors.New("authentication database unavailable")
		}
	})

	for _, tt := range []struct {
		name        string
		input       string
		wantOutcome core.LocalName
	}{
		{name: "認証成功", input: "correct", wantOutcome: "authenticated"},
		{name: "credential不一致", input: "mismatch", wantOutcome: "denied"},
		{name: "account lock", input: "locked", wantOutcome: "denied"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, tt.input)
			if err != nil {
				t.Fatalf("業務OutcomeをExecution Errorとして扱っています: %v", err)
			}
			if result.Mode != capabilityinvoke.ResultOutcome || result.Outcome != tt.wantOutcome {
				t.Fatalf("result=%#v want outcome=%q", result, tt.wantOutcome)
			}
		})
	}
}

func TestReferenceAuthenticationExecutionFailureDoesNotBecomeDenied(t *testing.T) {
	capability := referenceCapability(t, "security.authenticate")
	adapter := adapterFunc(func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error) {
		return capabilityinvoke.BusinessResult{}, errors.New("database connection failed")
	})

	result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, "correct")
	if err == nil {
		t.Fatalf("実行障害がerrorになりませんでした: result=%#v", result)
	}
	if result.Outcome == "denied" {
		t.Fatalf("実行障害がdeniedへ変換されています: result=%#v err=%v", result, err)
	}
}

func TestUndefinedOutcomeIsExecutionContractError(t *testing.T) {
	capability := referenceCapability(t, "security.authenticate")
	adapter := adapterFunc(func(_ context.Context, id core.CanonicalID, _ any) (capabilityinvoke.BusinessResult, error) {
		return capabilityinvoke.BusinessResult{
			Capability: id,
			Mode:       capabilityinvoke.ResultOutcome,
			Outcome:    "unknown",
			Payload:    map[string]string{"reason": "unexpected"},
		}, nil
	})

	_, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, nil)
	if err == nil || !strings.Contains(err.Error(), "定義していません") {
		t.Fatalf("未定義Outcomeを正常なBusiness Resultとして受理しています: %v", err)
	}
}

func TestExistingSingleOutputCapabilityUsesSameInvocationBoundary(t *testing.T) {
	capability := referenceCapability(t, "sales.order.submit")
	adapter := adapterFunc(func(_ context.Context, id core.CanonicalID, _ any) (capabilityinvoke.BusinessResult, error) {
		return capabilityinvoke.BusinessResult{
			Capability: id,
			Mode:       capabilityinvoke.ResultOutput,
			Payload:    map[string]string{"orderId": "order-001"},
		}, nil
	})

	result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, nil)
	if err != nil {
		t.Fatalf("既存output Capabilityを同じInvocation境界で扱えません: %v", err)
	}
	if result.Mode != capabilityinvoke.ResultOutput || result.Outcome != "" {
		t.Fatalf("既存output Capabilityの結果境界が変わっています: %#v", result)
	}
}

func TestOutcomeCannotBeReturnedForSingleOutputCapability(t *testing.T) {
	capability := referenceCapability(t, "sales.order.submit")
	adapter := adapterFunc(func(_ context.Context, id core.CanonicalID, _ any) (capabilityinvoke.BusinessResult, error) {
		return capabilityinvoke.BusinessResult{
			Capability: id,
			Mode:       capabilityinvoke.ResultOutcome,
			Outcome:    "submitted",
			Payload:    map[string]string{"orderId": "order-001"},
		}, nil
	})

	_, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, nil)
	if err == nil {
		t.Fatal("既存output Capabilityへ暗黙のOutcomeを追加できてしまいました")
	}
}

func referenceCapability(t *testing.T, id core.CanonicalID) semantic.Capability {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test fileの位置を取得できません")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}
	for _, namespace := range loaded.World.Semantics {
		for _, capability := range namespace.Capabilities {
			if capability.ID == id {
				return capability
			}
		}
	}
	t.Fatalf("Semantic Capability %sがReference Workspaceに見つかりません", id)
	return semantic.Capability{}
}

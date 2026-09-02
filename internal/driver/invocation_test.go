package driver_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/realization"
	"tei/internal/security/legacyauth"
	"tei/internal/semantic/capabilityinvoke"
)

type bindingAdapterResolverFunc func(context.Context, realization.CustomGoBinding) (capabilityinvoke.Adapter, error)

func (f bindingAdapterResolverFunc) ResolveAdapter(ctx context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
	return f(ctx, binding)
}

type driverCredentialVerifierFunc func(context.Context, string, string) (bool, error)

func (f driverCredentialVerifierFunc) Verify(ctx context.Context, ref, presented string) (bool, error) {
	return f(ctx, ref, presented)
}

type capabilityAdapterFunc func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error)

func (f capabilityAdapterFunc) Invoke(ctx context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	return f(ctx, capability, input)
}

func TestDriverInvokesReferenceAuthenticationThroughResolvedBinding(t *testing.T) {
	world := referenceWorld(t)
	resolver := driver.NewResolver(world)
	store, err := legacyauth.LoadFixtureStore(referenceLegacyAuthFixture(t))
	if err != nil {
		t.Fatalf("legacy authentication fixtureを読み込めません: %v", err)
	}

	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	adapter := legacyauth.Adapter{
		Store: store,
		Verifier: driverCredentialVerifierFunc(func(_ context.Context, ref, presented string) (bool, error) {
			if ref != "credential-ref://legacy/order-editor" {
				return false, fmt.Errorf("予期しないcredential参照です: %s", ref)
			}
			return presented == "runtime-editor-secret", nil
		}),
		Now: func() time.Time { return now },
	}
	adapters := bindingAdapterResolverFunc(func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
		if binding.Identity() != "order_entry/security.authenticate" {
			return nil, fmt.Errorf("未登録のBindingです: %s", binding.Identity())
		}
		return adapter, nil
	})

	t.Run("認証成功をBusiness Resultとして返す", func(t *testing.T) {
		const credential = "runtime-editor-secret"
		invocation, err := resolver.InvokeCapability(
			context.Background(),
			"order_entry",
			"security.authenticate",
			adapters,
			legacyauth.Input{LoginID: "legacy-order-editor", Credential: credential},
		)
		if err != nil {
			t.Fatalf("Authenticationを実行できません: %v", err)
		}
		if invocation.Binding.Binding.Identity() != "order_entry/security.authenticate" {
			t.Fatalf("Binding identity=%q", invocation.Binding.Binding.Identity())
		}
		if invocation.Result.Mode != capabilityinvoke.ResultOutcome || invocation.Result.Outcome != "authenticated" {
			t.Fatalf("Business Result=%#v", invocation.Result)
		}
		payload, ok := invocation.Result.Payload.(legacyauth.AuthenticatedSubject)
		if !ok || payload.SubjectID != "subject-order-editor" {
			t.Fatalf("authenticated payload=%#v", invocation.Result.Payload)
		}
		if strings.Contains(fmt.Sprintf("%#v", invocation), credential) {
			t.Fatalf("Driverの実行結果にcredentialが残っています: %#v", invocation)
		}
	})

	t.Run("credential不一致はExecution Errorではなくdenied Outcomeになる", func(t *testing.T) {
		invocation, err := resolver.InvokeCapability(
			context.Background(),
			"order_entry",
			"security.authenticate",
			adapters,
			legacyauth.Input{LoginID: "legacy-order-editor", Credential: "wrong-secret"},
		)
		if err != nil {
			t.Fatalf("業務上のdeniedがExecution Errorになっています: %v", err)
		}
		if invocation.Result.Outcome != "denied" {
			t.Fatalf("Business Result=%#v", invocation.Result)
		}
		payload, ok := invocation.Result.Payload.(legacyauth.AuthenticationDenial)
		if !ok || payload.Reason != "credential_mismatch" {
			t.Fatalf("denied payload=%#v", invocation.Result.Payload)
		}
	})
}

func TestDriverKeepsAdapterResolutionAndInvocationFailuresAsExecutionErrors(t *testing.T) {
	resolver := driver.NewResolver(referenceWorld(t))

	t.Run("Adapter resolverの障害をdeniedへ変換しない", func(t *testing.T) {
		adapters := bindingAdapterResolverFunc(func(context.Context, realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
			return nil, errors.New("adapter registry unavailable")
		})
		result, err := resolver.InvokeCapability(context.Background(), "order_entry", "security.authenticate", adapters, struct{}{})
		if err == nil {
			t.Fatalf("Adapter resolver障害がBusiness Resultになっています: %#v", result)
		}
		if result.Result.Outcome == "denied" {
			t.Fatalf("Adapter resolver障害がdeniedへ変換されています: %#v", result)
		}
	})

	t.Run("Adapterがnilなら実行準備の失敗にする", func(t *testing.T) {
		adapters := bindingAdapterResolverFunc(func(context.Context, realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
			return nil, nil
		})
		if _, err := resolver.InvokeCapability(context.Background(), "order_entry", "security.authenticate", adapters, struct{}{}); err == nil {
			t.Fatal("nil AdapterでCapabilityを実行しています")
		}
	})

	t.Run("Canonical契約にないOutcomeをExecution Errorとして返す", func(t *testing.T) {
		adapters := bindingAdapterResolverFunc(func(context.Context, realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
			return capabilityAdapterFunc(func(_ context.Context, capability core.CanonicalID, _ any) (capabilityinvoke.BusinessResult, error) {
				return capabilityinvoke.BusinessResult{
					Capability: capability,
					Mode:       capabilityinvoke.ResultOutcome,
					Outcome:    "unknown_outcome",
					Payload:    struct{}{},
				}, nil
			}), nil
		})
		result, err := resolver.InvokeCapability(context.Background(), "order_entry", "security.authenticate", adapters, struct{}{})
		if err == nil {
			t.Fatalf("未定義OutcomeがBusiness Resultとして通過しています: %#v", result)
		}
	})
}

func referenceLegacyAuthFixture(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test fileの位置を取得できません")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	return filepath.Join(root, "reference", "legacy-auth", "users.json")
}

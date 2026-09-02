package driver_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"tei/internal/driver"
	"tei/internal/realization"
	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
)

func TestDriverCarriesAuthenticatedSubjectIntoAuthorizationWithoutSecuritySpecificDriverCode(t *testing.T) {
	resolver := driver.NewResolver(referenceWorld(t))
	store, err := legacyauth.LoadFixtureStore(referenceLegacyAuthFixture(t))
	if err != nil {
		t.Fatalf("legacy authentication fixtureを読み込めません: %v", err)
	}
	rule, err := legacyauthorization.LoadFixtureRule(filepath.Join(filepath.Dir(referenceLegacyAuthFixture(t)), "authorization.json"))
	if err != nil {
		t.Fatalf("legacy authorization fixtureを読み込めません: %v", err)
	}

	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	authentication := legacyauth.Adapter{
		Store: store,
		Verifier: driverCredentialVerifierFunc(func(_ context.Context, ref, presented string) (bool, error) {
			switch ref {
			case "credential-ref://legacy/order-editor":
				return presented == "editor-secret", nil
			case "credential-ref://legacy/order-viewer":
				return presented == "viewer-secret", nil
			default:
				return false, fmt.Errorf("未登録のcredential参照です: %s", ref)
			}
		}),
		Now: func() time.Time { return now },
	}
	authorization := legacyauthorization.Adapter{Store: store, Rule: rule}

	adapters := bindingAdapterResolverFunc(func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
		switch binding.Identity() {
		case "order_entry/security.authenticate":
			return authentication, nil
		case "order_entry/security.authorize":
			return authorization, nil
		default:
			return nil, fmt.Errorf("未登録のBindingです: %s", binding.Identity())
		}
	})

	t.Run("ORDER_EDITORは認証後にbrowseを許可される", func(t *testing.T) {
		subjectID := authenticateSubject(t, resolver, adapters, "legacy-order-editor", "editor-secret")
		invocation, err := resolver.InvokeCapability(
			context.Background(),
			"order_entry",
			"security.authorize",
			adapters,
			legacyauthorization.Input{
				SubjectID: subjectID,
				Action:    "browse",
				Resource:  "order_entry.recent_orders",
			},
		)
		if err != nil {
			t.Fatalf("Authorizationを実行できません: %v", err)
		}
		if invocation.Binding.Binding.Identity() != "order_entry/security.authorize" {
			t.Fatalf("Binding identity=%q", invocation.Binding.Binding.Identity())
		}
		if invocation.Result.Outcome != "allowed" {
			t.Fatalf("Authorization result=%#v", invocation.Result)
		}
		grant, ok := invocation.Result.Payload.(legacyauthorization.AuthorizationGrant)
		if !ok {
			t.Fatalf("allowed payload=%#v", invocation.Result.Payload)
		}
		if grant.SubjectID != "subject-order-editor" || grant.Action != "browse" || grant.Resource != "order_entry.recent_orders" {
			t.Fatalf("Authorization grant=%#v", grant)
		}
	})

	t.Run("ORDER_VIEWERのedit拒否はExecution Errorではなくdeniedになる", func(t *testing.T) {
		subjectID := authenticateSubject(t, resolver, adapters, "legacy-order-viewer", "viewer-secret")
		invocation, err := resolver.InvokeCapability(
			context.Background(),
			"order_entry",
			"security.authorize",
			adapters,
			legacyauthorization.Input{
				SubjectID: subjectID,
				Action:    "edit",
				Resource:  "order_entry.order_record",
			},
		)
		if err != nil {
			t.Fatalf("業務上の認可拒否がExecution Errorになっています: %v", err)
		}
		if invocation.Result.Outcome != "denied" {
			t.Fatalf("Authorization result=%#v", invocation.Result)
		}
		denial, ok := invocation.Result.Payload.(legacyauthorization.AuthorizationDenial)
		if !ok || denial.Reason != "not_authorized" {
			t.Fatalf("denied payload=%#v", invocation.Result.Payload)
		}
	})
}

func authenticateSubject(
	t *testing.T,
	resolver driver.Resolver,
	adapters driver.BindingAdapterResolver,
	loginID, credential string,
) string {
	t.Helper()
	invocation, err := resolver.InvokeCapability(
		context.Background(),
		"order_entry",
		"security.authenticate",
		adapters,
		legacyauth.Input{LoginID: loginID, Credential: credential},
	)
	if err != nil {
		t.Fatalf("Authenticationを実行できません: %v", err)
	}
	if invocation.Result.Outcome != "authenticated" {
		t.Fatalf("Authentication result=%#v", invocation.Result)
	}
	payload, ok := invocation.Result.Payload.(legacyauth.AuthenticatedSubject)
	if !ok || payload.SubjectID == "" {
		t.Fatalf("authenticated payload=%#v", invocation.Result.Payload)
	}
	return payload.SubjectID
}

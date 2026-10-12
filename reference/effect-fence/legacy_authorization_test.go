package effectfence

import (
	"context"
	"errors"
	"testing"

	"tei/internal/core"
	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
)

type legacyAuthFunc func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error)

func (f legacyAuthFunc) Invoke(ctx context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	return f(ctx, capability, input)
}

type legacySubjectStoreFunc func(context.Context, string) (legacyauth.User, bool, error)

func (f legacySubjectStoreFunc) FindBySubject(ctx context.Context, subject string) (legacyauth.User, bool, error) {
	return f(ctx, subject)
}

type legacyRuleFunc func(context.Context, legacyauth.User, string, string) (bool, string, error)

func (f legacyRuleFunc) Evaluate(ctx context.Context, u legacyauth.User, action, resource string) (bool, string, error) {
	return f(ctx, u, action, resource)
}

func legacyBridgeScope() Scope {
	return Scope{Application: "sales", Capability: "sales.order.save", Subject: "editor-1", Action: "edit", Resource: "sales.order-5"}
}

func legacyBridgeRequest() Request {
	return Request{Scope: legacyBridgeScope(), OperationID: "save-01", Value: "保存内容"}
}

func newLegacyBridgeStore(t *testing.T) *MemoryStore {
	t.Helper()
	m := NewMemoryStore()
	if err := m.SetAuthority(legacyBridgeScope(), 1, true); err != nil {
		t.Fatal(err)
	}
	return m
}

func grantedLegacyResult(s Scope) capabilityinvoke.BusinessResult {
	return capabilityinvoke.BusinessResult{
		Capability: legacyauthorization.AuthorizationCapability,
		Mode:       capabilityinvoke.ResultOutcome,
		Outcome:    "allowed",
		Payload: legacyauthorization.AuthorizationGrant{
			SubjectID: s.Subject, Action: s.Action, Resource: s.Resource,
		},
	}
}

func TestLegacyAuthorizationBridgeUsesExistingRuleAndGrant(t *testing.T) {
	m := newLegacyBridgeStore(t)
	count := 0
	auth := legacyauthorization.Adapter{
		Store: legacySubjectStoreFunc(func(_ context.Context, subject string) (legacyauth.User, bool, error) {
			return legacyauth.User{SubjectID: subject, RoleCode: "EDITOR"}, true, nil
		}),
		Rule: legacyRuleFunc(func(_ context.Context, u legacyauth.User, action, resource string) (bool, string, error) {
			count++
			return u.RoleCode == "EDITOR" && action == "edit" && resource == "sales.order-5", "not_authorized", nil
		}),
	}
	permit, r, err := m.PrepareWithLegacyAuthorization(context.Background(), "editor-1", legacyBridgeRequest(), auth)
	if err != nil || r.Disposition != Prepared || count != 1 {
		t.Fatalf("existing rule not used: %+v %v count=%d", r, err, count)
	}
	r, err = m.Commit(context.Background(), permit)
	if err != nil || r.Disposition != Committed {
		t.Fatalf("legitimate update refused: %+v %v", r, err)
	}
	if val, version := m.Observe(legacyBridgeScope()); val != "保存内容" || version != 1 {
		t.Fatalf("missing effect: value=%q version=%d", val, version)
	}
}

func TestLegacyAuthorizationBridgeRejectsExistingRoleDenial(t *testing.T) {
	m := newLegacyBridgeStore(t)
	auth := legacyauthorization.Adapter{
		Store: legacySubjectStoreFunc(func(_ context.Context, subject string) (legacyauth.User, bool, error) {
			return legacyauth.User{SubjectID: subject, RoleCode: "VIEWER"}, true, nil
		}),
		Rule: legacyRuleFunc(func(_ context.Context, _ legacyauth.User, _, _ string) (bool, string, error) {
			return false, "sensitive-policy-detail", nil
		}),
	}
	permit, r, err := m.PrepareWithLegacyAuthorization(context.Background(), "editor-1", legacyBridgeRequest(), auth)
	if err != nil || r.Disposition != Denied || r.Reason != "authorization-denied" || permit != (Permit{}) {
		t.Fatalf("role rejection not handled: %+v %v", r, err)
	}
	if _, version := m.Observe(legacyBridgeScope()); version != 0 {
		t.Fatal("business denial wrote data")
	}
}

func TestLegacyAuthorizationBridgeRejectsForgedSubjectWithoutCallingProvider(t *testing.T) {
	m := newLegacyBridgeStore(t)
	calls := 0
	fake := legacyAuthFunc(func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error) {
		calls++
		return grantedLegacyResult(legacyBridgeScope()), nil
	})
	_, r, err := m.PrepareWithLegacyAuthorization(context.Background(), "another-user", legacyBridgeRequest(), fake)
	if err != nil || r.Disposition != Denied || r.Reason != "subject-mismatch" || calls != 0 {
		t.Fatalf("forged subject reached provider: %+v %v calls=%d", r, err, calls)
	}
}

func TestLegacyAuthorizationBridgeRejectsMismatchedGrantAndMalformedOutcomes(t *testing.T) {
	scope := legacyBridgeScope()
	wrongSubject := grantedLegacyResult(scope)
	wrongSubject.Payload = legacyauthorization.AuthorizationGrant{SubjectID: "another-user", Action: scope.Action, Resource: scope.Resource}
	wrongAction := grantedLegacyResult(scope)
	wrongAction.Payload = legacyauthorization.AuthorizationGrant{SubjectID: scope.Subject, Action: "browse", Resource: scope.Resource}
	wrongResource := grantedLegacyResult(scope)
	wrongResource.Payload = legacyauthorization.AuthorizationGrant{SubjectID: scope.Subject, Action: scope.Action, Resource: "another-record"}
	wrongCapability := grantedLegacyResult(scope)
	wrongCapability.Capability = "sales.order.save"
	wrongMode := grantedLegacyResult(scope)
	wrongMode.Mode = capabilityinvoke.ResultOutput
	wrongPayload := grantedLegacyResult(scope)
	wrongPayload.Payload = map[string]string{"subject": scope.Subject}
	unknown := grantedLegacyResult(scope)
	unknown.Outcome = "unknown"
	badDenial := grantedLegacyResult(scope)
	badDenial.Outcome = "denied"
	badDenial.Payload = "secret"
	missingDenialReason := grantedLegacyResult(scope)
	missingDenialReason.Outcome = "denied"
	missingDenialReason.Payload = legacyauthorization.AuthorizationDenial{}
	cases := map[string]capabilityinvoke.BusinessResult{
		"subject": wrongSubject, "action": wrongAction, "resource": wrongResource,
		"capability": wrongCapability, "mode": wrongMode, "payload": wrongPayload,
		"outcome": unknown, "denial-payload": badDenial, "denial-reason": missingDenialReason,
	}
	for name, fakeResult := range cases {
		t.Run(name, func(t *testing.T) {
			m := newLegacyBridgeStore(t)
			fake := legacyAuthFunc(func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error) {
				return fakeResult, nil
			})
			p, result, err := m.PrepareWithLegacyAuthorization(context.Background(), scope.Subject, legacyBridgeRequest(), fake)
			if err == nil || result.Disposition != "" || p != (Permit{}) {
				t.Fatalf("invalid authorization result accepted: %+v %v", result, err)
			}
			if _, version := m.Observe(scope); version != 0 {
				t.Fatal("invalid authorization result caused an effect")
			}
		})
	}
}

func TestLegacyAuthorizationBridgePreservesProviderErrorsAndMissingProvider(t *testing.T) {
	m := newLegacyBridgeStore(t)
	outage := errors.New("authorization store unavailable")
	fake := legacyAuthFunc(func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error) {
		return capabilityinvoke.BusinessResult{}, outage
	})
	_, result, err := m.PrepareWithLegacyAuthorization(context.Background(), "editor-1", legacyBridgeRequest(), fake)
	if !errors.Is(err, outage) || result.Disposition != "" {
		t.Fatalf("provider error hidden as a denial: %+v %v", result, err)
	}
	_, result, err = m.PrepareWithLegacyAuthorization(context.Background(), "editor-1", legacyBridgeRequest(), nil)
	if err == nil || result.Disposition != "" {
		t.Fatalf("missing provider accepted: %+v %v", result, err)
	}
}

func TestLegacyAuthorizationBridgeRevocationBeforeAndAfterPrepare(t *testing.T) {
	scope := legacyBridgeScope()
	m := newLegacyBridgeStore(t)
	revokeOnAuthorization := legacyAuthFunc(func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error) {
		if err := m.SetAuthority(scope, 2, false); err != nil {
			t.Fatal(err)
		}
		return grantedLegacyResult(scope), nil
	})
	_, outcome, err := m.PrepareWithLegacyAuthorization(context.Background(), scope.Subject, legacyBridgeRequest(), revokeOnAuthorization)
	if err != nil || outcome.Disposition != Denied {
		t.Fatalf("revocation during preflight was not detected: %+v %v", outcome, err)
	}
	m = newLegacyBridgeStore(t)
	allow := legacyAuthFunc(func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error) {
		return grantedLegacyResult(scope), nil
	})
	permit, outcome, err := m.PrepareWithLegacyAuthorization(context.Background(), scope.Subject, legacyBridgeRequest(), allow)
	if err != nil || outcome.Disposition != Prepared {
		t.Fatalf("prepare failed: %+v %v", outcome, err)
	}
	if err := m.SetAuthority(scope, 2, false); err != nil {
		t.Fatal(err)
	}
	outcome, err = m.Commit(context.Background(), permit)
	if err != nil || outcome.Disposition != Denied || outcome.Reason != "authority-changed" {
		t.Fatalf("revoked commit succeeded: %+v %v", outcome, err)
	}
	if _, version := m.Observe(scope); version != 0 {
		t.Fatalf("revoked operation committed: %d", version)
	}
}

func TestLegacyAuthorizationBridgeNeverClaimsDurableAtomicity(t *testing.T) {
	m := newLegacyBridgeStore(t)
	scope := legacyBridgeScope()
	permit, r, err := m.PrepareWithLegacyAuthorization(context.Background(), scope.Subject, legacyBridgeRequest(), legacyAuthFunc(func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error) {
		return grantedLegacyResult(scope), nil
	}))
	if err != nil || r.Disposition != Prepared {
		t.Fatalf("prepare: %+v %v", r, err)
	}
	applied := false
	_, err = CommitWithRequirement(context.Background(), m.BoundaryContract(), GuaranteeDurable, func(ctx context.Context) (Result, error) {
		applied = true
		return m.Commit(ctx, permit)
	})
	var unavailable *GuaranteeUnavailableError
	if !errors.As(err, &unavailable) || applied {
		t.Fatalf("durable effect was admitted: %v applied=%v", err, applied)
	}
	if _, version := m.Observe(scope); version != 0 {
		t.Fatalf("rejected durable commit changed data: %d", version)
	}
}

func TestLegacyAuthorizationBridgeStopsNewWriteAfterAccountRevocation(t *testing.T) {
	scope := legacyBridgeScope()
	m := newLegacyBridgeStore(t)
	user := legacyauth.User{SubjectID: scope.Subject, RoleCode: "EDITOR"}
	calls := 0
	auth := legacyauthorization.Adapter{
		Store: legacySubjectStoreFunc(func(_ context.Context, subject string) (legacyauth.User, bool, error) {
			return user, true, nil
		}),
		Rule: legacyRuleFunc(func(_ context.Context, _ legacyauth.User, _, _ string) (bool, string, error) {
			calls++
			return true, "", nil
		}),
	}
	initial := legacyBridgeRequest()
	p, result, err := m.PrepareWithLegacyAuthorization(context.Background(), scope.Subject, initial, auth)
	if err != nil || result.Disposition != Prepared {
		t.Fatalf("初回認可: %+v %v", result, err)
	}
	result, err = m.Commit(context.Background(), p)
	if err != nil || result.Disposition != Committed {
		t.Fatalf("初回保存: %+v %v", result, err)
	}
	user.Disabled = true
	later := legacyBridgeRequest()
	later.OperationID = "save-after-revocation"
	later.ExpectedResourceVersion = 1
	later.Value = "保存されてはいけない値"
	p, result, err = m.PrepareWithLegacyAuthorization(context.Background(), scope.Subject, later, auth)
	if err != nil || result.Disposition != Denied || result.Reason != "authorization-denied" || p != (Permit{}) {
		t.Fatalf("失効後の新しい操作が拒否されません: %+v %v", result, err)
	}
	if calls != 1 {
		t.Fatalf("無効化後に役割評価が呼ばれました: calls=%d", calls)
	}
	if value, version := m.Observe(scope); value != initial.Value || version != 1 {
		t.Fatalf("無効化後に更新されました: %q %d", value, version)
	}
}

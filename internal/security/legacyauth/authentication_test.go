package legacyauth_test

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
	"tei/internal/security/legacyauth"
	"tei/internal/semantic"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/workspace"
)

type storeFunc func(context.Context, string) (legacyauth.User, bool, error)

func (f storeFunc) FindByLogin(ctx context.Context, login string) (legacyauth.User, bool, error) {
	return f(ctx, login)
}

type verifierFunc func(context.Context, string, string) (bool, error)

func (f verifierFunc) Verify(ctx context.Context, ref, presented string) (bool, error) {
	return f(ctx, ref, presented)
}

type ruleFunc func(context.Context, legacyauth.User, time.Time) (bool, string, error)

func (f ruleFunc) Evaluate(ctx context.Context, user legacyauth.User, now time.Time) (bool, string, error) {
	return f(ctx, user, now)
}

func TestLegacyAuthenticationMapsExistingUserStatesToBusinessOutcomes(t *testing.T) {
	capability := referenceCapability(t, legacyauth.AuthenticationCapability)
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	base := legacyauth.User{
		LoginID:           "legacy-user-001",
		SubjectID:         "subject-001",
		CredentialRef:     "legacy-credential-ref-001",
		ValidFrom:         now.Add(-24 * time.Hour),
		ValidUntil:        now.Add(24 * time.Hour),
		LoginFailureCount: 2,
		OrganizationCode:  "SALES",
		RoleCode:          "ORDER_EDITOR",
	}

	cases := []struct {
		name        string
		mutate      func(*legacyauth.User)
		found       bool
		matched     bool
		rule        legacyauth.LegacyRule
		wantOutcome core.LocalName
		wantReason  string
	}{
		{name: "認証成功", found: true, matched: true, wantOutcome: "authenticated"},
		{name: "credential不一致", found: true, matched: false, wantOutcome: "denied", wantReason: "credential_mismatch"},
		{name: "利用者が見つからない", found: false, matched: false, wantOutcome: "denied", wantReason: "credential_mismatch"},
		{name: "無効化済み", found: true, matched: true, mutate: func(u *legacyauth.User) { u.Disabled = true }, wantOutcome: "denied", wantReason: "account_disabled"},
		{name: "有効期間前", found: true, matched: true, mutate: func(u *legacyauth.User) { u.ValidFrom = now.Add(time.Hour) }, wantOutcome: "denied", wantReason: "not_yet_valid"},
		{name: "有効期間切れ", found: true, matched: true, mutate: func(u *legacyauth.User) { u.ValidUntil = now }, wantOutcome: "denied", wantReason: "expired"},
		{name: "lock中", found: true, matched: true, mutate: func(u *legacyauth.User) { u.LockedUntil = now.Add(time.Hour) }, wantOutcome: "denied", wantReason: "account_locked"},
		{
			name:    "既存システム固有条件が不成立",
			found:   true,
			matched: true,
			rule: ruleFunc(func(_ context.Context, user legacyauth.User, _ time.Time) (bool, string, error) {
				if user.OrganizationCode != "SALES" || user.RoleCode != "ORDER_EDITOR" || user.LoginFailureCount != 2 {
					return false, "", fmt.Errorf("legacy metadataがadapterへ保存されていません: %#v", user)
				}
				return false, "organization_specific_condition", nil
			}),
			wantOutcome: "denied",
			wantReason:  "organization_specific_condition",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			user := base
			if tt.mutate != nil {
				tt.mutate(&user)
			}

			verifyCalls := 0
			adapter := legacyauth.Adapter{
				Store: storeFunc(func(_ context.Context, login string) (legacyauth.User, bool, error) {
					if login != "legacy-user-001" {
						t.Fatalf("login=%q", login)
					}
					return user, tt.found, nil
				}),
				Verifier: verifierFunc(func(_ context.Context, ref, presented string) (bool, error) {
					verifyCalls++
					if ref != "legacy-credential-ref-001" {
						t.Fatalf("credential ref=%q", ref)
					}
					if presented != "runtime-presented-value" {
						t.Fatalf("presented credential=%q", presented)
					}
					return tt.matched, nil
				}),
				Rule: tt.rule,
				Now:  func() time.Time { return now },
			}

			result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, legacyauth.Input{
				LoginID:    "legacy-user-001",
				Credential: "runtime-presented-value",
			})
			if err != nil {
				t.Fatalf("業務上の認証結果がExecution Errorになっています: %v", err)
			}
			if result.Mode != capabilityinvoke.ResultOutcome || result.Outcome != tt.wantOutcome {
				t.Fatalf("result=%#v want outcome=%q", result, tt.wantOutcome)
			}

			if tt.wantOutcome == "authenticated" {
				payload, ok := result.Payload.(legacyauth.AuthenticatedSubject)
				if !ok || payload.SubjectID != "subject-001" {
					t.Fatalf("authenticated payload=%#v", result.Payload)
				}
			} else {
				payload, ok := result.Payload.(legacyauth.AuthenticationDenial)
				if !ok || payload.Reason != tt.wantReason {
					t.Fatalf("denied payload=%#v want reason=%q", result.Payload, tt.wantReason)
				}
			}

			if strings.Contains(fmt.Sprintf("%#v", result), "runtime-presented-value") {
				t.Fatalf("Business Resultへ提示credentialが混入しています: %#v", result)
			}

			if tt.found && !user.Disabled && !now.Before(user.ValidFrom) && now.Before(user.ValidUntil) && (user.LockedUntil.IsZero() || !now.Before(user.LockedUntil)) && tt.rule == nil {
				if verifyCalls != 1 {
					t.Fatalf("credential verifier calls=%d want=1", verifyCalls)
				}
			}
		})
	}
}

func TestLegacyAuthenticationKeepsInfrastructureFailuresAsExecutionErrors(t *testing.T) {
	capability := referenceCapability(t, legacyauth.AuthenticationCapability)
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	user := legacyauth.User{
		LoginID:       "legacy-user-001",
		SubjectID:     "subject-001",
		CredentialRef: "legacy-credential-ref-001",
	}

	cases := []struct {
		name     string
		store    legacyauth.UserStore
		verifier legacyauth.CredentialVerifier
		rule     legacyauth.LegacyRule
	}{
		{
			name: "既存user store障害",
			store: storeFunc(func(context.Context, string) (legacyauth.User, bool, error) {
				return legacyauth.User{}, false, errors.New("legacy database unavailable")
			}),
			verifier: verifierFunc(func(context.Context, string, string) (bool, error) { return true, nil }),
		},
		{
			name: "credential verifier障害",
			store: storeFunc(func(context.Context, string) (legacyauth.User, bool, error) {
				return user, true, nil
			}),
			verifier: verifierFunc(func(context.Context, string, string) (bool, error) {
				return false, errors.New("credential service unavailable")
			}),
		},
		{
			name: "既存rule評価障害",
			store: storeFunc(func(context.Context, string) (legacyauth.User, bool, error) {
				return user, true, nil
			}),
			verifier: verifierFunc(func(context.Context, string, string) (bool, error) { return true, nil }),
			rule: ruleFunc(func(context.Context, legacyauth.User, time.Time) (bool, string, error) {
				return false, "", errors.New("legacy rule service unavailable")
			}),
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			adapter := legacyauth.Adapter{Store: tt.store, Verifier: tt.verifier, Rule: tt.rule, Now: func() time.Time { return now }}
			result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, legacyauth.Input{LoginID: "legacy-user-001", Credential: "runtime-presented-value"})
			if err == nil {
				t.Fatalf("実行障害がBusiness Resultになっています: %#v", result)
			}
			if result.Outcome == "denied" {
				t.Fatalf("実行障害がdeniedへ変換されています: result=%#v err=%v", result, err)
			}
		})
	}
}

func TestLegacyAuthenticationRejectsMalformedLegacyRecordsAsExecutionErrors(t *testing.T) {
	capability := referenceCapability(t, legacyauth.AuthenticationCapability)
	for _, tt := range []struct {
		name string
		user legacyauth.User
	}{
		{name: "subject identityなし", user: legacyauth.User{LoginID: "legacy-user-001", CredentialRef: "ref"}},
		{name: "credential参照なし", user: legacyauth.User{LoginID: "legacy-user-001", SubjectID: "subject-001"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			adapter := legacyauth.Adapter{
				Store:    storeFunc(func(context.Context, string) (legacyauth.User, bool, error) { return tt.user, true, nil }),
				Verifier: verifierFunc(func(context.Context, string, string) (bool, error) { return true, nil }),
			}
			result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, legacyauth.Input{LoginID: "legacy-user-001", Credential: "runtime-presented-value"})
			if err == nil {
				t.Fatalf("壊れたlegacy recordが正常結果になっています: %#v", result)
			}
		})
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

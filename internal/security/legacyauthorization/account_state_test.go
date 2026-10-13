package legacyauthorization_test

import (
	"context"
	"testing"
	"time"

	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
)

// 認証済みセッションが残っていても、現在の利用者状態が無効になれば役割を評価しない。
func TestLegacyAuthorizationRechecksAccountAvailability(t *testing.T) {
	now := time.Date(2026, time.October, 12, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		user legacyauth.User
		want string
	}{
		{"通常", legacyauth.User{SubjectID: "subject-editor"}, "allowed"},
		{"無効化", legacyauth.User{SubjectID: "subject-editor", Disabled: true}, "account_disabled"},
		{"開始前", legacyauth.User{SubjectID: "subject-editor", ValidFrom: now.Add(time.Nanosecond)}, "not_yet_valid"},
		{"開始境界", legacyauth.User{SubjectID: "subject-editor", ValidFrom: now}, "allowed"},
		{"期限直前", legacyauth.User{SubjectID: "subject-editor", ValidUntil: now.Add(time.Nanosecond)}, "allowed"},
		{"期限切れ", legacyauth.User{SubjectID: "subject-editor", ValidUntil: now}, "expired"},
		{"ロック中", legacyauth.User{SubjectID: "subject-editor", LockedUntil: now.Add(time.Nanosecond)}, "account_locked"},
		{"ロック解除境界", legacyauth.User{SubjectID: "subject-editor", LockedUntil: now}, "allowed"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			adapter := legacyauthorization.Adapter{
				Store: authorizationStoreFunc(func(context.Context, string) (legacyauth.User, bool, error) {
					return tt.user, true, nil
				}),
				Rule: authorizationRuleFunc(func(context.Context, legacyauth.User, string, string) (bool, string, error) {
					calls++
					return true, "", nil
				}),
				Now: func() time.Time { return now },
			}
			result, err := adapter.Invoke(context.Background(), legacyauthorization.AuthorizationCapability,
				legacyauthorization.Input{SubjectID: "subject-editor", Action: "write", Resource: "record-1"})
			if err != nil {
				t.Fatal(err)
			}
			if result.Mode != capabilityinvoke.ResultOutcome {
				t.Fatalf("unexpected result mode: %+v", result)
			}
			if tt.want == "allowed" {
				if result.Outcome != "allowed" || calls != 1 {
					t.Fatalf("active user refused: %+v, role evaluations=%d", result, calls)
				}
				return
			}
			denial, ok := result.Payload.(legacyauthorization.AuthorizationDenial)
			if result.Outcome != "denied" || !ok || denial.Reason != tt.want || calls != 0 {
				t.Fatalf("unavailable user passed role check: %+v, role evaluations=%d", result, calls)
			}
		})
	}
}

func TestLegacyAuthorizationRevocationAfterPriorGrant(t *testing.T) {
	now := time.Date(2026, time.October, 12, 12, 0, 0, 0, time.UTC)
	user := legacyauth.User{SubjectID: "subject-editor"}
	ruleCalls := 0
	adapter := legacyauthorization.Adapter{
		Store: authorizationStoreFunc(func(context.Context, string) (legacyauth.User, bool, error) {
			return user, true, nil
		}),
		Rule: authorizationRuleFunc(func(context.Context, legacyauth.User, string, string) (bool, string, error) {
			ruleCalls++
			return true, "", nil
		}),
		Now: func() time.Time { return now },
	}
	input := legacyauthorization.Input{SubjectID: "subject-editor", Action: "write", Resource: "record-1"}
	first, err := adapter.Invoke(context.Background(), legacyauthorization.AuthorizationCapability, input)
	if err != nil || first.Outcome != "allowed" || ruleCalls != 1 {
		t.Fatalf("initial authorization: %+v, %v, calls=%d", first, err, ruleCalls)
	}
	user.Disabled = true
	second, err := adapter.Invoke(context.Background(), legacyauthorization.AuthorizationCapability, input)
	if err != nil || second.Outcome != "denied" || ruleCalls != 1 {
		t.Fatalf("revoked user retained role grant: %+v, %v, calls=%d", second, err, ruleCalls)
	}
	user.Disabled = false
	user.ValidUntil = now
	third, err := adapter.Invoke(context.Background(), legacyauthorization.AuthorizationCapability, input)
	if err != nil || third.Outcome != "denied" || ruleCalls != 1 {
		t.Fatalf("expired user retained role grant: %+v, %v, calls=%d", third, err, ruleCalls)
	}
}

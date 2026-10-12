package legacyauth_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"tei/internal/security/legacyauth"
)

func TestLegacyAuthenticationBindsStoreRecordToRequestedLogin(t *testing.T) {
	for _, tt := range []struct {
		name              string
		requested         string
		returned          string
		found             bool
		wantError         bool
		wantOutcome       string
		wantStoreCalls    int
		wantVerifierCalls int
	}{
		{
			name: "matching login authenticates",
			requested: "user-a", returned: "user-a", found: true,
			wantOutcome: "authenticated", wantStoreCalls: 1, wantVerifierCalls: 1,
		},
		{
			name: "different login is rejected before verifying another user",
			requested: "user-a", returned: "user-b", found: true,
			wantError: true, wantStoreCalls: 1,
		},
		{
			name: "case mismatch requires an explicit canonical mapping",
			requested: "user-a", returned: "USER-A", found: true,
			wantError: true, wantStoreCalls: 1,
		},
		{
			name: "empty login is rejected before store lookup",
			requested: "", returned: "user-b", found: true,
			wantError: true,
		},
		{
			name: "blank login is rejected before store lookup",
			requested: " \t ", returned: "user-b", found: true,
			wantError: true,
		},
		{
			name: "not found remains credential mismatch",
			requested: "user-a", returned: "user-b", found: false,
			wantOutcome: "denied", wantStoreCalls: 1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			storeCalls, verifierCalls, ruleCalls := 0, 0, 0
			adapter := legacyauth.Adapter{
				Store: storeFunc(func(_ context.Context, login string) (legacyauth.User, bool, error) {
					storeCalls++
					if login != tt.requested {
						t.Fatalf("store request was changed: %q", login)
					}
					return legacyauth.User{
						LoginID: tt.returned, SubjectID: "subject-b",
						CredentialRef: "credential-ref-b",
					}, tt.found, nil
				}),
				Verifier: verifierFunc(func(context.Context, string, string) (bool, error) {
					verifierCalls++
					return true, nil
				}),
				Rule: ruleFunc(func(context.Context, legacyauth.User, time.Time) (bool, string, error) {
					ruleCalls++
					return true, "", nil
				}),
			}
			result, err := adapter.Invoke(context.Background(), legacyauth.AuthenticationCapability, legacyauth.Input{
				LoginID: tt.requested, Credential: "runtime-secret",
			})
			if tt.wantError {
				if err == nil {
					t.Fatalf("wrong or missing login identity was accepted: %#v", result)
				}
				if result.Outcome != "" {
					t.Fatalf("store contract violation returned a business outcome: %#v", result)
				}
				for _, secret := range []string{"runtime-secret", "credential-ref-b", "subject-b", "user-b"} {
					if strings.Contains(err.Error(), secret) {
						t.Fatalf("identity or secret leaked into diagnostic: %v", err)
					}
				}
			} else {
				if err != nil || string(result.Outcome) != tt.wantOutcome {
					t.Fatalf("result=%#v err=%v want=%s", result, err, tt.wantOutcome)
				}
			}
			if storeCalls != tt.wantStoreCalls || verifierCalls != tt.wantVerifierCalls {
				t.Fatalf("unexpected calls: store=%d verifier=%d", storeCalls, verifierCalls)
			}
			if tt.wantError && ruleCalls != 0 {
				t.Fatalf("legacy rule evaluated after identity mismatch: %d", ruleCalls)
			}
			if tt.wantOutcome == "authenticated" && ruleCalls != 1 {
				t.Fatalf("valid login did not reach legacy rule: %d", ruleCalls)
			}
			if tt.wantOutcome == "denied" && ruleCalls != 0 {
				t.Fatalf("unknown account reached legacy rule: %d", ruleCalls)
			}
		})
	}
}

package legacyauthorization_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
)

func TestAuthorizationEpochInactiveIsBusinessDenial(t *testing.T) {
	for _, stage := range []string{"before-user-read", "after-rule-evaluation"} {
		t.Run(stage, func(t *testing.T) {
			reads, users, rules := 0, 0, 0
			adapter := legacyauthorization.Adapter{
				Store: authorizationStoreFunc(func(context.Context, string) (legacyauth.User, bool, error) {
					users++
					return legacyauth.User{SubjectID: "editor", RoleCode: "EDITOR"}, true, nil
				}),
				Rule: authorizationRuleFunc(func(context.Context, legacyauth.User, string, string) (bool, string, error) {
					rules++
					return true, "", nil
				}),
				EpochReader: epochReadFunc(func(context.Context, legacyauthorization.Input) (legacyauthorization.AuthorityEpochs, error) {
					reads++
					if stage == "before-user-read" || reads == 2 {
						return legacyauthorization.AuthorityEpochs{}, fmt.Errorf("正本の失効が確認されました: %w", legacyauthorization.ErrAuthorityInactive)
					}
					return stableEpochs(), nil
				}),
			}
			got, err := adapter.Invoke(context.Background(), legacyauthorization.AuthorizationCapability, legacyauthorization.Input{
				SubjectID: "editor", Action: "write", Resource: "order-1",
			})
			denial, ok := got.Payload.(legacyauthorization.AuthorizationDenial)
			if err != nil || got.Outcome != "denied" || !ok || denial.Reason != "authority_inactive" {
				t.Fatalf("明示された失効を業務拒否へ変換できません: %+v %v", got, err)
			}
			if stage == "before-user-read" && (reads != 1 || users != 0 || rules != 0) {
				t.Fatalf("失効後に利用者・役割規則を評価しました: reads=%d users=%d rules=%d", reads, users, rules)
			}
			if stage == "after-rule-evaluation" && (reads != 2 || users != 1 || rules != 1) {
				t.Fatalf("判断中の失効を検出できません: reads=%d users=%d rules=%d", reads, users, rules)
			}
		})
	}
}

func TestAuthorizationEpochReadFailuresRemainExecutionErrors(t *testing.T) {
	for _, tt := range []struct {
		name string
		readError error
	}{
		{"provider-unavailable", errors.New("storage unavailable")},
		{"context-canceled", context.Canceled},
		{"context-deadline", context.DeadlineExceeded},
	} {
		t.Run(tt.name, func(t *testing.T) {
			adapter := legacyauthorization.Adapter{
				Store: authorizationStoreFunc(func(context.Context, string) (legacyauth.User, bool, error) {
					t.Fatal("版の読取り障害後に利用者を取得してはいけません")
					return legacyauth.User{}, false, nil
				}),
				Rule: authorizationRuleFunc(func(context.Context, legacyauth.User, string, string) (bool, string, error) {
					t.Fatal("版の読取り障害後に役割を評価してはいけません")
					return false, "", nil
				}),
				EpochReader: epochReadFunc(func(context.Context, legacyauthorization.Input) (legacyauthorization.AuthorityEpochs, error) {
					return legacyauthorization.AuthorityEpochs{}, tt.readError
				}),
			}
			got, err := adapter.Invoke(context.Background(), legacyauthorization.AuthorizationCapability, legacyauthorization.Input{
				SubjectID: "editor", Action: "write", Resource: "order-1",
			})
			if !errors.Is(err, tt.readError) || got.Outcome != "" || got.Payload != nil {
				t.Fatalf("Provider障害を許可または業務拒否へ変換しました: %+v %v", got, err)
			}
		})
	}
}

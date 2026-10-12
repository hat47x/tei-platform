package legacyauthorization_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
)

type epochReadFunc func(context.Context, legacyauthorization.Input) (legacyauthorization.AuthorityEpochs, error)

func (f epochReadFunc) ReadAuthorizationEpochs(ctx context.Context, input legacyauthorization.Input) (legacyauthorization.AuthorityEpochs, error) {
	return f(ctx, input)
}

func stableEpochs() legacyauthorization.AuthorityEpochs {
	return legacyauthorization.AuthorityEpochs{Source: "local-auth-store", SubjectLink: 10, Membership: 20, Policy: 30}
}

func TestAuthorizationGrantEpochProvenance(t *testing.T) {
	epochs := stableEpochs()
	reads, rules := 0, 0
	adapter := legacyauthorization.Adapter{
		Store: authorizationStoreFunc(func(context.Context, string) (legacyauth.User, bool, error) {
			return legacyauth.User{SubjectID: "editor", RoleCode: "EDITOR"}, true, nil
		}),
		Rule: authorizationRuleFunc(func(context.Context, legacyauth.User, string, string) (bool, string, error) {
			rules++
			return true, "", nil
		}),
		EpochReader: epochReadFunc(func(_ context.Context, input legacyauthorization.Input) (legacyauthorization.AuthorityEpochs, error) {
			if input.SubjectID != "editor" || input.Action != "write" || input.Resource != "record-7" {
				t.Fatalf("版読取り対象が認可要求と異なります: %+v", input)
			}
			reads++
			return epochs, nil
		}),
	}
	input := legacyauthorization.Input{SubjectID: "editor", Action: "write", Resource: "record-7"}
	got, err := adapter.Invoke(context.Background(), legacyauthorization.AuthorizationCapability, input)
	if err != nil || got.Outcome != "allowed" {
		t.Fatalf("有効な版付き許可を返せません: %+v %v", got, err)
	}
	grant, ok := got.Payload.(legacyauthorization.AuthorizationGrant)
	if !ok || grant.Epochs == nil || *grant.Epochs != epochs || rules != 1 || reads != 2 {
		t.Fatalf("認可根拠の版を結び付けられません: grant=%+v rules=%d reads=%d", grant, rules, reads)
	}
	adapter.EpochReader = nil
	got, err = adapter.Invoke(context.Background(), legacyauthorization.AuthorizationCapability, input)
	legacy := got.Payload.(legacyauthorization.AuthorizationGrant)
	if err != nil || got.Outcome != "allowed" || legacy.Epochs != nil {
		t.Fatalf("従来の許可に存在しない版が付与されました: %+v %v", got, err)
	}
}

func TestAuthorizationEpochChangesDuringRoleEvaluation(t *testing.T) {
	for _, dimension := range []string{"subject-link", "membership", "policy", "source"} {
		t.Run(dimension, func(t *testing.T) {
			epochs := stableEpochs()
			adapter := legacyauthorization.Adapter{
				Store: authorizationStoreFunc(func(context.Context, string) (legacyauth.User, bool, error) {
					return legacyauth.User{SubjectID: "editor"}, true, nil
				}),
				Rule: authorizationRuleFunc(func(context.Context, legacyauth.User, string, string) (bool, string, error) {
					switch dimension {
					case "subject-link":
						epochs.SubjectLink++
					case "membership":
						epochs.Membership++
					case "policy":
						epochs.Policy++
					case "source":
						epochs.Source = "replacement-store"
					}
					return true, "", nil
				}),
				EpochReader: epochReadFunc(func(context.Context, legacyauthorization.Input) (legacyauthorization.AuthorityEpochs, error) {
					return epochs, nil
				}),
			}
			result, err := adapter.Invoke(context.Background(), legacyauthorization.AuthorizationCapability, legacyauthorization.Input{
				SubjectID: "editor", Action: "write", Resource: "record-7",
			})
			if err != nil || result.Outcome != "denied" {
				t.Fatalf("途中で変更された許可根拠から成功を返しました: %+v %v", result, err)
			}
			if denial, ok := result.Payload.(legacyauthorization.AuthorizationDenial); !ok || denial.Reason != "authority_changed" {
				t.Fatalf("版不一致が別の結果に変換されました: %#v", result.Payload)
			}
		})
	}
}

func TestAuthorizationEpochReaderFailureAndIncompleteEvidence(t *testing.T) {
	outage := errors.New("authority provider unavailable")
	for _, tt := range []struct {
		name string
		read func(int) (legacyauthorization.AuthorityEpochs, error)
	}{
		{"read-error", func(int) (legacyauthorization.AuthorityEpochs, error) {
			return legacyauthorization.AuthorityEpochs{}, outage
		}},
		{"missing-link", func(int) (legacyauthorization.AuthorityEpochs, error) {
			return legacyauthorization.AuthorityEpochs{Source: "db", Membership: 1, Policy: 1}, nil
		}},
		{"blank-source", func(int) (legacyauthorization.AuthorityEpochs, error) {
			return legacyauthorization.AuthorityEpochs{SubjectLink: 1, Membership: 1, Policy: 1}, nil
		}},
		{"invalid-source", func(int) (legacyauthorization.AuthorityEpochs, error) {
			return legacyauthorization.AuthorityEpochs{Source: "db\nmalformed", SubjectLink: 1, Membership: 1, Policy: 1}, nil
		}},
		{"second-read-error", func(n int) (legacyauthorization.AuthorityEpochs, error) {
			if n == 2 {
				return legacyauthorization.AuthorityEpochs{}, outage
			}
			return stableEpochs(), nil
		}},
		{"second-read-incomplete", func(n int) (legacyauthorization.AuthorityEpochs, error) {
			if n == 2 {
				return legacyauthorization.AuthorityEpochs{Source: "db", SubjectLink: 1, Policy: 1}, nil
			}
			return stableEpochs(), nil
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reads := 0
			adapter := legacyauthorization.Adapter{
				Store: authorizationStoreFunc(func(context.Context, string) (legacyauth.User, bool, error) {
					return legacyauth.User{SubjectID: "editor"}, true, nil
				}),
				Rule: authorizationRuleFunc(func(context.Context, legacyauth.User, string, string) (bool, string, error) {
					return true, "", nil
				}),
				EpochReader: epochReadFunc(func(context.Context, legacyauthorization.Input) (legacyauthorization.AuthorityEpochs, error) {
					reads++
					return tt.read(reads)
				}),
			}
			result, err := adapter.Invoke(context.Background(), legacyauthorization.AuthorizationCapability, legacyauthorization.Input{
				SubjectID: "editor", Action: "write", Resource: "record-7",
			})
			if err == nil || result.Outcome != "" {
				t.Fatalf("版取得障害を許可または業務拒否として返しました: %+v %v", result, err)
			}
			if tt.name == "read-error" || tt.name == "second-read-error" {
				if !errors.Is(err, outage) {
					t.Fatalf("原障害を保持していません: %v", err)
				}
			}
		})
	}
}

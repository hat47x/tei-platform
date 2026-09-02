package legacyauth_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/security/legacyauth"
	"tei/internal/semantic/capabilityinvoke"
)

type authenticateFunctionFunc func(context.Context, string, string) (legacyauth.FunctionResult, error)

func (f authenticateFunctionFunc) Authenticate(ctx context.Context, login, credential string) (legacyauth.FunctionResult, error) {
	return f(ctx, login, credential)
}

func TestLegacyAuthenticationFunctionAdapterMapsReturnCodesToCanonicalOutcomes(t *testing.T) {
	mapping := loadReferenceFunctionMapping(t)
	capability := referenceCapability(t, legacyauth.AuthenticationCapability)

	cases := []struct {
		name        string
		returnCode  string
		subjectID   string
		wantOutcome string
		wantReason  string
	}{
		{name: "認証成功", returnCode: "0", subjectID: "subject-order-editor", wantOutcome: "authenticated"},
		{name: "credential不一致", returnCode: "10", wantOutcome: "denied", wantReason: "credential_mismatch"},
		{name: "無効化済み", returnCode: "20", wantOutcome: "denied", wantReason: "account_disabled"},
		{name: "lock中", returnCode: "30", wantOutcome: "denied", wantReason: "account_locked"},
		{name: "有効期間切れ", returnCode: "40", wantOutcome: "denied", wantReason: "expired"},
		{name: "有効期間前", returnCode: "50", wantOutcome: "denied", wantReason: "not_yet_valid"},
		{name: "既存システム固有条件が不成立", returnCode: "60", wantOutcome: "denied", wantReason: "organization_specific_condition"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			adapter := legacyauth.FunctionAdapter{
				Mapping: mapping,
				Function: authenticateFunctionFunc(func(_ context.Context, login, credential string) (legacyauth.FunctionResult, error) {
					if login != "legacy-order-editor" {
						t.Fatalf("login=%q", login)
					}
					if credential != "runtime-only-credential" {
						t.Fatalf("credential=%q", credential)
					}
					return legacyauth.FunctionResult{ReturnCode: tt.returnCode, SubjectID: tt.subjectID}, nil
				}),
			}

			result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, legacyauth.Input{
				LoginID:    "legacy-order-editor",
				Credential: "runtime-only-credential",
			})
			if err != nil {
				t.Fatalf("legacy authentication functionの業務結果がExecution Errorになっています: %v", err)
			}
			if result.Mode != capabilityinvoke.ResultOutcome || string(result.Outcome) != tt.wantOutcome {
				t.Fatalf("result=%#v want outcome=%q", result, tt.wantOutcome)
			}
			if tt.wantOutcome == "authenticated" {
				payload, ok := result.Payload.(legacyauth.AuthenticatedSubject)
				if !ok || payload.SubjectID != tt.subjectID {
					t.Fatalf("authenticated payload=%#v", result.Payload)
				}
			} else {
				payload, ok := result.Payload.(legacyauth.AuthenticationDenial)
				if !ok || payload.Reason != tt.wantReason {
					t.Fatalf("denied payload=%#v want reason=%q", result.Payload, tt.wantReason)
				}
			}
			if strings.Contains(fmt.Sprintf("%#v", result), tt.returnCode) && tt.returnCode != "0" {
				t.Fatalf("legacy return codeがBusiness Resultへ漏れています: %#v", result)
			}
			if strings.Contains(fmt.Sprintf("%#v", result), "runtime-only-credential") {
				t.Fatalf("実行時credentialがBusiness Resultへ漏れています: %#v", result)
			}
		})
	}
}

func TestLegacyAuthenticationFunctionAdapterKeepsFunctionFailuresAsExecutionErrors(t *testing.T) {
	mapping := loadReferenceFunctionMapping(t)
	capability := referenceCapability(t, legacyauth.AuthenticationCapability)

	cases := []struct {
		name     string
		function legacyauth.AuthenticateFunction
	}{
		{
			name: "既存認証関数の実行障害",
			function: authenticateFunctionFunc(func(context.Context, string, string) (legacyauth.FunctionResult, error) {
				return legacyauth.FunctionResult{}, errors.New("legacy procedure unavailable")
			}),
		},
		{
			name: "未対応return code",
			function: authenticateFunctionFunc(func(context.Context, string, string) (legacyauth.FunctionResult, error) {
				return legacyauth.FunctionResult{ReturnCode: "999"}, nil
			}),
		},
		{
			name: "認証成功だがsubject identityなし",
			function: authenticateFunctionFunc(func(context.Context, string, string) (legacyauth.FunctionResult, error) {
				return legacyauth.FunctionResult{ReturnCode: "0"}, nil
			}),
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			adapter := legacyauth.FunctionAdapter{Mapping: mapping, Function: tt.function}
			result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, legacyauth.Input{
				LoginID:    "legacy-order-editor",
				Credential: "runtime-only-credential",
			})
			if err == nil {
				t.Fatalf("既存認証関数の実行異常がBusiness Resultになっています: %#v", result)
			}
			if result.Outcome == "denied" {
				t.Fatalf("実行異常がdeniedへ変換されています: result=%#v err=%v", result, err)
			}
		})
	}
}

func TestLegacyAuthenticationFunctionMappingKeepsRepositoryProvenanceWithoutSecrets(t *testing.T) {
	mapping := loadReferenceFunctionMapping(t)
	if mapping.Source != "legacy_security.authenticate_user" {
		t.Fatalf("mapping source=%q", mapping.Source)
	}
	if mapping.SourcePath() == "" {
		t.Fatal("function mappingのsource pathが残っていません")
	}
	if mapping.AuthenticatedCode != "0" {
		t.Fatalf("authenticated code=%q", mapping.AuthenticatedCode)
	}
	if mapping.DeniedCodes["30"] != "account_locked" {
		t.Fatalf("lock code mapping=%#v", mapping.DeniedCodes)
	}

	data, err := os.ReadFile(mapping.SourcePath())
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, forbidden := range []string{`"password"`, `"password_hash"`, `"credential_hash"`, `"token"`, `"secret"`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("function mapping fixtureにsecret material用fieldが含まれています: %s", forbidden)
		}
	}
}

func loadReferenceFunctionMapping(t *testing.T) legacyauth.FunctionMapping {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test fileの位置を取得できません")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	path := filepath.Join(root, "reference", "legacy-auth", "authenticate-function.json")
	mapping, err := legacyauth.LoadFunctionMapping(path)
	if err != nil {
		t.Fatalf("legacy authentication function mappingを読み込めません: %v", err)
	}
	return mapping
}

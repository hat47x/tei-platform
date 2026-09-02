package legacyauth_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"tei/internal/security/legacyauth"
	"tei/internal/semantic/capabilityinvoke"
)

type functionInvokerFunc func(context.Context, string, string) (legacyauth.FunctionResult, error)

func (f functionInvokerFunc) Authenticate(ctx context.Context, loginID, credential string) (legacyauth.FunctionResult, error) {
	return f(ctx, loginID, credential)
}

func TestLegacyAuthenticationFunctionAdapterMapsExistingResultCodes(t *testing.T) {
	capability := referenceCapability(t, legacyauth.AuthenticationCapability)

	cases := []struct {
		name        string
		result      legacyauth.FunctionResult
		wantOutcome string
		wantReason  string
	}{
		{name: "認証成功", result: legacyauth.FunctionResult{ResultCode: "00", SubjectID: "subject-function-001"}, wantOutcome: "authenticated"},
		{name: "credential不一致", result: legacyauth.FunctionResult{ResultCode: "10"}, wantOutcome: "denied", wantReason: "credential_mismatch"},
		{name: "無効化済み", result: legacyauth.FunctionResult{ResultCode: "20"}, wantOutcome: "denied", wantReason: "account_disabled"},
		{name: "lock中", result: legacyauth.FunctionResult{ResultCode: "30"}, wantOutcome: "denied", wantReason: "account_locked"},
		{name: "有効期間切れ", result: legacyauth.FunctionResult{ResultCode: "40"}, wantOutcome: "denied", wantReason: "expired"},
		{name: "有効期間前", result: legacyauth.FunctionResult{ResultCode: "50"}, wantOutcome: "denied", wantReason: "not_yet_valid"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			adapter := legacyauth.FunctionAdapter{Invoker: functionInvokerFunc(func(_ context.Context, loginID, credential string) (legacyauth.FunctionResult, error) {
				if loginID != "legacy-user-001" {
					t.Fatalf("loginID=%q", loginID)
				}
				if credential != "runtime-presented-value" {
					t.Fatalf("credential=%q", credential)
				}
				return tt.result, nil
			})}

			result, err := capabilityinvoke.Invoke(context.Background(), capability, adapter, legacyauth.Input{
				LoginID:    "legacy-user-001",
				Credential: "runtime-presented-value",
			})
			if err != nil {
				t.Fatalf("既存関数が返した業務結果がExecution Errorになっています: %v", err)
			}
			if string(result.Outcome) != tt.wantOutcome {
				t.Fatalf("outcome=%q want=%q", result.Outcome, tt.wantOutcome)
			}
			if tt.wantOutcome == "authenticated" {
				payload, ok := result.Payload.(legacyauth.AuthenticatedSubject)
				if !ok || payload.SubjectID != "subject-function-001" {
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
		})
	}
}

func TestLegacyAuthenticationFunctionAdapterKeepsFunctionFailuresAsExecutionErrors(t *testing.T) {
	capability := referenceCapability(t, legacyauth.AuthenticationCapability)

	cases := []struct {
		name    string
		adapter legacyauth.FunctionAdapter
	}{
		{
			name: "既存関数の呼び出し障害",
			adapter: legacyauth.FunctionAdapter{Invoker: functionInvokerFunc(func(context.Context, string, string) (legacyauth.FunctionResult, error) {
				return legacyauth.FunctionResult{}, errors.New("legacy function unavailable")
			})},
		},
		{
			name: "未知のresult code",
			adapter: legacyauth.FunctionAdapter{Invoker: functionInvokerFunc(func(context.Context, string, string) (legacyauth.FunctionResult, error) {
				return legacyauth.FunctionResult{ResultCode: "99"}, nil
			})},
		},
		{
			name: "成功結果にsubject identityがない",
			adapter: legacyauth.FunctionAdapter{Invoker: functionInvokerFunc(func(context.Context, string, string) (legacyauth.FunctionResult, error) {
				return legacyauth.FunctionResult{ResultCode: "00"}, nil
			})},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result, err := capabilityinvoke.Invoke(context.Background(), capability, tt.adapter, legacyauth.Input{
				LoginID:    "legacy-user-001",
				Credential: "runtime-presented-value",
			})
			if err == nil {
				t.Fatalf("既存関数の実行異常がBusiness Resultになっています: %#v", result)
			}
			if result.Outcome == "denied" {
				t.Fatalf("実行異常がdeniedへ変換されています: result=%#v err=%v", result, err)
			}
		})
	}
}

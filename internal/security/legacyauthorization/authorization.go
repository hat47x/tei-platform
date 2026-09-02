package legacyauthorization

import (
	"context"
	"fmt"

	"tei/internal/core"
	"tei/internal/security/legacyauth"
	"tei/internal/semantic/capabilityinvoke"
)

const AuthorizationCapability core.CanonicalID = "security.authorize"

// SubjectStoreは、Authenticationが返したsubject identityから、
// 既存システム側の認可判断に必要な利用者情報を取得する境界である。
type SubjectStore interface {
	FindBySubject(context.Context, string) (legacyauth.User, bool, error)
}

// Ruleは、既存システム固有の組織・role・個別条件を使って、
// Application上のaction / resourceを実行できるか判定する。
// 正常に判定できた拒否はallowed=falseで返し、判定処理自体を完了できない場合だけerrorを返す。
type Rule interface {
	Evaluate(context.Context, legacyauth.User, string, string) (allowed bool, denialReason string, err error)
}

type Input struct {
	SubjectID string
	Action    string
	Resource  string
}

type AuthorizationGrant struct {
	SubjectID string
	Action    string
	Resource  string
}

type AuthorizationDenial struct {
	Reason string
}

// Adapterは、既存の利用者情報と認可ruleをsecurity.authorizeへ接続するReference実装である。
// Application / Driverはorganizationやroleを直接読み取らず、subject / action / resourceだけを渡す。
type Adapter struct {
	Store SubjectStore
	Rule  Rule
}

func (a Adapter) Invoke(ctx context.Context, capability core.CanonicalID, raw any) (capabilityinvoke.BusinessResult, error) {
	if capability != AuthorizationCapability {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authorization adapterはcapability %sを扱えません", capability)
	}
	input, ok := raw.(Input)
	if !ok {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("security.authorizeの入力型がlegacyauthorization.Inputではありません")
	}
	if input.SubjectID == "" {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("security.authorizeにsubject identityがありません")
	}
	if input.Action == "" {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("security.authorizeにactionがありません")
	}
	if input.Resource == "" {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("security.authorizeにresourceがありません")
	}
	if a.Store == nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authorization subject storeが指定されていません")
	}
	if a.Rule == nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authorization ruleが指定されていません")
	}

	user, found, err := a.Store.FindBySubject(ctx, input.SubjectID)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authorization用の利用者情報を取得できません: %w", err)
	}
	if !found {
		return denied("not_authorized"), nil
	}

	allowed, reason, err := a.Rule.Evaluate(ctx, user, input.Action, input.Resource)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authorization ruleを評価できません: %w", err)
	}
	if !allowed {
		if reason == "" {
			reason = "not_authorized"
		}
		return denied(reason), nil
	}

	return capabilityinvoke.BusinessResult{
		Capability: AuthorizationCapability,
		Mode:       capabilityinvoke.ResultOutcome,
		Outcome:    "allowed",
		Payload: AuthorizationGrant{
			SubjectID: input.SubjectID,
			Action:    input.Action,
			Resource:  input.Resource,
		},
	}, nil
}

func denied(reason string) capabilityinvoke.BusinessResult {
	return capabilityinvoke.BusinessResult{
		Capability: AuthorizationCapability,
		Mode:       capabilityinvoke.ResultOutcome,
		Outcome:    "denied",
		Payload:    AuthorizationDenial{Reason: reason},
	}
}

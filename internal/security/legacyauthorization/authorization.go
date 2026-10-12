package legacyauthorization

import (
	"context"
	"fmt"
	"time"

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
	// Now は試験用の時刻注入。省略時は実行時の現在時刻を用いる。
	Now func() time.Time
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
	// 保存先が別の主体のレコードを返した場合は、役割を評価せず停止する。
	// 欠落・取り違えを業務上の拒否へ丸めず、保存先契約の障害として返す。
	if user.SubjectID != input.SubjectID {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authorization subject storeの主体が要求と一致しません")
	}
	// 認証後に変更された利用者の無効化・期限・ロックを役割評価の前に反映する。
	// この再判定は書込みの確定との原子性を意味しない。
	now := time.Now()
	if a.Now != nil {
		now = a.Now()
	}
	if reason := user.AccessDenialAt(now); reason != "" {
		return denied(reason), nil
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

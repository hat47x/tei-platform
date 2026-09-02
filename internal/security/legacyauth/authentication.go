package legacyauth

import (
	"context"
	"fmt"
	"time"

	"tei/internal/core"
	"tei/internal/semantic/capabilityinvoke"
)

const AuthenticationCapability core.CanonicalID = "security.authenticate"

// Userは、既存システム側に存在する認証用recordの最小Referenceである。
// TEI標準User schemaへ変換するための型ではなく、既存fieldを失わずadapterへ渡す境界として扱う。
type User struct {
	LoginID           string
	SubjectID         string
	CredentialRef     string
	Disabled          bool
	ValidFrom         time.Time
	ValidUntil        time.Time
	LoginFailureCount int
	LockedUntil       time.Time
	OrganizationCode  string
	RoleCode          string
}

// UserStoreは、既存schemaや既存APIから認証対象recordを取得する境界である。
// password等のsecret materialをUserへ載せず、credentialの参照だけを返す。
type UserStore interface {
	FindByLogin(context.Context, string) (User, bool, error)
}

// CredentialVerifierは、実行時に提示されたcredentialを既存の検証方式へ接続する。
// 検証対象のsecret値をCanonical Worldやlegacy User recordへコピーしない。
type CredentialVerifier interface {
	Verify(context.Context, string, string) (bool, error)
}

// LegacyRuleは、field対応だけでは表しきれない既存システム固有の認証条件を評価する。
// 成立しない場合は業務上のdeniedとしてreasonを返し、評価自体を完了できない場合だけerrorを返す。
type LegacyRule interface {
	Evaluate(context.Context, User, time.Time) (allowed bool, denialReason string, err error)
}

type Input struct {
	LoginID    string
	Credential string
}

type AuthenticatedSubject struct {
	SubjectID string
}

type AuthenticationDenial struct {
	Reason string
}

// Adapterは、既存認証recordとcredential検証をsecurity.authenticateへ接続するReference実装である。
// credential不一致、無効化、期限外、lock、既存rule不成立は正常なdenied Outcomeとして返す。
// StoreやVerifier、rule評価そのものの障害はExecution Errorとしてerrorへ残す。
type Adapter struct {
	Store    UserStore
	Verifier CredentialVerifier
	Rule     LegacyRule
	Now      func() time.Time
}

func (a Adapter) Invoke(ctx context.Context, capability core.CanonicalID, raw any) (capabilityinvoke.BusinessResult, error) {
	if capability != AuthenticationCapability {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authentication adapterはcapability %sを扱えません", capability)
	}
	input, ok := raw.(Input)
	if !ok {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("security.authenticateの入力型がlegacyauth.Inputではありません")
	}
	if a.Store == nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authentication user storeが指定されていません")
	}
	if a.Verifier == nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authentication credential verifierが指定されていません")
	}

	now := time.Now()
	if a.Now != nil {
		now = a.Now()
	}

	user, found, err := a.Store.FindByLogin(ctx, input.LoginID)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy user recordを取得できません: %w", err)
	}
	if !found {
		return denied("credential_mismatch"), nil
	}
	if user.SubjectID == "" {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy user %sにsubject identityがありません", input.LoginID)
	}
	if user.CredentialRef == "" {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy user %sにcredential参照がありません", input.LoginID)
	}
	if user.Disabled {
		return denied("account_disabled"), nil
	}
	if !user.ValidFrom.IsZero() && now.Before(user.ValidFrom) {
		return denied("not_yet_valid"), nil
	}
	if !user.ValidUntil.IsZero() && !now.Before(user.ValidUntil) {
		return denied("expired"), nil
	}
	if !user.LockedUntil.IsZero() && now.Before(user.LockedUntil) {
		return denied("account_locked"), nil
	}
	if a.Rule != nil {
		allowed, reason, err := a.Rule.Evaluate(ctx, user, now)
		if err != nil {
			return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authentication ruleを評価できません: %w", err)
		}
		if !allowed {
			if reason == "" {
				reason = "legacy_rule_denied"
			}
			return denied(reason), nil
		}
	}

	matched, err := a.Verifier.Verify(ctx, user.CredentialRef, input.Credential)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy credentialを検証できません: %w", err)
	}
	if !matched {
		return denied("credential_mismatch"), nil
	}

	return capabilityinvoke.BusinessResult{
		Capability: AuthenticationCapability,
		Mode:       capabilityinvoke.ResultOutcome,
		Outcome:    "authenticated",
		Payload:    AuthenticatedSubject{SubjectID: user.SubjectID},
	}, nil
}

func denied(reason string) capabilityinvoke.BusinessResult {
	return capabilityinvoke.BusinessResult{
		Capability: AuthenticationCapability,
		Mode:       capabilityinvoke.ResultOutcome,
		Outcome:    "denied",
		Payload:    AuthenticationDenial{Reason: reason},
	}
}

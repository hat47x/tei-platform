package effectfence

import (
	"context"
	"errors"
	"fmt"

	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
)

// PrepareWithLegacyAuthorization は、既存の security.authorize の判断を使って
// 試験用 MemoryStore の操作を準備する。役割規則は再実装しない。
// trustedSubject は認証済み実行文脈から得ること。要求本文にある主体を信用しない。
// この接続口には認可の版が渡らないため、本番の永続的な原子性は保証しない。
func (m *MemoryStore) PrepareWithLegacyAuthorization(
	ctx context.Context,
	trustedSubject string,
	req Request,
	authorizer capabilityinvoke.Adapter,
) (Permit, Result, error) {
	if err := ctx.Err(); err != nil {
		return Permit{}, Result{}, err
	}
	if m == nil {
		return Permit{}, Result{}, errors.New("保存先が指定されていません")
	}
	if !req.Scope.valid() || !validToken(req.OperationID) || !validToken(trustedSubject) {
		return Permit{}, Result{Disposition: Denied, Reason: "invalid-request"}, nil
	}
	if req.Scope.Subject != trustedSubject {
		return Permit{}, Result{Disposition: Denied, Reason: "subject-mismatch"}, nil
	}
	if authorizer == nil {
		return Permit{}, Result{}, errors.New("既存の認可アダプターが指定されていません")
	}

	raw, err := authorizer.Invoke(ctx, legacyauthorization.AuthorizationCapability, legacyauthorization.Input{
		SubjectID: trustedSubject,
		Action:    req.Scope.Action,
		Resource:  req.Scope.Resource,
	})
	if err != nil {
		return Permit{}, Result{}, fmt.Errorf("既存の認可処理を完了できません: %w", err)
	}
	authorization, err := checkLegacyAuthorization(req.Scope, raw)
	if err != nil || authorization.Disposition != Prepared {
		return Permit{}, authorization, err
	}
	// この後も MemoryStore.Commit で模擬権限版と保存先版を再検査する。
	// 実認可の版と書込みを同一確定境界で照合できることは意味しない。
	return m.Prepare(ctx, req)
}

// checkLegacyAuthorization は許可結果の型・機能ID・三要素を厳密に確認する。
// 返却された拒否理由には業務上の機密が含まれ得るため外へ出さない。
func checkLegacyAuthorization(scope Scope, result capabilityinvoke.BusinessResult) (Result, error) {
	if result.Capability != legacyauthorization.AuthorizationCapability || result.Mode != capabilityinvoke.ResultOutcome {
		return Result{}, errors.New("既存認可の実行結果契約が一致しません")
	}
	switch result.Outcome {
	case "denied":
		denial, ok := result.Payload.(legacyauthorization.AuthorizationDenial)
		if !ok || !validToken(denial.Reason) {
			return Result{}, errors.New("既存認可の拒否結果が不正です")
		}
		return Result{Disposition: Denied, Reason: "authorization-denied"}, nil
	case "allowed":
		grant, ok := result.Payload.(legacyauthorization.AuthorizationGrant)
		if !ok || grant.SubjectID != scope.Subject || grant.Action != scope.Action || grant.Resource != scope.Resource {
			return Result{}, errors.New("既存認可の許可範囲が要求と一致しません")
		}
		return Result{Disposition: Prepared}, nil
	default:
		return Result{}, errors.New("既存認可の結果種別が未定義です")
	}
}

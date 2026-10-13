//go:build tei_sqlite_reference && cgo

package effectfence

import (
	"context"
	"errors"
	"fmt"

	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
)

// ReceiptLookupStatus は、現在の認可の下で確定記録を確認した結果である。
// not-recorded はこの保存先に該当する確定記録がないことだけを示す。
// 外部APIの副作用が生じなかったと断定する根拠には使わない。
type ReceiptLookupStatus string

const (
	ReceiptRecorded    ReceiptLookupStatus = "recorded"
	ReceiptNotRecorded ReceiptLookupStatus = "not-recorded"
	ReceiptNotAllowed  ReceiptLookupStatus = "not-authorized"
)

type ReceiptLookupResult struct {
	Status  ReceiptLookupStatus
	Receipt Receipt
}

// LookupWithVersionedAuthorization は、再接続後にも使える試験用の確定記録照会。
// 旧Lookupと違い、認証済み主体、security.authorizeの許可とその版、
// DBに実在する認可根拠の版、版更新を強制する制約を必ず再検査する。
// 実DBの読取りはBEGIN IMMEDIATEの同じ境界に収め、失効の競合を直列化する。
// この結果はこのSQLiteファイルだけに関するもので、外部効果の保証ではない。
func (f *sqliteFixture) LookupWithVersionedAuthorization(
	ctx context.Context,
	trustedSubject string,
	req Request,
	authorizer capabilityinvoke.Adapter,
	expectedSource string,
) (ReceiptLookupResult, error) {
	if err := ctx.Err(); err != nil {
		return ReceiptLookupResult{}, err
	}
	if f == nil {
		return ReceiptLookupResult{}, errors.New("SQLite保存先が指定されていません")
	}
	if !req.Scope.valid() || !validToken(req.OperationID) || !validToken(trustedSubject) {
		return ReceiptLookupResult{Status: ReceiptNotAllowed}, nil
	}
	if req.Scope.Subject != trustedSubject {
		return ReceiptLookupResult{Status: ReceiptNotAllowed}, nil
	}
	if authorizer == nil || !validToken(expectedSource) {
		return ReceiptLookupResult{}, errors.New("認可アダプターまたは認可正本の識別子がありません")
	}
	raw, err := authorizer.Invoke(ctx, legacyauthorization.AuthorizationCapability, legacyauthorization.Input{
		SubjectID: trustedSubject,
		Action:    req.Scope.Action,
		Resource:  req.Scope.Resource,
	})
	if err != nil {
		return ReceiptLookupResult{}, fmt.Errorf("再照会時の認可処理を完了できません: %w", err)
	}
	outcome, err := checkLegacyAuthorization(req.Scope, raw)
	if err != nil {
		return ReceiptLookupResult{}, err
	}
	if outcome.Disposition != Prepared {
		return ReceiptLookupResult{Status: ReceiptNotAllowed}, nil
	}
	grant := raw.Payload.(legacyauthorization.AuthorizationGrant)
	epochs := grant.Epochs
	if epochs == nil || epochs.Source != expectedSource ||
		epochs.SubjectLink == 0 || epochs.Membership == 0 || epochs.Policy == 0 {
		return ReceiptLookupResult{}, errors.New("再照会時の認可版証拠が不完全か正本と一致しません")
	}
	expected := authorityEpochVersions{link: epochs.SubjectLink, membership: epochs.Membership, policy: epochs.Policy}

	f.mu.Lock()
	defer f.mu.Unlock()
	var lookup ReceiptLookupResult
	_, err = f.transaction(func() (Result, error) {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		// schemaCheckは接続内の任意設定なので信用しない。再接続でも直接検証する。
		if err := f.verifyRequiredSQLiteLegacyEpochTriggers(); err != nil {
			return Result{}, err
		}
		row, exists, err := f.query("SELECT allowed FROM authority WHERE scope=" + scopeSQL(req.Scope))
		if err != nil {
			return Result{}, err
		}
		if !exists || row[0] != "1" {
			lookup.Status = ReceiptNotAllowed
			return Result{}, nil
		}
		current, active, err := f.readAuthorityDependencies(req.Scope)
		if err != nil {
			return Result{}, err
		}
		if !active || !expected.matches(current) {
			lookup.Status = ReceiptNotAllowed
			return Result{}, nil
		}
		recorded, found, err := f.lookupCommitted(req)
		if err != nil {
			return Result{}, err
		}
		if !found {
			lookup.Status = ReceiptNotRecorded
			return Result{}, nil
		}
		if recorded.Disposition != Replayed {
			lookup.Status = ReceiptNotAllowed
			return Result{}, nil
		}
		lookup = ReceiptLookupResult{Status: ReceiptRecorded, Receipt: recorded.Receipt}
		return Result{}, nil
	})
	if err != nil {
		// 保存先が不明なときに、部分的な確定記録を外へ出さない。
		return ReceiptLookupResult{}, err
	}
	return lookup, nil
}

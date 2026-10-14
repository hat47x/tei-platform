//go:build tei_sqlite_reference && cgo

package effectfence

import (
	"context"
	"errors"
	"fmt"

	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
)

// authorityEpochVersions は、権限評価で確認した正本の版を
// SQLite保存側が保留済み操作へ受け渡すための値型である。
type authorityEpochVersions struct {
	link       uint64
	membership uint64
	policy     uint64
}

func (e authorityEpochVersions) matches(current map[string]authorityDependencyState) bool {
	return e.link != 0 && e.membership != 0 && e.policy != 0 &&
		current["subject-link"] == (authorityDependencyState{version: e.link, allowed: true}) &&
		current["membership"] == (authorityDependencyState{version: e.membership, allowed: true}) &&
		current["policy"] == (authorityDependencyState{version: e.policy, allowed: true})
}

// PrepareWithVersionedAuthorization は認可の結果に版根拠を要求し、SQLiteの
// 保留操作へその版を束縛するテスト用の接続口。
// expectedSource は配備側が信頼して固定する正本識別子であり、要求本文に含めない。
// この接続口だけではReadAuthorizationEpochsの正本性や、全変更経路で
// 版更新が行われることを保証できない。
func (f *sqliteFixture) PrepareWithVersionedAuthorization(
	ctx context.Context,
	trustedSubject string,
	req Request,
	authorizer capabilityinvoke.Adapter,
	expectedSource string,
) (any, Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, Result{}, err
	}
	if f == nil {
		return nil, Result{}, errors.New("SQLite保存先が指定されていません")
	}
	if !req.Scope.valid() || !validToken(req.OperationID) || !validToken(trustedSubject) {
		return nil, Result{Disposition: Denied, Reason: "invalid-request"}, nil
	}
	if req.Scope.Subject != trustedSubject {
		return nil, Result{Disposition: Denied, Reason: "subject-mismatch"}, nil
	}
	if authorizer == nil || !validToken(expectedSource) {
		return nil, Result{}, errors.New("認可アダプターまたは認可正本の識別子がありません")
	}

	raw, err := authorizer.Invoke(ctx, legacyauthorization.AuthorizationCapability, legacyauthorization.Input{
		SubjectID: trustedSubject,
		Action:    req.Scope.Action,
		Resource:  req.Scope.Resource,
	})
	if err != nil {
		return nil, Result{}, fmt.Errorf("認可処理を完了できません: %w", err)
	}
	outcome, err := checkLegacyAuthorization(req.Scope, raw)
	if err != nil || outcome.Disposition != Prepared {
		return nil, outcome, err
	}
	grant := raw.Payload.(legacyauthorization.AuthorizationGrant)
	epochs := grant.Epochs
	if epochs == nil || epochs.Source != expectedSource ||
		epochs.SubjectLink == 0 || epochs.Membership == 0 || epochs.Policy == 0 {
		return nil, Result{}, errors.New("認可正本の版証拠が不完全か保存先と一致しません")
	}
	// 既存の許可だけでは永続的保証にならないため、3種類の照合を必須にする。
	f.RequireAuthorityDependencies()
	expected := authorityEpochVersions{link: epochs.SubjectLink, membership: epochs.Membership, policy: epochs.Policy}
	return f.prepareWithExpectedEpochs(ctx, req, &expected)
}

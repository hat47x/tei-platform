package effectfence

import (
	"context"
	"fmt"
)

// GuaranteeUnavailableError は実行時ポリシーに必要な確定保証がないことを表す。
// 業務拒否や未確定の外部書込みの状態に読み替えてはならない。
type GuaranteeUnavailableError struct {
	Required Guarantee
	Reason   string
}

func (e *GuaranteeUnavailableError) Error() string {
	return fmt.Sprintf("必要な副作用確定保証 %s を確認できません（%s）", e.Required, e.Reason)
}

// CommitWithRequirement は、実行可能な保存先の契約を検査してから確定を委ねる。
// この入口は宣言内容の真偽を監査しない。呼出元が信頼できるAdapterから取得した
// 正しい契約と、そこで実装された確定処理を結び付けなければ安全にはならない。
func CommitWithRequirement(ctx context.Context, contract BoundaryContract, required Guarantee, apply func(context.Context) (Result, error)) (Result, error) {
	return CommitWithVerifiedRequirement(ctx, contract, required, nil, apply)
}

// CommitWithVerifiedRequirement は配備側で検証されたProvider証拠を確定要求に結び付ける。
// 検証器は実行するAdapterと独立した信頼元から供給しなければならない。
func CommitWithVerifiedRequirement(ctx context.Context, contract BoundaryContract, required Guarantee, verify EvidenceVerifier, apply func(context.Context) (Result, error)) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	assessment := AssessVerifiedBoundary(contract, required, verify)
	if !assessment.Allowed {
		return Result{}, &GuaranteeUnavailableError{Required: required, Reason: assessment.Reason}
	}
	if apply == nil {
		return Result{}, fmt.Errorf("確定処理が指定されていません")
	}
	return apply(ctx)
}

package capabilityinvoke

import (
	"context"
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// ResultModeは、Capabilityを正常に実行した結果が単一outputなのか、
// 名前付きoutcomeなのかを区別する。
type ResultMode string

const (
	ResultOutput  ResultMode = "output"
	ResultOutcome ResultMode = "outcome"
)

// BusinessResultは、Capabilityを正常に実行できたときの業務上の結果を表す。
// 実行そのものを完了できなかった状態はこの構造へ変換せず、Go errorとして返す。
type BusinessResult struct {
	Capability core.CanonicalID
	Mode       ResultMode
	Outcome    core.LocalName
	Payload    any
}

// Adapterは、Semantic Capabilityを具体的な実装へ接続する最小境界である。
//
// credential不一致や業務上のrejectionのように、処理を正常に完了して得られた結末は
// BusinessResultとして返す。Database接続失敗、Binding異常、adapter障害など、
// Invocationを完了できなかった状態だけをerrorとして返す。
type Adapter interface {
	Invoke(context.Context, core.CanonicalID, any) (BusinessResult, error)
}

// InvokeはAdapterを呼び出し、返されたBusiness ResultがCanonical Capabilityの
// result契約に従っていることを確認する。
//
// Adapterが返したerror、未定義outcome、result modeの不一致、Capability identityの
// 不一致は、業務OutcomeではなくExecution Errorとして呼び出し側へ返す。
func Invoke(ctx context.Context, capability semantic.Capability, adapter Adapter, input any) (BusinessResult, error) {
	if adapter == nil {
		return BusinessResult{}, fmt.Errorf("capability %sを実行するAdapterが指定されていません", capability.ID)
	}

	result, err := adapter.Invoke(ctx, capability.ID, input)
	if err != nil {
		return BusinessResult{}, fmt.Errorf("capability %sの実行を完了できませんでした: %w", capability.ID, err)
	}
	if err := ValidateBusinessResult(capability, result); err != nil {
		return BusinessResult{}, err
	}
	return result, nil
}

// ValidateBusinessResultは、正常応答側のBusiness ResultがCapabilityのCanonical契約と
// 一致していることだけを確認する。Payloadの具体的な値表現や型変換はAdapter側の責務とし、
// Driverより先に汎用的な業務データ値モデルを固定しない。
func ValidateBusinessResult(capability semantic.Capability, result BusinessResult) error {
	if result.Capability != capability.ID {
		return fmt.Errorf("capability %sの実行結果が別のCapability %sを名乗っています", capability.ID, result.Capability)
	}

	if capability.Output != "" {
		if result.Mode != ResultOutput {
			return fmt.Errorf("capability %sは単一output契約ですが、result modeは%qです", capability.ID, result.Mode)
		}
		if result.Outcome != "" {
			return fmt.Errorf("capability %sの単一output結果にoutcome %qを指定できません", capability.ID, result.Outcome)
		}
		return nil
	}

	if len(capability.Outcomes) == 0 {
		return fmt.Errorf("capability %sには実行結果契約が定義されていません", capability.ID)
	}
	if result.Mode != ResultOutcome {
		return fmt.Errorf("capability %sは名前付きoutcome契約ですが、result modeは%qです", capability.ID, result.Mode)
	}
	if result.Outcome == "" {
		return fmt.Errorf("capability %sの名前付きoutcome結果にoutcome名がありません", capability.ID)
	}

	for _, outcome := range capability.Outcomes {
		if outcome.Name == result.Outcome {
			if result.Payload == nil {
				return fmt.Errorf("capability %sのoutcome %qにpayloadがありません", capability.ID, result.Outcome)
			}
			return nil
		}
	}
	return fmt.Errorf("capability %sはoutcome %qを定義していません", capability.ID, result.Outcome)
}

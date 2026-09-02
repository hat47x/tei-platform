package legacyauth

import (
	"context"
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic/capabilityinvoke"
)

// FunctionResultは、既存関数・stored procedure・外部API等が返す
// Reference用の認証結果を表す。ResultCodeは既存実装固有の値であり、
// TEIの標準Outcome名として扱わない。
type FunctionResult struct {
	ResultCode string
	SubjectID  string
}

// FunctionInvokerは、既存の関数呼び出し型認証を実行する境界である。
// credentialは実行時だけ渡し、Canonical WorldやBusiness Resultへ保存しない。
type FunctionInvoker interface {
	Authenticate(context.Context, string, string) (FunctionResult, error)
}

// FunctionAdapterは、既存関数が返すresult codeをsecurity.authenticateの
// Business Resultへ対応付けるReference adapterである。
//
// ここで扱うresult codeはこのReference固有の例であり、TEI共通の認証コード体系ではない。
// 未知のcodeや関数呼び出し障害はdeniedへ丸めず、Execution Errorとして返す。
type FunctionAdapter struct {
	Invoker FunctionInvoker
}

func (a FunctionAdapter) Invoke(ctx context.Context, capability core.CanonicalID, raw any) (capabilityinvoke.BusinessResult, error) {
	if capability != AuthenticationCapability {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authentication function adapterはcapability %sを扱えません", capability)
	}
	input, ok := raw.(Input)
	if !ok {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("security.authenticateの入力型がlegacyauth.Inputではありません")
	}
	if a.Invoker == nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authentication function invokerが指定されていません")
	}

	result, err := a.Invoker.Authenticate(ctx, input.LoginID, input.Credential)
	if err != nil {
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authentication functionを実行できません: %w", err)
	}

	switch result.ResultCode {
	case "00":
		if result.SubjectID == "" {
			return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authentication functionの成功結果にsubject identityがありません")
		}
		return capabilityinvoke.BusinessResult{
			Capability: AuthenticationCapability,
			Mode:       capabilityinvoke.ResultOutcome,
			Outcome:    "authenticated",
			Payload:    AuthenticatedSubject{SubjectID: result.SubjectID},
		}, nil
	case "10":
		return denied("credential_mismatch"), nil
	case "20":
		return denied("account_disabled"), nil
	case "30":
		return denied("account_locked"), nil
	case "40":
		return denied("expired"), nil
	case "50":
		return denied("not_yet_valid"), nil
	default:
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("legacy authentication functionが未知のresult code %qを返しました", result.ResultCode)
	}
}

package realization

import (
	"fmt"

	"tei/internal/core"
)

// BindingIdentityは、Application内で一つのCapabilityをどの実装へ接続するかを識別する。
// v1では新しい任意IDを増やさず、既存Validationが重複判定に使ってきた
// Application + Capabilityの組をそのまま安定したidentityとして扱う。
func BindingIdentity(applicationID, capabilityID core.CanonicalID) string {
	return string(applicationID) + "/" + string(capabilityID)
}

// Identityは、このCustom Go BindingのApplication-scoped identityを返す。
func (b CustomGoBinding) Identity() string {
	return BindingIdentity(b.Application, b.Capability)
}

// ResolveCustomGoBindingは、ApplicationとCapabilityを明示して一つのBindingを解決する。
// 呼び出し側がfile pathやGo symbolを推測して実装を選ばないための最小入口である。
// Bindingが存在しない場合や複数存在する場合は、業務結果ではなく実行準備上の失敗としてerrorを返す。
func ResolveCustomGoBinding(bindings []CustomGoBinding, applicationID, capabilityID core.CanonicalID) (CustomGoBinding, error) {
	var found CustomGoBinding
	count := 0
	for _, binding := range bindings {
		if binding.Application != applicationID || binding.Capability != capabilityID {
			continue
		}
		found = binding
		count++
	}

	switch count {
	case 0:
		return CustomGoBinding{}, fmt.Errorf("custom Go binding %s is not defined", BindingIdentity(applicationID, capabilityID))
	case 1:
		return found, nil
	default:
		return CustomGoBinding{}, fmt.Errorf("custom Go binding %s is ambiguous: %d definitions", BindingIdentity(applicationID, capabilityID), count)
	}
}

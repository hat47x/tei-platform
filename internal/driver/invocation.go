package driver

import (
	"context"
	"fmt"

	"tei/internal/core"
	"tei/internal/realization"
	"tei/internal/semantic/capabilityinvoke"
)

// BindingAdapterResolverは、解決済みのBindingに対応する実行Adapterを返す最小境界である。
//
// Driverは、この境界の内側にあるlegacy schema、外部サービス、Go hookの探索方法を知らない。
// Bindingのidentityから実装を選ぶ責務をcomposition側へ残すことで、Driver固有の実装選択規則を増やさない。
type BindingAdapterResolver interface {
	ResolveAdapter(context.Context, realization.CustomGoBinding) (capabilityinvoke.Adapter, error)
}

// CapabilityInvocationは、Driverが明示的なBindingを経由してCapabilityを実行した結果を表す。
// 実行時入力は保持しない。credential等のsecretをScenario Evidenceへ誤って残さないため、
// 追跡可能なBinding解決結果とBusiness Resultだけを返す。
type CapabilityInvocation struct {
	Binding CapabilityBindingResolution
	Result  capabilityinvoke.BusinessResult
}

// InvokeCapabilityは、Application + Capability identityからBindingを解決し、
// そのBindingに対応するAdapterを通してCanonical Capabilityを実行する。
//
// 業務上のdeniedやrejectionはcapabilityinvoke.BusinessResultとして返る。
// Binding解決失敗、Adapter解決失敗、Adapter障害、Canonical result契約違反は
// Invocationを完了できないExecution Errorとしてerrorに残す。
func (r Resolver) InvokeCapability(
	ctx context.Context,
	applicationID, capabilityID core.CanonicalID,
	adapters BindingAdapterResolver,
	input any,
) (CapabilityInvocation, error) {
	binding, err := r.ResolveCapabilityBinding(applicationID, capabilityID)
	if err != nil {
		return CapabilityInvocation{}, err
	}
	if adapters == nil {
		return CapabilityInvocation{}, fmt.Errorf("Binding %sに対応するAdapter resolverが指定されていません", binding.Binding.Identity())
	}

	adapter, err := adapters.ResolveAdapter(ctx, binding.Binding)
	if err != nil {
		return CapabilityInvocation{}, fmt.Errorf("Binding %sに対応するAdapterを解決できません: %w", binding.Binding.Identity(), err)
	}
	if adapter == nil {
		return CapabilityInvocation{}, fmt.Errorf("Binding %sに対応するAdapterがありません", binding.Binding.Identity())
	}

	result, err := capabilityinvoke.Invoke(ctx, binding.Capability, adapter, input)
	if err != nil {
		return CapabilityInvocation{}, fmt.Errorf("Binding %sからCapability %sの実行を完了できませんでした: %w", binding.Binding.Identity(), binding.Capability.ID, err)
	}
	return CapabilityInvocation{Binding: binding, Result: result}, nil
}

package driver

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/realization"
	"tei/internal/semantic"
)

// ResolvedCapabilityBindingは、Driverが実行準備として解決した
// Semantic CapabilityとApplication-scoped Bindingを保持する。
//
// Binding先のGo hook名やadapter file名はrealization層の派生情報であり、
// DriverはそれらをidentityとしてCapabilityや実装を探索しない。
type ResolvedCapabilityBinding struct {
	Capability semantic.Capability
	Binding    realization.CustomGoBinding
}

// ResolveCapabilityBindingは、ApplicationとCapabilityのCanonical identityを起点に、
// Semantic Capabilityと対応するCustom Go Bindingを一意に解決する。
//
// command、Authentication、Authorizationで同じ解決規則を利用し、
// Capabilityごとの特別な探索規則をDriverへ持ち込まない。
func (r Resolver) ResolveCapabilityBinding(
	applicationID core.CanonicalID,
	capabilityID core.CanonicalID,
) (ResolvedCapabilityBinding, error) {
	app, err := r.applicationByID(applicationID)
	if err != nil {
		return ResolvedCapabilityBinding{}, err
	}

	capability, err := r.capabilityByID(capabilityID)
	if err != nil {
		return ResolvedCapabilityBinding{}, fmt.Errorf("Capability %sを解決できません: %w", capabilityID, err)
	}

	binding, err := realization.ResolveCustomGoBinding(r.world.Bindings, app.ID, capability.ID)
	if err != nil {
		return ResolvedCapabilityBinding{}, fmt.Errorf("Capability %sのBindingを解決できません: %w", capabilityID, err)
	}

	return ResolvedCapabilityBinding{
		Capability: capability,
		Binding:    binding,
	}, nil
}

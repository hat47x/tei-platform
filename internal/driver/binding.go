package driver

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/realization"
	"tei/internal/semantic"
)

// CapabilityBindingResolutionは、Applicationから一つのSemantic Capabilityと
// その明示的なCustom Go Bindingを解決した結果を表す。
//
// DriverはBindingを解決する前にhook名やadapter file名を推測しない。
// それらは、解決済みBindingの実装詳細として必要な段階で参照する。
type CapabilityBindingResolution struct {
	Application application.Application
	Capability  semantic.Capability
	Binding     realization.CustomGoBinding
}

// ResolveCapabilityBindingは、Application + Capability identityを起点に、
// 既存の明示的なCustom Go Bindingを一意に解決する。
//
// Bindingが0件または複数件の場合は、Driverが候補を推測して選ばず、
// Capabilityを実行する準備を完了できない状態としてerrorを返す。
func (r Resolver) ResolveCapabilityBinding(applicationID, capabilityID core.CanonicalID) (CapabilityBindingResolution, error) {
	app, err := r.applicationByID(applicationID)
	if err != nil {
		return CapabilityBindingResolution{}, err
	}
	capability, err := r.capabilityByID(capabilityID)
	if err != nil {
		return CapabilityBindingResolution{}, err
	}
	binding, err := realization.ResolveCustomGoBinding(r.world.Bindings, app.ID, capability.ID)
	if err != nil {
		return CapabilityBindingResolution{}, fmt.Errorf("Application %sからCapability %sのBindingを解決できません: %w", app.ID, capability.ID, err)
	}
	return CapabilityBindingResolution{
		Application: app,
		Capability:  capability,
		Binding:     binding,
	}, nil
}

// CommandBindingResolutionは、command PatternからCapabilityとBindingまでを
// Canonical identityの参照順に解決した結果を表す。
type CommandBindingResolution struct {
	Pattern PatternResolution
	Binding CapabilityBindingResolution
}

// ResolveCommandBindingは、command Patternを起点に
// Pattern -> Capability -> Application-scoped Bindingの順で解決する。
//
// Patternからfile pathやGo symbolを直接導出してimplementationを発見することはしない。
func (r Resolver) ResolveCommandBinding(applicationID, patternID core.CanonicalID) (CommandBindingResolution, error) {
	pattern, err := r.ResolvePattern(applicationID, patternID)
	if err != nil {
		return CommandBindingResolution{}, err
	}
	if pattern.Pattern.Kind != application.PatternCommand {
		return CommandBindingResolution{}, fmt.Errorf("Pattern %sはcommandではありません: %s", pattern.Pattern.ID, pattern.Pattern.Kind)
	}
	if pattern.Capability == nil {
		return CommandBindingResolution{}, fmt.Errorf("command Pattern %sのCapabilityが解決されていません", pattern.Pattern.ID)
	}
	binding, err := r.ResolveCapabilityBinding(applicationID, pattern.Capability.ID)
	if err != nil {
		return CommandBindingResolution{}, fmt.Errorf("command Pattern %sのBindingを解決できません: %w", pattern.Pattern.ID, err)
	}
	return CommandBindingResolution{Pattern: pattern, Binding: binding}, nil
}

package main

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/realization"
)

const (
	runtimeCapabilitySelectionSchema    = "tei.reference.runtime-capability-selection/v0"
	runtimeCapabilitySelectionAuthority = "realization.contract-query-executor-binding/v1"
)

type RuntimeCapabilitySelectionBinding struct {
	Schema                              string                          `json:"schema"`
	Authority                           string                          `json:"authority"`
	Environment                         core.CanonicalID                `json:"environment"`
	Instance                            core.CanonicalID                `json:"instance"`
	Target                              realization.ContractQueryTarget `json:"target"`
	ExecutorCapability                  core.CanonicalID                `json:"executorCapability"`
	PluginID                            string                          `json:"pluginId"`
	PluginVersion                       string                          `json:"pluginVersion"`
	CapabilityLockFingerprint           string                          `json:"capabilityLockFingerprint"`
	RuntimeCapabilityBindingFingerprint string                          `json:"runtimeCapabilityBindingFingerprint"`
	SelectionFingerprint                string                          `json:"selectionFingerprint"`
}

func resolveLegacyContractQueryCapabilityForSelection(
	selected realization.ResolvedContractQueryExecutor,
	definition plugin.Definition,
	mapping LegacyCapabilityMapping,
	evidence []ConformanceEvidence,
) (CapabilityResolutionLock, error) {
	if mapping.Scope != ScopeApplicationRuntime {
		return CapabilityResolutionLock{}, fmt.Errorf(
			"selected Contract Query capability requires application-runtime scope, got %q",
			mapping.Scope,
		)
	}
	if mapping.Kind != plugin.CapabilityContractQueryExecutor {
		return CapabilityResolutionLock{}, fmt.Errorf(
			"selected Contract Query capability mapping kind %q is not contract-query-executor",
			mapping.Kind,
		)
	}
	if selected.Binding.Environment == "" ||
		selected.Binding.Instance == "" ||
		!selected.Binding.Target.Valid() {
		return CapabilityResolutionLock{}, fmt.Errorf(
			"resolved Contract Query executor selection identity is incomplete",
		)
	}
	if selected.Instance.ID != selected.Binding.Instance {
		return CapabilityResolutionLock{}, fmt.Errorf(
			"resolved Contract Query executor instance %s differs from binding instance %s",
			selected.Instance.ID,
			selected.Binding.Instance,
		)
	}
	if selected.Instance.Definition != selected.Binding.Target.Database {
		return CapabilityResolutionLock{}, fmt.Errorf(
			"resolved Contract Query executor instance realizes %s but target belongs to %s",
			selected.Instance.Definition,
			selected.Binding.Target.Database,
		)
	}
	if !selected.Capability.Capability.SupportsDatabaseProvider(selected.Instance.Provider) {
		return CapabilityResolutionLock{}, fmt.Errorf(
			"resolved Contract Query executor capability %s does not support instance provider %s",
			selected.Capability.Capability.ID,
			selected.Instance.Provider,
		)
	}
	if selected.Binding.Executor != selected.Capability.Capability.ID {
		return CapabilityResolutionLock{}, fmt.Errorf(
			"resolved Contract Query executor binding selects %s but capability is %s",
			selected.Binding.Executor,
			selected.Capability.Capability.ID,
		)
	}
	if selected.Capability.Capability.Kind != mapping.Kind ||
		selected.Capability.Capability.ID != mapping.CapabilityID {
		return CapabilityResolutionLock{}, fmt.Errorf(
			"resolved Contract Query executor capability %s/%s differs from migration mapping %s/%s",
			selected.Capability.Capability.Kind,
			selected.Capability.Capability.ID,
			mapping.Kind,
			mapping.CapabilityID,
		)
	}
	if selected.Capability.Plugin.ID != definition.ID ||
		selected.Capability.Plugin.Version != definition.Version {
		return CapabilityResolutionLock{}, fmt.Errorf(
			"resolved Contract Query executor Plugin %s@%s differs from authoritative definition %s@%s",
			selected.Capability.Plugin.ID,
			selected.Capability.Plugin.Version,
			definition.ID,
			definition.Version,
		)
	}

	projected, err := projectLegacyCapabilityProvider(definition, mapping)
	if err != nil {
		return CapabilityResolutionLock{}, err
	}
	selectedDeclarationFingerprint, err := normalizedLegacyCapabilityFingerprint(
		selected.Capability.Capability,
	)
	if err != nil {
		return CapabilityResolutionLock{}, err
	}
	if selectedDeclarationFingerprint != projected.SourceDeclarationFingerprint {
		return CapabilityResolutionLock{}, fmt.Errorf(
			"resolved Contract Query executor capability was not derived from the authoritative legacy declaration",
		)
	}

	contractVersion, err := parseVersion(mapping.Contract.Version)
	if err != nil {
		return CapabilityResolutionLock{}, err
	}
	input := ResolveInput{
		Scope: ScopeApplicationRuntime,
		Requirement: Requirement{
			ContractID:      mapping.Contract.ID,
			CompatibleMajor: contractVersion[0],
		},
		Contracts: []Contract{mapping.Contract},
		Providers: []Provider{projected},
		Evidence:  append([]ConformanceEvidence(nil), evidence...),
	}
	resolution, err := resolveCapability(input)
	if err != nil {
		return CapabilityResolutionLock{}, err
	}
	lock, err := buildCapabilityResolutionLock(input, resolution)
	if err != nil {
		return CapabilityResolutionLock{}, err
	}
	if lock.Wiring != nil {
		return CapabilityResolutionLock{}, fmt.Errorf(
			"target-selected runtime capability must not add generic provider wiring",
		)
	}
	return lock, nil
}

func buildRuntimeCapabilitySelectionBinding(
	runtimeBinding RuntimeCapabilityBinding,
	capabilityLock CapabilityResolutionLock,
	selected realization.ResolvedContractQueryExecutor,
) (RuntimeCapabilitySelectionBinding, error) {
	if err := validateRuntimeCapabilityBinding(runtimeBinding); err != nil {
		return RuntimeCapabilitySelectionBinding{}, err
	}
	if err := validateCapabilityResolutionLock(capabilityLock); err != nil {
		return RuntimeCapabilitySelectionBinding{}, err
	}
	if capabilityLock.Scope != ScopeApplicationRuntime {
		return RuntimeCapabilitySelectionBinding{}, fmt.Errorf(
			"runtime capability selection requires application-runtime scope, got %q",
			capabilityLock.Scope,
		)
	}
	if capabilityLock.Wiring != nil {
		return RuntimeCapabilitySelectionBinding{}, fmt.Errorf(
			"target-specific runtime selection cannot also use generic capability wiring",
		)
	}
	if runtimeBinding.CapabilityLockFingerprint != capabilityLock.LockFingerprint {
		return RuntimeCapabilitySelectionBinding{}, fmt.Errorf(
			"runtime capability binding references a different capability lock",
		)
	}
	if selected.Binding.Environment != runtimeBinding.Environment ||
		selected.Binding.Instance == "" ||
		selected.Instance.ID != selected.Binding.Instance ||
		!selected.Binding.Target.Valid() {
		return RuntimeCapabilitySelectionBinding{}, fmt.Errorf(
			"resolved Contract Query executor selection identity is inconsistent",
		)
	}
	if selected.Binding.Executor != selected.Capability.Capability.ID {
		return RuntimeCapabilitySelectionBinding{}, fmt.Errorf(
			"resolved Contract Query executor binding differs from selected capability",
		)
	}
	if selected.Instance.Definition != selected.Binding.Target.Database {
		return RuntimeCapabilitySelectionBinding{}, fmt.Errorf(
			"resolved Contract Query executor instance and target Database differ",
		)
	}
	if !selected.Capability.Capability.SupportsDatabaseProvider(selected.Instance.Provider) {
		return RuntimeCapabilitySelectionBinding{}, fmt.Errorf(
			"resolved Contract Query executor capability does not support instance provider",
		)
	}
	if string(selected.Capability.Plugin.ID) != runtimeBinding.PluginID ||
		selected.Capability.Plugin.Version != runtimeBinding.PluginVersion ||
		string(selected.Capability.Plugin.ID) != capabilityLock.Provider.PluginID ||
		selected.Capability.Plugin.Version != capabilityLock.Provider.PluginVersion {
		return RuntimeCapabilitySelectionBinding{}, fmt.Errorf(
			"resolved Contract Query executor Plugin differs from capability/runtime binding",
		)
	}

	binding := RuntimeCapabilitySelectionBinding{
		Schema:                              runtimeCapabilitySelectionSchema,
		Authority:                           runtimeCapabilitySelectionAuthority,
		Environment:                         selected.Binding.Environment,
		Instance:                            selected.Binding.Instance,
		Target:                              selected.Binding.Target,
		ExecutorCapability:                  selected.Binding.Executor,
		PluginID:                            string(selected.Capability.Plugin.ID),
		PluginVersion:                       selected.Capability.Plugin.Version,
		CapabilityLockFingerprint:           capabilityLock.LockFingerprint,
		RuntimeCapabilityBindingFingerprint: runtimeBinding.BindingFingerprint,
	}
	binding.SelectionFingerprint = runtimeCapabilitySelectionFingerprint(binding)
	return binding, nil
}

func verifyRuntimeCapabilitySelectionBinding(
	binding RuntimeCapabilitySelectionBinding,
	runtimeBinding RuntimeCapabilityBinding,
	capabilityLock CapabilityResolutionLock,
	selected realization.ResolvedContractQueryExecutor,
) error {
	if err := validateRuntimeCapabilitySelectionBinding(binding); err != nil {
		return err
	}
	expected, err := buildRuntimeCapabilitySelectionBinding(
		runtimeBinding,
		capabilityLock,
		selected,
	)
	if err != nil {
		return err
	}
	if binding != expected {
		return fmt.Errorf(
			"runtime capability selection binding differs from current authoritative selection",
		)
	}
	return nil
}

func validateRuntimeCapabilitySelectionBinding(
	binding RuntimeCapabilitySelectionBinding,
) error {
	if binding.Schema != runtimeCapabilitySelectionSchema {
		return fmt.Errorf(
			"runtime capability selection schema %q is not supported",
			binding.Schema,
		)
	}
	if binding.Authority != runtimeCapabilitySelectionAuthority {
		return fmt.Errorf(
			"runtime capability selection authority %q is not supported",
			binding.Authority,
		)
	}
	if binding.Environment == "" ||
		binding.Instance == "" ||
		!binding.Target.Valid() ||
		binding.ExecutorCapability == "" ||
		binding.PluginID == "" ||
		binding.PluginVersion == "" {
		return fmt.Errorf("runtime capability selection identity is incomplete")
	}
	if _, err := parseVersion(binding.PluginVersion); err != nil {
		return fmt.Errorf("runtime capability selection plugin version: %w", err)
	}
	if !validSHA256Fingerprint(binding.CapabilityLockFingerprint) ||
		!validSHA256Fingerprint(binding.RuntimeCapabilityBindingFingerprint) ||
		!validSHA256Fingerprint(binding.SelectionFingerprint) {
		return fmt.Errorf("runtime capability selection fingerprints are invalid")
	}
	if runtimeCapabilitySelectionFingerprint(binding) != binding.SelectionFingerprint {
		return fmt.Errorf("runtime capability selection fingerprint is stale")
	}
	return nil
}

func runtimeCapabilitySelectionFingerprint(
	binding RuntimeCapabilitySelectionBinding,
) string {
	binding.SelectionFingerprint = ""
	return fingerprint(binding)
}

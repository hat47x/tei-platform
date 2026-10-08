package main

import (
	"context"
	"fmt"

	"tei/internal/core"
	"tei/internal/plugin"
	pluginruntime "tei/internal/plugin/runtime"
)

const (
	productionExecutionBridgeSchema    = "tei.reference.production-execution-bridge/v0"
	productionExecutionBridgeAuthority = "derived-verification-not-runtime-authority"
)

// ProductionExecutionBridge proves that a generic Capability execution
// profile matches the already-selected production runtime boundary. It does
// not select a Provider, grant Permission, acquire Resource, or launch code.
type ProductionExecutionBridge struct {
	Schema                               string           `json:"schema"`
	Authority                            string           `json:"authority"`
	Environment                          core.CanonicalID `json:"environment"`
	CapabilityLockFingerprint            string           `json:"capabilityLockFingerprint"`
	RuntimeCapabilityBindingFingerprint  string           `json:"runtimeCapabilityBindingFingerprint"`
	ExecutionProfileFingerprint          string           `json:"executionProfileFingerprint"`
	ExecutionAuthorityBindingFingerprint string           `json:"executionAuthorityBindingFingerprint"`
	LegacyCapabilityKind                 string           `json:"legacyCapabilityKind"`
	LegacyCapabilityID                   core.CanonicalID `json:"legacyCapabilityId"`
	PluginID                             string           `json:"pluginId"`
	PluginVersion                        string           `json:"pluginVersion"`
	PluginRuntimeFingerprint             string           `json:"pluginRuntimeFingerprint"`
	CapabilityRuntimeFingerprint         string           `json:"capabilityRuntimeFingerprint"`
	BindingFingerprint                   string           `json:"bindingFingerprint"`
}

func buildLegacyProductionExecutionBridge(
	lock CapabilityResolutionLock,
	runtimeBinding RuntimeCapabilityBinding,
	profile CapabilityExecutionProfile,
	executionAuthority CapabilityExecutionAuthorityBinding,
	plan pluginruntime.EnvironmentPlan,
	mapping LegacyCapabilityMapping,
) (ProductionExecutionBridge, error) {
	if err := validateCapabilityResolutionLock(lock); err != nil {
		return ProductionExecutionBridge{}, err
	}
	if lock.Scope != ScopeApplicationRuntime {
		return ProductionExecutionBridge{}, fmt.Errorf(
			"production execution bridge requires application-runtime scope",
		)
	}
	if err := validateRuntimeCapabilityBinding(runtimeBinding); err != nil {
		return ProductionExecutionBridge{}, err
	}
	if err := validateCapabilityExecutionProfile(profile, true); err != nil {
		return ProductionExecutionBridge{}, err
	}
	if err := validateCapabilityExecutionAuthorityBinding(executionAuthority); err != nil {
		return ProductionExecutionBridge{}, err
	}
	if !mapping.Kind.Valid() || mapping.CapabilityID == "" {
		return ProductionExecutionBridge{}, fmt.Errorf(
			"production execution bridge requires exact legacy Capability identity",
		)
	}
	if runtimeBinding.Environment == "" ||
		runtimeBinding.Environment != plan.Environment {
		return ProductionExecutionBridge{}, fmt.Errorf(
			"runtime Capability binding Environment differs from production plan",
		)
	}
	if runtimeBinding.CapabilityLockFingerprint != lock.LockFingerprint ||
		executionAuthority.CapabilityLockFingerprint != lock.LockFingerprint {
		return ProductionExecutionBridge{}, fmt.Errorf(
			"production execution inputs reference different Capability Locks",
		)
	}
	if lock.Provider.ExecutionProfileFingerprint != profile.ProfileFingerprint ||
		executionAuthority.ExecutionProfileFingerprint != profile.ProfileFingerprint {
		return ProductionExecutionBridge{}, fmt.Errorf(
			"production execution inputs reference different execution profiles",
		)
	}
	if executionAuthority.IsolationPolicy != profile.IsolationPolicy ||
		!samePermissionSet(executionAuthority.Permissions, profile.Permissions) {
		return ProductionExecutionBridge{}, fmt.Errorf(
			"execution Authority binding differs from execution profile",
		)
	}
	if runtimeBinding.RealizationTarget != lock.RealizationTarget ||
		profile.RealizationTarget != lock.RealizationTarget {
		return ProductionExecutionBridge{}, fmt.Errorf(
			"production execution realization Target differs from Capability Lock",
		)
	}

	pluginRuntime, err := exactPluginRuntime(
		plan, lock.Provider.PluginID, lock.Provider.PluginVersion,
	)
	if err != nil {
		return ProductionExecutionBridge{}, err
	}
	if pluginRuntime.DescriptorFingerprint !=
		runtimeBinding.PluginDefinitionFingerprint ||
		pluginRuntime.Artifact.Digest != runtimeBinding.ArtifactDigest ||
		pluginRuntime.Artifact.Digest != lock.Provider.ArtifactDigest {
		return ProductionExecutionBridge{}, fmt.Errorf(
			"production Plugin runtime differs from Runtime Capability binding",
		)
	}

	capabilityRuntime, ok := plan.Capability(mapping.Kind, mapping.CapabilityID)
	if !ok {
		return ProductionExecutionBridge{}, fmt.Errorf(
			"production plan does not contain unique Capability %s %s",
			mapping.Kind,
			mapping.CapabilityID,
		)
	}
	if string(capabilityRuntime.Plugin.ID) != lock.Provider.PluginID ||
		capabilityRuntime.Plugin.Version != lock.Provider.PluginVersion {
		return ProductionExecutionBridge{}, fmt.Errorf(
			"production Capability is provided by a different Plugin",
		)
	}
	capabilityFingerprint, err :=
		normalizedLegacyCapabilityFingerprint(capabilityRuntime.Capability)
	if err != nil {
		return ProductionExecutionBridge{}, err
	}
	if capabilityFingerprint != lock.Provider.SourceDeclarationFingerprint {
		return ProductionExecutionBridge{}, fmt.Errorf(
			"production Capability declaration differs from locked Provider source declaration",
		)
	}
	if capabilityRuntime.Boundary != pluginRuntime.Boundary {
		return ProductionExecutionBridge{}, fmt.Errorf(
			"production Capability boundary differs from Plugin runtime boundary",
		)
	}
	grantedPermissions, err := normalizePermissionSet(capabilityRuntime.Permissions)
	if err != nil {
		return ProductionExecutionBridge{}, err
	}
	if !samePermissionSet(grantedPermissions, profile.Permissions) {
		return ProductionExecutionBridge{}, fmt.Errorf(
			"production Permission grant differs from generic execution profile",
		)
	}
	if err := verifyProductionIsolation(profile.IsolationPolicy, pluginRuntime); err != nil {
		return ProductionExecutionBridge{}, err
	}

	binding := ProductionExecutionBridge{
		Schema:                               productionExecutionBridgeSchema,
		Authority:                            productionExecutionBridgeAuthority,
		Environment:                          plan.Environment,
		CapabilityLockFingerprint:            lock.LockFingerprint,
		RuntimeCapabilityBindingFingerprint:  runtimeBinding.BindingFingerprint,
		ExecutionProfileFingerprint:          profile.ProfileFingerprint,
		ExecutionAuthorityBindingFingerprint: executionAuthority.BindingFingerprint,
		LegacyCapabilityKind:                 string(mapping.Kind),
		LegacyCapabilityID:                   mapping.CapabilityID,
		PluginID:                             lock.Provider.PluginID,
		PluginVersion:                        lock.Provider.PluginVersion,
		PluginRuntimeFingerprint:             fingerprint(pluginRuntime),
		CapabilityRuntimeFingerprint:         fingerprint(capabilityRuntime),
	}
	binding.BindingFingerprint = productionExecutionBridgeFingerprint(binding)
	if err := validateProductionExecutionBridge(binding); err != nil {
		return ProductionExecutionBridge{}, err
	}
	return binding, nil
}

func verifyLegacyProductionExecutionBridge(
	binding ProductionExecutionBridge,
	lock CapabilityResolutionLock,
	runtimeBinding RuntimeCapabilityBinding,
	profile CapabilityExecutionProfile,
	executionAuthority CapabilityExecutionAuthorityBinding,
	plan pluginruntime.EnvironmentPlan,
	mapping LegacyCapabilityMapping,
) error {
	if err := validateProductionExecutionBridge(binding); err != nil {
		return err
	}
	current, err := buildLegacyProductionExecutionBridge(
		lock,
		runtimeBinding,
		profile,
		executionAuthority,
		plan,
		mapping,
	)
	if err != nil {
		return err
	}
	if binding != current {
		return fmt.Errorf(
			"production execution bridge differs from current production runtime authority",
		)
	}
	return nil
}

func invokeLegacyCapabilityWithProductionBridge(
	ctx context.Context,
	binding ProductionExecutionBridge,
	lock CapabilityResolutionLock,
	runtimeBinding RuntimeCapabilityBinding,
	profile CapabilityExecutionProfile,
	executionAuthority CapabilityExecutionAuthorityBinding,
	registry pluginruntime.Registry,
	plan pluginruntime.EnvironmentPlan,
	mapping LegacyCapabilityMapping,
	operation string,
	requestID string,
	payload any,
) (pluginruntime.Invocation, error) {
	if err := verifyLegacyProductionExecutionBridge(
		binding,
		lock,
		runtimeBinding,
		profile,
		executionAuthority,
		plan,
		mapping,
	); err != nil {
		return pluginruntime.Invocation{}, err
	}
	return registry.InvokeWithProvenance(
		ctx,
		plan,
		mapping.Kind,
		mapping.CapabilityID,
		operation,
		requestID,
		payload,
	)
}

func exactPluginRuntime(
	plan pluginruntime.EnvironmentPlan,
	pluginID string,
	pluginVersion string,
) (pluginruntime.PluginRuntime, error) {
	var matches []pluginruntime.PluginRuntime
	for _, value := range plan.Plugins {
		if string(value.Plugin.ID) == pluginID &&
			value.Plugin.Version == pluginVersion {
			matches = append(matches, value)
		}
	}
	if len(matches) != 1 {
		return pluginruntime.PluginRuntime{}, fmt.Errorf(
			"production plan requires exactly one Plugin runtime %s@%s; found %d",
			pluginID,
			pluginVersion,
			len(matches),
		)
	}
	return matches[0], nil
}

func verifyProductionIsolation(
	policy CapabilityIsolationPolicy,
	pluginRuntime pluginruntime.PluginRuntime,
) error {
	switch policy {
	case CapabilityIsolationHostTrusted:
		if pluginRuntime.Boundary != pluginruntime.BoundaryInProcess ||
			pluginRuntime.Trust != pluginruntime.TrustHostTrusted {
			return fmt.Errorf(
				"host-trusted execution requires in-process host-trusted production runtime",
			)
		}
	case CapabilityIsolationHostEnforcedSandbox:
		if pluginRuntime.Boundary != pluginruntime.BoundaryProcess ||
			pluginRuntime.Trust != pluginruntime.TrustExternalProcess ||
			pluginRuntime.SandboxPolicy != plugin.SandboxRequired {
			return fmt.Errorf(
				"host-enforced sandbox execution requires external process with sandbox required",
			)
		}
	default:
		return fmt.Errorf("unknown execution isolation policy %q", policy)
	}
	return nil
}

func validateProductionExecutionBridge(
	binding ProductionExecutionBridge,
) error {
	if binding.Schema != productionExecutionBridgeSchema ||
		binding.Authority != productionExecutionBridgeAuthority {
		return fmt.Errorf("production execution bridge schema / authority is invalid")
	}
	if binding.Environment == "" ||
		binding.LegacyCapabilityKind == "" ||
		binding.LegacyCapabilityID == "" ||
		binding.PluginID == "" ||
		binding.PluginVersion == "" {
		return fmt.Errorf("production execution bridge identity is incomplete")
	}
	if !plugin.CapabilityKind(binding.LegacyCapabilityKind).Valid() {
		return fmt.Errorf(
			"production execution bridge legacy Capability kind %q is invalid",
			binding.LegacyCapabilityKind,
		)
	}
	if _, err := parseVersion(binding.PluginVersion); err != nil {
		return fmt.Errorf(
			"production execution bridge Plugin version: %w",
			err,
		)
	}
	for _, value := range []string{
		binding.CapabilityLockFingerprint,
		binding.RuntimeCapabilityBindingFingerprint,
		binding.ExecutionProfileFingerprint,
		binding.ExecutionAuthorityBindingFingerprint,
		binding.PluginRuntimeFingerprint,
		binding.CapabilityRuntimeFingerprint,
		binding.BindingFingerprint,
	} {
		if !validSHA256Fingerprint(value) {
			return fmt.Errorf("production execution bridge fingerprint is invalid")
		}
	}
	if productionExecutionBridgeFingerprint(binding) != binding.BindingFingerprint {
		return fmt.Errorf("production execution bridge fingerprint is stale")
	}
	return nil
}

func productionExecutionBridgeFingerprint(
	binding ProductionExecutionBridge,
) string {
	binding.BindingFingerprint = ""
	return fingerprint(binding)
}

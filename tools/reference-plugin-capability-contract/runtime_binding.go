package main

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/plugin"
)

const runtimeCapabilityBindingSchema = "tei.reference.runtime-capability-binding/v0"

type RuntimeCapabilityBinding struct {
	Schema                      string           `json:"schema"`
	Environment                 core.CanonicalID `json:"environment"`
	RealizationTarget           string           `json:"realizationTarget,omitempty"`
	CapabilityLockFingerprint   string           `json:"capabilityLockFingerprint"`
	PluginID                    string           `json:"pluginId"`
	PluginVersion               string           `json:"pluginVersion"`
	PluginDefinitionFingerprint string           `json:"pluginDefinitionFingerprint"`
	ArtifactDigest              string           `json:"artifactDigest"`
	BindingFingerprint          string           `json:"bindingFingerprint"`
}

func buildRuntimeCapabilityBinding(
	capabilityLock CapabilityResolutionLock,
	runtimeLock plugin.Lock,
	environmentID core.CanonicalID,
) (RuntimeCapabilityBinding, error) {
	if err := validateCapabilityResolutionLock(capabilityLock); err != nil {
		return RuntimeCapabilityBinding{}, err
	}
	if capabilityLock.Scope != ScopeApplicationRuntime {
		return RuntimeCapabilityBinding{}, fmt.Errorf(
			"runtime capability binding requires application-runtime scope, got %q",
			capabilityLock.Scope,
		)
	}
	if environmentID == "" {
		return RuntimeCapabilityBinding{}, fmt.Errorf("runtime capability binding requires environment")
	}
	locked, err := runtimeLockedPlugin(
		runtimeLock,
		environmentID,
		capabilityLock.Provider.PluginID,
		capabilityLock.Provider.PluginVersion,
	)
	if err != nil {
		return RuntimeCapabilityBinding{}, err
	}
	if locked.ArtifactDigest == "" ||
		locked.ArtifactDigest != capabilityLock.Provider.ArtifactDigest {
		return RuntimeCapabilityBinding{}, fmt.Errorf(
			"runtime Plugin Lock artifact %q does not match capability provider artifact %q",
			locked.ArtifactDigest,
			capabilityLock.Provider.ArtifactDigest,
		)
	}
	if !validHex64(locked.Fingerprint) {
		return RuntimeCapabilityBinding{}, fmt.Errorf(
			"runtime Plugin Lock definition fingerprint for %s@%s is invalid",
			locked.ID,
			locked.Version,
		)
	}

	binding := RuntimeCapabilityBinding{
		Schema:                      runtimeCapabilityBindingSchema,
		Environment:                 environmentID,
		RealizationTarget:           capabilityLock.RealizationTarget,
		CapabilityLockFingerprint:   capabilityLock.LockFingerprint,
		PluginID:                    capabilityLock.Provider.PluginID,
		PluginVersion:               capabilityLock.Provider.PluginVersion,
		PluginDefinitionFingerprint: locked.Fingerprint,
		ArtifactDigest:              locked.ArtifactDigest,
	}
	binding.BindingFingerprint = runtimeCapabilityBindingFingerprint(binding)
	return binding, nil
}

func buildLegacyRuntimeCapabilityBinding(
	capabilityLock CapabilityResolutionLock,
	runtimeLock plugin.Lock,
	environmentID core.CanonicalID,
	definition plugin.Definition,
	mapping LegacyCapabilityMapping,
) (RuntimeCapabilityBinding, error) {
	projected, err := projectLegacyCapabilityProvider(definition, mapping)
	if err != nil {
		return RuntimeCapabilityBinding{}, err
	}
	if projected != capabilityLock.Provider {
		return RuntimeCapabilityBinding{}, fmt.Errorf(
			"capability lock provider was not derived from the authoritative legacy Plugin declaration",
		)
	}
	locked, err := runtimeLockedPlugin(
		runtimeLock,
		environmentID,
		projected.PluginID,
		projected.PluginVersion,
	)
	if err != nil {
		return RuntimeCapabilityBinding{}, err
	}
	if locked.Fingerprint != plugin.DefinitionFingerprint(definition) {
		return RuntimeCapabilityBinding{}, fmt.Errorf(
			"runtime Plugin Lock definition fingerprint does not match authoritative legacy Plugin definition",
		)
	}
	return buildRuntimeCapabilityBinding(capabilityLock, runtimeLock, environmentID)
}

func verifyRuntimeCapabilityBinding(
	binding RuntimeCapabilityBinding,
	capabilityLock CapabilityResolutionLock,
	runtimeLock plugin.Lock,
) error {
	if err := validateRuntimeCapabilityBinding(binding); err != nil {
		return err
	}
	if err := validateCapabilityResolutionLock(capabilityLock); err != nil {
		return err
	}
	if capabilityLock.Scope != ScopeApplicationRuntime {
		return fmt.Errorf(
			"runtime capability binding cannot verify capability scope %q",
			capabilityLock.Scope,
		)
	}
	if binding.CapabilityLockFingerprint != capabilityLock.LockFingerprint {
		return fmt.Errorf("runtime capability binding references a different capability lock")
	}
	if binding.RealizationTarget != capabilityLock.RealizationTarget {
		return fmt.Errorf(
			"runtime capability binding target %q differs from capability lock target %q",
			binding.RealizationTarget,
			capabilityLock.RealizationTarget,
		)
	}
	if binding.PluginID != capabilityLock.Provider.PluginID ||
		binding.PluginVersion != capabilityLock.Provider.PluginVersion ||
		binding.ArtifactDigest != capabilityLock.Provider.ArtifactDigest {
		return fmt.Errorf("runtime capability binding provider identity differs from capability lock")
	}

	locked, err := runtimeLockedPlugin(
		runtimeLock,
		binding.Environment,
		binding.PluginID,
		binding.PluginVersion,
	)
	if err != nil {
		return err
	}
	if locked.Fingerprint != binding.PluginDefinitionFingerprint {
		return fmt.Errorf(
			"runtime Plugin Lock definition fingerprint for %s@%s diverged",
			binding.PluginID,
			binding.PluginVersion,
		)
	}
	if locked.ArtifactDigest != binding.ArtifactDigest {
		return fmt.Errorf(
			"runtime Plugin Lock artifact digest for %s@%s diverged",
			binding.PluginID,
			binding.PluginVersion,
		)
	}
	return nil
}

func runtimeLockedPlugin(
	lock plugin.Lock,
	environmentID core.CanonicalID,
	pluginID string,
	pluginVersion string,
) (plugin.LockedPlugin, error) {
	// Reuse the production lock validator without making the Reference depend
	// on an unexported EnvironmentLock helper.
	if _, err := plugin.EnvironmentLockFingerprint(lock, environmentID); err != nil {
		return plugin.LockedPlugin{}, err
	}

	var environmentLocks []plugin.EnvironmentLock
	for _, environmentLock := range lock.Environments {
		if environmentLock.Environment == environmentID {
			environmentLocks = append(environmentLocks, environmentLock)
		}
	}
	if len(environmentLocks) != 1 {
		return plugin.LockedPlugin{}, fmt.Errorf(
			"runtime Plugin Lock must contain exactly one environment %s; found %d",
			environmentID,
			len(environmentLocks),
		)
	}

	var matches []plugin.LockedPlugin
	for _, locked := range environmentLocks[0].Plugins {
		if string(locked.ID) == pluginID && locked.Version == pluginVersion {
			matches = append(matches, locked)
		}
	}
	if len(matches) != 1 {
		return plugin.LockedPlugin{}, fmt.Errorf(
			"runtime Plugin Lock must contain exactly one provider %s@%s in environment %s; found %d",
			pluginID,
			pluginVersion,
			environmentID,
			len(matches),
		)
	}
	return matches[0], nil
}

func validateRuntimeCapabilityBinding(binding RuntimeCapabilityBinding) error {
	if binding.Schema != runtimeCapabilityBindingSchema {
		return fmt.Errorf("runtime capability binding schema %q is not supported", binding.Schema)
	}
	if binding.Environment == "" ||
		binding.PluginID == "" ||
		binding.PluginVersion == "" {
		return fmt.Errorf("runtime capability binding identity is incomplete")
	}
	if binding.RealizationTarget != strings.TrimSpace(binding.RealizationTarget) {
		return fmt.Errorf("runtime capability binding target %q has surrounding whitespace", binding.RealizationTarget)
	}
	if _, err := parseVersion(binding.PluginVersion); err != nil {
		return fmt.Errorf("runtime capability binding plugin version: %w", err)
	}
	if !validSHA256Fingerprint(binding.CapabilityLockFingerprint) ||
		!validHex64(binding.PluginDefinitionFingerprint) ||
		!validSHA256Fingerprint(binding.ArtifactDigest) ||
		!validSHA256Fingerprint(binding.BindingFingerprint) {
		return fmt.Errorf("runtime capability binding fingerprints are invalid")
	}
	if runtimeCapabilityBindingFingerprint(binding) != binding.BindingFingerprint {
		return fmt.Errorf("runtime capability binding fingerprint is stale")
	}
	return nil
}

func runtimeCapabilityBindingFingerprint(binding RuntimeCapabilityBinding) string {
	binding.BindingFingerprint = ""
	return fingerprint(binding)
}

func validHex64(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

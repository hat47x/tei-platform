package main

import (
	"fmt"
	"strings"

	"tei/internal/permission"
)

const capabilityExecutionProfileSchema =
	"tei.reference.capability-execution-profile/v0"

const capabilityExecutionAuthorityBindingSchema =
	"tei.reference.capability-execution-authority-binding/v0"

type CapabilityIsolationPolicy string

const (
	CapabilityIsolationHostTrusted CapabilityIsolationPolicy =
		"host-trusted"
	CapabilityIsolationHostEnforcedSandbox CapabilityIsolationPolicy =
		"host-enforced-sandbox"
)

func (p CapabilityIsolationPolicy) Valid() bool {
	return p == CapabilityIsolationHostTrusted ||
		p == CapabilityIsolationHostEnforcedSandbox
}

type CapabilityExecutionProfile struct {
	Schema            string                    `json:"schema"`
	Scope             AuthorityScope            `json:"scope"`
	RealizationTarget string                    `json:"realizationTarget,omitempty"`
	IsolationPolicy   CapabilityIsolationPolicy `json:"isolationPolicy"`
	Permissions       []permission.Spec         `json:"permissions,omitempty"`
	ProfileFingerprint string                   `json:"profileFingerprint"`
}

type CapabilityExecutionGrant struct {
	CapabilityLockFingerprint string            `json:"capabilityLockFingerprint"`
	ExecutionProfileFingerprint string           `json:"executionProfileFingerprint"`
	Permissions               []permission.Spec `json:"permissions,omitempty"`
	AuthorityEvidenceFingerprint string          `json:"authorityEvidenceFingerprint"`
}

type CapabilityExecutionAuthorityBinding struct {
	Schema                       string                    `json:"schema"`
	CapabilityLockFingerprint    string                    `json:"capabilityLockFingerprint"`
	ExecutionProfileFingerprint  string                    `json:"executionProfileFingerprint"`
	IsolationPolicy              CapabilityIsolationPolicy `json:"isolationPolicy"`
	Permissions                  []permission.Spec         `json:"permissions,omitempty"`
	AuthorityEvidenceFingerprint string                    `json:"authorityEvidenceFingerprint"`
	BindingFingerprint           string                    `json:"bindingFingerprint"`
}

func newCapabilityExecutionProfile(
	scope AuthorityScope,
	realizationTarget string,
	isolation CapabilityIsolationPolicy,
	permissions []permission.Spec,
) (CapabilityExecutionProfile, error) {
	profile := CapabilityExecutionProfile{
		Schema:            capabilityExecutionProfileSchema,
		Scope:             scope,
		RealizationTarget: strings.TrimSpace(realizationTarget),
		IsolationPolicy:   isolation,
		Permissions:       append([]permission.Spec(nil), permissions...),
	}
	if err := validateCapabilityExecutionProfile(profile, false); err != nil {
		return CapabilityExecutionProfile{}, err
	}
	profile.Permissions = permission.Normalize(profile.Permissions)
	profile.ProfileFingerprint = capabilityExecutionProfileFingerprint(profile)
	return profile, nil
}

func bindProviderExecutionProfile(
	provider Provider,
	profile CapabilityExecutionProfile,
) (Provider, error) {
	if err := validateCapabilityExecutionProfile(profile, true); err != nil {
		return Provider{}, err
	}
	if provider.Scope != profile.Scope {
		return Provider{}, fmt.Errorf(
			"provider scope %q differs from execution profile scope %q",
			provider.Scope,
			profile.Scope,
		)
	}
	if provider.RealizationTarget != profile.RealizationTarget {
		return Provider{}, fmt.Errorf(
			"provider realization target %q differs from execution profile target %q",
			provider.RealizationTarget,
			profile.RealizationTarget,
		)
	}
	out := provider
	out.ExecutionProfileFingerprint = profile.ProfileFingerprint
	return out, nil
}

func verifyCapabilityExecutionAuthority(
	lock CapabilityResolutionLock,
	profile CapabilityExecutionProfile,
	grant CapabilityExecutionGrant,
) (CapabilityExecutionAuthorityBinding, error) {
	if err := validateCapabilityResolutionLock(lock); err != nil {
		return CapabilityExecutionAuthorityBinding{}, err
	}
	if err := validateCapabilityExecutionProfile(profile, true); err != nil {
		return CapabilityExecutionAuthorityBinding{}, err
	}
	if lock.Provider.ExecutionProfileFingerprint == "" {
		return CapabilityExecutionAuthorityBinding{}, fmt.Errorf(
			"capability provider has no execution profile binding",
		)
	}
	if lock.Provider.ExecutionProfileFingerprint != profile.ProfileFingerprint {
		return CapabilityExecutionAuthorityBinding{}, fmt.Errorf(
			"capability provider execution profile differs from supplied profile",
		)
	}
	if lock.Scope != profile.Scope ||
		lock.RealizationTarget != profile.RealizationTarget {
		return CapabilityExecutionAuthorityBinding{}, fmt.Errorf(
			"execution profile scope or realization target differs from capability lock",
		)
	}
	if grant.CapabilityLockFingerprint != lock.LockFingerprint {
		return CapabilityExecutionAuthorityBinding{}, fmt.Errorf(
			"execution grant references a different capability lock",
		)
	}
	if grant.ExecutionProfileFingerprint != profile.ProfileFingerprint {
		return CapabilityExecutionAuthorityBinding{}, fmt.Errorf(
			"execution grant references a different execution profile",
		)
	}
	if !validSHA256Fingerprint(grant.AuthorityEvidenceFingerprint) {
		return CapabilityExecutionAuthorityBinding{}, fmt.Errorf(
			"execution grant requires exact Authority Evidence fingerprint",
		)
	}

	required, err := normalizePermissionSet(profile.Permissions)
	if err != nil {
		return CapabilityExecutionAuthorityBinding{}, err
	}
	granted, err := normalizePermissionSet(grant.Permissions)
	if err != nil {
		return CapabilityExecutionAuthorityBinding{}, err
	}
	if !samePermissionSet(required, granted) {
		return CapabilityExecutionAuthorityBinding{}, fmt.Errorf(
			"execution grant permissions differ from exact execution profile requirements",
		)
	}

	binding := CapabilityExecutionAuthorityBinding{
		Schema:                       capabilityExecutionAuthorityBindingSchema,
		CapabilityLockFingerprint:    lock.LockFingerprint,
		ExecutionProfileFingerprint:  profile.ProfileFingerprint,
		IsolationPolicy:              profile.IsolationPolicy,
		Permissions:                  required,
		AuthorityEvidenceFingerprint: grant.AuthorityEvidenceFingerprint,
	}
	binding.BindingFingerprint =
		capabilityExecutionAuthorityBindingFingerprint(binding)
	return binding, nil
}

func validateCapabilityExecutionProfile(
	profile CapabilityExecutionProfile,
	requireFingerprint bool,
) error {
	if profile.Schema != capabilityExecutionProfileSchema {
		return fmt.Errorf(
			"execution profile schema %q is not supported",
			profile.Schema,
		)
	}
	if !profile.Scope.Valid() {
		return fmt.Errorf("execution profile scope %q is invalid", profile.Scope)
	}
	if profile.RealizationTarget != strings.TrimSpace(profile.RealizationTarget) {
		return fmt.Errorf(
			"execution profile realization target %q has surrounding whitespace",
			profile.RealizationTarget,
		)
	}
	if !profile.IsolationPolicy.Valid() {
		return fmt.Errorf(
			"execution profile isolation policy %q is invalid",
			profile.IsolationPolicy,
		)
	}
	normalized, err := normalizePermissionSet(profile.Permissions)
	if err != nil {
		return err
	}
	if len(normalized) != len(profile.Permissions) {
		return fmt.Errorf("execution profile permissions are not normalized")
	}
	for i := range normalized {
		if normalized[i] != profile.Permissions[i] {
			return fmt.Errorf("execution profile permissions are not normalized")
		}
	}
	if requireFingerprint {
		if !validSHA256Fingerprint(profile.ProfileFingerprint) {
			return fmt.Errorf("execution profile fingerprint is invalid")
		}
		if capabilityExecutionProfileFingerprint(profile) !=
			profile.ProfileFingerprint {
			return fmt.Errorf("execution profile fingerprint is stale")
		}
	} else if profile.ProfileFingerprint != "" {
		return fmt.Errorf("unsealed execution profile must not carry fingerprint")
	}
	return nil
}

func normalizePermissionSet(
	values []permission.Spec,
) ([]permission.Spec, error) {
	seen := map[string]struct{}{}
	out := append([]permission.Spec(nil), values...)
	for i, value := range out {
		if err := value.Validate(); err != nil {
			return nil, fmt.Errorf("permission[%d]: %w", i, err)
		}
		if _, duplicate := seen[value.Key()]; duplicate {
			return nil, fmt.Errorf(
				"permission %s is declared more than once",
				value.Key(),
			)
		}
		seen[value.Key()] = struct{}{}
	}
	out = permission.Normalize(out)
	return out, nil
}

func samePermissionSet(
	left []permission.Spec,
	right []permission.Spec,
) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func capabilityExecutionProfileFingerprint(
	profile CapabilityExecutionProfile,
) string {
	profile.ProfileFingerprint = ""
	profile.Permissions = permission.Normalize(profile.Permissions)
	return fingerprint(profile)
}

func capabilityExecutionAuthorityBindingFingerprint(
	binding CapabilityExecutionAuthorityBinding,
) string {
	binding.BindingFingerprint = ""
	binding.Permissions = permission.Normalize(binding.Permissions)
	return fingerprint(binding)
}

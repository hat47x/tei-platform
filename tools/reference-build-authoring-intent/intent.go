package main

// The Build Intent declaration and its strict reader now live in
// internal/buildintent (SB-C0 §4.13, I-1a), so cmd/tei can read the same
// declaration with the same implementation.
//
// This file keeps the names the reference resolver and its tests already use as
// thin aliases. Only the declaration surface moved; Resolve still lives here,
// because resolution needs a fixture catalog and moving that would make the
// fixture look like a product surface.

import (
	"reflect"

	"tei/internal/buildintent"
)

const (
	// IntentFormatVersion is the authoring format this resolver reads.
	IntentFormatVersion = buildintent.FormatVersion
	// ResolverVersion identifies these derivation rules. It is part of a
	// ResolutionContext and therefore part of a receipt's fingerprint.
	ResolverVersion = buildintent.ResolverVersion

	subjectKindPlugin     = "plugin"
	sourceKindDefinition  = "plugin-definition"
	builderAuthorityScope = "build-tool"
)

// BuildIntent is the declaration a person or a generator writes.
type BuildIntent = buildintent.BuildIntent

// IntentSubject identifies what is being built.
type IntentSubject = buildintent.IntentSubject

// IntentSource is a logical source reference.
type IntentSource = buildintent.IntentSource

// BuilderRequirement is a builder selection policy.
type BuilderRequirement = buildintent.BuilderRequirement

// ProviderWiring is an explicit logical provider selection.
type ProviderWiring = buildintent.ProviderWiring

// ArtifactPin is an exact pin written on purpose.
type ArtifactPin = buildintent.ArtifactPin

// DependencyRequirement is a logical build dependency.
type DependencyRequirement = buildintent.DependencyRequirement

// SecretRequirement declares a secret's name and providing authority.
type SecretRequirement = buildintent.SecretRequirement

// ParseIntent reads authoring JSON strictly.
func ParseIntent(raw []byte) (BuildIntent, error) {
	return buildintent.ParseIntent(raw)
}

// NormalizeIntent removes declaration order from the result.
func NormalizeIntent(i BuildIntent) BuildIntent {
	return buildintent.NormalizeIntent(i)
}

// IntentFingerprint is the declaration's own identity.
func IntentFingerprint(i BuildIntent) (string, error) {
	return buildintent.IntentFingerprint(i)
}

func validateSourceRef(ref string) error {
	return buildintent.ValidateSourceRef(ref)
}

func validateInputJSON(raw []byte) error {
	return buildintent.ValidateInputJSON(raw)
}

func rejectKeyCaseVariants(v any, t reflect.Type, path string) error {
	return buildintent.RejectKeyCaseVariants(v, t, path)
}

func fingerprintJSON(v any) (string, error) {
	return buildintent.FingerprintJSON(v)
}

package main

import (
	"reflect"

	"tei/internal/buildintent"
)

// This file keeps the reference resolver's own call sites unchanged while the
// declaration reader lives in internal/buildintent (SB-C0 §4.13 I-1a). The
// types are aliases, not copies: there is one BuildIntent definition in the
// repository, so the CLI entry and this reference cannot drift apart.
//
// Resolution stays here. A ResolutionContext needs source bytes, a builder
// catalog, a dependency catalog and a target catalog, which are fixtures of
// this reference rather than product concepts.

type (
	BuildIntent           = buildintent.BuildIntent
	IntentSubject         = buildintent.IntentSubject
	IntentSource          = buildintent.IntentSource
	BuilderRequirement    = buildintent.BuilderRequirement
	ProviderWiring        = buildintent.ProviderWiring
	ArtifactPin           = buildintent.ArtifactPin
	DependencyRequirement = buildintent.DependencyRequirement
	SecretRequirement     = buildintent.SecretRequirement
)

const (
	IntentFormatVersion = buildintent.IntentFormatVersion
	ResolverVersion     = buildintent.ResolverVersion

	builderAuthorityScope = buildintent.BuilderAuthorityScope
)

var sha256Digest = buildintent.SHA256Digest

// ParseIntent reads a declaration strictly. It delegates so that both this
// reference and cmd/tei read the same bytes with the same rules.
func ParseIntent(raw []byte) (BuildIntent, error) { return buildintent.ParseIntent(raw) }

// IntentFingerprint derives the declaration's identity.
func IntentFingerprint(i BuildIntent) (string, error) { return buildintent.IntentFingerprint(i) }

// NormalizeIntent removes declaration order from the result.
func NormalizeIntent(i BuildIntent) BuildIntent { return buildintent.NormalizeIntent(i) }

func fingerprintJSON(v any) (string, error) { return buildintent.FingerprintJSON(v) }

func validateInputJSON(raw []byte) error { return buildintent.ValidateInputJSON(raw) }

func validateSourceRef(ref string) error { return buildintent.ValidateSourceRef(ref) }

func rejectKeyCaseVariants(v any, t reflect.Type, path string) error {
	return buildintent.RejectKeyCaseVariants(v, t, path)
}

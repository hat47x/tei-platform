package main

// The resolver moved to internal/buildintent (SB-C0 §4.13, case I-1b): the
// interfaces, the receipt shape and Resolve now live beside the declaration
// reader, so a caller does not have to start a subprocess or duplicate the
// derivation. The fixture catalog stays here, which is what §4.13 objected to
// moving: a fixture in internal/ would look like a product surface.
//
// This file keeps the names the reference tool and its tests already use.

import (
	"tei/internal/buildintent"
)

const ReceiptFormatVersion = 1

// SourceResolver resolves a logical source reference into an exact BuildSource.
type SourceResolver = buildintent.SourceResolver

// ResolvedSource is the derived BuildSource with the bytes its digest came from.
type ResolvedSource = buildintent.ResolvedSource

// BuilderRequest is the input to builder contract resolution.
type BuilderRequest = buildintent.BuilderRequest

// BuilderResolution is the exact builder plus the identity of its evidence.
type BuilderResolution = buildintent.BuilderResolution

// BuilderResolver resolves a builder contract.
type BuilderResolver = buildintent.BuilderResolver

// DependencyResolver resolves a build dependency from the graph's producer.
type DependencyResolver = buildintent.DependencyResolver

// TargetCatalog checks that a target and profile exist.
type TargetCatalog = buildintent.TargetCatalog

// ResolutionContext bundles the observations a resolver may consult.
type ResolutionContext = buildintent.ResolutionContext

// ResolutionContextFingerprints identifies the context and the resolver version.
type ResolutionContextFingerprints = buildintent.ResolutionContextFingerprints

// ResolvedBuildReceipt binds an intent, the observed context and the plan.
type ResolvedBuildReceipt = buildintent.ResolvedBuildReceipt

// AuthorityDerivedCache marks a receipt as a derived cache, not a canonical
// source.
const AuthorityDerivedCache = buildintent.AuthorityDerivedCache

// Resolve derives an exact BuildPlan from an intent and a context.
func Resolve(intent BuildIntent, ctx ResolutionContext) (ResolvedBuildReceipt, error) {
	return buildintent.Resolve(intent, ctx)
}

// MarshalReceipt renders a receipt as JSON.
func MarshalReceipt(r ResolvedBuildReceipt) ([]byte, error) {
	return buildintent.MarshalReceipt(r)
}

// VerifyDerivedReceipt re-derives and compares a stored receipt.
func VerifyDerivedReceipt(intent BuildIntent, ctx ResolutionContext, stored []byte) error {
	return buildintent.VerifyDerivedReceipt(intent, ctx, stored)
}

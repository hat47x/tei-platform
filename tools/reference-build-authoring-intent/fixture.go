package main

import (
	"tei/internal/buildintent"
)

// This file keeps the reference resolver's own call sites unchanged while the
// catalog machinery lives in internal/buildintent (SB-C0 §4.22, migration step 1).
//
// The names here still say "fixture" because that is what this reference uses:
// a fixed catalog committed beside it. The mechanism itself is not a fixture —
// §4.22 decided the catalog is data the caller names and digests — so the code
// moved under product names and this file only aliases it.

type (
	FixtureCatalog     = buildintent.BuildCatalog
	FixtureProvider    = buildintent.BuildProvider
	FixtureDependency  = buildintent.BuildDependencyItem
	FixtureTargetEntry = buildintent.BuildTargetEntry
	DirSourceResolver  = buildintent.DirSourceResolver
)

// ParseFixtureCatalog reads this reference's catalog strictly.
func ParseFixtureCatalog(raw []byte) (FixtureCatalog, error) {
	return buildintent.ParseBuildCatalog(raw)
}

// NewFixtureContext assembles the resolution context this reference uses.
func NewFixtureContext(sourceRoot string, catalog FixtureCatalog) buildintent.ResolutionContext {
	return buildintent.NewBuildContext(sourceRoot, catalog)
}

//go:build studio_generated

package main

import (
	"tei/internal/studiohost"
	"tei/internal/studiohost/staticservices"
)

// Compile-time checked bindings to #1833's generated Go constructors.
// Only the host selects trustedRoot; no generated service is registered with Wails.
var (
	_ studiohost.CatalogService = (*staticservices.CatalogService)(nil)
	_ studiohost.ValidationService = (*staticservices.ValidateService)(nil)
)

func newStudioHost(paths []string) (*studiohost.ReadOnly, error) {
	return studiohost.NewReadOnlyWithFactory(paths, func(trustedRoot string) (studiohost.RootServices, error) {
		return studiohost.RootServices{
			Catalog: staticservices.NewCatalogService(trustedRoot),
			Validation: staticservices.NewValidateService(trustedRoot),
		}, nil
	})
}

//go:build studio_generated

package main

import (
	"tei/internal/inspectoradapter"
	"tei/internal/studiohost"
	"tei/internal/studiohost/staticservices"
)

// Generated Catalog/Validation and the local selected Inspector share the
// same Host-resolved Workspace root. The Inspector is an explicitly compiled
// reference Provider, not yet a generated or independently attested Plugin.
var (
	_ studiohost.CatalogService    = (*staticservices.CatalogService)(nil)
	_ studiohost.ValidationService = (*staticservices.ValidateService)(nil)
	_ studiohost.InspectionService = (*inspectoradapter.Selected)(nil)
)

func newStudioHost(paths []string) (*studiohost.ReadOnly, error) {
	inspector, err := inspectoradapter.NewLocalInspectorHost()
	if err != nil {
		return nil, err
	}
	return newStudioHostWithInspector(paths, inspector)
}

// newStudioHostWithInspector serves the Inspector the composing Host already
// selected. The renderer still supplies workspace IDs only; it cannot name a
// Provider, and every root reaches the Inspector through the allowlist.
func newStudioHostWithInspector(paths []string, inspector *inspectoradapter.LocalInspectorHost) (*studiohost.ReadOnly, error) {
	return studiohost.NewReadOnlyWithFactoryAndObserver(paths,
		func(trustedRoot string) (studiohost.RootServices, error) {
			selected, err := inspector.Bind(trustedRoot)
			if err != nil {
				return studiohost.RootServices{}, err
			}
			return studiohost.RootServices{
				Catalog:    staticservices.NewCatalogService(trustedRoot),
				Validation: staticservices.NewValidateService(trustedRoot),
				Inspection: selected,
			}, nil
		}, studioXeadObserver{})
}

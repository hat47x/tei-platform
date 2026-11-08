//go:build !studio_generated

package main

import (
	"tei/internal/inspectoradapter"
	"tei/internal/studiohost"
	"tei/internal/studiohost/staticservices"
)

// The normal Studio Host uses the same explicitly compiled reference
// Inspector Provider/Contract as the CLI. Workspace roots remain entirely
// host-owned and must pass studiohost's startup allowlist/identity checks.
// This is a static local Composition, not dynamic Plugin deployment.
func newStudioHost(paths []string) (*studiohost.ReadOnly, error) {
	inspector, err := inspectoradapter.NewLocalInspectorHost()
	if err != nil {
		return nil, err
	}
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

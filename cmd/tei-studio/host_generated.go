//go:build studio_generated

package main

import (
	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/studiohost"
	"tei/internal/studiohost/staticservices"
)

// selectedBuiltinInspector is an explicit Host-selected interim binding for
// Inspector. The generated Host has Catalog/Validation services but no
// generated Inspector service yet. Never let Studio silently fall back when
// RootServices.Inspection is absent; this adapter is visible in the Host
// composition and must be replaced by an exact Plugin Provider in D1b.
type selectedBuiltinInspector struct{ trustedRoot string }

func (s selectedBuiltinInspector) Inspect(subject sourceindex.Subject) (operations.SubjectInspection, error) {
	return operations.InspectSubject(s.trustedRoot, subject)
}

// Compile-time checked bindings to #1833's generated Go constructors.
// Only the host selects trustedRoot; no generated service is registered with Wails.
var (
	_ studiohost.CatalogService    = (*staticservices.CatalogService)(nil)
	_ studiohost.ValidationService = (*staticservices.ValidateService)(nil)
	_ studiohost.InspectionService = selectedBuiltinInspector{}
)

func newStudioHost(paths []string) (*studiohost.ReadOnly, error) {
	return studiohost.NewReadOnlyWithFactoryAndObserver(paths, func(trustedRoot string) (studiohost.RootServices, error) {
		return studiohost.RootServices{
			Catalog:    staticservices.NewCatalogService(trustedRoot),
			Validation: staticservices.NewValidateService(trustedRoot),
			Inspection: selectedBuiltinInspector{trustedRoot: trustedRoot},
		}, nil
	}, studioXeadObserver{})
}

package applicationview

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
)

// PackageViewは、Packageのメンバー・前提・投影・版束縛を解決した結果である。
type PackageView struct {
	SchemaVersion string                  `json:"schemaVersion"`
	Application   core.CanonicalID        `json:"application"`
	Package       core.CanonicalID        `json:"package"`
	Version       string                  `json:"version"`
	Members       []PackageMemberRef      `json:"members,omitempty"`
	Requires      []PackageRequirementRef `json:"requires,omitempty"`
	Projection    string                  `json:"projection,omitempty"`
	Conformance   string                  `json:"conformance,omitempty"`
	Bound         bool                    `json:"bound"`
}

// PackageMemberRefは、Packageが束ねるPatternである。
type PackageMemberRef struct {
	Kind    application.PatternKind `json:"kind"`
	Pattern core.CanonicalID        `json:"pattern"`
}

// PackageRequirementRefは、Packageが前提にするSemantic要素である。
type PackageRequirementRef struct {
	Kind string           `json:"kind"`
	ID   core.CanonicalID `json:"id"`
}

// BuildPackageは、Applicationが持つPackageを解決する。
func BuildPackage(world ir.World, applicationID, packageID core.CanonicalID) (PackageView, error) {
	var app *application.Application
	for i := range world.Applications {
		if world.Applications[i].ID == applicationID {
			app = &world.Applications[i]
			break
		}
	}
	if app == nil {
		return PackageView{}, fmt.Errorf("application %sがありません", applicationID)
	}
	for _, pkg := range app.Packages {
		if pkg.ID != packageID {
			continue
		}
		view := PackageView{
			SchemaVersion: SchemaVersion,
			Application:   applicationID,
			Package:       pkg.ID,
			Version:       pkg.Version,
			Projection:    pkg.Projection,
			Conformance:   pkg.Conformance,
		}
		for _, member := range pkg.Members {
			view.Members = append(view.Members, PackageMemberRef{Kind: member.Kind, Pattern: member.Pattern})
		}
		for _, requirement := range pkg.Requires {
			view.Requires = append(view.Requires, PackageRequirementRef{Kind: requirement.Kind, ID: requirement.ID})
		}
		for _, binding := range app.PackageBindings {
			if binding.ID == packageID && binding.Version == pkg.Version {
				view.Bound = true
			}
		}
		return view, nil
	}
	return PackageView{}, fmt.Errorf("package %sはapplication %sにありません", packageID, applicationID)
}

package studiohost

import (
	"tei/internal/applicationview"
	"tei/internal/core"
)

// PackageResponse is a read-only package projection for the Studio bridge.
type PackageResponse struct {
	OK      bool                         `json:"ok"`
	Value   *applicationview.PackageView `json:"value,omitempty"`
	Failure *Failure                     `json:"failure,omitempty"`
}

// ViewPackage resolves a package's members, requirements, and binding.
func (h *ReadOnly) ViewPackage(workspaceID, applicationID, packageID string) PackageResponse {
	loaded, failure := h.loadWorkspace(workspaceID)
	if failure != nil {
		return PackageResponse{Failure: failure}
	}
	value, err := applicationview.BuildPackage(loaded.World, core.CanonicalID(applicationID), core.CanonicalID(packageID))
	if err != nil {
		return PackageResponse{Failure: classifiedFailure("VIEW_FAILED", "view", err.Error())}
	}
	return PackageResponse{OK: true, Value: &value}
}

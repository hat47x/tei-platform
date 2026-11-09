package studiohost

import (
	"tei/internal/application"
	"tei/internal/applicationview"
	"tei/internal/core"
	"tei/internal/workspace"
)

// ViewResponse is a read-only projection of an application pattern or application.
// OK reports that the projection was produced; it does not accept the design.
type ViewResponse struct {
	OK      bool     `json:"ok"`
	Value   any      `json:"value,omitempty"`
	Failure *Failure `json:"failure,omitempty"`
}

// ViewPattern resolves a pattern into the same projection used by the CLI view.
func (h *ReadOnly) ViewPattern(workspaceID, applicationID, patternID string) ViewResponse {
	root, failure := h.workspace(workspaceID)
	if failure != nil {
		return ViewResponse{Failure: failure}
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return ViewResponse{Failure: &Failure{Code: "WORKSPACE_LOAD_FAILED", Stage: "view", Message: err.Error()}}
	}
	applicationIDValue := core.CanonicalID(applicationID)
	patternIDValue := core.CanonicalID(patternID)
	kind, err := applicationview.PatternKind(loaded.World, applicationIDValue, patternIDValue)
	if err != nil {
		return ViewResponse{Failure: &Failure{Code: "VIEW_FAILED", Stage: "view", Message: err.Error()}}
	}
	switch kind {
	case application.PatternRecord:
		value, err := applicationview.Build(loaded.World, applicationIDValue, patternIDValue)
		if err != nil {
			return ViewResponse{Failure: &Failure{Code: "VIEW_FAILED", Stage: "view", Message: err.Error()}}
		}
		return ViewResponse{OK: true, Value: value}
	case application.PatternHeaderDetail:
		value, err := applicationview.BuildHeaderDetail(loaded.World, applicationIDValue, patternIDValue)
		if err != nil {
			return ViewResponse{Failure: &Failure{Code: "VIEW_FAILED", Stage: "view", Message: err.Error()}}
		}
		return ViewResponse{OK: true, Value: value}
	case application.PatternBrowse:
		value, err := applicationview.BuildBrowse(loaded.World, applicationIDValue, patternIDValue)
		if err != nil {
			return ViewResponse{Failure: &Failure{Code: "VIEW_FAILED", Stage: "view", Message: err.Error()}}
		}
		return ViewResponse{OK: true, Value: value}
	case application.PatternCommand:
		value, err := applicationview.BuildCommandPattern(loaded.World, applicationIDValue, patternIDValue)
		if err != nil {
			return ViewResponse{Failure: &Failure{Code: "VIEW_FAILED", Stage: "view", Message: err.Error()}}
		}
		return ViewResponse{OK: true, Value: value}
	default:
		return ViewResponse{Failure: &Failure{Code: "VIEW_UNSUPPORTED", Stage: "view", Message: "pattern kind is not supported"}}
	}
}

// ViewApplication resolves an application summary.
func (h *ReadOnly) ViewApplication(workspaceID, applicationID string) ViewResponse {
	root, failure := h.workspace(workspaceID)
	if failure != nil {
		return ViewResponse{Failure: failure}
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return ViewResponse{Failure: &Failure{Code: "WORKSPACE_LOAD_FAILED", Stage: "view", Message: err.Error()}}
	}
	value, err := applicationview.BuildApplication(loaded.World, core.CanonicalID(applicationID))
	if err != nil {
		return ViewResponse{Failure: &Failure{Code: "VIEW_FAILED", Stage: "view", Message: err.Error()}}
	}
	return ViewResponse{OK: true, Value: value}
}

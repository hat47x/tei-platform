package studiohost

import (
	"errors"

	"tei/internal/operations"
)

// ThreeViewUnavailable is returned when the composing Host selected no
// three-view Provider. The Studio does not fall back to a built-in reading.
const ThreeViewUnavailable = "THREE_VIEW_UNAVAILABLE"

// ThreeViewService is the Host-selected, typed read-only three-view
// navigation. Its presence does not prove Plugin resolution, invocation
// provenance or current authority; the composing Host observes those.
type ThreeViewService interface {
	Navigate(request operations.ThreeViewRequest) (operations.ThreeViewNavigation, error)
}

// ThreeViewFactory is called only after the Host resolved an allowlisted
// Workspace ID. It is never exposed to the renderer.
type ThreeViewFactory func(trustedRoot string) (ThreeViewService, error)

type ThreeViewResponse struct {
	OK      bool                            `json:"ok"`
	Value   *operations.ThreeViewNavigation `json:"value,omitempty"`
	Failure *Failure                        `json:"failure,omitempty"`
}

// NavigateThreeView runs the three-view navigation Operation for one source of
// an allowlisted Workspace through the Provider the composing Host selected.
//
// pinOccurrence and pinSourceDigest carry a target the caller fixed on
// purpose. Both empty means no pin. The host keeps no selection state: the pin
// is the caller's value, and it comes back unchanged with its current status,
// whatever else the caller looks at in the meantime.
func (h *ReadOnly) NavigateThreeView(workspaceID, sourcePath, pinOccurrence, pinSourceDigest string) ThreeViewResponse {
	root, denied := h.workspace(workspaceID)
	if denied != nil {
		return ThreeViewResponse{Failure: denied}
	}
	if h.ThreeView == nil {
		return ThreeViewResponse{Failure: classifiedFailure(ThreeViewUnavailable, "", "no three-view Provider is selected for this host")}
	}
	request := operations.ThreeViewRequest{SourcePath: sourcePath}
	if pinOccurrence != "" || pinSourceDigest != "" {
		request.Pin = &operations.ThreeViewPin{Occurrence: pinOccurrence, SourceArtifactDigest: pinSourceDigest}
	}
	service, err := h.ThreeView(root)
	if err == nil && service == nil {
		err = errors.New("three-view service is missing")
	}
	var value operations.ThreeViewNavigation
	if err == nil {
		value, err = service.Navigate(request)
	}
	if err == nil {
		err = operations.ValidateThreeViewNavigation(request, value)
	}
	if err != nil {
		return ThreeViewResponse{Failure: threeViewFailure(err)}
	}
	return ThreeViewResponse{OK: true, Value: &value}
}

// threeViewFailure keeps only the closed set of Operation codes. A Provider's
// own error text may carry a private root or source text, so it is not shown.
func threeViewFailure(err error) *Failure {
	var typed *operations.ThreeViewError
	if errors.As(err, &typed) && typed != nil {
		switch typed.Code {
		case operations.ThreeViewSourceReadFailed,
			operations.ThreeViewSourceTooLarge,
			operations.ThreeViewUnsupportedSource,
			operations.ThreeViewSourceRejected,
			operations.ThreeViewProjectionFailed:
			return classifiedFailure(typed.Code, "", "three-view navigation failed")
		}
	}
	return classifiedFailure("OPERATION_FAILED", "", "workspace operation failed")
}

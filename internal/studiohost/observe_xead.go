package studiohost

import (
	"errors"

	"tei/internal/operations"
)

type XeadObservationResponse struct {
	OK      bool                        `json:"ok"`
	Value   *operations.XeadObservation `json:"value,omitempty"`
	Failure *Failure                    `json:"failure,omitempty"`
}

// ObserveXead reads a Workspace-relative .xead path from the configured
// allowlist and returns only the bounded read-only operation projection.
func (h *ReadOnly) ObserveXead(workspaceID, relativePath string) XeadObservationResponse {
	root, denied := h.workspace(workspaceID)
	if denied != nil {
		return XeadObservationResponse{Failure: denied}
	}
	value, err := operations.ObserveXeadSource(root, relativePath)
	if err != nil {
		var observationErr *operations.XeadObservationError
		if errors.As(err, &observationErr) {
			return XeadObservationResponse{Failure: &Failure{
				Code: observationErr.Code, Message: "X-TEA source observation failed",
			}}
		}
		return XeadObservationResponse{Failure: &Failure{
			Code: "OPERATION_FAILED", Message: "workspace operation failed",
		}}
	}
	return XeadObservationResponse{OK: true, Value: &value}
}

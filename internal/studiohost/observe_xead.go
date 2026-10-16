package studiohost

import (
	"errors"
	"path"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

const XeadObservationInputLimitBytes int64 = 4 << 20

const (
	XeadSourceReadFailed  = "XEAD_SOURCE_READ_FAILED"
	XeadSourceTooLarge    = "XEAD_SOURCE_TOO_LARGE"
	XeadUnsupportedFormat = "XEAD_UNSUPPORTED_FORMAT"
)

// XeadObservation is a Studio-only source projection. Its identifiers are
// provenance and it carries no Canonical or execution authority.
type XeadObservation struct {
	SchemaVersion   string         `json:"schemaVersion"`
	Effect          string         `json:"effect"`
	Observation     XeadSourceView `json:"observation"`
	AuthorityLimits []string       `json:"authorityLimits"`
}

type XeadSourceRef struct {
	Kind string `json:"sourceKind"`
	ID   string `json:"sourceID"`
}

type XeadSourceView struct {
	Tables      []XeadTable      `json:"tables,omitempty"`
	Diagnostics []XeadDiagnostic `json:"diagnostics,omitempty"`
}

type XeadTable struct {
	Source XeadSourceRef `json:"source"`
	Name   string        `json:"name,omitempty"`
	Fields []XeadField   `json:"fields,omitempty"`
}

type XeadField struct {
	Source   XeadSourceRef  `json:"source"`
	Name     string         `json:"name,omitempty"`
	DataType *XeadSourceRef `json:"dataType,omitempty"`
}

type XeadDiagnostic struct {
	Code        string `json:"code"`
	Element     string `json:"element"`
	Attribute   string `json:"attribute,omitempty"`
	Occurrences int    `json:"occurrences"`
}

// XeadObservationObserver receives bounded file bytes only. It has no path or
// filesystem capability and is implemented by the Studio-only parser adapter.
type XeadObservationObserver interface {
	Observe(content []byte) (XeadObservation, error)
}

type XeadObservationError struct{ Code string }

func (e *XeadObservationError) Error() string {
	if e == nil {
		return "XEAD_OPERATION_FAILED"
	}
	return e.Code
}

type XeadObservationResponse struct {
	OK      bool             `json:"ok"`
	Value   *XeadObservation `json:"value,omitempty"`
	Failure *Failure         `json:"failure,omitempty"`
}

func (h *ReadOnly) ObserveXead(workspaceID, relativePath string) XeadObservationResponse {
	root, denied := h.workspace(workspaceID)
	if denied != nil {
		return XeadObservationResponse{Failure: denied}
	}
	normalizedPath, err := sourceindex.NormalizeWorkspacePath(relativePath)
	if err != nil || normalizedPath != relativePath {
		return XeadObservationResponse{Failure: xeadFailure(XeadSourceReadFailed)}
	}
	if !strings.EqualFold(path.Ext(normalizedPath), ".xead") {
		return XeadObservationResponse{Failure: xeadFailure(XeadUnsupportedFormat)}
	}
	content, err := workspace.ReadSource(root, relativePath, XeadObservationInputLimitBytes)
	if err != nil {
		code := XeadSourceReadFailed
		if errors.Is(err, workspace.ErrSourceTooLarge) {
			code = XeadSourceTooLarge
		}
		return XeadObservationResponse{Failure: xeadFailure(code)}
	}
	if h.xeadObserver == nil {
		return XeadObservationResponse{Failure: &Failure{Code: "OPERATION_FAILED", Message: "workspace operation failed"}}
	}
	value, err := h.xeadObserver.Observe(content)
	if err != nil {
		var observationErr *XeadObservationError
		if errors.As(err, &observationErr) {
			return XeadObservationResponse{Failure: &Failure{Code: observationErr.Code, Message: "X-TEA source observation failed"}}
		}
		return XeadObservationResponse{Failure: &Failure{Code: "OPERATION_FAILED", Message: "workspace operation failed"}}
	}
	return XeadObservationResponse{OK: true, Value: &value}
}

func xeadFailure(code string) *Failure {
	return &Failure{Code: code, Message: "X-TEA source observation failed"}
}

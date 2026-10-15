package operations

import (
	"errors"
	"path"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
	"tei/internal/xeadsource"
)

const XeadObservationInputLimitBytes int64 = 4 << 20

const (
	ObserveXeadSourceReadFailed  = "XEAD_SOURCE_READ_FAILED"
	ObserveXeadSourceTooLarge    = "XEAD_SOURCE_TOO_LARGE"
	ObserveXeadUnsupportedFormat = "XEAD_UNSUPPORTED_FORMAT"
	ObserveXeadParseFailed       = "XEAD_PARSE_FAILED"
	ObserveXeadProjectionFailed  = "XEAD_PROJECTION_FAILED"
)

// XeadObservation is a bounded, read-only source projection. It is not a
// Canonical World, Source Index, edit target, or execution authority.
type XeadObservation struct {
	SchemaVersion   string                       `json:"schemaVersion"`
	Effect          string                       `json:"effect"`
	Observation     xeadsource.StudioObservation `json:"observation"`
	AuthorityLimits []string                     `json:"authorityLimits"`
}

// XeadObservationError exposes only a stable code. It deliberately does not
// retain or unwrap filesystem paths, source bytes, or parser errors.
type XeadObservationError struct {
	Code string
}

func (e *XeadObservationError) Error() string {
	if e == nil {
		return "XEAD_OPERATION_FAILED"
	}
	return e.Code
}

// ObserveXeadSource reads one Workspace-relative .xead file and returns only
// the bounded Studio projection. The full parsed source Document is discarded.
func ObserveXeadSource(root, relativePath string) (XeadObservation, error) {
	normalizedPath, err := sourceindex.NormalizeWorkspacePath(relativePath)
	if err != nil || normalizedPath != relativePath {
		return XeadObservation{}, &XeadObservationError{Code: ObserveXeadSourceReadFailed}
	}
	if !strings.EqualFold(path.Ext(normalizedPath), ".xead") {
		return XeadObservation{}, &XeadObservationError{Code: ObserveXeadUnsupportedFormat}
	}

	content, err := workspace.ReadSource(root, relativePath, XeadObservationInputLimitBytes)
	if err != nil {
		code := ObserveXeadSourceReadFailed
		if errors.Is(err, workspace.ErrSourceTooLarge) {
			code = ObserveXeadSourceTooLarge
		}
		return XeadObservation{}, &XeadObservationError{Code: code}
	}

	document, err := xeadsource.ParseWithLimits(xeadsource.FormatXead, content, xeadsource.ParseLimits{
		MaxBytes:    XeadObservationInputLimitBytes,
		MaxDepth:    128,
		MaxElements: 100000,
	})
	if err != nil {
		return XeadObservation{}, &XeadObservationError{Code: ObserveXeadParseFailed}
	}
	projection, err := xeadsource.ProjectStudioObservation(document)
	if err != nil {
		return XeadObservation{}, &XeadObservationError{Code: ObserveXeadProjectionFailed}
	}

	return XeadObservation{
		SchemaVersion: "tei.operations.xead-observation/v0",
		Effect:        "read-only",
		Observation:   projection,
		AuthorityLimits: []string{
			"xead-observation-is-not-canonical-meaning",
			"xead-source-identifiers-are-not-canonical-identities",
			"xead-observation-does-not-authorize-edits-or-execution",
			"unsupported-source-constructs-remain-unresolved",
		},
	}, nil
}

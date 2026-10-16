package main

import (
	"tei/internal/studiohost"
	"tei/internal/xeadsource"
)

const (
	xeadParseFailed      = "XEAD_PARSE_FAILED"
	xeadProjectionFailed = "XEAD_PROJECTION_FAILED"
)

// studioXeadObserver is the only product-facing consumer of the RMT parser.
// It belongs to this nested Studio module and returns a bounded display DTO.
type studioXeadObserver struct{}

func (studioXeadObserver) Observe(content []byte) (studiohost.XeadObservation, error) {
	document, err := xeadsource.ParseWithLimits(xeadsource.FormatXead, content, xeadsource.ParseLimits{
		MaxBytes: studiohost.XeadObservationInputLimitBytes, MaxDepth: 128, MaxElements: 100000,
	})
	if err != nil {
		return studiohost.XeadObservation{}, &studiohost.XeadObservationError{Code: xeadParseFailed}
	}
	projection, err := xeadsource.ProjectStudioObservation(document)
	if err != nil {
		return studiohost.XeadObservation{}, &studiohost.XeadObservationError{Code: xeadProjectionFailed}
	}
	result := studiohost.XeadObservation{
		SchemaVersion: "tei.operations.xead-observation/v0",
		Effect:        "read-only",
		Observation: studiohost.XeadSourceView{
			Tables: make([]studiohost.XeadTable, 0, len(projection.Tables)),
		},
		AuthorityLimits: []string{
			"xead-observation-is-not-canonical-meaning",
			"xead-source-identifiers-are-not-canonical-identities",
			"xead-observation-does-not-authorize-edits-or-execution",
			"unsupported-source-constructs-remain-unresolved",
		},
	}
	for _, table := range projection.Tables {
		projected := studiohost.XeadTable{
			Source: studiohost.XeadSourceRef{Kind: table.Source.Kind, ID: table.Source.ID},
			Name:   table.Name, Fields: make([]studiohost.XeadField, 0, len(table.Fields)),
		}
		for _, field := range table.Fields {
			converted := studiohost.XeadField{
				Source: studiohost.XeadSourceRef{Kind: field.Source.Kind, ID: field.Source.ID}, Name: field.Name,
			}
			if field.DataType != nil {
				converted.DataType = &studiohost.XeadSourceRef{Kind: field.DataType.Kind, ID: field.DataType.ID}
			}
			projected.Fields = append(projected.Fields, converted)
		}
		result.Observation.Tables = append(result.Observation.Tables, projected)
	}
	for _, diagnostic := range projection.Diagnostics {
		result.Observation.Diagnostics = append(result.Observation.Diagnostics, studiohost.XeadDiagnostic{
			Code: diagnostic.Code, Element: diagnostic.Element, Attribute: diagnostic.Attribute,
			Occurrences: diagnostic.Occurrences,
		})
	}
	return result, nil
}

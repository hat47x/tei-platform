package threeelement

import (
	"errors"
	"strings"
)

var (
	// These errors describe a model-local read-only inspection, not Source
	// navigation or permission to edit a declaration.
	ErrCRUDSnapshotRequired = errors.New("CRUD snapshot fingerprint is required")
	ErrCRUDSnapshotStale    = errors.New("CRUD model snapshot changed")
	ErrCRUDCellNotFound     = errors.New("CRUD cell not found in the model snapshot")
	ErrCRUDCellAmbiguous    = errors.New("CRUD cell identity is ambiguous")
)

// CRUDCellInspection is one selected Function/Data cell and every declaration
// that contributed to the corresponding model-local origin group. Neither
// position nor fingerprint is a qualified Source location or write grant.
type CRUDCellInspection struct {
	ModelFingerprint   string               `json:"modelFingerprint"`
	FunctionID         string               `json:"functionId"`
	DataID             string               `json:"dataId"`
	Row                CRUDTable            `json:"row"`
	Facts              []CRUDFactOccurrence `json:"facts"`
	SourceQualification string              `json:"sourceQualification"`
}

// InspectCRUDCellAtSnapshot performs the first consumer-side join against one
// freshly computed model projection. A UI must supply the exact fingerprint
// it previously displayed. Pair IDs alone cannot qualify a prior snapshot.
// This deliberately makes no claim about Source revision, Scope or authority.
func InspectCRUDCellAtSnapshot(model ThreeElementModel, expectedFingerprint, functionID, dataID string) (CRUDCellInspection, error) {
	var empty CRUDCellInspection
	if expectedFingerprint == "" || strings.TrimSpace(expectedFingerprint) != expectedFingerprint {
		return empty, ErrCRUDSnapshotRequired
	}
	if functionID == "" || dataID == "" || strings.TrimSpace(functionID) != functionID ||
		strings.TrimSpace(dataID) != dataID {
		return empty, ErrCRUDCellNotFound
	}
	projection := GenerateCRUDSnapshotProjection(model)
	if projection.ModelFingerprint != expectedFingerprint {
		return empty, ErrCRUDSnapshotStale
	}

	// The projection groups by Function ID x Data ID. Do not turn a
	// collision within an element kind into an apparently unique target.
	functionCount, dataCount := 0, 0
	for _, fn := range model.FunctionElements {
		if fn.ID == functionID {
			functionCount++
		}
	}
	for _, data := range model.DataElements {
		if data.ID == dataID {
			dataCount++
		}
	}
	if functionCount > 1 || dataCount > 1 {
		return empty, ErrCRUDCellAmbiguous
	}

	var row *CRUDTable
	for i := range projection.Rows {
		candidate := &projection.Rows[i]
		if candidate.FunctionID != functionID || candidate.DataID != dataID {
			continue
		}
		if row != nil {
			return empty, ErrCRUDCellAmbiguous
		}
		row = candidate
	}
	var origin *CRUDCellOrigin
	for i := range projection.Origins {
		candidate := &projection.Origins[i]
		if candidate.FunctionID != functionID || candidate.DataID != dataID {
			continue
		}
		if origin != nil {
			return empty, ErrCRUDCellAmbiguous
		}
		origin = candidate
	}
	if row == nil || origin == nil || len(origin.Facts) == 0 {
		return empty, ErrCRUDCellNotFound
	}
	facts := append([]CRUDFactOccurrence(nil), origin.Facts...)
	return CRUDCellInspection{
		ModelFingerprint:    projection.ModelFingerprint,
		FunctionID:          functionID,
		DataID:              dataID,
		Row:                 *row,
		Facts:               facts,
		SourceQualification: "model-local-only",
	}, nil
}

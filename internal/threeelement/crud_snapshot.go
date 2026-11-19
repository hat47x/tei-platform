package threeelement

// CRUDSnapshotProjection keeps the aggregated cells and their original
// declarations together under one model fingerprint. It is a read-only
// projection, not a Source locator or an authorization to edit.
type CRUDSnapshotProjection struct {
	ModelFingerprint string           `json:"modelFingerprint"`
	Rows             []CRUDTable      `json:"rows"`
	Origins          []CRUDCellOrigin `json:"origins"`
}

// GenerateCRUDSnapshotProjection derives both views from the same input
// snapshot. Consumers must not join a cell from one snapshot to origins from
// another solely by FunctionID and DataID.
func GenerateCRUDSnapshotProjection(model ThreeElementModel) CRUDSnapshotProjection {
	return CRUDSnapshotProjection{
		ModelFingerprint: ModelFingerprint(model),
		Rows:             GenerateCRUDTable(model),
		Origins:          GenerateCRUDCellOrigins(model),
	}
}

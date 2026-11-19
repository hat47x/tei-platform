package main

import (
	"flag"
	"io"

	"tei/internal/threeelement"
)

// runThreeElementCRUDSnapshot exposes the existing same-model read-only
// projection without changing the established "crud" response contract.
func runThreeElementCRUDSnapshot(args []string, out io.Writer) (int, error) {
	path, err := parseThreeElementModelPath(args)
	if err != nil {
		return failThreeElement("INVALID_ARGUMENT", err.Error(), 2, out)
	}
	if path == "" {
		printHelp(out, flag.NewFlagSet("three-element crud-snapshot", flag.ContinueOnError), "three-element crud-snapshot")
		return 0, nil
	}
	model, err := loadThreeElementModel(path)
	if err != nil {
		return failThreeElement("MODEL_LOAD_FAILED", err.Error(), 2, out)
	}
	projection := threeelement.GenerateCRUDSnapshotProjection(model)
	report := threeelement.ValidateModel(model)
	return writeThreeElementResponse(out, threeElementResponse{
		SchemaVersion: "tei.cli.three-element.crud-snapshot/v0",
		OK:            true,
		OKScope:       "projection-produced-only",
		Result: struct {
			SchemaVersion string `json:"schemaVersion"`
			threeelement.CRUDSnapshotProjection
			StructuralOK       bool   `json:"structuralOk"`
			CRUDConflictStatus string `json:"crudConflictStatus"`
		}{
			SchemaVersion:          "tei.three-element.crud-snapshot/v0",
			CRUDSnapshotProjection: projection,
			StructuralOK:           report.OK,
			CRUDConflictStatus:     report.CRUDConflictStatus,
		},
	}, 0)
}

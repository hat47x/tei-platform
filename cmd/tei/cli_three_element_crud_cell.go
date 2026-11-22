package main

import (
	"errors"
	"flag"
	"io"
	"strings"

	"tei/internal/threeelement"
)

// runThreeElementCRUDCell is a read-only model-local Consumer for the existing
// CRUD snapshot. It never treats a model fingerprint as a Source revision.
func runThreeElementCRUDCell(args []string, out io.Writer) (int, error) {
	fs := flag.NewFlagSet("three-element crud-cell", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	modelPath := fs.String("model", "", "path to three-element model JSON")
	expected := fs.String("expected-fingerprint", "", "fingerprint from the previous CRUD snapshot")
	functionID := fs.String("function", "", "exact Function ID")
	dataID := fs.String("data", "", "exact Data ID")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(out, fs, "three-element crud-cell")
			return 0, nil
		}
		return failThreeElement("INVALID_ARGUMENT", err.Error(), 2, out)
	}
	if strings.TrimSpace(*modelPath) == "" || strings.TrimSpace(*functionID) == "" ||
		strings.TrimSpace(*dataID) == "" || strings.TrimSpace(*expected) == "" {
		return failThreeElement("INVALID_ARGUMENT",
			"three-element crud-cell requires -model, -expected-fingerprint, -function and -data", 2, out)
	}
	model, err := loadThreeElementModel(*modelPath)
	if err != nil {
		return failThreeElement("MODEL_LOAD_FAILED", err.Error(), 2, out)
	}
	cell, err := threeelement.InspectCRUDCellAtSnapshot(model, *expected, *functionID, *dataID)
	if err != nil {
		switch {
		case errors.Is(err, threeelement.ErrCRUDSnapshotRequired):
			return failThreeElement("SNAPSHOT_REQUIRED", err.Error(), 2, out)
		case errors.Is(err, threeelement.ErrCRUDSnapshotStale):
			return failThreeElement("STALE_SNAPSHOT", err.Error(), 2, out)
		case errors.Is(err, threeelement.ErrCRUDCellAmbiguous):
			return failThreeElement("CRUD_CELL_AMBIGUOUS", err.Error(), 2, out)
		default:
			return failThreeElement("CRUD_CELL_NOT_FOUND", err.Error(), 2, out)
		}
	}
	report := threeelement.ValidateModel(model)
	return writeThreeElementResponse(out, threeElementResponse{
		SchemaVersion: "tei.cli.three-element.crud-cell/v0",
		OK:            true,
		OKScope:       "model-snapshot-cell-inspected-only",
		Result: struct {
			SchemaVersion      string `json:"schemaVersion"`
			threeelement.CRUDCellInspection
			StructuralOK       bool   `json:"structuralOk"`
			CRUDConflictStatus string `json:"crudConflictStatus"`
		}{
			SchemaVersion:      "tei.three-element.crud-cell/v0",
			CRUDCellInspection: cell,
			StructuralOK:       report.OK,
			CRUDConflictStatus: report.CRUDConflictStatus,
		},
	}, 0)
}

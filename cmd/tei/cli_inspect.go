package main

import (
	"encoding/json"
	"errors"
	"io"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

type cliInspectionFailure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type cliInspectionReport struct {
	SchemaVersion string                        `json:"schemaVersion"`
	OK            bool                          `json:"ok"`
	Result        *operations.SubjectInspection `json:"result,omitempty"`
	Error         *cliInspectionFailure         `json:"error,omitempty"`
}

// inspectWorkspaceJSON owns only CLI argument/JSON/exit transport concerns.
// Inspection semantics and provenance remain in the shared Go operation.
// A nonzero exit still emits exactly one machine-readable JSON document.
func inspectWorkspaceJSON(out io.Writer, root, world, kind, id string, positional int) (int, error) {
	report := cliInspectionReport{SchemaVersion: "tei.cli.inspect/v0"}
	if root == "" || world != "" || kind == "" || id == "" || positional != 0 {
		report.Error = &cliInspectionFailure{
			Code:    operations.InspectionInvalidSubject,
			Message: "inspect requires -workspace, -subject-kind and -subject-id; -world and positional arguments are not supported",
		}
		return writeInspectionReport(out, report, 2)
	}
	result, err := operations.InspectSubject(root, sourceindex.Subject{Kind: kind, ID: id})
	if err != nil {
		var inspection *operations.InspectionError
		var stage *operations.ValidationFailure
		code := 2
		failure := &cliInspectionFailure{Code: "INSPECTION_FAILED", Message: err.Error()}
		switch {
		case errors.As(err, &inspection):
			failure.Code = inspection.Code
			if inspection.Code == operations.InspectionNotFound || inspection.Code == operations.InspectionAmbiguous {
				code = 1
			}
		case errors.As(err, &stage):
			switch stage.Stage {
			case operations.StageLoad:
				failure.Code = "WORKSPACE_LOAD_FAILED"
			case operations.StageSourceIndex:
				failure.Code = "SOURCE_INDEX_FAILED"
			}
		}
		report.Error = failure
		return writeInspectionReport(out, report, code)
	}
	report.OK = true
	report.Result = &result
	return writeInspectionReport(out, report, 0)
}

func writeInspectionReport(out io.Writer, report cliInspectionReport, exitCode int) (int, error) {
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(report); err != nil {
		return 2, err
	}
	return exitCode, nil
}

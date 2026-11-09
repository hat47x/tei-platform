package main

import (
	"encoding/json"
	"errors"
	"io"

	"tei/internal/operations"
)

// cliValidationReport is the document `validate -workspace ... -json` writes.
// The type lives in internal/operations so the read-only MCP adapter writes
// the same document instead of reshaping the operation result (#3420).
type cliValidationReport = operations.WorkspaceValidationReport

// Source locations never synthesize an empty Semantic Subject for parse errors.
type cliValidationItem = operations.WorkspaceValidationItem

type cliValidationSource = operations.WorkspaceValidationSource

type cliValidationFailure = operations.ValidationReportFailure

// invalidJSONValidationArguments keeps an opt-in JSON response machine-readable
// even when mutually exclusive or missing input flags are supplied.
func invalidJSONValidationArguments(out io.Writer) (int, error) {
	return writeValidationReport(out, cliValidationReport{
		SchemaVersion: "tei.cli.validate/v0",
		Diagnostics:   []cliValidationItem{},
		Error: &cliValidationFailure{
			Code:    "INVALID_ARGUMENT",
			Message: "validate -json requires -workspace, forbids -world and positional arguments",
		},
	}, 2)
}

// invalidJSONValidationParse preserves the opt-in JSON transport contract even
// when FlagSet parsing fails (unknown flag or an invalid flag value).
func invalidJSONValidationParse(out io.Writer, cause error) (int, error) {
	return writeValidationReport(out, cliValidationReport{
		SchemaVersion: "tei.cli.validate/v0",
		Diagnostics:   []cliValidationItem{},
		Error: &cliValidationFailure{
			Code:    "INVALID_ARGUMENT",
			Message: cause.Error(),
		},
	}, 2)
}

// validateWorkspaceJSON preserves the existing text command and exit codes:
// 0 = validated, 1 = validation findings, 2 = load/index failure.
func validateWorkspaceJSON(out io.Writer, root string) (int, error) {
	result, err := operations.ValidateWorkspace(root)
	if err != nil {
		code := "WORKSPACE_VALIDATION_FAILED"
		var failure *operations.ValidationFailure
		if errors.As(err, &failure) {
			switch failure.Stage {
			case operations.StageLoad:
				code = "WORKSPACE_LOAD_FAILED"
			case operations.StageSourceIndex:
				code = "SOURCE_INDEX_FAILED"
			}
		}
		return writeValidationReport(out, cliValidationReport{
			SchemaVersion: operations.ValidationSchemaVersion,
			Diagnostics:   []cliValidationItem{},
			Error:         &cliValidationFailure{Code: code, Message: err.Error()},
		}, 2)
	}
	// One projection of the operation result, shared with the MCP adapter, so
	// both transports answer the same document (#3420).
	report := operations.WorkspaceValidationReportFrom(result)
	if result.Direct != nil {
		return writeValidationReport(out, report, 1)
	}
	if !report.OK {
		return writeValidationReport(out, report, 1)
	}
	return writeValidationReport(out, report, 0)
}

func writeValidationReport(out io.Writer, report cliValidationReport, code int) (int, error) {
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(report); err != nil {
		return 2, err
	}
	return code, nil
}

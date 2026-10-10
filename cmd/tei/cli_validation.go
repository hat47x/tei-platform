package main

import (
	"encoding/json"
	"io"

	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// cliValidationReport is a projection for automation, not a second semantic
// validation model or a grant of authority to modify accepted source.
type cliValidationReport struct {
	SchemaVersion string                          `json:"schemaVersion"`
	OK            bool                            `json:"ok"`
	Diagnostics   []workspace.ProjectedDiagnostic `json:"diagnostics"`
	Error         *cliValidationFailure           `json:"error,omitempty"`
}

type cliValidationFailure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// validateWorkspaceJSON preserves the existing text command and exit codes:
// 0 = validated, 1 = validation findings, 2 = load/index failure.
func validateWorkspaceJSON(out io.Writer, root string) (int, error) {
	report := cliValidationReport{
		SchemaVersion: "tei.cli.validate/v0",
		Diagnostics:   []workspace.ProjectedDiagnostic{},
	}
	result, err := workspace.Load(root)
	if err != nil {
		if direct, ok := workspace.SourceDiagnosticFromError(root, err); ok {
			report.Diagnostics = append(report.Diagnostics, workspace.ProjectedDiagnostic{
				Diagnostic: validation.Diagnostic{
					Code: direct.Code, Severity: direct.Severity, Message: direct.Message,
				},
				// There is no Semantic Subject before parsing completes.
				Sources: []sourceindex.Reference{{
					ArtifactPath: direct.Source.ArtifactPath,
					Range:        direct.Source.Range,
				}},
			})
			return writeValidationReport(out, report, 1)
		}
		report.Error = &cliValidationFailure{Code: "WORKSPACE_LOAD_FAILED", Message: err.Error()}
		return writeValidationReport(out, report, 2)
	}
	index, err := workspace.BuildSourceIndex(root, result)
	if err != nil {
		report.Error = &cliValidationFailure{Code: "SOURCE_INDEX_FAILED", Message: err.Error()}
		return writeValidationReport(out, report, 2)
	}
	report.Diagnostics = workspace.ProjectDiagnostics(validation.Validate(result.World), index)
	report.OK = true
	for _, item := range report.Diagnostics {
		if item.Diagnostic.Severity == validation.Error {
			report.OK = false
			break
		}
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

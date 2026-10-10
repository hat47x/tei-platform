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
	Diagnostics   []cliValidationItem             `json:"diagnostics"`
	Error         *cliValidationFailure           `json:"error,omitempty"`
}

// Source locations never synthesize an empty Semantic Subject for parse errors.
type cliValidationItem struct {
	Diagnostic validation.Diagnostic `json:"diagnostic"`
	Sources    []cliValidationSource `json:"sources,omitempty"`
}

type cliValidationSource struct {
	ArtifactPath string           `json:"artifactPath"`
	Range        sourceindex.Range `json:"range"`
	Role         sourceindex.Role  `json:"role,omitempty"`
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
		Diagnostics:   []cliValidationItem{},
	}
	result, err := workspace.Load(root)
	if err != nil {
		if direct, ok := workspace.SourceDiagnosticFromError(root, err); ok {
			report.Diagnostics = append(report.Diagnostics, cliValidationItem{
				Diagnostic: validation.Diagnostic{
					Code: direct.Code, Severity: direct.Severity, Message: direct.Message,
				},
				// There is no Semantic Subject before parsing completes.
				Sources: []cliValidationSource{{
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
	for _, projected := range workspace.ProjectDiagnostics(validation.Validate(result.World), index) {
		item := cliValidationItem{Diagnostic: projected.Diagnostic}
		for _, ref := range projected.Sources {
			item.Sources = append(item.Sources, cliValidationSource{
				ArtifactPath: ref.ArtifactPath,
				Range:        ref.Range,
				Role:         ref.Role,
			})
		}
		report.Diagnostics = append(report.Diagnostics, item)
	}
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

package operations

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// WorkspaceValidationReport is the transport-neutral projection of a
// WorkspaceValidation. The CLI writes it as `validate -workspace ... -json`,
// and the read-only MCP adapter writes it as `tei_validate`, so both adapters
// answer one document instead of each reshaping the operation result (#3420).
//
// The existing CLI field names are kept: `schemaVersion`, `ok`, `diagnostics`,
// `worldFingerprint`, `sourceIndexFingerprint`, `proposals`,
// `declarationCounts`, `workspace`. Adding them here does not change what the
// CLI already wrote.
type WorkspaceValidationReport struct {
	SchemaVersion          string                                 `json:"schemaVersion"`
	OK                     bool                                   `json:"ok"`
	Diagnostics            []WorkspaceValidationItem              `json:"diagnostics"`
	WorldFingerprint       string                                 `json:"worldFingerprint,omitempty"`
	SourceIndexFingerprint string                                 `json:"sourceIndexFingerprint,omitempty"`
	Error                  *ValidationReportFailure               `json:"error,omitempty"`
	Proposals              []validation.Proposal                  `json:"proposals,omitempty"`
	DeclarationCounts      *DeclarationCounts                     `json:"declarationCounts,omitempty"`
	SourceRootFindings     []workspace.WorkspaceSourceRootFinding `json:"sourceRootFindings,omitempty"`
	Workspace              *workspace.WorkspaceIdentity           `json:"workspace,omitempty"`
	// Incomplete states why coverage cannot be claimed. StoppedAt is present
	// for a fail-fast run, not for a collection whose coverage is unverified.
	// Absence does not prove every file was clean (#4178).
	Incomplete *WorkspaceValidationIncomplete `json:"incomplete,omitempty"`
}

// WorkspaceValidationIncomplete names the truncation of a validation run.
type WorkspaceValidationIncomplete struct {
	Reason    string `json:"reason"`
	Detail    string `json:"detail"`
	StoppedAt string `json:"stoppedAt,omitempty"`
}

// WorkspaceValidationItem is one diagnostic with its source locations. A
// syntax finding keeps its source range and never synthesizes a Subject.
type WorkspaceValidationItem struct {
	Diagnostic              validation.Diagnostic       `json:"diagnostic"`
	Sources                 []WorkspaceValidationSource `json:"sources,omitempty"`
	UnresolvedSourceOrigins []string                    `json:"unresolvedSourceOrigins,omitempty"`
}

// WorkspaceValidationSource is one Source range a diagnostic points at.
type WorkspaceValidationSource struct {
	ArtifactPath string               `json:"artifactPath"`
	Range        sourceindex.Range    `json:"range"`
	Role         sourceindex.Role     `json:"role,omitempty"`
	Subject      *sourceindex.Subject `json:"subject,omitempty"`
}

// ValidationReportFailure is the machine-readable failure of a validation read.
type ValidationReportFailure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ValidationSchemaVersion is the schemaVersion both adapters publish for the
// validate document.
const ValidationSchemaVersion = "tei.cli.validate/v0"

// WorkspaceValidationReportFrom projects the operation result into the shared
// document. It is a pure function of the result: no Workspace is read again.
func WorkspaceValidationReportFrom(result WorkspaceValidation) WorkspaceValidationReport {
	report := WorkspaceValidationReport{
		SchemaVersion: ValidationSchemaVersion,
		Diagnostics:   []WorkspaceValidationItem{},
	}
	if result.Direct != nil {
		directs := append([]workspace.SourceDiagnostic{*result.Direct}, result.AdditionalDirect...)
		for _, direct := range directs {
			report.Diagnostics = append(report.Diagnostics, WorkspaceValidationItem{
				Diagnostic: validation.Diagnostic{
					Code: direct.Code, Severity: direct.Severity, Message: direct.Message,
				},
				Sources: []WorkspaceValidationSource{{
					ArtifactPath: direct.Source.ArtifactPath,
					Range:        direct.Source.Range,
				}},
			})
		}
		if result.InspectionIncomplete {
			report.Incomplete = &WorkspaceValidationIncomplete{
				Reason: "syntax-coverage-unverified",
				Detail: "independent syntax findings were collected; coverage is unverified and semantic validation was not performed",
			}
		} else {
			// Preserve the prior projection for a caller that supplies one
			// fail-fast diagnostic without the collection marker.
			report.Incomplete = &WorkspaceValidationIncomplete{
				Reason:    "load-stopped-at-first-syntax-error",
				Detail:    "the run stopped at the first syntax error; files other than the one named in stoppedAt were not examined",
				StoppedAt: result.Direct.Source.ArtifactPath,
			}
		}
		return report
	}
	report.WorldFingerprint = result.WorldFingerprint
	report.SourceIndexFingerprint = result.SourceIndexFingerprint
	declarations := result.Declarations
	report.DeclarationCounts = &declarations
	report.SourceRootFindings = result.SourceRootFindings
	report.Workspace = result.Workspace
	if len(result.Diagnostics) > 0 && result.WorldFingerprint != "" {
		diagnostics := make([]validation.Diagnostic, 0, len(result.Diagnostics))
		for _, projected := range result.Diagnostics {
			diagnostics = append(diagnostics, projected.Diagnostic)
		}
		report.Proposals = validation.Propose(diagnostics, result.WorldFingerprint)
	}
	for _, projected := range result.Diagnostics {
		item := WorkspaceValidationItem{Diagnostic: projected.Diagnostic}
		if len(projected.UnresolvedSourceOrigins) > 0 {
			item.UnresolvedSourceOrigins = append([]string{}, projected.UnresolvedSourceOrigins...)
		}
		for _, ref := range projected.Sources {
			source := WorkspaceValidationSource{
				ArtifactPath: ref.ArtifactPath,
				Range:        ref.Range,
				Role:         ref.Role,
			}
			if projected.Diagnostic.Code == "DUPLICATE_SEMANTIC_ELEMENT" {
				subject := ref.Subject
				source.Subject = &subject
			}
			item.Sources = append(item.Sources, source)
		}
		report.Diagnostics = append(report.Diagnostics, item)
	}
	// A stable order makes the document usable in repeatable tool workflows.
	sort.Slice(report.Diagnostics, func(i, j int) bool {
		return ValidationDiagnosticSortKey(report.Diagnostics[i]) < ValidationDiagnosticSortKey(report.Diagnostics[j])
	})
	report.OK = true
	for _, finding := range report.SourceRootFindings {
		if finding.Severity == "error" {
			report.OK = false
			break
		}
	}
	for _, item := range report.Diagnostics {
		if item.Diagnostic.Severity == validation.Error {
			report.OK = false
			break
		}
	}
	return report
}

// WorkspaceValidationFailureReport is the document both adapters write when the
// validate read itself failed (load or Source Index). `diagnostics` is present
// and empty, as the CLI always writes it, so a caller never has to treat the
// missing field and an empty list as different answers.
func WorkspaceValidationFailureReport(code, message string) WorkspaceValidationReport {
	return WorkspaceValidationReport{
		SchemaVersion: ValidationSchemaVersion,
		Diagnostics:   []WorkspaceValidationItem{},
		Error:         &ValidationReportFailure{Code: code, Message: message},
	}
}

// ValidationDiagnosticSortKey is the length-prefixed key that orders one
// diagnostic item deterministically.
func ValidationDiagnosticSortKey(item WorkspaceValidationItem) string {
	d := item.Diagnostic
	fields := []string{string(d.Severity), d.Code, d.Message}
	if d.Subject != nil {
		fields = append(fields, d.Subject.Kind, d.Subject.ID)
	}
	for _, s := range item.Sources {
		fields = append(fields, s.ArtifactPath,
			fmt.Sprintf("%012d:%012d", s.Range.Start.Line, s.Range.Start.Column),
			string(s.Role))
	}
	for i, field := range fields {
		fields[i] = fmt.Sprintf("%d:%s", len(field), field)
	}
	return strings.Join(fields, "|")
}

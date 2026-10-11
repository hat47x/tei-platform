package operations

import (
	"tei/internal/validation"
	"tei/internal/workspace"
)

// WorkspaceValidation is a transport-independent read-only result shared by
// CLI and future HTTP / desktop adapters. It does not accept a proposal,
// mutate a workspace, or confer any execution authority.
type WorkspaceValidation struct {
	Direct      *workspace.SourceDiagnostic
	Diagnostics []workspace.ProjectedDiagnostic
}

func (r WorkspaceValidation) HasErrors() bool {
	if r.Direct != nil {
		return r.Direct.Severity == validation.Error
	}
	for _, d := range r.Diagnostics {
		if d.Diagnostic.Severity == validation.Error {
			return true
		}
	}
	return false
}

// ValidationStage explains which read-only boundary failed. It is not a
// business rule / semantic validation finding.
type ValidationStage string

const (
	StageLoad        ValidationStage = "load"
	StageSourceIndex ValidationStage = "source-index"
)

type ValidationFailure struct {
	Stage ValidationStage
	Err   error
}

func (f *ValidationFailure) Error() string {
	if f == nil {
		return "validation failure"
	}
	// Preserve legacy human-facing error text; callers inspect Stage via errors.As.
	return f.Err.Error()
}

func (f *ValidationFailure) Unwrap() error {
	if f == nil {
		return nil
	}
	return f.Err
}

// ValidateWorkspace parses the same authoritative source and projects
// existing validation diagnostics onto the existing Source Index. A syntax
// finding has no Canonical Subject and is kept distinct from semantic errors.
func ValidateWorkspace(root string) (WorkspaceValidation, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		if direct, ok := workspace.SourceDiagnosticFromError(root, err); ok {
			return WorkspaceValidation{Direct: &direct}, nil
		}
		return WorkspaceValidation{}, &ValidationFailure{Stage: StageLoad, Err: err}
	}
	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return WorkspaceValidation{}, &ValidationFailure{Stage: StageSourceIndex, Err: err}
	}
	return WorkspaceValidation{
		Diagnostics: workspace.ProjectDiagnostics(validation.Validate(loaded.World), index),
	}, nil
}

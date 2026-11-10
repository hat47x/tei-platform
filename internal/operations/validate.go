package operations

import (
	"errors"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// WorkspaceValidation is a transport-independent read-only result shared by
// CLI and future HTTP / desktop adapters. It does not accept a proposal,
// mutate a workspace, or confer any execution authority.
//
// WorldFingerprint and SourceIndexFingerprint identify the version that this
// single load produced the Diagnostics from, with the same computation as
// inspect and model catalog. They are empty when Direct is set, because a
// syntax finding means no World or Source Index was built. They are evidence
// of the read version only, not of correctness or acceptance.
type WorkspaceValidation struct {
	Direct                 *workspace.SourceDiagnostic
	AdditionalDirect       []workspace.SourceDiagnostic
	InspectionIncomplete   bool
	Diagnostics            []workspace.ProjectedDiagnostic
	WorldFingerprint       string
	SourceIndexFingerprint string
	// Declarations counts the Workspace-authored declarations. An empty
	// Workspace has Total 0 even though the standard semantic prelude is
	// loaded into World.Semantics (#2822).
	Declarations DeclarationCounts
	// SourceRootFindings are manifest source-root warnings or errors. They are
	// separate from semantic Diagnostics because an unknown or missing root has
	// no Canonical Subject (#2823).
	SourceRootFindings []workspace.WorkspaceSourceRootFinding
	// Workspace names the Workspace this validation read (#3181). It is absent
	// for a Direct syntax finding, which has no loaded Workspace identity.
	Workspace *workspace.WorkspaceIdentity
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
	loaded, err := workspace.LoadForValidation(root)
	if err != nil {
		var multiple *workspace.SourceErrors
		if errors.As(err, &multiple) && len(multiple.Errors) > 0 {
			result := WorkspaceValidation{InspectionIncomplete: true}
			for _, sourceError := range multiple.Errors {
				direct, ok := workspace.SourceDiagnosticFromError(root, sourceError)
				if !ok {
					return WorkspaceValidation{}, &ValidationFailure{Stage: StageLoad, Err: err}
				}
				if result.Direct == nil {
					result.Direct = &direct
				} else {
					result.AdditionalDirect = append(result.AdditionalDirect, direct)
				}
			}
			return result, nil
		}
		if direct, ok := workspace.SourceDiagnosticFromError(root, err); ok {
			return WorkspaceValidation{Direct: &direct, InspectionIncomplete: true}, nil
		}
		return WorkspaceValidation{}, &ValidationFailure{Stage: StageLoad, Err: err}
	}
	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return WorkspaceValidation{}, &ValidationFailure{Stage: StageSourceIndex, Err: err}
	}
	worldFP, sourceFP, err := workspaceProjectionFingerprints(loaded.World, index)
	if err != nil {
		return WorkspaceValidation{}, err
	}
	identity := loaded.Identity()
	return WorkspaceValidation{
		Diagnostics:            workspace.ProjectDiagnosticsWithLoadedDuplicates(validation.Validate(loaded.World), index, loaded),
		WorldFingerprint:       worldFP,
		SourceIndexFingerprint: sourceFP,
		Declarations:           CountDeclarations(loaded.World),
		SourceRootFindings:     loaded.SourceRootFindings,
		Workspace:              &identity,
	}, nil
}

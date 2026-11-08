package main

import (
	"fmt"
	"strings"

	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// sourceIndexStageError keeps the failure stage of a Source Index inspection so
// the CLI can map load failures to WORKSPACE_LOAD_FAILED and index/trace
// failures to SOURCE_INDEX_FAILED without string matching (#2354).
type sourceIndexStageError struct {
	Stage string
	Err   error
}

func (e *sourceIndexStageError) Error() string { return e.Err.Error() }
func (e *sourceIndexStageError) Unwrap() error { return e.Err }

const (
	sourceIndexStageLoad  = "load"
	sourceIndexStageBuild = "build"
)

type sourceIndexSubjectView = operations.SubjectSources

// sourceIndexValue is the whole-index projection. It carries the Workspace
// identity so a caller that reads several Workspaces can tell the indexes
// apart, and it always serializes both lists as arrays (#3455, #3456).
type sourceIndexValue struct {
	Artifacts  []sourceindex.Artifact       `json:"artifacts"`
	References []sourceindex.Reference      `json:"references"`
	Workspace  *workspace.WorkspaceIdentity `json:"workspace,omitempty"`
}

func buildSourceIndexInspection(root, subjectKind, subjectID string) (any, string, error) {
	if strings.TrimSpace(root) == "" {
		return nil, "", fmt.Errorf("source-index requires -workspace")
	}
	kind := strings.TrimSpace(subjectKind)
	id := strings.TrimSpace(subjectID)
	if (kind == "") != (id == "") {
		return nil, "", fmt.Errorf("-subject-kind and -subject-id must be supplied together")
	}

	result, err := workspace.Load(root)
	if err != nil {
		return nil, "", &sourceIndexStageError{Stage: sourceIndexStageLoad, Err: err}
	}
	index, err := workspace.BuildSourceIndex(root, result)
	if err != nil {
		return nil, "", &sourceIndexStageError{Stage: sourceIndexStageBuild, Err: err}
	}
	if kind == "" {
		return sourceIndexValue{
			Artifacts:  nonNilArtifacts(index.Artifacts),
			References: nonNilReferences(index.References),
			Workspace:  workspaceIdentityPointer(result),
		}, resultModeIndex, nil
	}

	// Reuse the index built for this invocation, rather than reloading the
	// Workspace and potentially mixing two different source snapshots.
	subjects, err := operations.TraceSubjectSourcesFromIndex(index, sourceindex.Subject{Kind: kind, ID: id})
	if err != nil {
		return nil, "", &sourceIndexStageError{Stage: sourceIndexStageBuild, Err: err}
	}
	return sourceIndexSubjectValue{
		SubjectSources: subjects,
		Workspace:      workspaceIdentityPointer(result),
	}, resultModeSubject, nil
}

// sourceIndexSubjectValue is the single-subject projection with the identity
// the read observed (#3455).
type sourceIndexSubjectValue struct {
	operations.SubjectSources
	Workspace *workspace.WorkspaceIdentity `json:"workspace,omitempty"`
}

// nonNilArtifacts and nonNilReferences keep the JSON arrays stable: an empty
// list is [] rather than null, so a caller can iterate without special-casing
// a missing field (#3456).
func nonNilArtifacts(values []sourceindex.Artifact) []sourceindex.Artifact {
	if values == nil {
		return []sourceindex.Artifact{}
	}
	return values
}

func nonNilReferences(values []sourceindex.Reference) []sourceindex.Reference {
	if values == nil {
		return []sourceindex.Reference{}
	}
	return values
}

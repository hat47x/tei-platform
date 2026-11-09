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

// sourceIndexIndexView is the whole-index projection the CLI writes. Empty
// lists are emitted as [] because internal/sourceindex.Index keeps a nil slice
// for "no references", which encoding/json writes as null; a reader that
// iterates result.references then fails on the empty case only (#3456).
// The Source Index fingerprint is computed from the index itself, not from this
// projection, so normalizing here does not change any reported fingerprint.
type sourceIndexIndexView struct {
	Artifacts  []sourceindex.Artifact       `json:"artifacts"`
	References []sourceindex.Reference      `json:"references"`
	Workspace  *workspace.WorkspaceIdentity `json:"workspace,omitempty"`
}

// sourceIndexSubjectValue is the single-subject projection with the Workspace
// identity the read observed (#3455).
type sourceIndexSubjectValue struct {
	operations.SubjectSources
	Workspace *workspace.WorkspaceIdentity `json:"workspace,omitempty"`
}

func newSourceIndexIndexView(index sourceindex.Index) sourceIndexIndexView {
	view := sourceIndexIndexView{
		Artifacts:  index.Artifacts,
		References: index.References,
	}
	if view.Artifacts == nil {
		view.Artifacts = []sourceindex.Artifact{}
	}
	if view.References == nil {
		view.References = []sourceindex.Reference{}
	}
	return view
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
		view := newSourceIndexIndexView(index)
		view.Workspace = workspaceIdentityPointer(result)
		return view, resultModeIndex, nil
	}

	// Reuse the index built for this invocation, rather than reloading the
	// Workspace and potentially mixing two different source snapshots.
	subjects, err := operations.TraceSubjectSourcesFromIndex(index, sourceindex.Subject{Kind: kind, ID: id})
	if err != nil {
		return nil, "", &sourceIndexStageError{Stage: sourceIndexStageBuild, Err: err}
	}
	// A subject with no Source references leaves References nil; write [] so
	// both modes answer an empty list the same way (#3456).
	if subjects.References == nil {
		subjects.References = []sourceindex.Reference{}
	}
	if subjects.Artifacts == nil {
		subjects.Artifacts = []sourceindex.Artifact{}
	}
	return sourceIndexSubjectValue{
		SubjectSources: subjects,
		Workspace:      workspaceIdentityPointer(result),
	}, resultModeSubject, nil
}

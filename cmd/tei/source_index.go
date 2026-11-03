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
		return index, resultModeIndex, nil
	}

	// Reuse the index built for this invocation, rather than reloading the
	// Workspace and potentially mixing two different source snapshots.
	subjects, err := operations.TraceSubjectSourcesFromIndex(index, sourceindex.Subject{Kind: kind, ID: id})
	if err != nil {
		return nil, "", &sourceIndexStageError{Stage: sourceIndexStageBuild, Err: err}
	}
	return subjects, resultModeSubject, nil
}

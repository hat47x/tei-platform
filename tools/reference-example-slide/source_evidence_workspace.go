package exampleslide

import (
	"fmt"
	"reflect"

	"tei/internal/workspace"
)

// AuditLoadedWorkspaceAnswerSourceEvidence links an illustrative review to
// source bytes from the *same* Workspace Load that produced its semantic
// Catalog and Source Index. A user-provided Catalog cannot impersonate that
// Workspace merely by copying its revision string.
//
// The audit reads a frozen authoring snapshot, not the current filesystem,
// and makes no claim about source loading's OS-level symlink race boundary.
func AuditLoadedWorkspaceAnswerSourceEvidence(root string, input Input, answers AnswerInput) (SourceEvidenceAudit, error) {
	var empty SourceEvidenceAudit
	loaded, err := workspace.Load(root)
	if err != nil {
		return empty, fmt.Errorf("load Workspace for source audit: %w", err)
	}
	catalog, err := CatalogFromWorld(loaded.World)
	if err != nil {
		return empty, fmt.Errorf("derive checked Workspace Catalog: %w", err)
	}
	if !reflect.DeepEqual(input.Catalog, catalog) {
		return empty, fmt.Errorf("illustrative Catalog differs from loaded Workspace semantics")
	}
	index, read, err := workspace.IndexedLoadedSnapshot(root, loaded)
	if err != nil {
		return empty, fmt.Errorf("bind Workspace Source Index snapshot: %w", err)
	}
	return AuditAnswerSourceEvidence(input, answers, index, read)
}

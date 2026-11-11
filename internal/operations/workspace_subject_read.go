package operations

import (
	"encoding/json"

	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// WorkspaceSubjectRead is ONE Workspace load that answers both the canonical
// projection of one subject and the managed Source revision that projection was
// read at. It exists because a caller that loaded the Workspace twice could pair
// a current World with an older Source Index and call the combination one
// snapshot; the document draft projection reads model identity, source
// provenance and revision together, so it takes them from a single read.
//
// The returned values are read-only observations. ModelRevision is the revision
// of the managed Source set, not of the World, and grants neither acceptance nor
// write authority.
type WorkspaceSubjectRead struct {
	Inspection    SubjectInspection
	ModelRevision string
	World         ir.World
}

// ReadWorkspaceSubject loads one Workspace, builds its Source Index and resolves
// one exact Canonical subject from that same snapshot. The canonical subject is
// resolved BEFORE the Source Index is built, so an unbound or failed index never
// masks a missing subject, matching InspectSubject's order.
func ReadWorkspaceSubject(root string, subject sourceindex.Subject) (WorkspaceSubjectRead, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return WorkspaceSubjectRead{}, &ValidationFailure{Stage: StageLoad, Err: err}
	}
	// Resolve the Canonical Subject before deriving provenance, exactly as
	// InspectSubject does: a source reference alone is never proof of a
	// canonical declaration.
	canonical, err := inspectCanonical(loaded.World, subject)
	if err != nil {
		return WorkspaceSubjectRead{}, err
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return WorkspaceSubjectRead{}, &ValidationFailure{Stage: StageSourceIndex, Err: err}
	}
	inspection, err := inspectionFromResolved(loaded, idx, subject, canonical)
	if err != nil {
		return WorkspaceSubjectRead{}, err
	}
	return WorkspaceSubjectRead{
		Inspection:    inspection,
		ModelRevision: sourceindex.ModelRevision(idx),
		World:         loaded.World,
	}, nil
}

// InspectionFromLoad projects one subject from an already loaded Workspace and
// its Source Index. It performs no filesystem read, so a caller that holds one
// load can answer several subjects from the same snapshot instead of reloading.
func InspectionFromLoad(loaded workspace.Result, idx sourceindex.Index, subject sourceindex.Subject) (SubjectInspection, error) {
	canonical, err := inspectCanonical(loaded.World, subject)
	if err != nil {
		return SubjectInspection{}, err
	}
	return inspectionFromResolved(loaded, idx, subject, canonical)
}

// inspectionFromResolved builds the projection once the canonical object is
// known. Both entry points above resolve the canonical subject first, so the
// failure they report for a missing subject stays SUBJECT_NOT_FOUND rather than
// a Source Index failure.
func inspectionFromResolved(
	loaded workspace.Result, idx sourceindex.Index,
	subject sourceindex.Subject, canonical json.RawMessage,
) (SubjectInspection, error) {
	sources, err := TraceSubjectSourcesFromIndex(idx, subject)
	if err != nil {
		return SubjectInspection{}, err
	}
	worldFP, sourceFP, err := WorkspaceProjectionFingerprints(loaded.World, idx)
	if err != nil {
		return SubjectInspection{}, err
	}
	status := "not-indexed"
	if len(sources.References) > 0 {
		status = "indexed"
	}
	identity := loaded.Identity()
	return SubjectInspection{
		SchemaVersion:          "tei.operations.inspect/v0",
		Effect:                 "read-only",
		Subject:                subject,
		Canonical:              canonical,
		WorldFingerprint:       worldFP,
		SourceIndexFingerprint: sourceFP,
		SourceStatus:           status,
		Workspace:              &identity,
		Sources:                sources,
		AuthorityLimits:        InspectionAuthorityLimits(),
	}, nil
}

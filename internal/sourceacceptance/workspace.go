package sourceacceptance

import (
	"context"
	"errors"
	"fmt"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// VerifiedWorkspace pairs one actual Workspace Load with the accepted Source
// membership checked against the Host's CURRENT witness. Consumers must use
// Loaded.World from this result, not a separate Workspace Load.
//
// SourceIndex is built only from the bytes Load captured (never re-read from
// the filesystem), so it identifies the bytes Loaded.World came from. This is
// still not atomic across files, detects no files omitted from membership,
// and G1/G2 stay UNQUALIFIED. It does not authenticate a Reader supplied by
// the caller or grant effect permission.
//
// A Workspace that declares imports is refused (SOURCE_WORKSPACE_IMPORTS_UNBOUND):
// the provider's content is not in the Source Index.
//
// Manifest is the sha256 of the workspace manifest bytes Load read. The
// manifest decides source roots and imports but is NOT part of the accepted
// Source Index, so Acceptance does not bind it. Manifest is Load-time evidence
// for the caller, not something the Host witness checked.
type VerifiedWorkspace struct {
	Loaded      workspace.Result
	SourceIndex sourceindex.Index
	Acceptance  Record
	Manifest    workspace.ManifestIdentity
}

// LoadVerifiedWorkspace is the reusable seam for a Target's pre-handler
// startup check. No Workspace/World is returned when the Reader is missing,
// the Source Index is unavailable, or current Host acceptance does not match.
//
// A trusted Host must supply Reader; a CLI string, scratch Decision record,
// or local JSON file must never be silently promoted to that authority.
func LoadVerifiedWorkspace(ctx context.Context, reader Reader, applicationID, root string) (VerifiedWorkspace, error) {
	return loadVerifiedWorkspace(ctx, reader, applicationID, root, nil)
}

// loadVerifiedWorkspace is the private seam behind LoadVerifiedWorkspace.
// afterLoad, when non-nil, runs between workspace.Load and the Source Index
// build so tests can change files in exactly that window. The exported
// function passes nil; there is no package-level hook.
func loadVerifiedWorkspace(ctx context.Context, reader Reader, applicationID, root string, afterLoad func()) (VerifiedWorkspace, error) {
	if reader == nil {
		return VerifiedWorkspace{}, errors.New("SOURCE_ACCEPTANCE_READER_REQUIRED")
	}
	if err := ctx.Err(); err != nil {
		return VerifiedWorkspace{}, fmt.Errorf("SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE: %w", err)
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return VerifiedWorkspace{}, fmt.Errorf("SOURCE_WORKSPACE_LOAD_FAILED: %w", err)
	}
	if afterLoad != nil {
		afterLoad()
	}
	index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if errors.Is(err, workspace.ErrImportsNotBound) {
		return VerifiedWorkspace{}, fmt.Errorf("SOURCE_WORKSPACE_IMPORTS_UNBOUND: %w", err)
	}
	if err != nil {
		return VerifiedWorkspace{}, fmt.Errorf("SOURCE_INDEX_BUILD_FAILED: source index is not bound to Load-time bytes: %w", err)
	}
	record, err := VerifyCurrent(ctx, reader, applicationID, index)
	if err != nil {
		return VerifiedWorkspace{}, err
	}
	return VerifiedWorkspace{Loaded: loaded, SourceIndex: index, Acceptance: record, Manifest: loaded.ManifestIdentity()}, nil
}

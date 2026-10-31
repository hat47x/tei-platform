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
// does not cover imported Workspaces, and G1/G2 stay UNQUALIFIED. It does not
// authenticate a Reader supplied by the caller or grant effect permission.
type VerifiedWorkspace struct {
	Loaded      workspace.Result
	SourceIndex sourceindex.Index
	Acceptance  Record
}

// LoadVerifiedWorkspace is the reusable seam for a Target's pre-handler
// startup check. No Workspace/World is returned when the Reader is missing,
// the Source Index is unavailable, or current Host acceptance does not match.
//
// A trusted Host must supply Reader; a CLI string, scratch Decision record,
// or local JSON file must never be silently promoted to that authority.
func LoadVerifiedWorkspace(ctx context.Context, reader Reader, applicationID, root string) (VerifiedWorkspace, error) {
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
	index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		return VerifiedWorkspace{}, fmt.Errorf("SOURCE_INDEX_BUILD_FAILED: source index is not bound to Load-time bytes: %w", err)
	}
	record, err := VerifyCurrent(ctx, reader, applicationID, index)
	if err != nil {
		return VerifiedWorkspace{}, err
	}
	return VerifiedWorkspace{Loaded: loaded, SourceIndex: index, Acceptance: record}, nil
}

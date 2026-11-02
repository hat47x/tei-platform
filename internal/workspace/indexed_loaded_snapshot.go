package workspace

import (
	"bytes"
	"fmt"

	"tei/internal/sourceindex"
)

// IndexedLoadedSnapshot binds one Source Index to exactly the bytes consumed
// by Load. It returns a read-only, indexed, in-memory source reader. It never
// opens a file during evidence inspection, so a subsequent filesystem rename
// or symlink change cannot redirect that inspection to a different file.
//
// This is an authoring snapshot, NOT proof of the *current* filesystem state.
// Load reads manifests and every managed Source (parsed and forward-compatible)
// from a single pinned os.Root. However inventory/Source enumeration still use
// filesystem paths, and in-place modifications remain possible; Load is not an
// atomic multi-file snapshot.
func IndexedLoadedSnapshot(root string, loaded Result) (sourceindex.Index, func(string) ([]byte, error), error) {
	var empty sourceindex.Index
	physical, err := canonicalWorkspaceRoot(root)
	if err != nil {
		return empty, nil, err
	}
	if loaded.sourceSnapshotRoot == "" || loaded.sourceSnapshotRoot != physical || loaded.sourceSnapshot == nil {
		return empty, nil, fmt.Errorf("Source evidence requires a Load snapshot of the same Workspace")
	}
	index, err := BuildSourceIndex(root, loaded)
	if err != nil {
		return empty, nil, err
	}
	snapshot := make(map[string][]byte, len(index.Artifacts))
	for _, artifact := range index.Artifacts {
		content, ok := loaded.sourceSnapshot[artifact.Path]
		if !ok {
			// A BuildSourceIndex fallback that read files after Load is not
			// sufficient: it cannot be tied to Load-time bytes. A partial
			// snapshot (for example from a forged Result) fails closed.
			return empty, nil, fmt.Errorf("source %q has no Load-time snapshot", artifact.Path)
		}
		copied := bytes.Clone(content)
		if got := sourceindex.Fingerprint(copied); got != artifact.Fingerprint {
			return empty, nil, fmt.Errorf("source %q snapshot does not match Source Index", artifact.Path)
		}
		snapshot[artifact.Path] = copied
	}
	read := func(raw string) ([]byte, error) {
		p, err := sourceindex.NormalizeWorkspacePath(raw)
		if err != nil {
			return nil, err
		}
		if p != raw {
			return nil, fmt.Errorf("source path %q is not normalized", raw)
		}
		content, exists := snapshot[p]
		if !exists {
			return nil, fmt.Errorf("source %q is not indexed in this Load snapshot", p)
		}
		return bytes.Clone(content), nil
	}
	if err := index.Validate(read); err != nil {
		return empty, nil, fmt.Errorf("Load snapshot Source Index: %w", err)
	}
	return index, read, nil
}

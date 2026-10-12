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
// Source loading itself still uses the best-effort containment reader tracked
// by issue #1017; this function makes no race-free Load claim.
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
			// sufficient: it cannot be tied to parsed loader bytes.
			return empty, nil, fmt.Errorf("source %q has no parsed Load snapshot", artifact.Path)
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

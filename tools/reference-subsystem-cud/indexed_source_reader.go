package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tei/internal/sourceindex"
)

// indexedSourceReader confines indexed-source reads to an explicitly selected
// source root, even when the indexed path traverses symlinks. The caller still
// decides which root to trust. This is a best-effort filesystem guard, not an
// atomic sandbox against concurrent replacement of source files.
func indexedSourceReader(root string) (func(string) ([]byte, error), error) {
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve indexed source root: %w", err)
	}
	canonicalRoot, err = filepath.Abs(canonicalRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute indexed source root: %w", err)
	}
	info, err := os.Stat(canonicalRoot)
	if err != nil {
		return nil, fmt.Errorf("stat indexed source root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("indexed source root is not a directory: %q", root)
	}

	return func(raw string) ([]byte, error) {
		normalized, err := sourceindex.NormalizeWorkspacePath(raw)
		if err != nil || normalized != raw {
			return nil, fmt.Errorf("invalid indexed source path %q", raw)
		}
		resolved, err := filepath.EvalSymlinks(filepath.Join(canonicalRoot, filepath.FromSlash(normalized)))
		if err != nil {
			return nil, fmt.Errorf("resolve indexed source path %q: %w", raw, err)
		}
		relative, err := filepath.Rel(canonicalRoot, resolved)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
			return nil, fmt.Errorf("indexed source path %q resolves outside the source root", raw)
		}
		return os.ReadFile(resolved)
	}, nil
}

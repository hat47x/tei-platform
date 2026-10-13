package workspace

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"

	"tei/internal/sourceindex"
)

// canonicalWorkspaceRoot identifies a Workspace independently of its caller's
// relative/absolute spelling. It does not freeze the underlying filesystem.
func canonicalWorkspaceRoot(root string) (string, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("absolute Workspace root: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve Workspace root: %w", err)
	}
	return resolved, nil
}

// readSourceForLoad retains exactly the bytes consumed by a domain loader.
// It does not provide an open-time containment guarantee: the shared
// race-free Source Reader remains tracked by issue #1017.
func readSourceForLoad(root string, result *Result, filename string) ([]byte, error) {
	relative, err := filepath.Rel(root, filename)
	if err != nil {
		return nil, fmt.Errorf("source %q relative to Workspace: %w", filename, err)
	}
	path, err := sourceindex.NormalizeWorkspacePath(filepath.ToSlash(relative))
	if err != nil {
		return nil, fmt.Errorf("source %q: %w", filename, err)
	}
	if result.sourceSnapshotRoot == "" {
		return nil, fmt.Errorf("source reader has no Workspace root")
	}
	content, err := readWorkspaceFile(result.sourceSnapshotRoot, path)
	if err != nil {
		return nil, err
	}
	if result.sourceSnapshot != nil {
		if previous, exists := result.sourceSnapshot[path]; exists {
			if !bytes.Equal(previous, content) {
				return nil, fmt.Errorf("source %q changed while loading Workspace", path)
			}
		} else {
			// Keep a private copy so later parser operations cannot alter the
			// bytes used for the Source Index fingerprint and ranges.
			result.sourceSnapshot[path] = bytes.Clone(content)
		}
	}
	return content, nil
}

// readWorkspaceFile reads from a directory-handle-relative, symlink-denying
// boundary. On targets without a race-free implementation it fails closed.
// This deliberately disallows even symlinks resolving inside the Workspace.
func readWorkspaceFile(rootReal, rawPath string) ([]byte, error) {
	path, err := sourceindex.NormalizeWorkspacePath(rawPath)
	if err != nil {
		return nil, err
	}
	file, err := openRootedSource(rootReal, path)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("stat source %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		file.Close()
		return nil, fmt.Errorf("source %q is not a regular file", path)
	}
	content, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read source %q: %w", path, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close source %q: %w", path, closeErr)
	}
	return content, nil
}

package workspace

import (
	"bytes"
	"fmt"
	"io"
	"os"
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
// During Load, all domain Source files come from the same pinned directory
// handle used for the manifest. A returned Result retains only frozen bytes,
// not the OS handle. Files can still be modified in place between reads.
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
	// Load has a single pinned root. Direct calls to a specific loader
	// (including its standalone tests) retain the safe per-file os.Root
	// boundary without pretending to have a whole-Load pin.
	var content []byte
	if result.sourceReadRoot != nil {
		content, err = readPinnedWorkspaceFile(result.sourceReadRoot, path)
	} else {
		content, err = readWorkspaceFile(result.sourceSnapshotRoot, path)
	}
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

// readWorkspaceFile opens a source through a traversal-resistant root.
// Relative symlinks that remain inside the root are supported; escapes are
// rejected. Targets without a trustworthy rooted-open implementation fail
// closed rather than downgrading to EvalSymlinks followed by os.Open.
func readWorkspaceFile(rootReal, rawPath string) ([]byte, error) {
	path, err := sourceindex.NormalizeWorkspacePath(rawPath)
	if err != nil {
		return nil, err
	}
	file, err := openRootedSource(rootReal, path)
	if err != nil {
		return nil, err
	}
	return readOpenedWorkspaceSource(file, path)
}

// readPinnedWorkspaceFile does not re-resolve a Workspace pathname. Every
// domain Source read and the manifest itself use one Root for the entire Load.
func readPinnedWorkspaceFile(root *os.Root, rawPath string) ([]byte, error) {
	path, err := sourceindex.NormalizeWorkspacePath(rawPath)
	if err != nil {
		return nil, err
	}
	file, err := root.Open(filepath.FromSlash(path))
	if err != nil {
		return nil, fmt.Errorf("open source %q in pinned Workspace: %w", path, err)
	}
	return readOpenedWorkspaceSource(file, path)
}

func readOpenedWorkspaceSource(file *os.File, path string) ([]byte, error) {
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

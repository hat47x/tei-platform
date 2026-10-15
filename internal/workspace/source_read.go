package workspace

import (
	"errors"
	"fmt"
	"io"
	"math"

	"tei/internal/sourceindex"
)

var (
	ErrInvalidSourceReadLimit = errors.New("invalid-source-read-limit")
	ErrSourceTooLarge         = errors.New("workspace-source-too-large")
)

// ReadSource reads one regular Workspace-relative file through the existing
// traversal-resistant root boundary. The limit bounds bytes retained in
// memory; an oversized file is rejected instead of truncated.
func ReadSource(root, relativePath string, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return nil, ErrInvalidSourceReadLimit
	}
	path, err := sourceindex.NormalizeWorkspacePath(relativePath)
	if err != nil {
		return nil, fmt.Errorf("normalize Workspace source path: %w", err)
	}
	if path != relativePath {
		return nil, errors.New("Workspace source path is not normalized")
	}
	file, err := openRootedSource(root, path)
	if err != nil {
		return nil, fmt.Errorf("open Workspace source: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("stat Workspace source: %w", err)
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, errors.New("Workspace source is not a regular file")
	}
	if info.Size() > maxBytes {
		closeErr := file.Close()
		if closeErr != nil {
			return nil, fmt.Errorf("close oversized Workspace source: %w", closeErr)
		}
		return nil, ErrSourceTooLarge
	}

	readLimit := maxBytes
	if maxBytes < math.MaxInt64 {
		readLimit++
	}
	content, readErr := io.ReadAll(io.LimitReader(file, readLimit))
	closeErr := file.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read Workspace source: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close Workspace source: %w", closeErr)
	}
	if int64(len(content)) > maxBytes {
		return nil, ErrSourceTooLarge
	}
	return content, nil
}

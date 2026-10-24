package designoutput

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
	"strings"

	"tei/internal/environment"
)

// Verify compares every saved file with the current deterministic derivation.
// It does not use saved metadata as trusted instructions or grant acceptance.
func Verify(workspaceDir, bundleDir string, provider environment.DatabaseProvider) (Manifest, error) {
	return verify(workspaceDir, bundleDir, provider, nil, "", "")
}

// VerifyWithGenerator requires caller-selected generation inputs rather than
// inferring them from the untrusted bundle's contract or candidate manifest.
func VerifyWithGenerator(workspaceDir, bundleDir string, provider environment.DatabaseProvider, contractJSON []byte, strategy, version string) (Manifest, error) {
	if len(contractJSON) == 0 || strings.TrimSpace(strategy) == "" || strings.TrimSpace(version) == "" {
		return Manifest{}, fmt.Errorf("contract JSON, strategy, and generator version are required")
	}
	return verify(workspaceDir, bundleDir, provider, contractJSON, strategy, version)
}

func verify(workspaceDir, bundleDir string, provider environment.DatabaseProvider, contractJSON []byte, strategy, version string) (Manifest, error) {
	if strings.TrimSpace(workspaceDir) == "" || strings.TrimSpace(bundleDir) == "" {
		return Manifest{}, fmt.Errorf("workspace and bundle are required")
	}
	bundle, err := buildFromWorkspace(workspaceDir, provider, contractJSON, strategy, version)
	if err != nil {
		return Manifest{}, err
	}
	files, err := bundleFiles(bundle)
	if err != nil {
		return Manifest{}, err
	}
	expected := make(map[string][]byte, len(files))
	for _, file := range files {
		if !fs.ValidPath(file.Path) || strings.Contains(file.Path, "\\") {
			return Manifest{}, fmt.Errorf("invalid derived path %q", file.Path)
		}
		if _, duplicate := expected[file.Path]; duplicate {
			return Manifest{}, fmt.Errorf("duplicate derived path %q", file.Path)
		}
		expected[file.Path] = file.Content
	}
	root, err := os.OpenRoot(bundleDir)
	if err != nil {
		return Manifest{}, fmt.Errorf("open bundle: %w", err)
	}
	defer root.Close()
	err = fs.WalkDir(root.FS(), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("bundle file %q is not a regular file", path)
		}
		want, ok := expected[path]
		if !ok {
			return fmt.Errorf("unexpected bundle file %q", path)
		}
		if info.Size() != int64(len(want)) {
			return fmt.Errorf("bundle file %q differs from current derivation", path)
		}
		file, err := root.Open(path)
		if err != nil {
			return fmt.Errorf("open bundle file %q: %w", path, err)
		}
		// Read at most the expected content plus one byte, even if it grows.
		actual, readErr := io.ReadAll(io.LimitReader(file, int64(len(want))+1))
		closeErr := file.Close()
		if readErr != nil {
			return fmt.Errorf("read bundle file %q: %w", path, readErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close bundle file %q: %w", path, closeErr)
		}
		if !bytes.Equal(actual, want) {
			return fmt.Errorf("bundle file %q differs from current derivation", path)
		}
		delete(expected, path)
		return nil
	})
	if err != nil {
		return Manifest{}, err
	}
	if len(expected) > 0 {
		missing := make([]string, 0, len(expected))
		for name := range expected {
			missing = append(missing, name)
		}
		sort.Strings(missing)
		return Manifest{}, fmt.Errorf("missing bundle file %q", missing[0])
	}
	return bundle.Manifest, nil
}

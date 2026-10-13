package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"tei/internal/sourceindex"
)

const maxAcceptanceManifestBytes = 2 << 20

// manifestFingerprint binds the Workspace selector's physical bytes to a
// Review and the Source write fence. Model revision intentionally remains
// independent: comments or selector bytes can change without changing IR.
type manifestFingerprint struct {
	Path        string
	Fingerprint string
}

// readAcceptanceManifest observes the Workspace manifest through the same
// pinned os.Root used by mutating Acceptance. The Workspace loader remains
// the sole semantic parser; this function does not interpret its contents.
func readAcceptanceManifest(files *os.Root) (manifestFingerprint, error) {
	for _, name := range []string{"tei.workspace.yaml", "tei.workspace.tei"} {
		info, err := files.Lstat(name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return manifestFingerprint{}, fmt.Errorf("stat Workspace manifest: %w", err)
		}
		if !info.Mode().IsRegular() {
			return manifestFingerprint{}, fmt.Errorf("Workspace manifest %s must be a regular file", name)
		}
		f, err := files.Open(name)
		if err != nil {
			return manifestFingerprint{}, fmt.Errorf("open Workspace manifest: %w", err)
		}
		opened, statErr := f.Stat()
		if statErr != nil || !opened.Mode().IsRegular() {
			_ = f.Close()
			return manifestFingerprint{}, fmt.Errorf("Workspace manifest %s changed file type", name)
		}
		content, readErr := io.ReadAll(io.LimitReader(f, maxAcceptanceManifestBytes+1))
		closeErr := f.Close()
		if readErr != nil || closeErr != nil || len(content) > maxAcceptanceManifestBytes {
			return manifestFingerprint{}, fmt.Errorf("cannot read bounded Workspace manifest %s", name)
		}
		return manifestFingerprint{Path: name, Fingerprint: sourceindex.Fingerprint(content)}, nil
	}
	return manifestFingerprint{}, errors.New("Workspace manifest not found")
}

// checkAcceptanceManifest is a byte-level freshness check, not a filesystem
// transaction or source authorization. External writers are not serialized.
func checkAcceptanceManifest(files *os.Root, expected manifestFingerprint) error {
	current, err := readAcceptanceManifest(files)
	if err != nil {
		return fmt.Errorf("Workspace manifest became unavailable: %w", err)
	}
	if expected.Path == "" || expected.Fingerprint == "" || current != expected {
		return fmt.Errorf("Workspace manifest changed: expected %s %s; current %s %s",
			expected.Path, expected.Fingerprint, current.Path, current.Fingerprint)
	}
	return nil
}

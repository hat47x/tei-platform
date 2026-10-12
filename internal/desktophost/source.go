// Package desktophost contains non-authoritative Desktop Studio host state.
// It never changes Accepted Source, authorizes a mutation, or executes a command.
package desktophost

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"

	"tei/internal/workspace"
)

// ObservedArtifact is a fingerprint of bytes consumed by workspace.Load, not
// a write lock, an authority token, or evidence of a race-free file read.
type ObservedArtifact struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

// SourceObservation is derived from the existing Workspace loader and Source
// Index. WorkspaceRoot is sensitive local metadata and must not be sent to an
// external AI or telemetry destination without a separate disclosure decision.
type SourceObservation struct {
	WorkspaceRoot string             `json:"workspaceRoot"`
	Artifacts     []ObservedArtifact `json:"artifacts"`
}

var ErrSourceChanged = errors.New("desktop source observation changed")

// ObserveSource reuses the authoritative loader's byte snapshot and Source
// Index. It does not read raw source files through a second desktop reader.
// Filesystem containment at Load is currently best-effort (issue #1017).
func ObserveSource(root string) (SourceObservation, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return SourceObservation{}, fmt.Errorf("load Workspace: %w", err)
	}
	index, _, err := workspace.IndexedLoadedSnapshot(root, loaded)
	if err != nil {
		return SourceObservation{}, fmt.Errorf("index Load snapshot: %w", err)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return SourceObservation{}, fmt.Errorf("resolve Workspace root: %w", err)
	}
	physicalRoot, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return SourceObservation{}, fmt.Errorf("resolve Workspace root: %w", err)
	}
	result := SourceObservation{WorkspaceRoot: physicalRoot, Artifacts: make([]ObservedArtifact, 0, len(index.Artifacts))}
	for _, artifact := range index.Artifacts {
		result.Artifacts = append(result.Artifacts, ObservedArtifact{Path: artifact.Path, Fingerprint: artifact.Fingerprint})
	}
	sort.Slice(result.Artifacts, func(i, j int) bool { return result.Artifacts[i].Path < result.Artifacts[j].Path })
	return result, nil
}

// CompareObservations reports whether two read-only observations differ.
// Equality is NOT permission to write: shared Acceptance still must check the
// actual Source, revision, and authority within its own mutation fence.
func CompareObservations(before, after SourceObservation) error {
	if before.WorkspaceRoot == "" || after.WorkspaceRoot == "" || before.WorkspaceRoot != after.WorkspaceRoot {
		return ErrSourceChanged
	}
	if len(before.Artifacts) != len(after.Artifacts) {
		return ErrSourceChanged
	}
	for i := range before.Artifacts {
		a, b := before.Artifacts[i], after.Artifacts[i]
		if a.Path == "" || a.Fingerprint == "" || a != b {
			return ErrSourceChanged
		}
		if i > 0 && (before.Artifacts[i-1].Path >= a.Path || after.Artifacts[i-1].Path >= b.Path) {
			return ErrSourceChanged
		}
	}
	return nil
}

// CheckSourceFresh rereads the Workspace; Watcher notifications are not
// consulted. This is a preflight observation, not a safe accept transaction.
func CheckSourceFresh(before SourceObservation) (SourceObservation, error) {
	if before.WorkspaceRoot == "" {
		return SourceObservation{}, ErrSourceChanged
	}
	current, err := ObserveSource(before.WorkspaceRoot)
	if err != nil {
		return SourceObservation{}, err
	}
	return current, CompareObservations(before, current)
}

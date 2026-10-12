// Package desktophost contains non-authoritative Desktop Studio host state.
// It never changes Accepted Source, authorizes a mutation, or executes a command.
package desktophost

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"tei/internal/sourceindex"
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

// ObserveSource uses BuildSourceIndex, which uses retained parser bytes for
// parsed Source and its existing checked reader for unparsed managed files.
// IndexedLoadedSnapshot intentionally rejects such unparsed assets; it cannot
// be used as a general inventory. The Workspace manifest is also tracked as
// a Host observation. Reads are not a transaction or write permission.
// Filesystem containment at Load is currently best-effort (issue #1017).
func ObserveSource(root string) (SourceObservation, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return SourceObservation{}, fmt.Errorf("resolve Workspace root: %w", err)
	}
	physicalRoot, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return SourceObservation{}, fmt.Errorf("resolve Workspace root: %w", err)
	}
	manifestBefore, err := observeWorkspaceManifest(physicalRoot)
	if err != nil {
		return SourceObservation{}, err
	}
	loaded, err := workspace.Load(physicalRoot)
	if err != nil {
		return SourceObservation{}, fmt.Errorf("load Workspace: %w", err)
	}
	index, err := workspace.BuildSourceIndex(physicalRoot, loaded)
	if err != nil {
		return SourceObservation{}, fmt.Errorf("index Workspace Source: %w", err)
	}
	manifestAfter, err := observeWorkspaceManifest(physicalRoot)
	if err != nil {
		return SourceObservation{}, err
	}
	if manifestBefore != manifestAfter {
		return SourceObservation{}, ErrSourceChanged
	}
	result := SourceObservation{WorkspaceRoot: physicalRoot, Artifacts: make([]ObservedArtifact, 0, len(index.Artifacts)+1)}
	result.Artifacts = append(result.Artifacts, manifestAfter)
	for _, artifact := range index.Artifacts {
		result.Artifacts = append(result.Artifacts, ObservedArtifact{Path: artifact.Path, Fingerprint: artifact.Fingerprint})
	}
	sort.Slice(result.Artifacts, func(i, j int) bool { return result.Artifacts[i].Path < result.Artifacts[j].Path })
	if err := CompareObservations(result, result); err != nil {
		return SourceObservation{}, err
	}
	return result, nil
}

// observeWorkspaceManifest tracks the Workspace selector used by Load. Using
// os.Root confines this additional host read to the opened Workspace root.
// The loader's own manifest read is not yet an atomic snapshot with this read.
func observeWorkspaceManifest(root string) (ObservedArtifact, error) {
	files, err := os.OpenRoot(root)
	if err != nil {
		return ObservedArtifact{}, fmt.Errorf("open Workspace root: %w", err)
	}
	defer files.Close()
	for _, name := range []string{"tei.workspace.yaml", "tei.workspace.tei"} {
		file, err := files.Open(name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return ObservedArtifact{}, fmt.Errorf("open Workspace manifest: %w", err)
		}
		info, statErr := file.Stat()
		if statErr != nil || !info.Mode().IsRegular() {
			file.Close()
			return ObservedArtifact{}, fmt.Errorf("Workspace manifest is not a regular file")
		}
		// A bounded Host observation, not a new Workspace parser.
		content, readErr := io.ReadAll(io.LimitReader(file, 2<<20+1))
		closeErr := file.Close()
		if readErr != nil || closeErr != nil || len(content) > 2<<20 {
			return ObservedArtifact{}, fmt.Errorf("cannot read bounded Workspace manifest")
		}
		return ObservedArtifact{Path: name, Fingerprint: sourceindex.Fingerprint(content)}, nil
	}
	return ObservedArtifact{}, fmt.Errorf("Workspace manifest not found")
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

package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tei/internal/sourceindex"
)

// BuildSourceIndex projects the workspace loader's current source metadata into
// the M0.23 source-side index. It deliberately keeps physical source metadata
// outside ir.World. Parser-level exact ranges will replace the current point
// anchors incrementally without changing this ownership boundary.
func BuildSourceIndex(root string, result Result) (sourceindex.Index, error) {
	idx := sourceindex.Index{}
	contents := map[string][]byte{}
	artifacts := map[string]sourceindex.Artifact{}

	read := func(rawPath string) ([]byte, error) {
		p, err := sourceindex.NormalizeWorkspacePath(rawPath)
		if err != nil {
			return nil, err
		}
		if content, ok := contents[p]; ok {
			return content, nil
		}
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil {
			return nil, err
		}
		contents[p] = content
		return content, nil
	}

	ensureArtifact := func(rawPath string) (sourceindex.Artifact, error) {
		p, err := sourceindex.NormalizeWorkspacePath(rawPath)
		if err != nil {
			return sourceindex.Artifact{}, err
		}
		if artifact, ok := artifacts[p]; ok {
			return artifact, nil
		}
		content, err := read(p)
		if err != nil {
			return sourceindex.Artifact{}, fmt.Errorf("read managed source %s: %w", p, err)
		}
		artifact, err := sourceindex.NewArtifact(p, content)
		if err != nil {
			return sourceindex.Artifact{}, err
		}
		artifacts[p] = artifact
		idx.Artifacts = append(idx.Artifacts, artifact)
		return artifact, nil
	}

	for _, managed := range result.ManagedFiles {
		if _, err := ensureArtifact(managed.Path); err != nil {
			return sourceindex.Index{}, err
		}
	}

	for key, location := range result.Sources {
		artifact, err := ensureArtifact(location.File)
		if err != nil {
			return sourceindex.Index{}, fmt.Errorf("source %s: %w", key, err)
		}
		content := contents[artifact.Path]
		line, column := location.Line, location.Column
		if line < 1 {
			line = 1
		}
		if column < 1 {
			column = 1
		}
		rng, err := sourceindex.PointRangeAt(content, line, column)
		if err != nil {
			return sourceindex.Index{}, fmt.Errorf("source %s at %s:%d:%d: %w", key, artifact.Path, line, column, err)
		}
		idx.References = append(idx.References, sourceindex.Reference{
			Subject:      subjectFromSourceKey(key),
			ArtifactPath: artifact.Path,
			Range:        rng,
			Role:         sourceindex.RoleDefinition,
		})
	}

	idx.Normalize()
	if err := idx.Validate(read); err != nil {
		return sourceindex.Index{}, err
	}
	return idx, nil
}

func subjectFromSourceKey(key string) sourceindex.Subject {
	kind, id, ok := strings.Cut(key, ":")
	if !ok || strings.TrimSpace(kind) == "" || strings.TrimSpace(id) == "" {
		return sourceindex.Subject{Kind: "workspace", ID: key}
	}
	return sourceindex.Subject{Kind: kind, ID: id}
}

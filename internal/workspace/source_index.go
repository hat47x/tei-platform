package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/syntax"
)

// BuildSourceIndex projects workspace source metadata into the M0.23
// source-side index. Physical source metadata remains outside ir.World.
// Subjects with an exact syntax anchor use the identifier token range; other
// subjects continue to use the loader's legacy point location until their
// parser anchors are introduced.
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
		subject := subjectFromSourceKey(key)
		rng, exact, err := exactSubjectRange(content, subject)
		if err != nil {
			return sourceindex.Index{}, fmt.Errorf("source %s in %s: %w", key, artifact.Path, err)
		}
		if !exact {
			line, column := location.Line, location.Column
			if line < 1 {
				line = 1
			}
			if column < 1 {
				column = 1
			}
			rng, err = sourceindex.PointRangeAt(content, line, column)
			if err != nil {
				return sourceindex.Index{}, fmt.Errorf("source %s at %s:%d:%d: %w", key, artifact.Path, line, column, err)
			}
		}
		idx.References = append(idx.References, sourceindex.Reference{
			Subject:      subject,
			ArtifactPath: artifact.Path,
			Range:        rng,
			Role:         sourceindex.RoleDefinition,
		})
	}

	for _, contribution := range result.SourceContributions {
		artifact, err := ensureArtifact(contribution.File)
		if err != nil {
			return sourceindex.Index{}, fmt.Errorf("source contribution %s:%s: %w", contribution.Subject.Kind, contribution.Subject.ID, err)
		}
		content := contents[artifact.Path]
		rng, err := sourceindex.RangeFromOffsets(content, 0, len(content))
		if err != nil {
			return sourceindex.Index{}, err
		}
		idx.References = append(idx.References, sourceindex.Reference{
			Subject:      contribution.Subject,
			ArtifactPath: artifact.Path,
			Range:        rng,
			Role:         contribution.Role,
		})
	}

	idx.Normalize()
	if err := idx.Validate(read); err != nil {
		return sourceindex.Index{}, err
	}
	return idx, nil
}

func exactSubjectRange(content []byte, subject sourceindex.Subject) (sourceindex.Range, bool, error) {
	if isSemanticSubjectKind(subject.Kind) {
		anchored, err := syntax.ParseSemanticFileAnchored(string(content))
		if err != nil {
			return sourceindex.Range{}, true, err
		}
		for _, item := range anchored.Anchors {
			if item.Kind == subject.Kind && item.ID == subject.ID {
				rng, err := sourceindex.RangeFromOffsets(content, item.Anchor.Start.Offset, item.Anchor.End.Offset)
				return rng, true, err
			}
		}
		return sourceindex.Range{}, true, fmt.Errorf("semantic subject %s:%s has no syntax anchor", subject.Kind, subject.ID)
	}
	return declarationIdentifierRange(content, subject)
}

func isSemanticSubjectKind(kind string) bool {
	switch kind {
	case "semantic", "semantic-type", "semantic-data", "semantic-attribute", "semantic-relation", "semantic-capability":
		return true
	default:
		return false
	}
}

func declarationIdentifierRange(content []byte, subject sourceindex.Subject) (sourceindex.Range, bool, error) {
	var keyword string
	switch subject.Kind {
	case "application":
		keyword = "application"
	case "database":
		keyword = "database"
	case "database-contract":
		keyword = "contract"
	case "query":
		keyword = "query"
	default:
		return sourceindex.Range{}, false, nil
	}
	anchor, err := syntax.DeclarationIdentifierAnchor(string(content), keyword, subject.ID)
	if err != nil {
		return sourceindex.Range{}, true, err
	}
	rng, err := sourceindex.RangeFromOffsets(content, anchor.Start.Offset, anchor.End.Offset)
	return rng, true, err
}

func subjectFromSourceKey(key string) sourceindex.Subject {
	kind, id, ok := strings.Cut(key, ":")
	if !ok || strings.TrimSpace(kind) == "" || strings.TrimSpace(id) == "" {
		return sourceindex.Subject{Kind: "workspace", ID: key}
	}
	return sourceindex.Subject{Kind: kind, ID: id}
}

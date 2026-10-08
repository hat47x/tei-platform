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
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return sourceindex.Index{}, fmt.Errorf("resolve Workspace root: %w", err)
	}
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
		// A caller may supply a Result without passing through Load. Reject
		// physical escapes here as well as in the managed-source preflight.
		// This check is not atomic with ReadFile; see issue #1017 for the
		// platform-specific read-time race-free boundary.
		physical, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil {
			return nil, fmt.Errorf("resolve source %q: %w", p, err)
		}
		rel, err := filepath.Rel(rootReal, physical)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return nil, fmt.Errorf("source %q resolves outside Workspace", p)
		}
		info, err := os.Stat(physical)
		if err != nil {
			return nil, fmt.Errorf("stat source %q: %w", p, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("source %q is not a regular file", p)
		}
		content, err := os.ReadFile(physical)
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

	// Index declarations are nested inside Persistence Mapping files and do not
	// have their own loader entry. Derive their source references from the
	// canonical Mapping plus its already-known source artifact. The subject is
	// path-independent so file moves only change the Source Index locator.
	for _, mapping := range result.World.Persistence {
		location, ok := result.Sources["persistence:"+string(mapping.ID)]
		if !ok {
			continue
		}
		artifact, err := ensureArtifact(location.File)
		if err != nil {
			return sourceindex.Index{}, fmt.Errorf("persistence %s: %w", mapping.ID, err)
		}
		content := contents[artifact.Path]
		seenIndexNames := make(map[string]struct{}, len(mapping.Indexes))
		for _, index := range mapping.Indexes {
			name := strings.TrimSpace(index.Name)
			if _, exists := seenIndexNames[name]; exists {
				return sourceindex.Index{}, fmt.Errorf("persistence %s declares duplicate index name %q; persistence-index source identity must be unique within a mapping", mapping.ID, index.Name)
			}
			seenIndexNames[name] = struct{}{}
			anchor, err := syntax.PersistenceIndexIdentifierAnchor(string(content), string(mapping.ID), index.Name)
			if err != nil {
				return sourceindex.Index{}, fmt.Errorf("persistence index %s/%s in %s: %w", mapping.ID, index.Name, artifact.Path, err)
			}
			rng, err := sourceindex.RangeFromOffsets(content, anchor.Start.Offset, anchor.End.Offset)
			if err != nil {
				return sourceindex.Index{}, err
			}
			idx.References = append(idx.References, sourceindex.Reference{
				Subject: sourceindex.Subject{
					Kind: "persistence-index",
					ID:   persistenceIndexSubjectID(string(mapping.ID), index.Name),
				},
				ArtifactPath: artifact.Path,
				Range:        rng,
				Role:         sourceindex.RoleDefinition,
			})
		}
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
	if subject.Kind == "binding" {
		_, capabilityID, ok := strings.Cut(subject.ID, "/")
		if !ok || strings.TrimSpace(capabilityID) == "" {
			return sourceindex.Range{}, true, fmt.Errorf("binding subject %q must contain application/capability identity", subject.ID)
		}
		anchor, err := syntax.DeclarationIdentifierAnchor(string(content), "realize", capabilityID)
		if err != nil {
			return sourceindex.Range{}, true, err
		}
		rng, err := sourceindex.RangeFromOffsets(content, anchor.Start.Offset, anchor.End.Offset)
		return rng, true, err
	}
	return declarationIdentifierRange(content, subject)
}

func isSemanticSubjectKind(kind string) bool {
	switch kind {
	case "semantic", "semantic-type", "semantic-data", "semantic-attribute", "semantic-relation", "semantic-capability", "semantic-validation":
		return true
	default:
		return false
	}
}

func declarationIdentifierRange(content []byte, subject sourceindex.Subject) (sourceindex.Range, bool, error) {
	if subject.Kind == "database-contract-query" {
		_, queryName, ok := strings.Cut(subject.ID, "/")
		if !ok || strings.TrimSpace(queryName) == "" {
			return sourceindex.Range{}, true, fmt.Errorf("database-contract-query subject %q must contain contract/query identity", subject.ID)
		}
		anchor, err := syntax.ContractQueryDeclarationAnchor(string(content), queryName)
		if err != nil {
			return sourceindex.Range{}, true, err
		}
		rng, err := sourceindex.RangeFromOffsets(content, anchor.Start.Offset, anchor.End.Offset)
		return rng, true, err
	}

	var keyword string
	switch subject.Kind {
	case "application":
		keyword = "application"
	case "application-pattern":
		keyword = "pattern"
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

func persistenceIndexSubjectID(mappingID, indexName string) string {
	return mappingID + "/" + indexName
}

func subjectFromSourceKey(key string) sourceindex.Subject {
	kind, id, ok := strings.Cut(key, ":")
	if !ok || strings.TrimSpace(kind) == "" || strings.TrimSpace(id) == "" {
		return sourceindex.Subject{Kind: "workspace", ID: key}
	}
	return sourceindex.Subject{Kind: kind, ID: id}
}

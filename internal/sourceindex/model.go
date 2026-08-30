package sourceindex

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"

	"tei/internal/subject"
)

type Role string

const (
	RoleDefinition     Role = "definition"
	RoleAuthoring      Role = "authoring"
	RoleImplementation Role = "implementation"
	RoleDerived        Role = "derived"
)

type Artifact struct {
	Path        string `json:"path"`
	Format      string `json:"format,omitempty"`
	Fingerprint string `json:"fingerprint"`
}

type Position struct {
	Offset int `json:"offset"`
	Line   int `json:"line"`
	Column int `json:"column"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Subject remains as a compatibility alias for existing Source Index callers.
// The semantic identity itself is owned by the neutral subject package, not by
// Source Index.
type Subject = subject.Subject

type Reference struct {
	Subject      Subject `json:"subject"`
	ArtifactPath string  `json:"artifactPath"`
	Range        Range   `json:"range"`
	Role         Role    `json:"role"`
}

type Index struct {
	Artifacts  []Artifact  `json:"artifacts"`
	References []Reference `json:"references"`
}

func NewArtifact(rawPath string, content []byte) (Artifact, error) {
	p, err := NormalizeWorkspacePath(rawPath)
	if err != nil {
		return Artifact{}, err
	}
	format := strings.TrimPrefix(strings.ToLower(path.Ext(p)), ".")
	return Artifact{Path: p, Format: format, Fingerprint: Fingerprint(content)}, nil
}

func NormalizeWorkspacePath(raw string) (string, error) {
	p := strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/")
	if p == "" {
		return "", fmt.Errorf("source path is empty")
	}
	if strings.HasPrefix(p, "/") {
		return "", fmt.Errorf("source path %q must be workspace-relative", raw)
	}
	clean := path.Clean(p)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("source path %q escapes the workspace", raw)
	}
	first := strings.SplitN(clean, "/", 2)[0]
	if strings.Contains(first, ":") {
		return "", fmt.Errorf("source path %q must not be an absolute drive path", raw)
	}
	return clean, nil
}

func Fingerprint(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func PositionAt(content []byte, offset int) (Position, error) {
	if offset < 0 || offset > len(content) {
		return Position{}, fmt.Errorf("source offset %d is outside [0,%d]", offset, len(content))
	}
	line, column := 1, 1
	for i := 0; i < offset; i++ {
		if content[i] == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	return Position{Offset: offset, Line: line, Column: column}, nil
}

func OffsetAt(content []byte, line, column int) (int, error) {
	if line < 1 || column < 1 {
		return 0, fmt.Errorf("source position must be 1-based, got %d:%d", line, column)
	}
	currentLine, currentColumn := 1, 1
	for offset := 0; offset <= len(content); offset++ {
		if currentLine == line && currentColumn == column {
			return offset, nil
		}
		if offset == len(content) {
			break
		}
		if content[offset] == '\n' {
			currentLine++
			currentColumn = 1
		} else {
			currentColumn++
		}
	}
	return 0, fmt.Errorf("source position %d:%d is outside content", line, column)
}

func RangeFromOffsets(content []byte, start, end int) (Range, error) {
	if start < 0 || end < start || end > len(content) {
		return Range{}, fmt.Errorf("invalid source range [%d,%d) for content length %d", start, end, len(content))
	}
	startPos, err := PositionAt(content, start)
	if err != nil {
		return Range{}, err
	}
	endPos, err := PositionAt(content, end)
	if err != nil {
		return Range{}, err
	}
	return Range{Start: startPos, End: endPos}, nil
}

func PointRangeAt(content []byte, line, column int) (Range, error) {
	offset, err := OffsetAt(content, line, column)
	if err != nil {
		return Range{}, err
	}
	return RangeFromOffsets(content, offset, offset)
}

func (r Range) Validate(content []byte) error {
	if r.Start.Offset < 0 || r.End.Offset < r.Start.Offset || r.End.Offset > len(content) {
		return fmt.Errorf("source range [%d,%d) is outside content length %d", r.Start.Offset, r.End.Offset, len(content))
	}
	start, _ := PositionAt(content, r.Start.Offset)
	end, _ := PositionAt(content, r.End.Offset)
	if r.Start != start {
		return fmt.Errorf("source range start metadata is stale: have %+v want %+v", r.Start, start)
	}
	if r.End != end {
		return fmt.Errorf("source range end metadata is stale: have %+v want %+v", r.End, end)
	}
	return nil
}

func (i *Index) Normalize() {
	sort.Slice(i.Artifacts, func(a, b int) bool {
		return i.Artifacts[a].Path < i.Artifacts[b].Path
	})
	sort.Slice(i.References, func(a, b int) bool {
		left, right := i.References[a], i.References[b]
		if left.Subject.Kind != right.Subject.Kind {
			return left.Subject.Kind < right.Subject.Kind
		}
		if left.Subject.ID != right.Subject.ID {
			return left.Subject.ID < right.Subject.ID
		}
		if left.Role != right.Role {
			return left.Role < right.Role
		}
		if left.ArtifactPath != right.ArtifactPath {
			return left.ArtifactPath < right.ArtifactPath
		}
		return left.Range.Start.Offset < right.Range.Start.Offset
	})
}

func (i Index) ReferencesFor(subject Subject) []Reference {
	out := make([]Reference, 0)
	for _, ref := range i.References {
		if ref.Subject == subject {
			out = append(out, ref)
		}
	}
	return out
}

func (i Index) Validate(read func(string) ([]byte, error)) error {
	artifacts := make(map[string]Artifact, len(i.Artifacts))
	for _, artifact := range i.Artifacts {
		p, err := NormalizeWorkspacePath(artifact.Path)
		if err != nil {
			return err
		}
		if p != artifact.Path {
			return fmt.Errorf("source artifact path %q is not normalized", artifact.Path)
		}
		if _, exists := artifacts[p]; exists {
			return fmt.Errorf("source artifact %s is indexed more than once", p)
		}
		content, err := read(p)
		if err != nil {
			return fmt.Errorf("read source artifact %s: %w", p, err)
		}
		if got := Fingerprint(content); got != artifact.Fingerprint {
			return fmt.Errorf("source artifact %s fingerprint is stale: have %s want %s", p, artifact.Fingerprint, got)
		}
		artifacts[p] = artifact
	}
	for _, ref := range i.References {
		artifact, ok := artifacts[ref.ArtifactPath]
		if !ok {
			return fmt.Errorf("source reference %s:%s points to unindexed artifact %s", ref.Subject.Kind, ref.Subject.ID, ref.ArtifactPath)
		}
		content, err := read(artifact.Path)
		if err != nil {
			return fmt.Errorf("read source artifact %s: %w", artifact.Path, err)
		}
		if err := ref.Range.Validate(content); err != nil {
			return fmt.Errorf("source reference %s:%s: %w", ref.Subject.Kind, ref.Subject.ID, err)
		}
	}
	return nil
}

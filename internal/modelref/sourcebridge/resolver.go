package sourcebridge

import (
	"errors"
	"fmt"
	"sort"

	"tei/internal/modelref"
	"tei/internal/sourceindex"
	"tei/internal/subject"
)

// Resolver indexes a validated, immutable Source Index snapshot for many
// navigation operations. The caller must pin graph, index and reader to the
// same Workspace revision; validating an index cannot lock a live filesystem.
type Resolver struct {
	nodes       map[modelref.Key]int
	collisions  map[subject.Subject]int
	references  map[subject.Subject][]sourceindex.Reference
	artifacts   map[string]sourceindex.Artifact
}

// NewResolver validates artifact digests and exact ranges once rather than
// repeating O(index-size) filesystem work for each diagram/table cell.
func NewResolver(graph modelref.Graph, idx sourceindex.Index, read func(string) ([]byte, error)) (*Resolver, error) {
	if read == nil {
		return nil, errors.New("source reader unavailable")
	}
	if graph.Schema != "tei.modelref.v0" {
		return nil, fmt.Errorf("unsupported source navigation graph schema %q", graph.Schema)
	}
	if err := idx.Validate(read); err != nil {
		return nil, err
	}
	r := &Resolver{
		nodes: make(map[modelref.Key]int, len(graph.Nodes)),
		collisions: make(map[subject.Subject]int, len(graph.Nodes)),
		references: make(map[subject.Subject][]sourceindex.Reference),
		artifacts: make(map[string]sourceindex.Artifact, len(idx.Artifacts)),
	}
	for _, n := range graph.Nodes {
		r.nodes[n.Key]++
		r.collisions[n.Key.Subject]++
	}
	for _, a := range idx.Artifacts {
		r.artifacts[a.Path] = a
	}
	for _, ref := range idx.References {
		r.references[ref.Subject] = append(r.references[ref.Subject], ref)
	}
	for key := range r.references {
		refs := r.references[key]
		sort.Slice(refs, func(i, j int) bool {
			a, b := refs[i], refs[j]
			if a.ArtifactPath != b.ArtifactPath {
				return a.ArtifactPath < b.ArtifactPath
			}
			if a.Role != b.Role {
				return a.Role < b.Role
			}
			if a.Range.Start.Offset != b.Range.Start.Offset {
				return a.Range.Start.Offset < b.Range.Start.Offset
			}
			return a.Range.End.Offset < b.Range.End.Offset
		})
	}
	return r, nil
}

// Locate only returns a Source location, not editability, Acceptance,
// confirmation, or runtime authority. Read-only and derived references can
// never become a source definition merely by successful resolution.
func (r *Resolver) Locate(key modelref.Key) Navigation {
	out := Navigation{Key: key, Status: "unresolved", AuthorityLimits: []string{
		"source-location-is-not-semantic-acceptance",
		"source-location-is-not-write-authority",
	}}
	if r == nil {
		out.Reason = "source-resolver-unavailable"
		return out
	}
	if r.nodes[key] != 1 {
		out.Reason = "subject-not-unique-in-graph"
		return out
	}
	if r.collisions[key.Subject] != 1 {
		out.Reason = "subject-scope-ambiguous"
		return out
	}
	refs := r.references[key.Subject]
	if len(refs) == 0 {
		out.Reason = "source-anchor-unavailable"
		return out
	}
	// Keep all cross-references for read-only inspection; do not choose a
	// definition arbitrarily where duplicate editable anchors conflict.
	definitionCount, authoringCount := 0, 0
	for _, ref := range refs {
		switch ref.Role {
		case sourceindex.RoleDefinition:
			definitionCount++
		case sourceindex.RoleAuthoring:
			authoringCount++
		}
		a, ok := r.artifacts[ref.ArtifactPath]
		if !ok {
			out.Reason = "source-artifact-unindexed"
			return out
		}
		out.Artifacts = append(out.Artifacts, a)
	}
	out.References = append(out.References, refs...)
	if definitionCount > 1 || authoringCount > 1 {
		out.Reason = "ambiguous-source-anchor"
		return out
	}
	if definitionCount+authoringCount == 0 {
		out.Reason = "derived-only-source"
		return out
	}
	// An indexed definition is an origin hint; actual ability to edit is
	// controlled by the existing Shared Acceptance path.
	seen := make(map[string]bool, len(out.Artifacts))
	var unique []sourceindex.Artifact
	for _, artifact := range out.Artifacts {
		if !seen[artifact.Path] {
			unique = append(unique, artifact)
			seen[artifact.Path] = true
		}
	}
	out.Artifacts = unique
	out.Status = "located"
	return out
}

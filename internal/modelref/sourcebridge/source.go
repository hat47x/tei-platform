// Package sourcebridge resolves typed logical subjects into exact, read-only
// Source Index locations. It does not create an authoring or Acceptance path.
package sourcebridge

import (
	"fmt"
	"sort"

	"tei/internal/modelref"
	"tei/internal/sourceindex"
)

// Navigation is a derived Source Index projection. A located anchor is not
// proof of Accepted meaning, write authority, or source parser correctness.
type Navigation struct {
	Key             modelref.Key            `json:"key"`
	Status          string                  `json:"status"` // located, unresolved
	Reason          string                  `json:"reason,omitempty"`
	References      []sourceindex.Reference `json:"references,omitempty"`
	Artifacts       []sourceindex.Artifact  `json:"artifacts,omitempty"`
	AuthorityLimits []string                `json:"authorityLimits"`
}

// Locate validates the source snapshot before returning any navigation hints.
// The Source Index does not yet carry a scope field; ambiguous same-kind/ID
// subjects across graph scopes must not silently share an anchor.
func Locate(g modelref.Graph, idx sourceindex.Index, target modelref.Key, read func(string) ([]byte, error)) Navigation {
	out := Navigation{Key: target, Status: "unresolved", AuthorityLimits: []string{
		"source-location-is-not-semantic-acceptance",
		"source-location-is-not-write-authority",
	}}
	if read == nil {
		out.Reason = "source-reader-unavailable"
		return out
	}
	if err := idx.Validate(read); err != nil {
		out.Reason = "source-index-invalid"
		return out
	}
	count, collisions := 0, 0
	for _, n := range g.Nodes {
		if n.Key == target {
			count++
		}
		if n.Key.Subject == target.Subject {
			collisions++
		}
	}
	if count != 1 {
		out.Reason = "subject-not-unique-in-graph"
		return out
	}
	if collisions != 1 {
		out.Reason = "subject-scope-ambiguous"
		return out
	}
	out.References = idx.ReferencesFor(target.Subject)
	if len(out.References) == 0 {
		out.Reason = "source-anchor-unavailable"
		return out
	}
	hasSource := false
	artifactByPath := make(map[string]sourceindex.Artifact)
	for _, a := range idx.Artifacts {
		artifactByPath[a.Path] = a
	}
	artifacts := make(map[string]sourceindex.Artifact)
	for _, ref := range out.References {
		if ref.Role == sourceindex.RoleDefinition || ref.Role == sourceindex.RoleAuthoring {
			hasSource = true
		}
		a, ok := artifactByPath[ref.ArtifactPath]
		if !ok {
			out.Reason = "source-artifact-unindexed"
			return out
		}
		artifacts[a.Path] = a
	}
	for _, a := range artifacts {
		out.Artifacts = append(out.Artifacts, a)
	}
	sort.Slice(out.Artifacts, func(i, j int) bool { return out.Artifacts[i].Path < out.Artifacts[j].Path })
	sort.Slice(out.References, func(i, j int) bool {
		a, b := out.References[i], out.References[j]
		ak := fmt.Sprintf("%s/%s/%012d/%012d", a.ArtifactPath, a.Role, a.Range.Start.Offset, a.Range.End.Offset)
		bk := fmt.Sprintf("%s/%s/%012d/%012d", b.ArtifactPath, b.Role, b.Range.Start.Offset, b.Range.End.Offset)
		return ak < bk
	})
	if !hasSource {
		out.Reason = "derived-only-source"
		return out
	}
	out.Status = "located"
	return out
}

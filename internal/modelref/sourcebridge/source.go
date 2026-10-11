// Package sourcebridge resolves typed logical subjects into exact, read-only
// Source Index locations. It does not create an authoring or Acceptance path.
package sourcebridge

import (
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
	resolver, err := NewResolver(g, idx, read)
	if err != nil {
		reason := "source-index-invalid"
		if read == nil {
			reason = "source-reader-unavailable"
		}
		return Navigation{
			Key: target, Status: "unresolved", Reason: reason,
			AuthorityLimits: []string{"source-location-is-not-semantic-acceptance", "source-location-is-not-write-authority"},
		}
	}
	return resolver.Locate(target)
}

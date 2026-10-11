package modelref

import (
	"encoding/json"
	"fmt"
	"sort"
)

// traceGraphSnapshotBytes encodes equivalent typed graphs identically
// regardless of iteration order. Duplicate declarations and edges are retained
// because removing them would hide ambiguity or change candidate cardinality.
// Source evidence is treated as an unordered list of observations.
func traceGraphSnapshotBytes(graph Graph) ([]byte, error) {
	nodes := make([]string, 0, len(graph.Nodes))
	for _, original := range graph.Nodes {
		node := original
		node.Evidence = sortedTraceEvidence(original.Evidence)
		raw, err := json.Marshal(node)
		if err != nil {
			return nil, fmt.Errorf("serialize modelref node: %w", err)
		}
		nodes = append(nodes, string(raw))
	}
	edges := make([]string, 0, len(graph.Edges))
	for _, original := range graph.Edges {
		edge := original
		edge.Evidence = sortedTraceEvidence(original.Evidence)
		raw, err := json.Marshal(edge)
		if err != nil {
			return nil, fmt.Errorf("serialize modelref edge: %w", err)
		}
		edges = append(edges, string(raw))
	}
	sort.Strings(nodes)
	sort.Strings(edges)
	return json.Marshal(struct {
		Schema string   `json:"schema"`
		Nodes  []string `json:"nodes"`
		Edges  []string `json:"edges"`
	}{Schema: graph.Schema, Nodes: nodes, Edges: edges})
}

func sortedTraceEvidence(values []Evidence) []Evidence {
	copied := append([]Evidence(nil), values...)
	sort.Slice(copied, func(i, j int) bool {
		a, b := copied[i], copied[j]
		if a.Authority != b.Authority {
			return a.Authority < b.Authority
		}
		if a.Context != b.Context {
			return a.Context < b.Context
		}
		if a.SourceToken != b.SourceToken {
			return a.SourceToken < b.SourceToken
		}
		if a.ArtifactFingerprint != b.ArtifactFingerprint {
			return a.ArtifactFingerprint < b.ArtifactFingerprint
		}
		return a.RuleFingerprint < b.RuleFingerprint
	})
	return copied
}

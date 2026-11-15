package teimdreader

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
)

// CandidateShapeFingerprint identifies only the Graph/Rule structure consumed
// by this reference generator. It does not identify Source bytes, accepted
// TEI Meaning, a selected Provider or a Build/Execution attempt.
//
// Node labels, Markdown layout, Mermaid projections and source positions do
// not participate: they do not affect the currently generated Go behavior.
func CandidateShapeFingerprint(c RuleCandidate) (string, error) {
	if c.Rule.ID != "node.edge_ids" || (c.Rule.Endpoint != "from" && c.Rule.Endpoint != "to") {
		return "", fmt.Errorf("unsupported reference rule")
	}
	type payload struct {
		Nodes []string    `json:"nodes"`
		Edges [][3]string `json:"edges"`
		Rule  [2]string   `json:"rule"`
	}
	var normalized payload
	normalized.Nodes = make([]string, 0, len(c.Graph.Nodes))
	normalized.Edges = make([][3]string, 0, len(c.Graph.Edges))
	normalized.Rule = [2]string{c.Rule.ID, c.Rule.Endpoint}
	nodes := map[string]bool{}
	edges := map[string]bool{}
	for _, n := range c.Graph.Nodes {
		if n.ID == "" || nodes[n.ID] {
			return "", fmt.Errorf("invalid node identity")
		}
		nodes[n.ID] = true
		normalized.Nodes = append(normalized.Nodes, n.ID)
	}
	if len(nodes) == 0 {
		return "", fmt.Errorf("no nodes")
	}
	for _, e := range c.Graph.Edges {
		if e.ID == "" || edges[e.ID] || !nodes[e.From] || !nodes[e.To] {
			return "", fmt.Errorf("invalid edge identity or endpoint")
		}
		edges[e.ID] = true
		normalized.Edges = append(normalized.Edges, [3]string{e.ID, e.From, e.To})
	}
	sort.Strings(normalized.Nodes)
	sort.Slice(normalized.Edges, func(i, j int) bool {
		return normalized.Edges[i][0] < normalized.Edges[j][0]
	})
	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("canonical reference candidate: %w", err)
	}
	const domain = "tei-md-graph-reference/shape-v1\n"
	digest := sha256.Sum256(append([]byte(domain), raw...))
	return fmt.Sprintf("%x", digest), nil
}

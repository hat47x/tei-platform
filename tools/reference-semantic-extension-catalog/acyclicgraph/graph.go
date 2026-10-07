package acyclicgraph

import (
	"fmt"
	"sort"
	"strings"
)

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Snapshot struct {
	Nodes []string `json:"nodes"`
	Edges []Edge   `json:"edges,omitempty"`
}

type Diagnostic struct {
	Code    string `json:"code"`
	Node    string `json:"node,omitempty"`
	Message string `json:"message"`
}

func Normalize(snapshot Snapshot) (Snapshot, error) {
	nodes := make([]string, 0, len(snapshot.Nodes))
	nodeSet := make(map[string]struct{}, len(snapshot.Nodes))
	for i, raw := range snapshot.Nodes {
		node := strings.TrimSpace(raw)
		if node == "" {
			return Snapshot{}, fmt.Errorf("node[%d] is empty", i)
		}
		if _, exists := nodeSet[node]; exists {
			return Snapshot{}, fmt.Errorf("node %q is declared more than once", node)
		}
		nodeSet[node] = struct{}{}
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)

	edges := make([]Edge, 0, len(snapshot.Edges))
	edgeSet := map[string]struct{}{}
	for i, raw := range snapshot.Edges {
		edge := Edge{From: strings.TrimSpace(raw.From), To: strings.TrimSpace(raw.To)}
		if edge.From == "" || edge.To == "" {
			return Snapshot{}, fmt.Errorf("edge[%d] requires from and to", i)
		}
		if _, ok := nodeSet[edge.From]; !ok {
			return Snapshot{}, fmt.Errorf("edge[%d] source %q is not in nodes", i, edge.From)
		}
		if _, ok := nodeSet[edge.To]; !ok {
			return Snapshot{}, fmt.Errorf("edge[%d] target %q is not in nodes", i, edge.To)
		}
		key := edge.From + "\x00" + edge.To
		if _, duplicate := edgeSet[key]; duplicate {
			continue
		}
		edgeSet[key] = struct{}{}
		edges = append(edges, edge)
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		return edges[i].To < edges[j].To
	})

	return Snapshot{Nodes: nodes, Edges: edges}, nil
}

func ValidateAcyclic(snapshot Snapshot) ([]Diagnostic, error) {
	normalized, err := Normalize(snapshot)
	if err != nil {
		return nil, err
	}

	adjacency := make(map[string][]string, len(normalized.Nodes))
	for _, node := range normalized.Nodes {
		adjacency[node] = nil
	}
	for _, edge := range normalized.Edges {
		adjacency[edge.From] = append(adjacency[edge.From], edge.To)
	}
	for node := range adjacency {
		sort.Strings(adjacency[node])
	}

	const (
		visiting = iota + 1
		visited
	)
	state := map[string]int{}
	var visit func(string) (string, bool)
	visit = func(node string) (string, bool) {
		switch state[node] {
		case visiting:
			return node, true
		case visited:
			return "", false
		}
		state[node] = visiting
		for _, next := range adjacency[node] {
			if cycleNode, cycle := visit(next); cycle {
				return cycleNode, true
			}
		}
		state[node] = visited
		return "", false
	}

	for _, node := range normalized.Nodes {
		if cycleNode, cycle := visit(node); cycle {
			return []Diagnostic{{
				Code:    "DIRECTED_GRAPH_CYCLE",
				Node:    cycleNode,
				Message: "directed relation graph contains a cycle",
			}}, nil
		}
	}
	return nil, nil
}

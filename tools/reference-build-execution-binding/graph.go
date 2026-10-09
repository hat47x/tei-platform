package main

import (
	"fmt"
	"sort"
	"strings"
)

const buildGraphFormatVersion = 1

type BuildGraphNode struct {
	ID                       string `json:"id"`
	ExecutionPlanFingerprint string `json:"executionPlanFingerprint"`
}

type BuildGraphEdge struct {
	FromNode     string `json:"fromNode"`
	OutputRole   string `json:"outputRole"`
	ToNode       string `json:"toNode"`
	DependencyID string `json:"dependencyId"`
}

type BuildGraphSeed struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	Version        string `json:"version"`
	ArtifactDigest string `json:"artifactDigest"`
	TrustBoundary  string `json:"trustBoundary"`
	ToNode         string `json:"toNode"`
	DependencyID   string `json:"dependencyId"`
}

type BuildGraph struct {
	FormatVersion int              `json:"formatVersion"`
	Nodes         []BuildGraphNode `json:"nodes"`
	Edges         []BuildGraphEdge `json:"edges,omitempty"`
	Seeds         []BuildGraphSeed `json:"seeds,omitempty"`
}

func (g BuildGraph) Validate() error {
	if g.FormatVersion != buildGraphFormatVersion {
		return fmt.Errorf("unsupported build graph format %d", g.FormatVersion)
	}
	if len(g.Nodes) == 0 {
		return fmt.Errorf("build graph requires at least one node")
	}
	nodes := make(map[string]BuildGraphNode, len(g.Nodes))
	for _, node := range g.Nodes {
		if strings.TrimSpace(node.ID) == "" {
			return fmt.Errorf("build graph node id is required")
		}
		if _, duplicate := nodes[node.ID]; duplicate {
			return fmt.Errorf("build graph node %s is declared more than once", node.ID)
		}
		if err := validateFingerprint("build graph node "+node.ID+" execution plan fingerprint", node.ExecutionPlanFingerprint); err != nil {
			return err
		}
		nodes[node.ID] = node
	}

	edges := map[string]struct{}{}
	for _, edge := range g.Edges {
		if strings.TrimSpace(edge.FromNode) == "" ||
			strings.TrimSpace(edge.ToNode) == "" ||
			strings.TrimSpace(edge.OutputRole) == "" ||
			strings.TrimSpace(edge.DependencyID) == "" {
			return fmt.Errorf("build graph edge requires fromNode, outputRole, toNode, and dependencyId")
		}
		if _, ok := nodes[edge.FromNode]; !ok {
			return fmt.Errorf("build graph edge references unknown producer node %s", edge.FromNode)
		}
		if _, ok := nodes[edge.ToNode]; !ok {
			return fmt.Errorf("build graph edge references unknown consumer node %s", edge.ToNode)
		}
		key := strings.Join([]string{edge.FromNode, edge.OutputRole, edge.ToNode, edge.DependencyID}, "\x00")
		if _, duplicate := edges[key]; duplicate {
			return fmt.Errorf("build graph edge %s -> %s dependency %s is declared more than once", edge.FromNode, edge.ToNode, edge.DependencyID)
		}
		edges[key] = struct{}{}
	}

	seeds := map[string]struct{}{}
	for _, seed := range g.Seeds {
		if strings.TrimSpace(seed.ID) == "" ||
			strings.TrimSpace(seed.Kind) == "" ||
			strings.TrimSpace(seed.Version) == "" ||
			strings.TrimSpace(seed.TrustBoundary) == "" ||
			strings.TrimSpace(seed.ToNode) == "" ||
			strings.TrimSpace(seed.DependencyID) == "" {
			return fmt.Errorf("build graph seed identity, trust boundary, consumer, and dependency id are required")
		}
		if err := validateDigest("build graph seed "+seed.ID+" artifact digest", seed.ArtifactDigest); err != nil {
			return err
		}
		if _, ok := nodes[seed.ToNode]; !ok {
			return fmt.Errorf("build graph seed %s references unknown consumer node %s", seed.ID, seed.ToNode)
		}
		key := strings.Join([]string{seed.ID, seed.Kind, seed.Version, seed.ToNode, seed.DependencyID}, "\x00")
		if _, duplicate := seeds[key]; duplicate {
			return fmt.Errorf("build graph seed %s is declared more than once for node %s", seed.ID, seed.ToNode)
		}
		seeds[key] = struct{}{}
	}

	if _, err := topologicalBuildOrderValidated(g); err != nil {
		return err
	}
	return nil
}

func normalizeBuildGraph(g BuildGraph) BuildGraph {
	out := g
	out.Nodes = append([]BuildGraphNode(nil), g.Nodes...)
	out.Edges = append([]BuildGraphEdge(nil), g.Edges...)
	out.Seeds = append([]BuildGraphSeed(nil), g.Seeds...)

	for i := range out.Nodes {
		out.Nodes[i].ExecutionPlanFingerprint = normalizeFingerprint(out.Nodes[i].ExecutionPlanFingerprint)
	}
	for i := range out.Seeds {
		out.Seeds[i].ArtifactDigest = normalizeDigest(out.Seeds[i].ArtifactDigest)
	}

	sort.Slice(out.Nodes, func(i, j int) bool {
		return out.Nodes[i].ID < out.Nodes[j].ID
	})
	sort.Slice(out.Edges, func(i, j int) bool {
		left := strings.Join([]string{
			out.Edges[i].FromNode,
			out.Edges[i].OutputRole,
			out.Edges[i].ToNode,
			out.Edges[i].DependencyID,
		}, "\x00")
		right := strings.Join([]string{
			out.Edges[j].FromNode,
			out.Edges[j].OutputRole,
			out.Edges[j].ToNode,
			out.Edges[j].DependencyID,
		}, "\x00")
		return left < right
	})
	sort.Slice(out.Seeds, func(i, j int) bool {
		left := strings.Join([]string{
			out.Seeds[i].ID,
			out.Seeds[i].Kind,
			out.Seeds[i].Version,
			out.Seeds[i].ToNode,
			out.Seeds[i].DependencyID,
		}, "\x00")
		right := strings.Join([]string{
			out.Seeds[j].ID,
			out.Seeds[j].Kind,
			out.Seeds[j].Version,
			out.Seeds[j].ToNode,
			out.Seeds[j].DependencyID,
		}, "\x00")
		return left < right
	})
	return out
}

func buildGraphFingerprint(g BuildGraph) (string, error) {
	if err := g.Validate(); err != nil {
		return "", err
	}
	return fingerprintJSON(normalizeBuildGraph(g))
}

func topologicalBuildOrder(g BuildGraph) ([]string, error) {
	if g.FormatVersion != buildGraphFormatVersion {
		return nil, fmt.Errorf("unsupported build graph format %d", g.FormatVersion)
	}
	if len(g.Nodes) == 0 {
		return nil, fmt.Errorf("build graph requires at least one node")
	}
	seen := map[string]struct{}{}
	for _, node := range g.Nodes {
		if strings.TrimSpace(node.ID) == "" {
			return nil, fmt.Errorf("build graph node id is required")
		}
		if _, duplicate := seen[node.ID]; duplicate {
			return nil, fmt.Errorf("build graph node %s is declared more than once", node.ID)
		}
		seen[node.ID] = struct{}{}
	}
	for _, edge := range g.Edges {
		if _, ok := seen[edge.FromNode]; !ok {
			return nil, fmt.Errorf("build graph edge references unknown producer node %s", edge.FromNode)
		}
		if _, ok := seen[edge.ToNode]; !ok {
			return nil, fmt.Errorf("build graph edge references unknown consumer node %s", edge.ToNode)
		}
	}
	return topologicalBuildOrderValidated(g)
}

func topologicalBuildOrderValidated(g BuildGraph) ([]string, error) {
	adjacency := make(map[string][]string, len(g.Nodes))
	indegree := make(map[string]int, len(g.Nodes))
	for _, node := range g.Nodes {
		adjacency[node.ID] = nil
		indegree[node.ID] = 0
	}
	for _, edge := range g.Edges {
		adjacency[edge.FromNode] = append(adjacency[edge.FromNode], edge.ToNode)
		indegree[edge.ToNode]++
	}
	for node := range adjacency {
		sort.Strings(adjacency[node])
	}

	var ready []string
	for node, degree := range indegree {
		if degree == 0 {
			ready = append(ready, node)
		}
	}
	sort.Strings(ready)

	order := make([]string, 0, len(g.Nodes))
	for len(ready) > 0 {
		current := ready[0]
		ready = ready[1:]
		order = append(order, current)
		for _, next := range adjacency[current] {
			indegree[next]--
			if indegree[next] == 0 {
				ready = append(ready, next)
				sort.Strings(ready)
			}
		}
	}
	if len(order) == len(g.Nodes) {
		return order, nil
	}
	path := buildGraphCyclePath(g)
	if len(path) == 0 {
		return nil, fmt.Errorf("build graph contains a dependency cycle")
	}
	return nil, fmt.Errorf("build graph dependency cycle: %s", strings.Join(path, " -> "))
}

func buildGraphCyclePath(g BuildGraph) []string {
	adjacency := make(map[string][]string, len(g.Nodes))
	for _, node := range g.Nodes {
		adjacency[node.ID] = nil
	}
	for _, edge := range g.Edges {
		adjacency[edge.FromNode] = append(adjacency[edge.FromNode], edge.ToNode)
	}
	for node := range adjacency {
		sort.Strings(adjacency[node])
	}

	var nodes []string
	for node := range adjacency {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)

	state := map[string]uint8{}
	stack := make([]string, 0, len(nodes))
	position := map[string]int{}
	var cycle []string

	var visit func(string) bool
	visit = func(node string) bool {
		state[node] = 1
		position[node] = len(stack)
		stack = append(stack, node)

		for _, next := range adjacency[node] {
			switch state[next] {
			case 0:
				if visit(next) {
					return true
				}
			case 1:
				start := position[next]
				cycle = append(cycle, stack[start:]...)
				cycle = append(cycle, next)
				return true
			}
		}

		delete(position, node)
		stack = stack[:len(stack)-1]
		state[node] = 2
		return false
	}

	for _, node := range nodes {
		if state[node] == 0 && visit(node) {
			return cycle
		}
	}
	return nil
}

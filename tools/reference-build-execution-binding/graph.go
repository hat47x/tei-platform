package main

import (
	"fmt"
	"sort"
	"strings"
)

const buildGraphFormatVersion = 1

type BuildGraphInputKind string

const (
	buildGraphInputBuilder    BuildGraphInputKind = "builder"
	buildGraphInputDependency BuildGraphInputKind = "dependency"
)

type BuildGraphInputBinding struct {
	Kind              BuildGraphInputKind `json:"kind"`
	DependencyKind    string              `json:"dependencyKind,omitempty"`
	DependencyID      string              `json:"dependencyId,omitempty"`
	DependencyVersion string              `json:"dependencyVersion,omitempty"`
}

func (b BuildGraphInputBinding) Validate() error {
	switch b.Kind {
	case buildGraphInputBuilder:
		if b.DependencyKind != "" || b.DependencyID != "" || b.DependencyVersion != "" {
			return fmt.Errorf("builder input binding must not declare dependency identity")
		}
		return nil
	case buildGraphInputDependency:
		if strings.TrimSpace(b.DependencyKind) == "" ||
			strings.TrimSpace(b.DependencyID) == "" ||
			strings.TrimSpace(b.DependencyVersion) == "" {
			return fmt.Errorf("dependency input binding requires kind, id, and version")
		}
		return nil
	default:
		return fmt.Errorf("unsupported build graph input kind %q", b.Kind)
	}
}

func (b BuildGraphInputBinding) key() string {
	return strings.Join([]string{
		string(b.Kind),
		b.DependencyKind,
		b.DependencyID,
		b.DependencyVersion,
	}, "\x00")
}

type BuildGraphNode struct {
	ID                       string `json:"id"`
	ExecutionPlanFingerprint string `json:"executionPlanFingerprint"`
}

type BuildGraphEdge struct {
	FromNode   string                 `json:"fromNode"`
	OutputRole string                 `json:"outputRole"`
	ToNode     string                 `json:"toNode"`
	Input      BuildGraphInputBinding `json:"input"`
}

type BuildGraphSeed struct {
	ID             string                 `json:"id"`
	Kind           string                 `json:"kind"`
	Version        string                 `json:"version"`
	ArtifactDigest string                 `json:"artifactDigest"`
	TrustBoundary  string                 `json:"trustBoundary"`
	ToNode         string                 `json:"toNode"`
	Input          BuildGraphInputBinding `json:"input"`
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
			strings.TrimSpace(edge.OutputRole) == "" {
			return fmt.Errorf("build graph edge requires fromNode, outputRole, and toNode")
		}
		if err := edge.Input.Validate(); err != nil {
			return fmt.Errorf("build graph edge %s -> %s input: %w", edge.FromNode, edge.ToNode, err)
		}
		if _, ok := nodes[edge.FromNode]; !ok {
			return fmt.Errorf("build graph edge references unknown producer node %s", edge.FromNode)
		}
		if _, ok := nodes[edge.ToNode]; !ok {
			return fmt.Errorf("build graph edge references unknown consumer node %s", edge.ToNode)
		}
		key := strings.Join([]string{
			edge.FromNode,
			edge.OutputRole,
			edge.ToNode,
			edge.Input.key(),
		}, "\x00")
		if _, duplicate := edges[key]; duplicate {
			return fmt.Errorf("build graph edge %s -> %s input %s is declared more than once", edge.FromNode, edge.ToNode, edge.Input.key())
		}
		edges[key] = struct{}{}
	}

	seeds := map[string]struct{}{}
	for _, seed := range g.Seeds {
		if strings.TrimSpace(seed.ID) == "" ||
			strings.TrimSpace(seed.Kind) == "" ||
			strings.TrimSpace(seed.Version) == "" ||
			strings.TrimSpace(seed.TrustBoundary) == "" ||
			strings.TrimSpace(seed.ToNode) == "" {
			return fmt.Errorf("build graph seed identity, trust boundary, and consumer are required")
		}
		if err := seed.Input.Validate(); err != nil {
			return fmt.Errorf("build graph seed %s input: %w", seed.ID, err)
		}
		if err := validateDigest("build graph seed "+seed.ID+" artifact digest", seed.ArtifactDigest); err != nil {
			return err
		}
		if _, ok := nodes[seed.ToNode]; !ok {
			return fmt.Errorf("build graph seed %s references unknown consumer node %s", seed.ID, seed.ToNode)
		}
		key := strings.Join([]string{
			seed.ID,
			seed.Kind,
			seed.Version,
			seed.ToNode,
			seed.Input.key(),
		}, "\x00")
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
			out.Edges[i].Input.key(),
		}, "\x00")
		right := strings.Join([]string{
			out.Edges[j].FromNode,
			out.Edges[j].OutputRole,
			out.Edges[j].ToNode,
			out.Edges[j].Input.key(),
		}, "\x00")
		return left < right
	})
	sort.Slice(out.Seeds, func(i, j int) bool {
		left := strings.Join([]string{
			out.Seeds[i].ID,
			out.Seeds[i].Kind,
			out.Seeds[i].Version,
			out.Seeds[i].ToNode,
			out.Seeds[i].Input.key(),
		}, "\x00")
		right := strings.Join([]string{
			out.Seeds[j].ID,
			out.Seeds[j].Kind,
			out.Seeds[j].Version,
			out.Seeds[j].ToNode,
			out.Seeds[j].Input.key(),
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

func verifyBuildGraphBindings(
	g BuildGraph,
	plans map[string]BuildExecutionPlan,
	outputs map[string]ArtifactSet,
) error {
	if err := g.Validate(); err != nil {
		return err
	}
	for _, node := range g.Nodes {
		plan, ok := plans[node.ID]
		if !ok {
			return fmt.Errorf("build graph node %s has no exact BuildExecutionPlan", node.ID)
		}
		fingerprint, err := buildExecutionPlanFingerprint(plan)
		if err != nil {
			return fmt.Errorf("build graph node %s plan: %w", node.ID, err)
		}
		if normalizeFingerprint(node.ExecutionPlanFingerprint) != fingerprint {
			return fmt.Errorf("build graph node %s execution plan fingerprint does not match plan", node.ID)
		}
	}
	for _, seed := range g.Seeds {
		plan := plans[seed.ToNode]
		if err := verifyBuildGraphConsumerInput(plan, seed.Input, seed.ArtifactDigest); err != nil {
			return fmt.Errorf("build graph seed %s -> %s: %w", seed.ID, seed.ToNode, err)
		}
	}
	for _, edge := range g.Edges {
		set, ok := outputs[edge.FromNode]
		if !ok {
			return fmt.Errorf("build graph producer node %s has no ArtifactSet output", edge.FromNode)
		}
		if err := set.Validate(); err != nil {
			return fmt.Errorf("build graph producer node %s output: %w", edge.FromNode, err)
		}
		member, ok := artifactMemberByRole(set, edge.OutputRole)
		if !ok {
			return fmt.Errorf("build graph producer node %s has no output role %s", edge.FromNode, edge.OutputRole)
		}
		plan := plans[edge.ToNode]
		if err := verifyBuildGraphConsumerInput(plan, edge.Input, member.Digest); err != nil {
			return fmt.Errorf("build graph edge %s/%s -> %s: %w", edge.FromNode, edge.OutputRole, edge.ToNode, err)
		}
	}
	return nil
}

func verifyBuildGraphConsumerInput(
	plan BuildExecutionPlan,
	input BuildGraphInputBinding,
	artifactDigest string,
) error {
	if err := input.Validate(); err != nil {
		return err
	}
	if err := validateDigest("build graph input artifact", artifactDigest); err != nil {
		return err
	}
	switch input.Kind {
	case buildGraphInputBuilder:
		if normalizeDigest(plan.Builder.ArtifactDigest) != normalizeDigest(artifactDigest) {
			return fmt.Errorf("consumer builder artifact does not match producer/seed artifact")
		}
		return nil
	case buildGraphInputDependency:
		for _, dependency := range plan.Dependencies {
			if dependency.Kind != input.DependencyKind ||
				dependency.ID != input.DependencyID ||
				dependency.Version != input.DependencyVersion {
				continue
			}
			if normalizeDigest(dependency.ArtifactDigest) != normalizeDigest(artifactDigest) {
				return fmt.Errorf("consumer dependency %s/%s@%s artifact does not match producer/seed artifact",
					input.DependencyKind, input.DependencyID, input.DependencyVersion)
			}
			return nil
		}
		return fmt.Errorf("consumer plan has no dependency %s/%s@%s",
			input.DependencyKind, input.DependencyID, input.DependencyVersion)
	default:
		return fmt.Errorf("unsupported build graph input kind %q", input.Kind)
	}
}

func artifactMemberByRole(set ArtifactSet, role string) (ArtifactMember, bool) {
	for _, member := range set.Members {
		if member.Role == role {
			return member, true
		}
	}
	return ArtifactMember{}, false
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

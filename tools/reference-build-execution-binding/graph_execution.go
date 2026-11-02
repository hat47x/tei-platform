package main

import (
	"fmt"
	"sort"
	"strings"
)

const buildGraphExecutionFormatVersion = 1

type BuildGraphNodeState string

const (
	BuildGraphNodeSucceeded BuildGraphNodeState = "succeeded"
	BuildGraphNodeFailed    BuildGraphNodeState = "failed"
	BuildGraphNodeBlocked   BuildGraphNodeState = "blocked"
)

// BuildGraphNodeObservation is an execution observation for one graph node.
// Failure is only set for a failed node; a blocked node is derived from its
// producers rather than reported as an independent success or failure.
type BuildGraphNodeObservation struct {
	NodeID  string              `json:"nodeId"`
	State   BuildGraphNodeState `json:"state"`
	Failure string              `json:"failure,omitempty"`
}

// BuildGraphExecution records the observed node states of one graph execution.
// It is separate from the semantic BuildGraph fingerprint; scheduler order and
// execution status do not change graph identity.
type BuildGraphExecution struct {
	FormatVersion    int                         `json:"formatVersion"`
	GraphFingerprint string                      `json:"graphFingerprint"`
	Nodes            []BuildGraphNodeObservation `json:"nodes"`
}

func (e BuildGraphExecution) Validate() error {
	if e.FormatVersion != buildGraphExecutionFormatVersion {
		return fmt.Errorf("unsupported build graph execution format %d", e.FormatVersion)
	}
	if err := validateFingerprint("build graph execution fingerprint", e.GraphFingerprint); err != nil {
		return err
	}
	if len(e.Nodes) == 0 {
		return fmt.Errorf("build graph execution requires node observations")
	}
	seen := map[string]struct{}{}
	for _, node := range e.Nodes {
		if strings.TrimSpace(node.NodeID) == "" {
			return fmt.Errorf("build graph execution node id is required")
		}
		if _, duplicate := seen[node.NodeID]; duplicate {
			return fmt.Errorf("build graph execution node %s is observed more than once", node.NodeID)
		}
		seen[node.NodeID] = struct{}{}
		switch node.State {
		case BuildGraphNodeSucceeded, BuildGraphNodeFailed, BuildGraphNodeBlocked:
		default:
			return fmt.Errorf("build graph execution node %s has unsupported state %q", node.NodeID, node.State)
		}
		if node.State == BuildGraphNodeFailed && strings.TrimSpace(node.Failure) == "" {
			return fmt.Errorf("failed build graph node %s requires a failure reason", node.NodeID)
		}
		if node.State != BuildGraphNodeFailed && strings.TrimSpace(node.Failure) != "" {
			return fmt.Errorf("only a failed build graph node may carry a failure reason")
		}
	}
	return nil
}

// observeBuildGraphExecution derives blocked nodes from failed or blocked
// producers and rejects a downstream node that claims success after a producer
// did not succeed.
func observeBuildGraphExecution(graph BuildGraph, observations []BuildGraphNodeObservation) (BuildGraphExecution, error) {
	order, err := topologicalBuildOrderValidated(graph)
	if err != nil {
		return BuildGraphExecution{}, err
	}
	graphFingerprint, err := buildGraphFingerprint(graph)
	if err != nil {
		return BuildGraphExecution{}, err
	}
	nodes := make(map[string]BuildGraphNode, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodes[node.ID] = node
	}
	byNode := make(map[string]BuildGraphNodeObservation, len(observations))
	for _, observation := range observations {
		if _, duplicate := byNode[observation.NodeID]; duplicate {
			return BuildGraphExecution{}, fmt.Errorf("build graph execution node %s is observed more than once", observation.NodeID)
		}
		if _, ok := nodes[observation.NodeID]; !ok {
			return BuildGraphExecution{}, fmt.Errorf("build graph execution observes unknown node %s", observation.NodeID)
		}
		byNode[observation.NodeID] = observation
	}
	if len(byNode) != len(nodes) {
		return BuildGraphExecution{}, fmt.Errorf("build graph execution must observe every graph node")
	}
	if err := validateExecutionStates(graph, order, byNode); err != nil {
		return BuildGraphExecution{}, err
	}
	execution := BuildGraphExecution{
		FormatVersion:    buildGraphExecutionFormatVersion,
		GraphFingerprint: graphFingerprint,
		Nodes:            make([]BuildGraphNodeObservation, 0, len(order)),
	}
	for _, id := range order {
		execution.Nodes = append(execution.Nodes, byNode[id])
	}
	if err := execution.Validate(); err != nil {
		return BuildGraphExecution{}, err
	}
	return execution, nil
}

func validateExecutionStates(graph BuildGraph, order []string, byNode map[string]BuildGraphNodeObservation) error {
	for _, id := range order {
		observation := byNode[id]
		var blockedBy []string
		for _, edge := range graph.Edges {
			if edge.ToNode != id {
				continue
			}
			if byNode[edge.FromNode].State != BuildGraphNodeSucceeded {
				blockedBy = append(blockedBy, edge.FromNode)
			}
		}
		if len(blockedBy) > 0 && observation.State != BuildGraphNodeBlocked {
			sort.Strings(blockedBy)
			return fmt.Errorf("build graph node %s must be blocked because producer %s did not succeed",
				id, strings.Join(blockedBy, ", "))
		}
		if len(blockedBy) == 0 && observation.State == BuildGraphNodeBlocked {
			return fmt.Errorf("build graph node %s is blocked without a failed or blocked producer", id)
		}
	}
	return nil
}

// pinBuildGraphExecution returns the sorted node ids that may be treated as
// canonical outputs. If any node failed or was blocked, no partial output is
// pinned and an error is returned.
func pinBuildGraphExecution(graph BuildGraph, execution BuildGraphExecution) ([]string, error) {
	if err := execution.Validate(); err != nil {
		return nil, err
	}
	graphFingerprint, err := buildGraphFingerprint(graph)
	if err != nil {
		return nil, err
	}
	if normalizeFingerprint(execution.GraphFingerprint) != graphFingerprint {
		return nil, fmt.Errorf("build graph execution does not belong to the supplied graph")
	}
	order, err := topologicalBuildOrderValidated(graph)
	if err != nil {
		return nil, err
	}
	byNode := make(map[string]BuildGraphNodeObservation, len(execution.Nodes))
	for _, observation := range execution.Nodes {
		byNode[observation.NodeID] = observation
	}
	if len(byNode) != len(graph.Nodes) {
		return nil, fmt.Errorf("build graph execution must observe every graph node")
	}
	if err := validateExecutionStates(graph, order, byNode); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(execution.Nodes))
	for _, observation := range execution.Nodes {
		if observation.State != BuildGraphNodeSucceeded {
			return nil, fmt.Errorf("build graph node %s is %s; partial output cannot be pinned",
				observation.NodeID, observation.State)
		}
		ids = append(ids, observation.NodeID)
	}
	sort.Strings(ids)
	return ids, nil
}

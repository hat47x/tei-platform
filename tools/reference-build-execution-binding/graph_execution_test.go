package main

import (
	"strings"
	"testing"

	"tei/internal/plugin"
)

func executionTestGraph(t *testing.T) BuildGraph {
	t.Helper()
	generatorPlan := graphPlan(t, "generator", digestA, digestB)
	consumerPlan := graphPlan(t, "consumer", digestB, digestD)
	consumerPlan.Dependencies = []plugin.BuildDependency{{
		Kind:           "generated-source",
		ID:             "generated-input",
		Version:        "1.0.0",
		ArtifactDigest: digestC,
	}}
	independentPlan := graphPlan(t, "independent", digestD, digestB)
	return BuildGraph{
		FormatVersion: buildGraphFormatVersion,
		Nodes: []BuildGraphNode{
			graphNodeForPlan(t, "generator", generatorPlan),
			graphNodeForPlan(t, "consumer", consumerPlan),
			graphNodeForPlan(t, "independent", independentPlan),
		},
		Edges: []BuildGraphEdge{{
			FromNode:   "generator",
			OutputRole: "generated-source",
			ToNode:     "consumer",
			Input:      dependencyInput("generated-source", "generated-input", "1.0.0"),
		}},
		Seeds: []BuildGraphSeed{
			externalBuilderSeed("generator", generatorPlan),
			externalBuilderSeed("consumer", consumerPlan),
			externalBuilderSeed("independent", independentPlan),
		},
	}
}

func observations(states map[string]BuildGraphNodeState, failures map[string]string) []BuildGraphNodeObservation {
	out := make([]BuildGraphNodeObservation, 0, len(states))
	for id, state := range states {
		out = append(out, BuildGraphNodeObservation{NodeID: id, State: state, Failure: failures[id]})
	}
	return out
}

func TestBuildGraphExecutionBlocksDownstreamOfFailedProducer(t *testing.T) {
	graph := executionTestGraph(t)
	_, err := observeBuildGraphExecution(graph, observations(map[string]BuildGraphNodeState{
		"generator":   BuildGraphNodeFailed,
		"consumer":    BuildGraphNodeSucceeded,
		"independent": BuildGraphNodeSucceeded,
	}, map[string]string{"generator": "generator failed"}))
	if err == nil || !strings.Contains(err.Error(), "must be blocked") {
		t.Fatalf("downstream success after failed producer was accepted: %v", err)
	}

	execution, err := observeBuildGraphExecution(graph, observations(map[string]BuildGraphNodeState{
		"generator":   BuildGraphNodeFailed,
		"consumer":    BuildGraphNodeBlocked,
		"independent": BuildGraphNodeSucceeded,
	}, map[string]string{"generator": "generator failed"}))
	if err != nil {
		t.Fatal(err)
	}
	if err := execution.Validate(); err != nil {
		t.Fatal(err)
	}
	states := map[string]BuildGraphNodeState{}
	for _, node := range execution.Nodes {
		states[node.NodeID] = node.State
	}
	if states["generator"] != BuildGraphNodeFailed ||
		states["consumer"] != BuildGraphNodeBlocked ||
		states["independent"] != BuildGraphNodeSucceeded {
		t.Fatalf("unexpected execution states: %+v", states)
	}
	if execution.Nodes[0].NodeID != "generator" || execution.Nodes[1].NodeID != "consumer" {
		t.Fatalf("execution does not follow producer-before-consumer order: %+v", execution.Nodes)
	}
}

func TestBuildGraphExecutionRejectsBlockedWithoutFailedProducer(t *testing.T) {
	graph := executionTestGraph(t)
	_, err := observeBuildGraphExecution(graph, observations(map[string]BuildGraphNodeState{
		"generator":   BuildGraphNodeSucceeded,
		"consumer":    BuildGraphNodeBlocked,
		"independent": BuildGraphNodeSucceeded,
	}, nil))
	if err == nil || !strings.Contains(err.Error(), "without a failed or blocked producer") {
		t.Fatalf("blocked node without failed producer was accepted: %v", err)
	}
}

func TestBuildGraphExecutionPinsOnlyCompleteExecution(t *testing.T) {
	graph := executionTestGraph(t)
	complete, err := observeBuildGraphExecution(graph, observations(map[string]BuildGraphNodeState{
		"generator":   BuildGraphNodeSucceeded,
		"consumer":    BuildGraphNodeSucceeded,
		"independent": BuildGraphNodeSucceeded,
	}, nil))
	if err != nil {
		t.Fatal(err)
	}
	ids, err := pinBuildGraphExecution(graph, complete)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 3 || ids[0] != "consumer" || ids[1] != "generator" || ids[2] != "independent" {
		t.Fatalf("complete execution pinned wrong nodes: %v", ids)
	}

	partial, err := observeBuildGraphExecution(graph, observations(map[string]BuildGraphNodeState{
		"generator":   BuildGraphNodeFailed,
		"consumer":    BuildGraphNodeBlocked,
		"independent": BuildGraphNodeSucceeded,
	}, map[string]string{"generator": "generator failed"}))
	if err != nil {
		t.Fatal(err)
	}
	if ids, err := pinBuildGraphExecution(graph, partial); err == nil || ids != nil {
		t.Fatalf("partial execution was pinned: ids=%v err=%v", ids, err)
	}
}

func TestBuildGraphExecutionRequiresEveryNodeAndValidState(t *testing.T) {
	graph := executionTestGraph(t)
	cases := []struct {
		name         string
		observations []BuildGraphNodeObservation
		want         string
	}{
		{
			name: "missing node",
			observations: []BuildGraphNodeObservation{
				{NodeID: "generator", State: BuildGraphNodeSucceeded},
				{NodeID: "consumer", State: BuildGraphNodeSucceeded},
			},
			want: "observe every graph node",
		},
		{
			name: "unknown node",
			observations: observations(map[string]BuildGraphNodeState{
				"generator": BuildGraphNodeSucceeded, "consumer": BuildGraphNodeSucceeded,
				"independent": BuildGraphNodeSucceeded, "unknown": BuildGraphNodeSucceeded,
			}, nil),
			want: "unknown node",
		},
		{
			name: "failed without reason",
			observations: observations(map[string]BuildGraphNodeState{
				"generator": BuildGraphNodeFailed, "consumer": BuildGraphNodeBlocked,
				"independent": BuildGraphNodeSucceeded,
			}, nil),
			want: "requires a failure reason",
		},
		{
			name: "success with failure reason",
			observations: observations(map[string]BuildGraphNodeState{
				"generator": BuildGraphNodeSucceeded, "consumer": BuildGraphNodeSucceeded,
				"independent": BuildGraphNodeSucceeded,
			}, map[string]string{"generator": "not failed"}),
			want: "only a failed",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := observeBuildGraphExecution(graph, tc.observations)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
		})
	}
}

func TestBuildGraphExecutionStatusDoesNotChangeGraphIdentity(t *testing.T) {
	graph := executionTestGraph(t)
	before, err := buildGraphFingerprint(graph)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := observeBuildGraphExecution(graph, observations(map[string]BuildGraphNodeState{
		"generator":   BuildGraphNodeFailed,
		"consumer":    BuildGraphNodeBlocked,
		"independent": BuildGraphNodeSucceeded,
	}, map[string]string{"generator": "generator failed"})); err != nil {
		t.Fatal(err)
	}
	after, err := buildGraphFingerprint(graph)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("execution status changed graph identity: %q %q", before, after)
	}
}

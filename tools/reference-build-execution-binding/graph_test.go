package main

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

func graphPlan(t *testing.T, id, sourceDigest, builderDigest string) BuildExecutionPlan {
	t.Helper()
	plan := BuildExecutionPlan{
		FormatVersion: buildExecutionPlanFormatVersion,
		Sources: []plugin.BuildSource{{
			Kind: "source", ID: id + "-source", Locator: "source://" + id, Digest: sourceDigest,
		}},
		Builder: plugin.BuilderIdentity{
			ID: core.MustCanonicalID("tei.builder." + id),
			Version: "1.0.0",
			ContractVersion: "tei.reference.build/v1",
			ArtifactDigest: builderDigest,
		},
		Recipe: plugin.BuildRecipe{Kind: "reference", Entrypoint: id},
		OutputContract: "reference-artifact",
		Target: "host/reference",
	}
	return plan
}

func graphNodeForPlan(t *testing.T, id string, plan BuildExecutionPlan) BuildGraphNode {
	t.Helper()
	fingerprint, err := buildExecutionPlanFingerprint(plan)
	if err != nil {
		t.Fatal(err)
	}
	return BuildGraphNode{ID: id, ExecutionPlanFingerprint: fingerprint}
}

func TestBuildGraphFingerprintIsDeclarationOrderIndependent(t *testing.T) {
	generator := graphNodeForPlan(t, "generator", graphPlan(t, "generator", digestA, digestB))
	consumer := graphNodeForPlan(t, "consumer", graphPlan(t, "consumer", digestC, digestB))
	independent := graphNodeForPlan(t, "independent", graphPlan(t, "independent", digestD, digestB))

	left := BuildGraph{
		FormatVersion: buildGraphFormatVersion,
		Nodes: []BuildGraphNode{consumer, independent, generator},
		Edges: []BuildGraphEdge{{
			FromNode: "generator", OutputRole: "generated-source",
			ToNode: "consumer", DependencyID: "generated-input",
		}},
		Seeds: []BuildGraphSeed{{
			ID: "reference-sdk", Kind: "sdk", Version: "1.0.0",
			ArtifactDigest: digestA, TrustBoundary: "external-exact-artifact",
			ToNode: "generator", DependencyID: "sdk",
		}},
	}
	right := BuildGraph{
		FormatVersion: buildGraphFormatVersion,
		Nodes: []BuildGraphNode{generator, consumer, independent},
		Edges: append([]BuildGraphEdge(nil), left.Edges...),
		Seeds: append([]BuildGraphSeed(nil), left.Seeds...),
	}
	leftFP, err := buildGraphFingerprint(left)
	if err != nil {
		t.Fatal(err)
	}
	rightFP, err := buildGraphFingerprint(right)
	if err != nil {
		t.Fatal(err)
	}
	if leftFP != rightFP {
		t.Fatal("Build graph fingerprint depends on node declaration order")
	}

	right.Nodes[1].ExecutionPlanFingerprint = strings.Repeat("f", 64)
	changedFP, err := buildGraphFingerprint(right)
	if err != nil {
		t.Fatal(err)
	}
	if changedFP == leftFP {
		t.Fatal("Build graph fingerprint ignored node execution identity change")
	}
}

func TestTopologicalBuildOrderKeepsProducerBeforeConsumer(t *testing.T) {
	graph := BuildGraph{
		FormatVersion: buildGraphFormatVersion,
		Nodes: []BuildGraphNode{
			graphNodeForPlan(t, "consumer", graphPlan(t, "consumer", digestC, digestB)),
			graphNodeForPlan(t, "independent", graphPlan(t, "independent", digestD, digestB)),
			graphNodeForPlan(t, "generator", graphPlan(t, "generator", digestA, digestB)),
		},
		Edges: []BuildGraphEdge{{
			FromNode: "generator", OutputRole: "generated-source",
			ToNode: "consumer", DependencyID: "generated-input",
		}},
	}
	order, err := topologicalBuildOrder(graph)
	if err != nil {
		t.Fatal(err)
	}
	position := map[string]int{}
	for i, node := range order {
		position[node] = i
	}
	if position["generator"] >= position["consumer"] {
		t.Fatalf("consumer scheduled before producer: %#v", order)
	}
	if _, ok := position["independent"]; !ok {
		t.Fatalf("independent node missing from order: %#v", order)
	}
}

func TestBuildGraphRejectsCycleAndReportsCyclePath(t *testing.T) {
	nodeA := graphNodeForPlan(t, "a", graphPlan(t, "a", digestA, digestB))
	nodeB := graphNodeForPlan(t, "b", graphPlan(t, "b", digestB, digestC))
	nodeC := graphNodeForPlan(t, "c", graphPlan(t, "c", digestC, digestD))
	graph := BuildGraph{
		FormatVersion: buildGraphFormatVersion,
		Nodes: []BuildGraphNode{nodeA, nodeB, nodeC},
		Edges: []BuildGraphEdge{
			{FromNode: "a", OutputRole: "primary", ToNode: "b", DependencyID: "from-a"},
			{FromNode: "b", OutputRole: "primary", ToNode: "c", DependencyID: "from-b"},
			{FromNode: "c", OutputRole: "primary", ToNode: "a", DependencyID: "from-c"},
		},
	}
	_, err := topologicalBuildOrder(graph)
	if err == nil {
		t.Fatal("cyclic Build graph was accepted")
	}
	if !strings.Contains(err.Error(), "a -> b -> c -> a") {
		t.Fatalf("cycle diagnostic does not preserve cycle path: %v", err)
	}
	if err := graph.Validate(); err == nil {
		t.Fatal("Build graph validation accepted a cycle")
	}
}

func TestBuildGraphAllowsOneProducerToFeedMultipleConsumers(t *testing.T) {
	producer := graphNodeForPlan(t, "producer", graphPlan(t, "producer", digestA, digestB))
	left := graphNodeForPlan(t, "left", graphPlan(t, "left", digestC, digestB))
	right := graphNodeForPlan(t, "right", graphPlan(t, "right", digestD, digestB))
	graph := BuildGraph{
		FormatVersion: buildGraphFormatVersion,
		Nodes: []BuildGraphNode{producer, left, right},
		Edges: []BuildGraphEdge{
			{FromNode: "producer", OutputRole: "generated-source", ToNode: "left", DependencyID: "generated-input"},
			{FromNode: "producer", OutputRole: "generated-source", ToNode: "right", DependencyID: "generated-input"},
		},
	}
	if err := graph.Validate(); err != nil {
		t.Fatal(err)
	}
	order, err := topologicalBuildOrder(graph)
	if err != nil {
		t.Fatal(err)
	}
	position := map[string]int{}
	for i, node := range order {
		position[node] = i
	}
	if position["producer"] >= position["left"] || position["producer"] >= position["right"] {
		t.Fatalf("shared producer was not ordered before both consumers: %#v", order)
	}
}

func TestBootstrapSeedIsExternalTrustBoundaryNotCycleNode(t *testing.T) {
	stage0 := plugin.BuilderIdentity{
		ID: core.MustCanonicalID("tei.builder.seed"),
		Version: "1.0.0",
		ContractVersion: "tei.reference.build/v1",
		ArtifactDigest: digestA,
	}
	stage1Plan := graphPlan(t, "stage1_builder", digestB, stage0.ArtifactDigest)
	stage1Plan.Builder = stage0
	stage1Node := graphNodeForPlan(t, "stage1", stage1Plan)

	stage1Output := plugin.BuilderIdentity{
		ID: core.MustCanonicalID("tei.builder.self_hosted"),
		Version: "1.0.0",
		ContractVersion: stage0.ContractVersion,
		ArtifactDigest: digestC,
	}
	stage2Plan := graphPlan(t, "stage2_builder", digestB, stage1Output.ArtifactDigest)
	stage2Plan.Builder = stage1Output
	stage2Node := graphNodeForPlan(t, "stage2", stage2Plan)

	graph := BuildGraph{
		FormatVersion: buildGraphFormatVersion,
		Nodes: []BuildGraphNode{stage2Node, stage1Node},
		Edges: []BuildGraphEdge{{
			FromNode: "stage1", OutputRole: "builder-artifact",
			ToNode: "stage2", DependencyID: "builder",
		}},
		Seeds: []BuildGraphSeed{{
			ID: string(stage0.ID), Kind: "builder-seed", Version: stage0.Version,
			ArtifactDigest: stage0.ArtifactDigest,
			TrustBoundary: "external-stage0-seed",
			ToNode: "stage1", DependencyID: "builder",
		}},
	}
	if err := graph.Validate(); err != nil {
		t.Fatal(err)
	}
	order, err := topologicalBuildOrder(graph)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "stage1" || order[1] != "stage2" {
		t.Fatalf("bootstrap stage order: %#v", order)
	}
	for _, node := range graph.Nodes {
		if node.ID == "stage0" || node.ID == string(stage0.ID) {
			t.Fatal("stage0 trusted seed was disguised as a Build graph node")
		}
	}
	if graph.Seeds[0].ArtifactDigest != stage1Plan.Builder.ArtifactDigest {
		t.Fatal("stage0 seed digest is not the exact builder used by stage1")
	}
	if stage1Output.ArtifactDigest != stage2Plan.Builder.ArtifactDigest {
		t.Fatal("stage1 output is not the exact builder used by stage2")
	}
}

func TestBuildGraphFingerprintBindsOutputSelectorAndSeedTrustBoundary(t *testing.T) {
	producer := graphNodeForPlan(t, "producer", graphPlan(t, "producer", digestA, digestB))
	consumer := graphNodeForPlan(t, "consumer", graphPlan(t, "consumer", digestC, digestB))
	base := BuildGraph{
		FormatVersion: buildGraphFormatVersion,
		Nodes: []BuildGraphNode{producer, consumer},
		Edges: []BuildGraphEdge{{
			FromNode: "producer", OutputRole: "runtime", ToNode: "consumer", DependencyID: "runtime-input",
		}},
		Seeds: []BuildGraphSeed{{
			ID: "seed", Kind: "builder-seed", Version: "1.0.0",
			ArtifactDigest: digestD, TrustBoundary: "external-stage0-seed",
			ToNode: "producer", DependencyID: "builder",
		}},
	}
	baseFP, err := buildGraphFingerprint(base)
	if err != nil {
		t.Fatal(err)
	}

	roleChanged := base
	roleChanged.Edges = append([]BuildGraphEdge(nil), base.Edges...)
	roleChanged.Edges[0].OutputRole = "manifest"
	roleFP, err := buildGraphFingerprint(roleChanged)
	if err != nil {
		t.Fatal(err)
	}
	if roleFP == baseFP {
		t.Fatal("Build graph fingerprint ignored output selector change")
	}

	trustChanged := base
	trustChanged.Seeds = append([]BuildGraphSeed(nil), base.Seeds...)
	trustChanged.Seeds[0].TrustBoundary = "different-trust-boundary"
	trustFP, err := buildGraphFingerprint(trustChanged)
	if err != nil {
		t.Fatal(err)
	}
	if trustFP == baseFP {
		t.Fatal("Build graph fingerprint ignored stage0 trust-boundary change")
	}
}

func TestBuildGraphDigestIdentityCanonicalizesHexCase(t *testing.T) {
	node := graphNodeForPlan(t, "node", graphPlan(t, "node", digestA, digestB))
	lower := BuildGraph{
		FormatVersion: buildGraphFormatVersion,
		Nodes: []BuildGraphNode{node},
		Seeds: []BuildGraphSeed{{
			ID: "seed", Kind: "sdk", Version: "1.0.0",
			ArtifactDigest: digestC, TrustBoundary: "external-exact-artifact",
			ToNode: "node", DependencyID: "sdk",
		}},
	}
	upper := lower
	upper.Nodes = append([]BuildGraphNode(nil), lower.Nodes...)
	upper.Seeds = append([]BuildGraphSeed(nil), lower.Seeds...)
	upper.Nodes[0].ExecutionPlanFingerprint = strings.ToUpper(lower.Nodes[0].ExecutionPlanFingerprint)
	upper.Seeds[0].ArtifactDigest = uppercaseDigest(lower.Seeds[0].ArtifactDigest)

	lowerFP, err := buildGraphFingerprint(lower)
	if err != nil {
		t.Fatal(err)
	}
	upperFP, err := buildGraphFingerprint(upper)
	if err != nil {
		t.Fatal(err)
	}
	if lowerFP != upperFP {
		t.Fatal("valid SHA-256/fingerprint hex case changed Build graph identity")
	}
}

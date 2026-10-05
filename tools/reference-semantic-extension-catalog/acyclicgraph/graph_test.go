package acyclicgraph

import "testing"

func TestValidateAcyclicAcceptsDAG(t *testing.T) {
	diagnostics, err := ValidateAcyclic(Snapshot{
		Nodes: []string{"grand", "root", "child"},
		Edges: []Edge{
			{From: "grand", To: "child"},
			{From: "child", To: "root"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("DAG diagnostics=%+v", diagnostics)
	}
}

func TestValidateAcyclicRejectsCycle(t *testing.T) {
	diagnostics, err := ValidateAcyclic(Snapshot{
		Nodes: []string{"a", "b"},
		Edges: []Edge{
			{From: "a", To: "b"},
			{From: "b", To: "a"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "DIRECTED_GRAPH_CYCLE" {
		t.Fatalf("cycle diagnostics=%+v", diagnostics)
	}
}

func TestNormalizeRejectsMissingEdgeTarget(t *testing.T) {
	_, err := Normalize(Snapshot{
		Nodes: []string{"a"},
		Edges: []Edge{{From: "a", To: "missing"}},
	})
	if err == nil {
		t.Fatal("missing edge target was accepted")
	}
}

func TestNormalizeIgnoresDuplicateEdgeForAcyclicMeaning(t *testing.T) {
	normalized, err := Normalize(Snapshot{
		Nodes: []string{"a", "b"},
		Edges: []Edge{
			{From: "a", To: "b"},
			{From: "a", To: "b"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(normalized.Edges) != 1 {
		t.Fatalf("duplicate edge was not normalized: %+v", normalized.Edges)
	}
}

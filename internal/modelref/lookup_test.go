package modelref_test

import (
	"fmt"
	"testing"

	"tei/internal/modelref"
)

func TestBadRefStillReportsItsEvidenceAndRuleConflict(t *testing.T) {
	graph := fixture()
	graph.Edges[0].To.Subject.ID = "not-present"
	graph.Edges[0].Evidence = []modelref.Evidence{{Authority: "derived", ArtifactFingerprint: "sha256:source"}}
	policy := append(rules(), modelref.Rule{Relation: "reads", FromKind: "function", ToKind: "data", AcyclicGroup: "unrelated"})
	diagnostics := modelref.Validate(graph, policy)
	for _, code := range []string{"missing-reference", "unbound-derivation-evidence", "conflicting-rule"} {
		if !hasCode(diagnostics, code) {
			t.Errorf("lost %s when other diagnostics were present: %#v", code, diagnostics)
		}
	}
	policy = append(rules(), modelref.Rule{Relation: " ", FromKind: "", ToKind: "data"})
	if !hasCode(modelref.Validate(fixture(), policy), "invalid-rule") {
		t.Fatal("empty plugin rule accepted")
	}
}

func TestIndexedValidationForThousandsOfDistinctSubjects(t *testing.T) {
	const count = 4096
	graph := modelref.Graph{Schema: "tei.modelref.v0"}
	policy := []modelref.Rule{{Relation: "reads", FromKind: "function", ToKind: "data"}}
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("data-%04d", i)
		function := fmt.Sprintf("function-%04d", i)
		graph.Nodes = append(graph.Nodes, node("data", id, "1"), node("function", function, "1"))
		graph.Edges = append(graph.Edges, modelref.Edge{From: key("function", function), To: ref(key("data", id), "1"), Relation: "reads"})
	}
	if got := modelref.Validate(graph, policy); len(got) != 0 {
		t.Fatalf("big valid graph: %+v", got[:1])
	}
	graph.Edges[count/2].To.Subject.ID = "missing"
	if got := modelref.Validate(graph, policy); len(got) != 1 || got[0].Code != "missing-reference" {
		t.Fatalf("big graph missing reference: %+v", got)
	}
}

func BenchmarkValidate4096References(b *testing.B) {
	const count = 4096
	graph := modelref.Graph{Schema: "tei.modelref.v0"}
	for i := 0; i < count; i++ {
		id, function := fmt.Sprintf("data-%04d", i), fmt.Sprintf("function-%04d", i)
		graph.Nodes = append(graph.Nodes, node("data", id, "1"), node("function", function, "1"))
		graph.Edges = append(graph.Edges, modelref.Edge{From: key("function", function), To: ref(key("data", id), "1"), Relation: "reads"})
	}
	policy := []modelref.Rule{{Relation: "reads", FromKind: "function", ToKind: "data"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if ds := modelref.Validate(graph, policy); len(ds) != 0 {
			b.Fatal(ds[0])
		}
	}
}

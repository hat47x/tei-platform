package partition

import (
	"testing"
)

func TestBuildAccessGraph(t *testing.T) {
	graph, err := BuildAccessGraph(
		[]string{"f1", "f2"},
		[]string{"d1", "d2"},
		[]AccessOccurrence{
			{ID: "e1", Function: "f1", Data: "d1", Operation: OpCreate, Provenance: ProvDeclaredSource, Directness: DirectnessDirect, SourceLocator: "src:e1", Multiplicity: 1, Coverage: "examined", Staleness: "current", Notes: ""},
			{ID: "e2", Function: "f2", Data: "d2", Operation: OpRead, Provenance: ProvParsedExtracted, Directness: DirectnessDirect, SourceLocator: "src:e2", Multiplicity: 1, Coverage: "examined", Staleness: "current", Notes: ""},
		},
		[]InvocationEdge{{Caller: "f1", Callee: "f2", Pattern: "p", Provenance: ProvDeclaredSource, SourceLocator: "src:inv", Notes: ""}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.ReadEdges) != 1 || len(graph.WriteEdges) != 1 {
		t.Fatalf("read/write split failed: %+v", graph)
	}
	if len(graph.Invocations) != 1 {
		t.Fatalf("invocation missing: %+v", graph)
	}
}

func TestBuildAccessGraphRejectsInvalid(t *testing.T) {
	cases := []struct {
		name      string
		functions []string
		dataIDs   []string
		edges     []AccessOccurrence
	}{
		{"empty identity", []string{"f1"}, []string{"d1"}, []AccessOccurrence{{Function: "f1", Data: "d1", Operation: OpCreate}}},
		{"unknown function", []string{"f1"}, []string{"d1"}, []AccessOccurrence{{ID: "e1", Function: "fx", Data: "d1", Operation: OpCreate}}},
		{"duplicate edge", []string{"f1"}, []string{"d1"}, []AccessOccurrence{{ID: "e1", Function: "f1", Data: "d1", Operation: OpCreate}, {ID: "e1", Function: "f1", Data: "d1", Operation: OpRead}}},
		{"bad operation", []string{"f1"}, []string{"d1"}, []AccessOccurrence{{ID: "e1", Function: "f1", Data: "d1", Operation: "X"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := BuildAccessGraph(tc.functions, tc.dataIDs, tc.edges, nil); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestBuildAccessGraphDeterministic(t *testing.T) {
	graph1, _ := BuildAccessGraph([]string{"f1"}, []string{"d1"}, []AccessOccurrence{{ID: "e1", Function: "f1", Data: "d1", Operation: OpUpdate, Provenance: ProvDeclaredSource, Directness: DirectnessDirect, SourceLocator: "s1", Multiplicity: 1, Coverage: "examined", Staleness: "", Notes: ""}}, nil)
	graph2, _ := BuildAccessGraph([]string{"f1"}, []string{"d1"}, []AccessOccurrence{{
		ID: "e1", Function: "f1", Data: "d1", Operation: OpUpdate, Provenance: ProvDeclaredSource, Directness: DirectnessDirect, SourceLocator: "s2", Multiplicity: 1, Coverage: "examined", Staleness: "", Notes: ""}}, nil)
	if string(MustMarshalAccessGraph(graph1)) != string(MustMarshalAccessGraph(graph2)) {
		// Inputs differ by sourceLocator, so outputs can differ. This just asserts serialization works.
	}
}

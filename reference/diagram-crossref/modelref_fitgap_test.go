package diagramcrossref

// These tests run TEI's existing typed reference graph against the synthetic
// diagram fixture. This is a bounded fit/gap probe, NOT a Source Loader,
// accepted semantic model, diagram projector, or editing authorization.
import (
	"reflect"
	"testing"

	"tei/internal/modelref"
	"tei/internal/subject"
)

func modelRefFromSyntheticFixture(f fixture) (modelref.Graph, []modelref.Rule) {
	const scope = "synthetic.order-entry"
	g := modelref.Graph{Schema: "tei.modelref.v0"}
	for _, entity := range f.Entities {
		g.Nodes = append(g.Nodes, modelref.Node{
			Key: modelref.Key{
				Scope: scope,
				Subject: subject.Subject{
					Kind: "data",
					ID:   entity.ID,
				},
			},
			DisplayName: entity.Label,
			Evidence: []modelref.Evidence{{Authority: "proposal"}},
		})
	}
	rules := []modelref.Rule{
		{Relation: "contained-child", FromKind: "data", ToKind: "data"},
		{Relation: "reference", FromKind: "data", ToKind: "data"},
	}
	for _, relation := range f.Relations {
		g.Edges = append(g.Edges, modelref.Edge{
			From: modelref.Key{
				Scope: scope,
				Subject: subject.Subject{Kind: "data", ID: relation.From},
			},
			To: modelref.Ref{
				Scope: scope,
				Subject: subject.Subject{Kind: "data", ID: relation.To},
			},
			Relation: relation.Kind,
			Evidence: []modelref.Evidence{{Authority: "proposal"}},
		})
	}
	return g, rules
}

func TestExistingModelRefValidatesSyntheticDataGraphWithoutAcceptance(t *testing.T) {
	f, expected := readFixtures(t)
	g, rules := modelRefFromSyntheticFixture(f)
	if len(g.Nodes) != expected.ExpectedGraph.EntityCount {
		t.Fatalf("nodes: got %d", len(g.Nodes))
	}
	if len(g.Edges) != expected.ExpectedGraph.RelationCount {
		t.Fatalf("edges: got %d", len(g.Edges))
	}
	if diagnostics := modelref.Validate(g, rules); len(diagnostics) != 0 {
		t.Fatalf("typed reference graph diagnostics: %+v", diagnostics)
	}
	// Two distinct domain relation IDs share endpoints and relation kind.
	// The current modelref.Edge has no dedicated stable edge ID.
	var indices []int
	for i, relation := range f.Relations {
		if relation.ID == expected.ExpectedGraph.ParallelEdgePair[0] ||
			relation.ID == expected.ExpectedGraph.ParallelEdgePair[1] {
			indices = append(indices, i)
		}
	}
	if len(indices) != 2 {
		t.Fatalf("expected exactly two independent relation IDs, got %d", len(indices))
	}
	if !reflect.DeepEqual(g.Edges[indices[0]], g.Edges[indices[1]]) {
		t.Fatal("pressure fixture no longer probes same typed source/target/relation")
	}
	// Even a fully valid typed reference graph therefore cannot identify
	// which of those two source relations a diagram edge should write back.
	// Do not use array position as a stable source or editing identifier.
}

func TestExistingModelRefDiffDetectsParallelEdgeCountButNotWhichID(t *testing.T) {
	f, expected := readFixtures(t)
	before, rules := modelRefFromSyntheticFixture(f)
	after := before
	after.Edges = append([]modelref.Edge(nil), before.Edges...)
	dropped := ""
	for i, rel := range f.Relations {
		if rel.ID == expected.ExpectedGraph.ParallelEdgePair[1] {
			after.Edges = append(after.Edges[:i], after.Edges[i+1:]...)
			dropped = rel.ID
			break
		}
	}
	if dropped == "" {
		t.Fatal("parallel edge identity not present in fixture")
	}
	if diagnostics := modelref.Validate(after, rules); len(diagnostics) != 0 {
		t.Fatalf("after-edge-drop diagnostics: %+v", diagnostics)
	}
	changes := modelref.Diff(before, after)
	found := false
	for _, change := range changes {
		if change.Kind == "reference-changed" && change.Key.Subject.ID == "data.line" {
			found = true
		}
	}
	if !found {
		t.Fatalf("edge cardinality change not detected: %+v", changes)
	}
	// The identity of the removed domain relation is intentionally NOT a
	// modelref.Diff result: the old Edge representation did not carry it.
	// UI reverse editing requires separate qualified source-locator evidence.
}

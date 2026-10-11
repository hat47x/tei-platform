package modelref_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"tei/internal/modelref"
	"tei/internal/subject"
)

func key(kind, id string) modelref.Key {
	return modelref.Key{Scope: "orders-v1", Subject: subject.Subject{Kind: kind, ID: id}}
}
func node(kind, id, version string) modelref.Node {
	return modelref.Node{Key: key(kind, id), Version: version, DisplayName: id}
}
func ref(k modelref.Key, version string) modelref.Ref {
	return modelref.Ref{Scope: k.Scope, Subject: k.Subject, Version: version}
}
func rules() []modelref.Rule {
	return []modelref.Rule{
		{Relation: "reads", FromKind: "function", ToKind: "data"},
		{Relation: "renders", FromKind: "form", ToKind: "function", DisplayAffects: true},
		{Relation: "triggers", FromKind: "task", ToKind: "function", AcyclicGroup: "procedure"},
		{Relation: "precedes", FromKind: "task", ToKind: "task", AcyclicGroup: "procedure"},
		{Relation: "depends-on", FromKind: "data", ToKind: "data", AcyclicGroup: "data-dependencies"},
	}
}
func fixture() modelref.Graph {
	return modelref.Graph{Schema: "tei.modelref.v0", Nodes: []modelref.Node{
		node("data", "order-line", "1"), node("function", "submit-order", "2"),
		node("task", "confirm-order", "1"), node("form", "order-form", "1"),
	}, Edges: []modelref.Edge{
		{From: key("function", "submit-order"), To: ref(key("data", "order-line"), "1"), Relation: "reads"},
		{From: key("task", "confirm-order"), To: ref(key("function", "submit-order"), "2"), Relation: "triggers"},
		{From: key("form", "order-form"), To: ref(key("function", "submit-order"), "2"), Relation: "renders"},
	}}
}
func hasCode(ds []modelref.Diagnostic, code string) bool {
	for _, d := range ds {
		if d.Code == code {
			return true
		}
	}
	return false
}

func TestThreeViewsJSONAndStableIdentity(t *testing.T) {
	g := fixture()
	if ds := modelref.Validate(g, rules()); len(ds) != 0 {
		t.Fatalf("valid cross-view graph: %+v", ds)
	}
	b, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	var restored modelref.Graph
	if err := json.Unmarshal(b, &restored); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g, restored) {
		t.Fatalf("roundtrip mismatch: %#v", restored)
	}
	renamed := fixture()
	renamed.Nodes[0].DisplayName = "注文内訳"
	changes := modelref.Diff(g, renamed)
	if len(changes) != 1 || changes[0].Kind != "display-renamed" || changes[0].Key != key("data", "order-line") {
		t.Fatalf("display rename: %+v", changes)
	}
	if got := modelref.Impact(g, changes, rules()); len(got) != 0 {
		t.Fatalf("read edge must not imply display impact: %+v", got)
	}
}

func TestTypedResolutionNegativeCases(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*modelref.Graph)
		code   string
	}{
		{"missing", func(g *modelref.Graph) { g.Edges[0].To.Subject.ID = "unknown" }, "missing-reference"},
		{"wrong-kind", func(g *modelref.Graph) { g.Edges[0].To.Subject.Kind = "function" }, "wrong-kind"},
		{"wrong-version", func(g *modelref.Graph) { g.Edges[0].To.Version = "3" }, "version-mismatch"},
		{"invalid-edge", func(g *modelref.Graph) { g.Edges[0].Relation = "writes" }, "invalid-edge-type"},
		{"duplicate", func(g *modelref.Graph) { g.Nodes = append(g.Nodes, g.Nodes[0]) }, "duplicate-node-key"},
		{"unscoped-ambiguous", func(g *modelref.Graph) {
			other := g.Nodes[0]
			other.Key.Scope = "returns-v1"
			g.Nodes = append(g.Nodes, other)
			g.Edges[0].To.Scope = ""
		}, "ambiguous-reference"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := fixture()
			tc.mutate(&g)
			if !hasCode(modelref.Validate(g, rules()), tc.code) {
				t.Fatalf("want %q", tc.code)
			}
		})
	}
}

func TestTypedCyclePolicy(t *testing.T) {
	g := fixture()
	g.Nodes = append(g.Nodes, node("task", "pick-order", "1"))
	g.Edges = append(g.Edges, modelref.Edge{From: key("task", "confirm-order"), To: ref(key("task", "pick-order"), "1"), Relation: "precedes"}, modelref.Edge{From: key("task", "pick-order"), To: ref(key("task", "confirm-order"), "1"), Relation: "precedes"})
	if !hasCode(modelref.Validate(g, rules()), "cycle") {
		t.Fatal("procedure cycle not reported")
	}
	allow := rules()
	for i := range allow {
		allow[i].AcyclicGroup = ""
	}
	if hasCode(modelref.Validate(g, allow), "cycle") {
		t.Fatal("cycle was globally prohibited")
	}
}

func TestImpactPropagationAndExplicitIdentityMigration(t *testing.T) {
	before := fixture()
	after := fixture()
	after.Nodes[1].DisplayName = "送信処理"
	impacts := modelref.Impact(before, modelref.Diff(before, after), rules())
	if len(impacts) != 1 || impacts[0].Consumer != key("form", "order-form") {
		t.Fatalf("render-only display impact: %+v", impacts)
	}
	after = fixture()
	after.Nodes[0].Version = "2"
	impacts = modelref.Impact(before, modelref.Diff(before, after), rules())
	if len(impacts) != 3 {
		t.Fatalf("transitive impact: %+v", impacts)
	}
	after = fixture()
	after.Nodes[0].Key = key("data", "order-detail")
	if !hasCode(modelref.Validate(after, rules()), "missing-reference") {
		t.Fatal("dangling edge was hidden")
	}
	transition := modelref.Transition{Kind: "rekey", From: []modelref.Key{key("data", "order-line")}, To: []modelref.Key{key("data", "order-detail")}}
	if got := modelref.ValidateTransition(before, after, transition); got != "" {
		t.Fatal(got)
	}
	if got := modelref.ValidateTransition(before, after, modelref.Transition{Kind: "rekey", To: transition.To}); got != "invalid-cardinality" {
		t.Fatal(got)
	}
	changes := modelref.Diff(before, after)
	if len(changes) != 2 {
		t.Fatalf("rekey must be add/remove: %+v", changes)
	}
}

func TestSourceEvidenceCannotSilentlyBecomeIdentity(t *testing.T) {
	g := fixture()
	g.Nodes[0].Evidence = []modelref.Evidence{{Authority: "source-observation", SourceToken: "legacy:table#12", ArtifactFingerprint: "sha256:source"}}
	g.Nodes[1].Evidence = []modelref.Evidence{{Authority: "proposal", Context: "project:orders"}}
	if ds := modelref.Validate(g, rules()); len(ds) != 0 {
		t.Fatalf("legacy and AI remain non-accepted evidence: %+v", ds)
	}
	if g.Nodes[0].Key.Subject.ID != "order-line" || g.Nodes[1].Evidence[0].Authority != "proposal" {
		t.Fatal("identity/authority mutated")
	}
	g.Nodes[0].Evidence[0].ArtifactFingerprint = ""
	if !hasCode(modelref.Validate(g, rules()), "unbound-source-evidence") {
		t.Fatal("exact source binding missing")
	}
	g.Nodes[0].Evidence = []modelref.Evidence{{Authority: "derived", ArtifactFingerprint: "sha256:source"}}
	if !hasCode(modelref.Validate(g, rules()), "unbound-derivation-evidence") {
		t.Fatal("derivation rule binding missing")
	}
}

func TestSplitAndMergeRequireExplicitLineage(t *testing.T) {
	before := fixture()
	after := fixture()
	after.Nodes[0].Key = key("data", "order-header")
	after.Nodes = append(after.Nodes, node("data", "order-detail", "1"))
	split := modelref.Transition{Kind: "split", From: []modelref.Key{key("data", "order-line")}, To: []modelref.Key{key("data", "order-header"), key("data", "order-detail")}}
	if got := modelref.ValidateTransition(before, after, split); got != "" {
		t.Fatal(got)
	}
	if got := modelref.ValidateTransition(before, after, modelref.Transition{Kind: "merge", From: split.From, To: split.To}); got != "invalid-cardinality" {
		t.Fatal(got)
	}
}

func TestChangedReferenceAndEvidenceAreNotLost(t *testing.T) {
	before := fixture()
	after := fixture()
	after.Edges = after.Edges[1:]
	changes := modelref.Diff(before, after)
	if len(changes) != 1 || changes[0].Kind != "reference-changed" || changes[0].Key != key("function", "submit-order") {
		t.Fatalf("edge removal must be visible: %+v", changes)
	}
	after = fixture()
	after.Nodes[0].Evidence = []modelref.Evidence{{Authority: "proposal", Context: "orders changed"}}
	changes = modelref.Diff(before, after)
	if len(changes) != 1 || changes[0].Kind != "evidence-changed" {
		t.Fatalf("evidence change needs review: %+v", changes)
	}
}

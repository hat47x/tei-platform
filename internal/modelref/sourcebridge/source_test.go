package sourcebridge_test

import (
	"tei/internal/modelref"
	"tei/internal/modelref/sourcebridge"
	"tei/internal/sourceindex"
	"tei/internal/subject"
	"testing"
)

func mkKey(scope, id string) modelref.Key {
	return modelref.Key{Scope: scope, Subject: subject.Subject{Kind: "data", ID: id}}
}
func TestSourceNavigationAndDrift(t *testing.T) {
	content := []byte("data order-line")
	a, err := sourceindex.NewArtifact("model/order.tei", content)
	if err != nil {
		t.Fatal(err)
	}
	rng, err := sourceindex.RangeFromOffsets(content, 5, 15)
	if err != nil {
		t.Fatal(err)
	}
	k := mkKey("orders-v1", "order-line")
	g := modelref.Graph{Schema: "tei.modelref.v0", Nodes: []modelref.Node{{Key: k}}}
	idx := sourceindex.Index{Artifacts: []sourceindex.Artifact{a}, References: []sourceindex.Reference{{Subject: k.Subject, ArtifactPath: a.Path, Range: rng, Role: sourceindex.RoleDefinition}}}
	read := func(path string) ([]byte, error) { return content, nil }
	got := sourcebridge.Locate(g, idx, k, read)
	if got.Status != "located" || len(got.Artifacts) != 1 || len(got.References) != 1 {
		t.Fatalf("lost exact anchor: %+v", got)
	}
	readDrift := func(path string) ([]byte, error) { return []byte("data order-detail"), nil }
	if got := sourcebridge.Locate(g, idx, k, readDrift); got.Status != "unresolved" || got.Reason != "source-index-invalid" {
		t.Fatalf("accepted stale anchor: %+v", got)
	}
	idx.References[0].Role = sourceindex.RoleDerived
	if got := sourcebridge.Locate(g, idx, k, read); got.Reason != "derived-only-source" {
		t.Fatalf("derived-only anchor promoted: %+v", got)
	}
}
func TestSameSubjectInDifferentScopesMustNotAlias(t *testing.T) {
	content := []byte("order-line")
	a, _ := sourceindex.NewArtifact("model/a.tei", content)
	rng, _ := sourceindex.RangeFromOffsets(content, 0, len(content))
	k := mkKey("orders-v1", "order-line")
	other := mkKey("returns-v1", "order-line")
	g := modelref.Graph{Schema: "tei.modelref.v0", Nodes: []modelref.Node{{Key: k}, {Key: other}}}
	idx := sourceindex.Index{Artifacts: []sourceindex.Artifact{a}, References: []sourceindex.Reference{{Subject: k.Subject, ArtifactPath: a.Path, Range: rng, Role: sourceindex.RoleDefinition}}}
	read := func(string) ([]byte, error) { return content, nil }
	if got := sourcebridge.Locate(g, idx, k, read); got.Reason != "subject-scope-ambiguous" {
		t.Fatalf("scope alias accepted: %+v", got)
	}
	g.Nodes = g.Nodes[:1]
	k2 := mkKey("orders-v1", "customer")
	if got := sourcebridge.Locate(g, idx, k2, read); got.Reason != "subject-not-unique-in-graph" {
		t.Fatalf("invented subject: %+v", got)
	}
	idx.References = nil
	if got := sourcebridge.Locate(g, idx, k, read); got.Reason != "source-anchor-unavailable" {
		t.Fatalf("invented source anchor: %+v", got)
	}
}

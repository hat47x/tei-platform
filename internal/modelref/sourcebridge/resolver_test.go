package sourcebridge_test

import (
	"fmt"
	"testing"

	"tei/internal/modelref"
	"tei/internal/modelref/sourcebridge"
	"tei/internal/sourceindex"
	"tei/internal/subject"
)

func TestResolverValidatesOneSnapshotForManyNavigations(t *testing.T) {
	content := []byte("data order-line")
	a, err := sourceindex.NewArtifact("semantic/sales/order.tei", content)
	if err != nil {
		t.Fatal(err)
	}
	rng, err := sourceindex.RangeFromOffsets(content, 5, len(content))
	if err != nil {
		t.Fatal(err)
	}
	nodes := make([]modelref.Node, 0, 1000)
	for i := 0; i < 1000; i++ {
		nodes = append(nodes, modelref.Node{Key: modelref.Key{Scope: "orders", Subject: subject.Subject{Kind: "semantic-data", ID: fmt.Sprintf("order-%04d", i)}}})
	}
	const focus = 420
	key := nodes[focus].Key
	idx := sourceindex.Index{Artifacts: []sourceindex.Artifact{a}, References: []sourceindex.Reference{
		{Subject: key.Subject, ArtifactPath: a.Path, Range: rng, Role: sourceindex.RoleDefinition},
	}}
	reads := 0
	read := func(string) ([]byte, error) { reads++; return content, nil }
	resolver, err := sourcebridge.NewResolver(modelref.Graph{Schema: "tei.modelref.v0", Nodes: nodes}, idx, read)
	if err != nil {
		t.Fatal(err)
	}
	validatedReads := reads
	if validatedReads == 0 {
		t.Fatal("Source Index was never validated")
	}
	for i := 0; i < len(nodes); i++ {
		navigation := resolver.Locate(nodes[i].Key)
		if i == focus && navigation.Status != "located" {
			t.Fatalf("lost source at %d: %+v", i, navigation)
		}
		if i != focus && navigation.Status != "unresolved" {
			t.Fatalf("invented source at %d: %+v", i, navigation)
		}
	}
	if reads != validatedReads {
		t.Fatalf("repeated Source Index IO on navigation: before=%d after=%d", validatedReads, reads)
	}
}

func TestResolverRejectsAmbiguousDefinitionAnchors(t *testing.T) {
	content := []byte("order-line")
	a, _ := sourceindex.NewArtifact("semantic/sales/order.tei", content)
	rng, _ := sourceindex.RangeFromOffsets(content, 0, len(content))
	key := modelref.Key{Scope: "orders", Subject: subject.Subject{Kind: "semantic-data", ID: "order-line"}}
	index := sourceindex.Index{Artifacts: []sourceindex.Artifact{a}, References: []sourceindex.Reference{
		{Subject: key.Subject, ArtifactPath: a.Path, Range: rng, Role: sourceindex.RoleDefinition},
		{Subject: key.Subject, ArtifactPath: a.Path, Range: rng, Role: sourceindex.RoleDefinition},
	}}
	read := func(string) ([]byte, error) { return content, nil }
	resolver, err := sourcebridge.NewResolver(modelref.Graph{Schema: "tei.modelref.v0", Nodes: []modelref.Node{{Key: key}}}, index, read)
	if err != nil {
		t.Fatal(err)
	}
	if got := resolver.Locate(key); got.Reason != "ambiguous-source-anchor" || got.Status != "unresolved" {
		t.Fatalf("ambiguous source was promoted: %+v", got)
	}
	if _, err := sourcebridge.NewResolver(modelref.Graph{Schema: "tei.modelref.v0"}, index, func(string) ([]byte, error) { return []byte("stale"), nil }); err == nil {
		t.Fatal("stale Source Index qualified as a snapshot")
	}
	if _, err := sourcebridge.NewResolver(modelref.Graph{Schema: "tei.modelref.v9"}, index, read); err == nil {
		t.Fatal("unsupported Graph schema was accepted")
	}
}

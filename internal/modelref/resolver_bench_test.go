package modelref

import (
	"fmt"
	"testing"

	"tei/internal/subject"
)

// resolverBenchFixture builds a graph of n nodes and n references that resolve
// to them, which is the shape a plan over a real Workspace has: many references,
// one snapshot.
func resolverBenchFixture(n int) ([]Node, []Ref) {
	nodes := make([]Node, 0, n)
	refs := make([]Ref, 0, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("subject.%06d", i)
		nodes = append(nodes, Node{Key: Key{Scope: "c", Subject: subject.Subject{Kind: "semantic-data", ID: id}}})
		refs = append(refs, Ref{Scope: "c",
			Subject: subject.Subject{Kind: "semantic-data", ID: id}})
	}
	return nodes, refs
}

// BenchmarkResolvePerReference measures calling Resolve for every reference.
// That entry point re-indexes all nodes on each call, so the cost is quadratic
// in the graph size for a caller that resolves many references.
func BenchmarkResolvePerReference(b *testing.B) {
	for _, n := range []int{200, 2000} {
		nodes, refs := resolverBenchFixture(n)
		b.Run(fmt.Sprintf("nodes=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				for _, ref := range refs {
					if _, issue := Resolve(nodes, ref); issue != "" {
						b.Fatalf("issue %q", issue)
					}
				}
			}
		})
	}
}

// BenchmarkResolverReused measures one index for the same references. This is
// what internal/realization/derivationbinding does now.
func BenchmarkResolverReused(b *testing.B) {
	for _, n := range []int{200, 2000} {
		nodes, refs := resolverBenchFixture(n)
		b.Run(fmt.Sprintf("nodes=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				resolver := NewResolver(nodes)
				for _, ref := range refs {
					if _, issue := resolver.Resolve(ref); issue != "" {
						b.Fatalf("issue %q", issue)
					}
				}
			}
		})
	}
}

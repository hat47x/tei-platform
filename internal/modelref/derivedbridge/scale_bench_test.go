package derivedbridge

import (
	"fmt"
	"testing"

	"tei/internal/core"
	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/subject"
)

// scaleDerived builds an index of the given size: n namespaces, each holding one
// Data with one Attribute, and a dependency from each Data to the next. It is
// deliberately the shape a Workspace grows in — many small elements, and edge
// count proportional to element count — so a non-linear stage shows up as the
// size doubles.
func scaleDerived(n int) index.Derived {
	d := index.Derived{}
	d.SemanticElements = make([]index.SemanticElementEntry, 0, 2*n)
	d.SemanticDependencies = make([]index.SemanticDependency, 0, n)
	for i := 0; i < n; i++ {
		ns := core.CanonicalID(fmt.Sprintf("ns%05d", i))
		data := core.CanonicalID(fmt.Sprintf("ns%05d.data", i))
		attr := core.CanonicalID(fmt.Sprintf("ns%05d.data.value", i))
		d.SemanticElements = append(d.SemanticElements,
			index.SemanticElementEntry{Kind: "namespace", ID: ns},
			index.SemanticElementEntry{Kind: "data", ID: data},
			index.SemanticElementEntry{Kind: "attribute", ID: attr},
		)
		if i > 0 {
			prev := core.CanonicalID(fmt.Sprintf("ns%05d.data", i-1))
			d.SemanticDependencies = append(d.SemanticDependencies,
				index.SemanticDependency{FromKind: "semantic-data", From: data,
					Relation: "underlying", ToKind: "semantic-data", To: prev})
		}
	}
	return d
}

// BenchmarkFromIndexScale measures projection cost against size. #1177 needs to
// know whether the read path grows linearly before anyone optimises it, and the
// reference Workspace is far too small to show it (measured at 160ms per command,
// dominated by process start).
func BenchmarkFromIndexScale(b *testing.B) {
	for _, n := range []int{100, 1000, 5000} {
		d := scaleDerived(n)
		b.Run(fmt.Sprintf("elements=%d", 3*n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				p := FromIndex(d, "c")
				if len(p.Graph.Nodes) == 0 {
					b.Fatal("empty graph")
				}
			}
		})
	}
}

// BenchmarkTraceResolutionScale measures resolving one row against graph size.
// Resolution is what `trace -mode full` does per row, so a quadratic lookup here
// would make paging cost grow with the workspace.
func BenchmarkTraceResolutionScale(b *testing.B) {
	for _, n := range []int{100, 1000, 5000} {
		p := FromIndex(scaleDerived(n), "c")
		target := modelref.Key{Scope: "c", Subject: subject.Subject{
			Kind: "semantic-data", ID: fmt.Sprintf("ns%05d.data", n/2)}}
		b.Run(fmt.Sprintf("elements=%d", 3*n), func(b *testing.B) {
			// The trace path builds one resolver per snapshot and reuses it, which
			// is what a large graph must do to stay linear. Building it is not part
			// of resolving one row, so the timer starts after it.
			resolver := modelref.NewResolver(p.Graph.Nodes)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key, issue := resolver.Resolve(modelref.Ref{Scope: "c", Subject: subject.Subject{Kind: target.Subject.Kind, ID: target.Subject.ID}})
				if issue != "" {
					b.Fatalf("issue %q", issue)
				}
				_ = key
			}
		})
	}
}

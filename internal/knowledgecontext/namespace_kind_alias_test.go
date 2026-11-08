package knowledgecontext_test

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/knowledgecontext"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/subject"
)

// TestBuildResolvesANamespaceUnderBothKindNames pins the namespace kind alias.
// The catalog, inspect and trace call a namespace "semantic"; knowledge-context
// used "semantic-namespace". Without the alias, a caller that reads
// `entries[].subject.kind` from `model catalog` and passes it to
// `knowledge-context` gets a context whose `semanticElement` is silently absent
// while `ok` stays true (#3242 follow-up).
func TestBuildResolvesANamespaceUnderBothKindNames(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{ID: "sales"}}}
	sources := sourceindex.Index{}
	for _, kind := range []string{"semantic", "semantic-namespace"} {
		ctx := knowledgecontext.Build(world, sources, subject.Subject{Kind: kind, ID: "sales"})
		if ctx.SemanticElement == nil {
			t.Fatalf("kind %q did not resolve the namespace element: %+v", kind, ctx.Unresolved)
		}
		if ctx.SemanticElement.Kind != "namespace" || string(ctx.SemanticElement.ID) != "sales" {
			t.Fatalf("kind %q resolved %+v", kind, ctx.SemanticElement)
		}
		if len(ctx.Unresolved) != 1 || ctx.Unresolved[0] != "source-reference-unavailable" {
			t.Fatalf("kind %q unresolved=%v, want only the source gap", kind, ctx.Unresolved)
		}
	}
}

// A kind the context does not model stays unresolved rather than being guessed.
func TestBuildLeavesAnUnknownKindUnresolved(t *testing.T) {
	world := ir.World{}
	ctx := knowledgecontext.Build(world, sourceindex.Index{}, subject.Subject{Kind: "not-a-kind", ID: "sales"})
	if ctx.SemanticElement != nil {
		t.Fatalf("unknown kind resolved an element: %+v", ctx.SemanticElement)
	}
	if ctx.Subject.Kind != "not-a-kind" {
		t.Fatalf("subject was rewritten: %+v", ctx.Subject)
	}
}

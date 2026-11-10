package modelref

import (
	"fmt"
	"math/rand"
	"testing"

	"tei/internal/subject"
)

// resolverFixture builds a graph with duplicate IDs across kinds and scopes, so
// the ambiguous and wrong-kind paths are exercised as well as the simple hit.
func resolverFixture() []Node {
	nodes := make([]Node, 0, 40)
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("subject.%02d", i)
		nodes = append(nodes,
			Node{Key: Key{Scope: "c", Subject: subjectOf("semantic-data", id)}},
			Node{Key: Key{Scope: "c", Subject: subjectOf("semantic-attribute", id)}},
			Node{Key: Key{Scope: "s", Subject: subjectOf("semantic-data", id)}},
		)
	}
	// One ID that exists only in scope c, to distinguish scope filtering.
	nodes = append(nodes, Node{Key: Key{Scope: "c", Subject: subjectOf("query", "only.c")}})
	return nodes
}

// TestResolverAgreesWithResolve pins that the reusable resolver answers exactly
// what the one-shot call answers, including the failure issues. Resolve used to
// be the only entry point; it is now a wrapper, so the two must not drift.
func TestResolverAgreesWithResolve(t *testing.T) {
	nodes := resolverFixture()
	resolver := NewResolver(nodes)

	refs := []Ref{
		{Scope: "c", Subject: subjectOf("semantic-data", "subject.00")},
		{Scope: "s", Subject: subjectOf("semantic-data", "subject.00")},
		{Scope: "c", Subject: subjectOf("query", "only.c")},
		// present in two scopes: an omitted scope is ambiguous, not "first wins".
		{Subject: subjectOf("semantic-data", "subject.01")},
		// kind mismatch
		{Scope: "c", Subject: subjectOf("application", "subject.02")},
		// absent
		{Scope: "c", Subject: subjectOf("semantic-data", "absent")},
		// version constraint
		{Scope: "c", Subject: subjectOf("semantic-data", "subject.03"), Version: "2"},
	}
	for _, ref := range refs {
		wantKey, wantIssue := Resolve(nodes, ref)
		gotKey, gotIssue := resolver.Resolve(ref)
		if gotKey != wantKey || gotIssue != wantIssue {
			t.Fatalf("ref %+v: resolver=(%v,%q) resolve=(%v,%q)",
				ref, gotKey, gotIssue, wantKey, wantIssue)
		}
	}
}

// A resolver built from a snapshot must stay correct when asked the same
// question many times, and must not depend on the order of the questions.
func TestResolverIsRepeatableAndOrderIndependent(t *testing.T) {
	nodes := resolverFixture()
	random := rand.New(rand.NewSource(20261111))

	refs := make([]Ref, 0, 64)
	for i := 0; i < 64; i++ {
		id := fmt.Sprintf("subject.%02d", random.Intn(12))
		kinds := []string{"semantic-data", "semantic-attribute", "query"}
		scopes := []string{"c", "s", ""}
		refs = append(refs, Ref{
			Scope:   scopes[random.Intn(len(scopes))],
			Subject: subjectOf(kinds[random.Intn(len(kinds))], id),
		})
	}

	first := make([]string, len(refs))
	resolver := NewResolver(nodes)
	for i, ref := range refs {
		key, issue := resolver.Resolve(ref)
		first[i] = fmt.Sprintf("%v/%s", key, issue)
	}
	for i := len(refs) - 1; i >= 0; i-- {
		key, issue := resolver.Resolve(refs[i])
		if got := fmt.Sprintf("%v/%s", key, issue); got != first[i] {
			t.Fatalf("ref %d answered %q in reverse order, %q forwards", i, got, first[i])
		}
	}
}

// subjectOf builds the subject value a reference carries.
func subjectOf(kind, id string) subject.Subject {
	return subject.Subject{Kind: kind, ID: id}
}

package modelref_test

import (
	"tei/internal/modelref"
	"testing"
)

func TestNamespaceMoveKeepsLogicalSubjectButRequiresExplicitRebinding(t *testing.T) {
	before, after, transition, move := rekeyFixture()
	after.Nodes[0].Key = key("data", "order-line")
	after.Nodes[0].Key.Scope = "another-workspace"
	after.Edges[0].To = ref(after.Nodes[0].Key, "1")
	transition.To = []modelref.Key{after.Nodes[0].Key}
	move.New = ref(after.Nodes[0].Key, "1")
	if result := modelref.ReviewMigration(before, after, transition, []modelref.EdgeRetarget{move}, rules(), "revision:base"); !result.Consistent {
		t.Fatalf("explicit scope move: %+v", result.Diagnostics)
	}
	if result := modelref.ReviewMigration(before, after, transition, nil, rules(), "revision:base"); result.Consistent || !hasCode(result.Diagnostics, "unreviewed-incoming-reference") {
		t.Fatalf("scope move inferred: %+v", result.Diagnostics)
	}
}
func TestMergeRequiresSeparateConsumerRebinds(t *testing.T) {
	before := modelref.Graph{Schema: "tei.modelref.v0", Nodes: []modelref.Node{node("data", "a", "1"), node("data", "b", "1"), node("function", "left", "1"), node("function", "right", "1")}, Edges: []modelref.Edge{
		{From: key("function", "left"), To: ref(key("data", "a"), "1"), Relation: "reads"},
		{From: key("function", "right"), To: ref(key("data", "b"), "1"), Relation: "reads"},
	}}
	after := modelref.Graph{Schema: before.Schema, Nodes: []modelref.Node{node("data", "merged", "2"), node("function", "left", "1"), node("function", "right", "1")}, Edges: []modelref.Edge{
		{From: key("function", "left"), To: ref(key("data", "merged"), "2"), Relation: "reads"},
		{From: key("function", "right"), To: ref(key("data", "merged"), "2"), Relation: "reads"},
	}}
	tr := modelref.Transition{Kind: "merge", From: []modelref.Key{key("data", "a"), key("data", "b")}, To: []modelref.Key{key("data", "merged")}}
	moves := []modelref.EdgeRetarget{
		{From: key("function", "left"), Relation: "reads", Old: ref(key("data", "a"), "1"), New: ref(key("data", "merged"), "2")},
		{From: key("function", "right"), Relation: "reads", Old: ref(key("data", "b"), "1"), New: ref(key("data", "merged"), "2")},
	}
	if out := modelref.ReviewMigration(before, after, tr, moves, rules(), "rev"); !out.Consistent {
		t.Fatalf("explicit merge: %+v", out.Diagnostics)
	}
	if out := modelref.ReviewMigration(before, after, tr, moves[:1], rules(), "rev"); out.Consistent || !hasCode(out.Diagnostics, "unreviewed-incoming-reference") {
		t.Fatalf("silent loss of second consumer: %+v", out.Diagnostics)
	}
}
func TestMigrationRejectsUnrelatedNodeChange(t *testing.T) {
	before, after, tr, move := rekeyFixture()
	after.Nodes[1].DisplayName = "changed-other-function"
	out := modelref.ReviewMigration(before, after, tr, []modelref.EdgeRetarget{move}, rules(), "rev")
	if out.Consistent || !hasCode(out.Diagnostics, "unrelated-node-change") {
		t.Fatalf("unrelated edit was approved by narrow review: %+v", out.Diagnostics)
	}
}

func TestMigrationCannotSilentlyReplaceReferenceEvidence(t *testing.T) {
	before, after, transition, move := rekeyFixture()
	before.Edges[0].Evidence = []modelref.Evidence{{Authority: "source-observation", SourceToken: "old", ArtifactFingerprint: "sha256:source"}}
	after.Edges[0].Evidence = []modelref.Evidence{{Authority: "derived", ArtifactFingerprint: "sha256:other", RuleFingerprint: "sha256:rule"}}
	r := modelref.ReviewMigration(before, after, transition, []modelref.EdgeRetarget{move}, rules(), "revision")
	if r.Consistent || !hasCode(r.Diagnostics, "retarget-evidence-changed") {
		t.Fatalf("evidence substitution concealed by retarget: %+v", r.Diagnostics)
	}
	after.Edges[0].Evidence = append([]modelref.Evidence(nil), before.Edges[0].Evidence...)
	before.Edges[1].Evidence = []modelref.Evidence{{Authority: "proposal", Context: "before"}}
	after.Edges[1].Evidence = []modelref.Evidence{{Authority: "proposal", Context: "after"}}
	r = modelref.ReviewMigration(before, after, transition, []modelref.EdgeRetarget{move}, rules(), "revision")
	if r.Consistent || !hasCode(r.Diagnostics, "unreviewed-reference-evidence") {
		t.Fatalf("unrelated edge proof substitution concealed: %+v", r.Diagnostics)
	}
}

// All evidence for the same reference signature must be accounted for. Neither
// the last edge in the slice nor ordering of edge declarations is authoritative.
func TestMigrationDuplicateUnrelatedReferenceEvidenceIsNotHidden(t *testing.T) {
	before, after, transition, move := rekeyFixture()
	before.Edges[1].Evidence = []modelref.Evidence{{Authority: "proposal", Context: "alpha"}}
	after.Edges[1].Evidence = []modelref.Evidence{{Authority: "proposal", Context: "alpha"}}
	duplicate := before.Edges[1]
	duplicate.Evidence = []modelref.Evidence{{Authority: "proposal", Context: "beta"}}
	before.Edges = append(before.Edges, duplicate)
	after.Edges = append(after.Edges, duplicate)

	moves := []modelref.EdgeRetarget{move}
	if out := modelref.ReviewMigration(before, after, transition, moves, rules(), "revision"); !out.Consistent {
		t.Fatalf("unchanged duplicate references cannot block migration: %+v", out.Diagnostics)
	}
	last := len(after.Edges) - 1
	after.Edges[1], after.Edges[last] = after.Edges[last], after.Edges[1]
	if out := modelref.ReviewMigration(before, after, transition, moves, rules(), "revision"); !out.Consistent {
		t.Fatalf("permuting duplicate edges changes evidence meaning: %+v", out.Diagnostics)
	}
	after.Edges[1], after.Edges[last] = after.Edges[last], after.Edges[1]

	// Mutate the first occurrence, preserving the last matching edge. The
	// former map[edgeID]string implementation compared only the last one.
	after.Edges[1].Evidence = []modelref.Evidence{{Authority: "proposal", Context: "changed"}}
	out := modelref.ReviewMigration(before, after, transition, moves, rules(), "revision")
	if out.Consistent || !hasCode(out.Diagnostics, "unreviewed-reference-evidence") {
		t.Fatalf("modified non-final duplicate was approved: %+v", out.Diagnostics)
	}
}

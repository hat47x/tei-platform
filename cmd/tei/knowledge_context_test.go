package main

import (
	"path/filepath"
	"testing"

	"tei/internal/knowledgecontext"
)

func TestBuildKnowledgeContextReturnsScopedApplicationPatternContext(t *testing.T) {
	root := testRepositoryRoot(t)
	got, err := buildKnowledgeContext(
		filepath.Join(root, "reference", "workspace"),
		"application-pattern",
		"order_entry.customer_reference",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ApplicationPattern == nil {
		t.Fatalf("application pattern detail is missing: %+v", got)
	}
	if got.ApplicationPattern.Application != "order_entry" ||
		got.ApplicationPattern.Pattern != "order_entry.customer_reference" ||
		got.ApplicationPattern.Kind != "reference" {
		t.Fatalf("application pattern detail drifted: %+v", got.ApplicationPattern)
	}
	if len(got.References) == 0 || len(got.Artifacts) == 0 {
		t.Fatalf("source trace is missing: references=%+v artifacts=%+v", got.References, got.Artifacts)
	}
	if len(got.Unresolved) != 0 {
		t.Fatalf("known application pattern has unresolved context: %+v", got.Unresolved)
	}

	for _, want := range []struct {
		direction string
		relation  string
		kind      string
		id        string
	}{
		{direction: "incoming", relation: "owns-pattern", kind: "application", id: "order_entry"},
		{direction: "outgoing", relation: "relation", kind: "semantic-relation", id: "sales.order.customer"},
		{direction: "outgoing", relation: "contract", kind: "database-contract", id: "master_data.customer"},
		{direction: "outgoing", relation: "contract-query", kind: "database-contract-query", id: "master_data.customer/find_by_id"},
	} {
		if !hasKnowledgeContextEdge(got, want.direction, want.relation, want.kind, want.id) {
			t.Fatalf("knowledge context edge is missing: %+v context=%+v", want, got)
		}
	}

	for _, limit := range []string{
		"derived-context-is-not-canonical-authority",
		"dependency-edge-is-not-semantic-applicability",
		"dependency-edge-is-not-write-or-runtime-authority",
	} {
		if !containsKnowledgeContextString(got.AuthorityLimits, limit) {
			t.Fatalf("authority limit %q is missing: %+v", limit, got.AuthorityLimits)
		}
	}
}

func TestBuildKnowledgeContextRequiresWorkspaceAndSubject(t *testing.T) {
	if _, err := buildKnowledgeContext("", "application-pattern", "order_entry.customer_reference"); err == nil {
		t.Fatal("empty workspace was accepted")
	}
	if _, err := buildKnowledgeContext("unused", "", "order_entry.customer_reference"); err == nil {
		t.Fatal("empty subject kind was accepted")
	}
	if _, err := buildKnowledgeContext("unused", "application-pattern", ""); err == nil {
		t.Fatal("empty subject id was accepted")
	}
}

func hasKnowledgeContextEdge(ctx knowledgecontext.Context, direction, relation, otherKind, otherID string) bool {
	edges := ctx.Outgoing
	if direction == "incoming" {
		edges = ctx.Incoming
	}
	for _, edge := range edges {
		if edge.Relation != relation {
			continue
		}
		if direction == "incoming" && edge.From.Kind == otherKind && edge.From.ID == otherID {
			return true
		}
		if direction == "outgoing" && edge.To.Kind == otherKind && edge.To.ID == otherID {
			return true
		}
	}
	return false
}

func containsKnowledgeContextString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

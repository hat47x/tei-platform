package main

import (
	"path/filepath"
	"testing"

	"tei/internal/modelref"
	"tei/internal/subject"
)

func workspaceKey(kind, id string) modelref.Key {
	return modelref.Key{Scope: "order-entry-reference", Subject: subject.Subject{Kind: kind, ID: id}}
}

func TestRealWorkspaceImpactDoesNotInventEvidenceReuse(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	relation := workspaceKey("semantic-relation", "sales.order.customer")
	pattern := workspaceKey("application-pattern", "order_entry.customer_reference")
	application := workspaceKey("application", "order_entry")
	customer := workspaceKey("semantic-data", "customer.customer")
	for _, tc := range []struct {
		name     string
		changed  modelref.Key
		kind     string
		consumer modelref.Key
		action   string
	}{
		{"relation-affects-pattern", relation, "version-changed", pattern, "recheck"},
		{"customer-affects-application-transitively", customer, "version-changed", application, "recheck"},
		{"display-rename-without-verified-receipt", relation, "display-renamed", pattern, "unresolved"},
		{"directly-changed-subject", relation, "display-renamed", relation, "recheck"},
		{"missing-consumer", relation, "version-changed", workspaceKey("application-pattern", "not-declared"), "unresolved"},
		{"unsupported-change-kind", relation, "future-kind", pattern, "unresolved"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report, err := reviewWorkspaceChange(root, "order-entry-reference", tc.changed, tc.kind, tc.consumer)
			if err != nil {
				t.Fatal(err)
			}
			if report.Decision.Action != tc.action || report.Decision.ExecutionAllowed ||
				report.Decision.ReceiptRef != "" || len(report.Decision.ReasonCodes) == 0 {
				t.Fatalf("unsafe decision: %+v", report)
			}
			if report.GraphNodes == 0 || report.GraphEdges == 0 {
				t.Fatal("not backed by real Workspace typed index")
			}
			// Today, the real Workspace contains Database Contract
			// projection gaps. They must remain visible for inspection,
			// but resolving them later must not break this regression.
			if tc.name == "display-rename-without-verified-receipt" {
				wantReason := "no-bound-receipt-or-proven-dependency-closure"
				if len(report.ProjectionUnresolved) > 0 {
					wantReason = "workspace-projection-unresolved"
				}
				if report.Decision.ReasonCodes[0] != wantReason {
					t.Fatalf("negative impact bypassed the evidence boundary: %+v", report.Decision)
				}
			}
		})
	}
}

func TestWorkspaceAdapterFailClosedBeforeAndAfterGraphReview(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	relation := workspaceKey("semantic-relation", "sales.order.customer")
	pattern := workspaceKey("application-pattern", "order_entry.customer_reference")
	if _, err := reviewWorkspaceChange(root, "", relation, "version-changed", pattern); err == nil {
		t.Fatal("accepted absent scope")
	}
	if _, err := reviewWorkspaceChange(root, "other-scope", relation, "version-changed", pattern); err == nil {
		t.Fatal("accepted mismatched scope")
	}
	r, err := reviewWorkspaceChange(root, "order-entry-reference", relation, "unknown-change", pattern)
	if err != nil {
		t.Fatal(err)
	}
	if r.Impact.GraphValid || r.Decision.Action != "unresolved" || len(r.Impact.Diagnostics) == 0 {
		t.Fatalf("invalid change promoted: %+v", r)
	}
	r.ProjectionUnresolved = []string{"unknown-native-effect"}
	r.Impact.GraphValid = true
	d := classifyWorkspaceImpact(r, modelref.Graph{Nodes: []modelref.Node{{Key: pattern}}})
	if d.Action != "unresolved" || d.ReasonCodes[0] != "workspace-projection-unresolved" {
		t.Fatalf("opaque dependency promoted: %+v", d)
	}
	// A positive impact is conservative even when another, unrelated
	// projection gap exists; it does not assert complete closure.
	r.Impact.Affected = []modelref.Affected{{Changed: relation, Consumer: pattern, Reason: "version-changed"}}
	r.Changed.Kind = "version-changed"
	d = classifyWorkspaceImpact(r, modelref.Graph{Nodes: []modelref.Node{{Key: pattern}}})
	if d.Action != "recheck" || d.ExecutionAllowed || d.ReceiptRef != "" {
		t.Fatalf("positive observed typed impact was obscured or promoted: %+v", d)
	}
}

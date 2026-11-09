package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/modelref"
	"tei/internal/sourceindex"
)

func TestTwoRealWorkspaceSnapshotsDetectSQLChangeOmittedByTypedGraph(t *testing.T) {
	original := filepath.Join("..", "..", "reference", "workspace")
	beforeRoot, afterRoot := t.TempDir(), t.TempDir()
	for _, root := range []string{beforeRoot, afterRoot} {
		if err := os.CopyFS(root, os.DirFS(original)); err != nil {
			t.Fatalf("copy real Workspace: %v", err)
		}
	}
	const relative = "applications/order-entry/queries/recent_orders.sql"
	sqlFile := filepath.Join(afterRoot, filepath.FromSlash(relative))
	body, err := os.ReadFile(sqlFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "ORDER BY o.ordered_at DESC") {
		t.Fatal("real reference SQL fixture changed")
	}
	replacement := strings.Replace(string(body), "ORDER BY o.ordered_at DESC", "ORDER BY o.ordered_at ASC", 1)
	if err := os.WriteFile(sqlFile, []byte(replacement), 0600); err != nil {
		t.Fatal(err)
	}
	consumer := workspaceKey("application", "order_entry")
	report, err := reviewWorkspaceDiff(beforeRoot, afterRoot, "order-entry-reference", consumer)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Impact.GraphValid || len(report.Impact.Changes) != 0 {
		t.Fatalf("test requires a valid unchanged typed projection: %+v", report.Impact)
	}
	if len(report.SourceChanges) != 1 || report.SourceChanges[0].Path != relative ||
		report.SourceChanges[0].Kind != "content-changed" || report.SourceChanges[0].BeforeFingerprint == report.SourceChanges[0].AfterFingerprint ||
		report.BeforeSourceRevision == report.AfterSourceRevision {
		t.Fatalf("changed SQL bytes were not captured by existing Source Index: %+v", report)
	}
	if report.Decision.Action != "unresolved" ||
		report.Decision.ReasonCodes[0] != "source-changed-without-typed-graph-diff" ||
		report.Decision.ExecutionAllowed || report.Decision.ReceiptRef != "" {
		t.Fatalf("typed-graph no-op incorrectly certified source change: %+v", report.Decision)
	}
}

func TestSameRealWorkspaceSnapshotIsNotCertifiedForEvidenceReuse(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	consumer := workspaceKey("application", "order_entry")
	report, err := reviewWorkspaceDiff(root, root, "order-entry-reference", consumer)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.SourceChanges) != 0 || report.BeforeSourceRevision != report.AfterSourceRevision ||
		report.Decision.Action != "unresolved" || report.Decision.ExecutionAllowed {
		t.Fatalf("unchanged Workspace was falsely accepted as sufficient receipt evidence: %+v", report)
	}
}

func TestWorkspaceDiffRejectsInvalidInputsOrAfterWorkspace(t *testing.T) {
	original := filepath.Join("..", "..", "reference", "workspace")
	consumer := workspaceKey("application", "order_entry")
	if _, err := reviewWorkspaceDiff(original, "", "order-entry-reference", consumer); err == nil {
		t.Fatal("accepted absent after root")
	}
	if _, err := reviewWorkspaceDiff(original, original, "wrong-scope", consumer); err == nil {
		t.Fatal("accepted mismatched consumer scope")
	}
	afterRoot := t.TempDir()
	if err := os.CopyFS(afterRoot, os.DirFS(original)); err != nil {
		t.Fatal(err)
	}
	semanticPath := filepath.Join(afterRoot, "semantic", "sales", "order.tei")
	if err := os.WriteFile(semanticPath, []byte("not a valid TEI semantic declaration"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := reviewWorkspaceDiff(original, afterRoot, "order-entry-reference", consumer); err == nil {
		t.Fatal("malformed after Workspace was accepted")
	}
}

func TestSourceArtifactDeltaIsSortedAndKeepsAddRemove(t *testing.T) {
	before := sourceindex.Index{Artifacts: []sourceindex.Artifact{
		{Path: "z.tei", Fingerprint: "sha256:unchanged"},
		{Path: "b.tei", Fingerprint: "sha256:old"},
		{Path: "a.tei", Fingerprint: "sha256:removed"},
	}}
	after := sourceindex.Index{Artifacts: []sourceindex.Artifact{
		{Path: "b.tei", Fingerprint: "sha256:new"},
		{Path: "c.tei", Fingerprint: "sha256:added"},
		{Path: "z.tei", Fingerprint: "sha256:unchanged"},
	}}
	got := diffSourceArtifacts(before, after)
	if len(got) != 3 || got[0].Path != "a.tei" || got[0].Kind != "removed" ||
		got[1].Path != "b.tei" || got[1].Kind != "content-changed" ||
		got[2].Path != "c.tei" || got[2].Kind != "added" ||
		got[0].AfterFingerprint != "" || got[2].BeforeFingerprint != "" {
		t.Fatalf("source additions/removals concealed or unordered: %+v", got)
	}
}

func TestWorkspaceDiffDecisionMaintainsEvidenceAndAuthorityBoundaries(t *testing.T) {
	key := workspaceKey("semantic-data", "sales.order")
	consumer := workspaceKey("application-pattern", "order_entry.order_record")
	graph := modelref.Graph{Nodes: []modelref.Node{{Key: key}, {Key: consumer}}}
	base := WorkspaceDiffReport{
		Consumer:              consumer,
		BeforeRuleFingerprint: "same",
		AfterRuleFingerprint:  "same",
		SourceChanges:         []SourceArtifactChange{{Path: "semantic/sales/order.tei", Kind: "content-changed"}},
		Impact:                modelref.ImpactReview{GraphValid: true},
	}
	assert := func(name string, report WorkspaceDiffReport, wantAction, wantReason string, indexed bool) {
		t.Helper()
		got := classifyWorkspaceDiff(report, graph, graph, indexed)
		if got.Action != wantAction || got.ReasonCodes[0] != wantReason ||
			got.ExecutionAllowed || got.ReceiptRef != "" {
			t.Errorf("%s: %+v", name, got)
		}
	}
	assert("untyped source change", base, "unresolved", "source-changed-without-typed-graph-diff", true)
	direct := base
	direct.Impact.Changes = []modelref.Change{{Key: consumer, Kind: "reference-changed"}}
	assert("directly changed consumer", direct, "recheck", "consumer-directly-changed", true)
	affected := base
	affected.Impact.Changes = []modelref.Change{{Key: key, Kind: "version-changed"}}
	affected.Impact.Affected = []modelref.Affected{{Changed: key, Consumer: consumer, Reason: "version-changed"}}
	assert("typed transitive impact", affected, "recheck", "typed-reference-impact-candidate", true)
	unrelated := base
	unrelated.Impact.Changes = []modelref.Change{{Key: key, Kind: "version-changed"}}
	assert("source changed but no consumer impact", unrelated, "unresolved", "source-change-without-proven-consumer-impact", true)
	drift := affected
	drift.AfterRuleFingerprint = "different"
	assert("rule drift never proves closed impact", drift, "unresolved", "typed-projection-rules-changed", true)
	badGraph := affected
	badGraph.Impact.GraphValid = false
	assert("invalid graph", badGraph, "unresolved", "typed-before-or-after-graph-invalid", true)
	assert("unindexed sources", affected, "unresolved", "managed-source-inventory-missing", false)
	noConsumer := base
	noConsumer.Consumer = workspaceKey("application", "missing")
	assert("missing consumer", noConsumer, "unresolved", "consumer-not-in-either-typed-graph", true)
	noChanges := base
	noChanges.SourceChanges = nil
	assert("no changes no receipts", noChanges, "unresolved", "no-independent-receipt-or-proven-dependency-closure", true)
}

func TestWorkspaceDiffRuleDriftRetainsKnownConsumerImpact(t *testing.T) {
	changed := workspaceKey("semantic-data", "sales.order")
	consumer := workspaceKey("application-pattern", "order_entry.order_record")
	graph := modelref.Graph{Nodes: []modelref.Node{{Key: changed}, {Key: consumer}}}
	report := WorkspaceDiffReport{
		Consumer:              consumer,
		BeforeRuleFingerprint: "old-rules",
		AfterRuleFingerprint:  "new-rules",
		SourceChanges:         []SourceArtifactChange{{Path: "semantic/sales/order.tei", Kind: "content-changed"}},
		Impact: modelref.ImpactReview{
			GraphValid: true,
			Changes:    []modelref.Change{{Key: changed, Kind: "version-changed"}},
			Affected:   []modelref.Affected{{Changed: changed, Consumer: consumer, Reason: "version-changed"}},
		},
	}

	got := classifyWorkspaceDiff(report, graph, graph, true)
	if got.Action != "unresolved" || got.ExecutionAllowed ||
		len(got.ReasonCodes) != 2 ||
		got.ReasonCodes[0] != "typed-projection-rules-changed" ||
		got.ReasonCodes[1] != "typed-reference-impact-candidate" {
		t.Fatalf("known recheck impact must survive unresolved rule coverage: %+v", got)
	}
	if len(report.Impact.Affected) != 1 {
		t.Fatalf("classifying assurance must not mutate typed impact evidence: %+v", report.Impact)
	}

	// A purported impact for a consumer absent from both typed projections
	// is not sufficient to claim that a known consumer was affected.
	phantom := workspaceKey("application-pattern", "order_entry.unknown")
	report.Consumer = phantom
	report.Impact.Affected = []modelref.Affected{{Changed: changed, Consumer: phantom, Reason: "version-changed"}}
	got = classifyWorkspaceDiff(report, graph, graph, true)
	if got.Action != "unresolved" || len(got.ReasonCodes) != 1 ||
		got.ReasonCodes[0] != "typed-projection-rules-changed" || got.ExecutionAllowed {
		t.Fatalf("missing consumer must not be reported as a known typed impact: %+v", got)
	}

	report.Consumer = consumer
	report.Impact.Affected = nil
	got = classifyWorkspaceDiff(report, graph, graph, true)
	if got.Action != "unresolved" || len(got.ReasonCodes) != 1 ||
		got.ReasonCodes[0] != "typed-projection-rules-changed" {
		t.Fatalf("unknown scope must not invent a known consumer impact: %+v", got)
	}

	report.Impact.Changes = []modelref.Change{{Key: consumer, Kind: "reference-changed"}}
	got = classifyWorkspaceDiff(report, graph, graph, true)
	if got.Action != "unresolved" || len(got.ReasonCodes) != 2 ||
		got.ReasonCodes[1] != "consumer-directly-changed" {
		t.Fatalf("direct impact must remain visible without claiming closure: %+v", got)
	}
}

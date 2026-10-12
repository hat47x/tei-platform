package main

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/modelref/derivedbridge"
	"tei/internal/modelref/sourcebridge"
	"tei/internal/sourceindex"
	"tei/internal/subject"
)

// #1120: A proposal's origin is not Canonical authority. Review and Accept
// must preserve existing typed subjects while a new validation becomes a
// Source-backed definition. This exercise uses the real Workspace loader,
// existing Shared Acceptance boundary, Decision ledger and modelref bridge.
func TestIssue1120AIProposalIdentityAcrossSharedAcceptance(t *testing.T) {
	root := scratchWorkspaceBeforeSecondApplication(t)
	before, err := takeSnapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	const scope = "order-entry-observation"
	project := func(s snapshot) derivedbridge.Projection {
		return derivedbridge.FromIndex(index.Build(s.loaded.World), scope)
	}
	first := project(before)
	if ds := modelref.Validate(first.Graph, first.Rules); len(ds) != 0 {
		t.Fatalf("invalid before projection: %+v", ds)
	}
	key := func(kind, id string) modelref.Key {
		return modelref.Key{Scope: scope, Subject: subject.Subject{Kind: kind, ID: id}}
	}
	stable := []modelref.Key{
		key("semantic-data", "sales.order_line"),
		key("semantic-attribute", "sales.order_line.unit_price"),
		key("semantic-validation", "sales.order_line.quantity_positive"),
		key("application-pattern", "order_entry.order_record"),
	}
	const addedID = "sales.order_line.unit_price_positive"
	added := key("semantic-validation", addedID)
	hasNode := func(g modelref.Graph, k modelref.Key) bool {
		for _, node := range g.Nodes {
			if node.Key == k {
				return true
			}
		}
		return false
	}
	for _, k := range stable {
		if !hasNode(first.Graph, k) {
			t.Fatalf("missing pre-existing typed identity: %+v", k)
		}
	}
	if hasNode(first.Graph, added) {
		t.Fatalf("new validation existed before proposal: %+v", added)
	}
	if len(before.index.ReferencesFor(sourceindex.Subject{Kind: added.Subject.Kind, ID: added.Subject.ID})) != 0 {
		t.Fatal("source location fabricated for an unaccepted proposal")
	}

	candidate := unitPriceCandidate(before.revision)
	candidate.Origin.Kind = "ai"
	candidate.Origin.Actor = "proposal-only"
	// Preserve a caller-supplied provenance note. Neither that note nor the
	// declared AI actor confers approval or accepted meaning.
	candidate.Evidence[0].Note += " / legacy-record:line-17"
	input := mustJSON(t, candidate)
	unchanged := treeDigest(t, root)
	evaluation, err := evaluate(root, input)
	if err != nil || !evaluation.preview.Acceptable {
		t.Fatalf("proposal preview failed: %v / %+v", err, evaluation.preview.Stages)
	}
	assertTreeUnchanged(t, unchanged, treeDigest(t, root))
	stillBefore, err := takeSnapshot(root)
	if err != nil || stillBefore.revision != before.revision {
		t.Fatalf("preview changed authoritative source: %v / %s", err, stillBefore.revision)
	}

	decision := acceptViaCLI(t, root, candidateFile(t, candidate))
	if decision.Decision != "accepted" || !decision.SourceChanged ||
		decision.BaseModelRevision != before.revision ||
		decision.ModelRevisionBefore != before.revision ||
		decision.Origin == nil || decision.Origin.Kind != "ai" ||
		decision.Origin.Actor != "proposal-only" {
		t.Fatalf("accepted decision lost base revision / AI origin: %+v", decision)
	}
	if len(decision.Evidence) == 0 || decision.Evidence[0].Note != candidate.Evidence[0].Note {
		t.Fatalf("proposal provenance did not survive acceptance: %+v", decision.Evidence)
	}
	after, err := takeSnapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	if after.revision == before.revision || after.revision != decision.ModelRevisionAfter {
		t.Fatalf("accepted source revision mismatch: before=%s after=%s decision=%s",
			before.revision, after.revision, decision.ModelRevisionAfter)
	}
	second := project(after)
	if ds := modelref.Validate(second.Graph, second.Rules); len(ds) != 0 {
		t.Fatalf("invalid after projection: %+v", ds)
	}
	for _, k := range stable {
		if !hasNode(second.Graph, k) {
			t.Fatalf("accepted change displaced an existing scoped identity: %+v", k)
		}
	}
	if !hasNode(second.Graph, added) {
		t.Fatalf("accepted validation did not materialize: %+v", added)
	}
	impact := modelref.ReviewImpactDiff(first.Graph, second.Graph, second.Rules)
	if !impact.GraphValid {
		t.Fatalf("accepted typed graph could not be compared: %+v", impact.Diagnostics)
	}
	sawAdded := false
	for _, change := range impact.Changes {
		if change.Key == added && change.Kind == "added" {
			sawAdded = true
		}
		for _, existing := range stable {
			if change.Key == existing && (change.Kind == "added" || change.Kind == "removed") {
				t.Fatalf("existing subject was silently rekeyed: %+v", change)
			}
		}
	}
	if !sawAdded {
		t.Fatalf("new validation was not observed as a new ID: %+v", impact.Changes)
	}

	read := func(relative string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	}
	resolver, err := sourcebridge.NewResolver(second.Graph, after.index, read)
	if err != nil {
		t.Fatalf("accepted Source Index cannot be verified: %v", err)
	}
	location := resolver.Locate(added)
	if location.Status != "located" {
		t.Fatalf("accepted new identity has no definition anchor: %+v", location)
	}
	foundDefinition := false
	for _, anchor := range location.References {
		if anchor.Role == sourceindex.RoleDefinition &&
			anchor.ArtifactPath == "semantic/sales/validation.tei" {
			foundDefinition = true
		}
	}
	if !foundDefinition {
		t.Fatalf("accepted identity did not resolve to its Source: %+v", location.References)
	}
	records := decisionsOf(t, root)
	if len(records.Records) != 1 ||
		records.Records[0].Decision.CandidateFingerprint != decision.CandidateFingerprint ||
		records.Records[0].DecisionActorStatus != "unresolved" ||
		records.Records[0].DecisionAuthorityStatus != "unresolved" {
		t.Fatalf("AI origin promoted to decision authority or ledger detached: %+v", records.Records)
	}
}

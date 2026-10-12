package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/modelref/derivedbridge"
	"tei/internal/modelref/sourcebridge"
	"tei/internal/sourceindex"
	"tei/internal/subject"
	"tei/tools/internal/rmtevidence"
	"tei/tools/internal/rmtlink"
)

// This binds a source adapter's genuine, source-derived token from independent
// synthetic X-TEA XML to a candidate, the exact Review, an Acceptance Decision
// and the resulting typed Source definition. The synthetic XML does NOT prove
// semantic equivalence to the TEI validation or original-system behavior.
func TestIssue1120RMTEvidenceTokenToAcceptedTypedIdentity(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	before, err := takeSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	const scope = "order-entry-reference"
	typed := func(kind, id string) modelref.Key {
		return modelref.Key{Scope: scope, Subject: subject.Subject{Kind: kind, ID: id}}
	}
	const newID = "sales.order_line.unit_price_positive"
	newKey := typed("semantic-validation", newID)
	beforeProjection := derivedbridge.FromIndex(index.Build(before.loaded.World), scope)
	for _, node := range beforeProjection.Graph.Nodes {
		if node.Key == newKey {
			t.Fatal("new target was already authoritative before Acceptance")
		}
	}
	if refs := before.index.ReferencesFor(sourceindex.Subject{Kind: newKey.Subject.Kind, ID: newID}); len(refs) != 0 {
		t.Fatalf("unaccepted target acquired a Source definition: %+v", refs)
	}
	stable := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order_line"}
	stableBefore := before.index.ReferencesFor(stable)
	if len(stableBefore) == 0 {
		t.Fatal("existing Data has no Source anchor")
	}

	// The adapter computes a source-bound token from independently observed
	// local XML, rather than using a human-written 'legacy note' as evidence.
	externalBytes := rmtIndependentSyntheticSourceEvidence(t)
	sourceEvidence, err := rmtevidence.Decode(externalBytes)
	if err != nil {
		t.Fatal(err)
	}
	if sourceEvidence.SourceToken == "" || sourceEvidence.SourceToken == newID ||
		sourceEvidence.SourceArtifactDigest == "" || sourceEvidence.MappingContractFingerprint == "" {
		t.Fatalf("external identity conflated with canonical subject: %+v", sourceEvidence)
	}

	proposal := unitPriceCandidate(before.revision)
	proposal.Origin.Kind, proposal.Origin.Actor = "ai", "unverified-proposal-author"
	proposal.Evidence = nil // only the independent source-bound artifact
	grounded, err := attachRMTTranscriptionEvidence(mustJSON(t, proposal), externalBytes)
	if err != nil {
		t.Fatal(err)
	}
	original := treeDigest(t, dir)
	ev, err := evaluate(dir, mustJSON(t, grounded))
	if err != nil || !ev.preview.Acceptable {
		t.Fatalf("typed proposal is not reviewable: %v; %+v", err, ev.preview.Stages)
	}
	assertTreeUnchanged(t, original, treeDigest(t, dir))
	if len(ev.preview.SemanticDiff) != 1 || ev.preview.SemanticDiff[0].ID != newID ||
		ev.preview.SemanticDiff[0].Kind != "validation" || ev.preview.SemanticDiff[0].Change != "added" {
		t.Fatalf("reviewed semantic target is not the exact proposal: %+v", ev.preview.SemanticDiff)
	}
	change := ev.preview.SemanticDiff[0]
	link, context, err := prepareRMTReviewedLink(ev, sourceEvidence, rmtlink.SemanticTarget{
		Kind: change.Kind, ID: change.ID, Change: change.Change,
	})
	if err != nil {
		t.Fatal(err)
	}
	if link.Status != rmtlink.StatusProposed || link.MeaningStatus != rmtlink.MeaningUnresolved ||
		context.Source.SourceToken != sourceEvidence.SourceToken {
		t.Fatalf("proposed source correspondence became accepted authority: %+v", link)
	}
	if err := rmtlink.Verify(link, context); err != nil {
		t.Fatal(err)
	}
	stale := context
	stale.Source.SourceArtifactDigest = "sha256:" + strings.Repeat("0", 64)
	if err := rmtlink.Verify(link, stale); err == nil {
		t.Fatal("different source artifact was bound to reviewed target")
	}

	decision := acceptViaCLI(t, dir, candidateFile(t, grounded))
	if decision.Decision != "accepted" || !decision.SourceChanged ||
		decision.BaseModelRevision != before.revision ||
		decision.ModelRevisionBefore != before.revision ||
		decision.Origin == nil || decision.Origin.Kind != "ai" {
		t.Fatalf("Decision lost proposal source or revision: %+v", decision)
	}
	persisted := false
	for _, item := range decision.Evidence {
		existing, ok := rmtTranscriptionFromEvidence(item)
		if !ok {
			continue
		}
		if existing.SourceToken == sourceEvidence.SourceToken &&
			existing.SourceArtifactDigest == sourceEvidence.SourceArtifactDigest &&
			existing.MappingContractFingerprint == sourceEvidence.MappingContractFingerprint &&
			existing.EvidenceFingerprint == sourceEvidence.EvidenceFingerprint {
			persisted = true
		}
	}
	if !persisted {
		t.Fatalf("Decision did not retain source-bound evidence: %+v", decision.Evidence)
	}
	after, err := takeSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if after.revision == before.revision || after.revision != decision.ModelRevisionAfter {
		t.Fatalf("Revision was not bound to Decision: before=%s after=%s decision=%s",
			before.revision, after.revision, decision.ModelRevisionAfter)
	}
	afterProjection := derivedbridge.FromIndex(index.Build(after.loaded.World), scope)
	impact := modelref.ReviewImpactDiff(beforeProjection.Graph, afterProjection.Graph, afterProjection.Rules)
	if !impact.GraphValid {
		t.Fatalf("accepted projection does not validate: %+v", impact.Diagnostics)
	}
	foundAdded := false
	for _, delta := range impact.Changes {
		if delta.Kind == "added" && delta.Key == newKey {
			foundAdded = true
		}
	}
	if !foundAdded {
		t.Fatalf("actual accepted identity was not an addition: %+v", impact.Changes)
	}
	if !reflect.DeepEqual(stableBefore, after.index.ReferencesFor(stable)) {
		t.Fatal("unchanged Data Source identity moved during unrelated acceptance")
	}

	read := func(rel string) ([]byte, error) {
		return os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	}
	resolver, err := sourcebridge.NewResolver(afterProjection.Graph, after.index, read)
	if err != nil {
		t.Fatalf("accepted Source index failed to validate: %v", err)
	}
	location := resolver.Locate(newKey)
	if location.Status != "located" {
		t.Fatalf("new Source definition could not be located: %+v", location)
	}
	foundSource := false
	for _, anchor := range location.References {
		if anchor.Role == sourceindex.RoleDefinition &&
			anchor.ArtifactPath == "semantic/sales/validation.tei" {
			foundSource = true
		}
	}
	if !foundSource {
		t.Fatalf("Source location did not bind the accepted ID: %+v", location.References)
	}
	ledger := decisionsOf(t, dir)
	if len(ledger.Records) != 1 {
		t.Fatalf("Decision ledger is incomplete: %+v", ledger.Records)
	}
	record := ledger.Records[0]
	if record.Decision.CandidateFingerprint != decision.CandidateFingerprint ||
		record.Decision.EvidenceFingerprint != decision.EvidenceFingerprint ||
		record.EvidenceStatus != "recorded" ||
		record.ArtifactEvidenceStatus != "recorded" ||
		record.SourceEvidenceStatus != "none" ||
		record.DecisionActorStatus != "unresolved" ||
		record.DecisionAuthorityStatus != "unresolved" {
		t.Fatalf("source observation was promoted or disconnected: %+v", record)
	}
	// The candidate-only semantic correlation remains proposed even after
	// TEI source acceptance. A human assertion of legacy equivalence was not
	// given and therefore cannot be inferred from the accepted TEI source.
	if link.MeaningStatus != rmtlink.MeaningUnresolved {
		t.Fatal("accepted TEI meaning implied equivalence with legacy observation")
	}

	lineage, err := traceRMTIdentityAcrossAcceptance(link, context, before, after, scope)
	if err != nil {
		t.Fatalf("recorded acceptance cannot be traced to typed Source: %v", err)
	}
	if lineage.Target != newKey || lineage.SourceToken != sourceEvidence.SourceToken ||
		lineage.SourceDigest != sourceEvidence.SourceArtifactDigest ||
		lineage.CandidateFingerprint != decision.CandidateFingerprint ||
		lineage.ReviewFingerprint != decision.ReviewBinding.Fingerprint ||
		lineage.DefinitionPath != "semantic/sales/validation.tei" ||
		lineage.BeforeRevision != before.revision || lineage.AfterRevision != after.revision ||
		lineage.MeaningStatus != rmtlink.MeaningUnresolved {
		t.Fatalf("accepted source identity / legacy provenance drifted: %+v", lineage)
	}
	t.Run("reject altered source correlation", func(t *testing.T) {
		tampered := link
		tampered.SourceToken = "sha256:" + strings.Repeat("0", 64)
		if _, err := traceRMTIdentityAcrossAcceptance(tampered, context, before, after, scope); err == nil {
			t.Fatal("tampered source token was accepted")
		}
	})
	t.Run("reject stale before revision", func(t *testing.T) {
		staleBefore := before
		staleBefore.revision = "sha256:" + strings.Repeat("0", 64)
		if _, err := traceRMTIdentityAcrossAcceptance(link, context, staleBefore, after, scope); err == nil {
			t.Fatal("stale base revision was accepted")
		}
	})
	t.Run("reject unbound after revision", func(t *testing.T) {
		staleAfter := after
		staleAfter.revision = "sha256:" + strings.Repeat("0", 64)
		if _, err := traceRMTIdentityAcrossAcceptance(link, context, before, staleAfter, scope); err == nil {
			t.Fatal("unbound decision ledger revision was accepted")
		}
	})
	t.Run("reject source mutation after snapshot", func(t *testing.T) {
		path := filepath.Join(dir, filepath.FromSlash(lineage.DefinitionPath))
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(bytes, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := traceRMTIdentityAcrossAcceptance(link, context, before, after, scope); err == nil {
			t.Fatal("mutated accepted Source was treated as unchanged")
		}
	})
}

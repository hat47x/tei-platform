package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/desktophost"
	"tei/internal/workspace"
)

// The Desktop Host does not own Acceptance. These tests deliberately invoke
// the existing reference-shared-acceptance evaluator and accept implementation
// in a scratch Workspace rather than synthesizing another write path.
func TestDesktopHostObservationSharedAcceptanceRoundTrip(t *testing.T) {
	root := scratchWorkspace(t)
	before, err := desktophost.ObserveSource(root)
	if err != nil {
		t.Fatal(err)
	}
	revision := revisionOfDir(t, root)
	candidate := mustJSON(t, unitPriceCandidate(revision))
	ev, err := evaluate(root, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if !ev.preview.Acceptable {
		t.Fatalf("shared preview refused: %+v", allDiagnostics(ev.preview))
	}
	review := buildReviewProjection(ev)
	if review.ReviewFingerprint == "" || !review.PreviewAcceptable {
		t.Fatalf("shared review was not usable: %+v", review)
	}
	if _, err := desktophost.CheckSourceFresh(before); err != nil {
		t.Fatalf("read-only evaluation modified Desktop Source observation: %v", err)
	}
	marker := desktophost.DraftMarker{
		Version: desktophost.MarkerVersion, CandidateRef: "draft-validation-1",
		BaseModelRevision: revision, Baseline: before,
	}
	raw, err := desktophost.EncodeDraftMarker(marker)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := desktophost.RecoverDraftMarker(raw, before.WorkspaceRoot, desktophost.ObserveSource)
	if err != nil || recovered.Status != desktophost.RecoveryNeedsReview {
		t.Fatalf("draft restore must not imply approval: %+v, %v", recovered, err)
	}

	// This is a TEST HARNESS acceptance; review fingerprint equality is neither
	// evidence of a human's decision nor runtime authorization.
	decision, err := accept(root, candidate, acceptOptions{
		ExpectedReviewFingerprint: review.ReviewFingerprint,
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "accepted" || !decision.SourceChanged ||
		decision.ReviewBinding == nil || decision.ReviewBinding.Fingerprint != review.ReviewFingerprint {
		t.Fatalf("shared Acceptance was not bound to review: %+v", decision)
	}
	if _, err := workspace.Load(root); err != nil {
		t.Fatalf("accepted Workspace did not reload through the ordinary loader: %v", err)
	}
	if _, err := desktophost.CheckSourceFresh(before); !errors.Is(err, desktophost.ErrSourceChanged) {
		t.Fatalf("Desktop observation did not detect accepted Source update: %v", err)
	}
	afterRecovery, err := desktophost.RecoverDraftMarker(raw, before.WorkspaceRoot, desktophost.ObserveSource)
	if err != nil || afterRecovery.Status != desktophost.RecoverySourceChanged {
		t.Fatalf("old Draft remained apparently current after acceptance: %+v, %v", afterRecovery, err)
	}
}

func TestDesktopHostAndSharedAcceptanceBothRejectStaleManagedSource(t *testing.T) {
	root := scratchWorkspace(t)
	before, err := desktophost.ObserveSource(root)
	if err != nil {
		t.Fatal(err)
	}
	revision := revisionOfDir(t, root)
	candidate := mustJSON(t, unitPriceCandidate(revision))
	ev, err := evaluate(root, candidate)
	if err != nil || !ev.preview.Acceptable {
		t.Fatalf("shared Preview failed: %+v, %v", ev.preview, err)
	}
	review := buildReviewProjection(ev)

	// Simulate another editor changing source bytes WITHOUT a watcher event.
	target := filepath.Join(root, "semantic", "sales", "validation.tei")
	original, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	external := append(append([]byte(nil), original...), '\n')
	if err := os.WriteFile(target, external, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := desktophost.CheckSourceFresh(before); !errors.Is(err, desktophost.ErrSourceChanged) {
		t.Fatalf("Desktop preflight missed external edit: %v", err)
	}
	decision, err := accept(root, candidate, acceptOptions{
		ExpectedReviewFingerprint: review.ReviewFingerprint,
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "refused" || decision.SourceChanged {
		t.Fatalf("stale Shared Acceptance modified Source: %+v", decision)
	}
	found := false
	for _, reason := range decision.Refusals {
		if reason.Code == "STALE_BASE_MODEL_REVISION" {
			found = true
		}
	}
	if !found {
		t.Fatalf("stale revision was not the refusal cause: %+v", decision.Refusals)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != string(external) {
		t.Fatalf("refusal overwrote another editor's change: %v", err)
	}
}

func TestDesktopManifestObservationDetectsChangeOmittedByModelRevision(t *testing.T) {
	root := scratchWorkspace(t)
	before, err := desktophost.ObserveSource(root)
	if err != nil {
		t.Fatal(err)
	}
	revision := revisionOfDir(t, root)
	marker := desktophost.DraftMarker{
		Version: desktophost.MarkerVersion, CandidateRef: "draft-manifest-1",
		BaseModelRevision: revision, Baseline: before,
	}
	raw, err := desktophost.EncodeDraftMarker(marker)
	if err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, "tei.workspace.yaml")
	file, err := os.OpenFile(manifest, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("\n# external Desktop workspace configuration edit\n"); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	// A comment is a byte change in the authoritative Workspace selector.
	// The reference model revision covers Source Index artifacts, not this
	// manifest. Do not claim the current Acceptance automatically fences it.
	if after := revisionOfDir(t, root); after != revision {
		t.Fatalf("fixture ceased demonstrating independent manifest bytes: %s != %s", after, revision)
	}
	if _, err := desktophost.CheckSourceFresh(before); !errors.Is(err, desktophost.ErrSourceChanged) {
		t.Fatalf("unchanged model revision hid manifest byte change: %v", err)
	}
	recovered, err := desktophost.RecoverDraftMarker(raw, before.WorkspaceRoot, desktophost.ObserveSource)
	if err != nil || recovered.Status != desktophost.RecoverySourceChanged {
		t.Fatalf("manifest change did not invalidate draft recovery: %+v %v", recovered, err)
	}
	// This is a read-only boundary test: there is no atomic link between
	// Desktop preflight and Shared Acceptance's mutation fence yet. In
	// particular, callers must not reuse the old review for Accept here.
	if strings.TrimSpace(recovered.Marker.CandidateRef) != marker.CandidateRef {
		t.Fatal("recovery unexpectedly replaced the draft identity")
	}
}

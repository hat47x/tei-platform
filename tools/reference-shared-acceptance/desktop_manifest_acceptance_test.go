package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/desktophost"
)

// Revisions cover managed Source Index files, not the Workspace selector.
// The latter still has to participate in the exact human Review binding.
func TestDesktopManifestByteChangeInvalidatesOldReviewWithoutModelRevisionChange(t *testing.T) {
	root := scratchWorkspace(t)
	revision := revisionOfDir(t, root)
	candidate := mustJSON(t, unitPriceCandidate(revision))
	before, err := evaluate(root, candidate)
	if err != nil || !before.preview.Acceptable {
		t.Fatalf("initial Preview failed: %+v %v", before.preview, err)
	}
	reviewBefore := buildReviewProjection(before)
	if reviewBefore.ManifestFingerprint == "" || reviewBefore.ManifestPath != "tei.workspace.yaml" {
		t.Fatalf("Review does not bind the Workspace selector: %+v", reviewBefore)
	}
	manifest := filepath.Join(root, reviewBefore.ManifestPath)
	appendManifestComment(t, manifest)
	if got := revisionOfDir(t, root); got != revision {
		t.Fatalf("test requires a byte change independent of model revision: %s vs %s", got, revision)
	}
	after, err := evaluate(root, candidate)
	if err != nil || !after.preview.Acceptable {
		t.Fatalf("manifest-only comment should keep Semantic Preview valid: %+v %v", after.preview, err)
	}
	reviewAfter := buildReviewProjection(after)
	if reviewAfter.ManifestFingerprint == reviewBefore.ManifestFingerprint ||
		reviewAfter.ReviewFingerprint == reviewBefore.ReviewFingerprint ||
		reviewAfter.CandidateFingerprint != reviewBefore.CandidateFingerprint {
		t.Fatalf("manifest must invalidate Review without changing candidate identity: before=%+v after=%+v", reviewBefore, reviewAfter)
	}
	target := filepath.Join(root, "semantic", "sales", "validation.tei")
	sourceBefore, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := accept(root, candidate, acceptOptions{ExpectedReviewFingerprint: reviewBefore.ReviewFingerprint})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "refused" || decision.SourceChanged ||
		!hasCode(refusalCodes(decision), "REVIEW_FINGERPRINT_MISMATCH") {
		t.Fatalf("old Review was accepted after byte-only Workspace selector change: %+v", decision)
	}
	sourceAfter, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(sourceAfter, sourceBefore) {
		t.Fatalf("refused review modified Accepted Source: %v", err)
	}
}

func TestDesktopManifestChangeAtAcceptWriteBoundaryRefusesWithoutMutatingSource(t *testing.T) {
	root := scratchWorkspace(t)
	revision := revisionOfDir(t, root)
	candidate := mustJSON(t, unitPriceCandidate(revision))
	ev, err := evaluate(root, candidate)
	if err != nil || !ev.preview.Acceptable {
		t.Fatalf("initial Preview failed: %+v %v", ev.preview, err)
	}
	review := buildReviewProjection(ev)
	target := filepath.Join(root, "semantic", "sales", "validation.tei")
	before, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, review.ManifestPath)
	didChange := false
	beforeEachWriteHook = func(index int, edit SourceEdit) {
		if index == 0 {
			appendManifestComment(t, manifest)
			didChange = true
		}
	}
	defer func() { beforeEachWriteHook = nil }()
	decision, err := accept(root, candidate, acceptOptions{ExpectedReviewFingerprint: review.ReviewFingerprint})
	if err != nil {
		t.Fatal(err)
	}
	if !didChange || decision.Decision != "refused" || decision.SourceChanged ||
		!hasCode(refusalCodes(decision), "SOURCE_MANIFEST_CHANGED_DURING_ACCEPT") {
		t.Fatalf("manifest byte swap at mutation boundary was not refused: %+v", decision)
	}
	got, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(before, got) {
		t.Fatalf("manifest refusal modified Source: %v", err)
	}
}

func TestDesktopManifestChangeBetweenTwoWritesRollsBackOwnSource(t *testing.T) {
	root := scratchWorkspace(t)
	revision := revisionOfDir(t, root)
	c := unitPriceCandidate(revision)
	c.Delta = append(c.Delta, Operation{
		Op: opAddValidation, Namespace: "sales", ID: "sales.order.total_positive",
		Kind: "positive", Attribute: "sales.order.total",
		Placement: "semantic/sales/order.tei",
	})
	candidate := mustJSON(t, c)
	ev, err := evaluate(root, candidate)
	if err != nil || !ev.preview.Acceptable || len(ev.edits) != 2 {
		t.Fatalf("initial Preview must produce two writes: %+v %v", ev.preview, err)
	}
	review := buildReviewProjection(ev)
	first := filepath.Join(root, "semantic", "sales", "order.tei")
	second := filepath.Join(root, "semantic", "sales", "validation.tei")
	beforeFirst, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	beforeSecond, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, review.ManifestPath)
	beforeManifest, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	didChange := false
	beforeEachWriteHook = func(index int, edit SourceEdit) {
		if index == 1 {
			appendManifestComment(t, manifest)
			didChange = true
		}
	}
	defer func() { beforeEachWriteHook = nil }()
	decision, err := accept(root, candidate, acceptOptions{ExpectedReviewFingerprint: review.ReviewFingerprint})
	if err != nil {
		t.Fatal(err)
	}
	if !didChange || decision.Decision != "refused" || decision.SourceChanged ||
		!hasCode(refusalCodes(decision), "SOURCE_MANIFEST_CHANGED_DURING_ACCEPT") {
		t.Fatalf("Accept did not restore its first edit on second-write manifest change: %+v", decision)
	}
	for _, fixture := range []struct {
		path string
		want []byte
	}{{first, beforeFirst}, {second, beforeSecond}} {
		got, err := os.ReadFile(fixture.path)
		if err != nil || !bytes.Equal(got, fixture.want) {
			t.Fatalf("refused Accept changed %s: %v", fixture.path, err)
		}
	}
	gotManifest, err := os.ReadFile(manifest)
	if err != nil || bytes.Equal(gotManifest, beforeManifest) {
		t.Fatalf("another actor's manifest change was overwritten: %v", err)
	}
}

func TestDesktopManifestChangeAfterAllWritesIsNotAccepted(t *testing.T) {
	root := scratchWorkspace(t)
	revision := revisionOfDir(t, root)
	candidate := mustJSON(t, unitPriceCandidate(revision))
	ev, err := evaluate(root, candidate)
	if err != nil || !ev.preview.Acceptable {
		t.Fatalf("initial Preview failed: %+v %v", ev.preview, err)
	}
	review := buildReviewProjection(ev)
	target := filepath.Join(root, "semantic", "sales", "validation.tei")
	sourceBefore, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, review.ManifestPath)
	afterWritesHook = func(root string) { appendManifestComment(t, manifest) }
	defer func() { afterWritesHook = nil }()
	decision, err := accept(root, candidate, acceptOptions{ExpectedReviewFingerprint: review.ReviewFingerprint})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "refused" || decision.SourceChanged ||
		!hasCode(refusalCodes(decision), "SOURCE_MANIFEST_CHANGED_DURING_ACCEPT") {
		t.Fatalf("post-write manifest change committed stale Accepted Source: %+v", decision)
	}
	sourceAfter, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(sourceAfter, sourceBefore) {
		t.Fatalf("failed Accept did not restore own Source: %v", err)
	}
}

func TestDesktopManifestReviewBindingMatchesReadOnlyHostObservation(t *testing.T) {
	root := scratchWorkspace(t)
	observed, err := desktophost.ObserveSource(root)
	if err != nil {
		t.Fatal(err)
	}
	ev, err := evaluate(root, mustJSON(t, unitPriceCandidate(revisionOfDir(t, root))))
	if err != nil || !ev.preview.Acceptable {
		t.Fatalf("Preview failed: %+v %v", ev.preview, err)
	}
	review := buildReviewProjection(ev)
	found := false
	for _, artifact := range observed.Artifacts {
		if artifact.Path == review.ManifestPath {
			if artifact.Fingerprint != review.ManifestFingerprint {
				t.Fatalf("Host and Shared Acceptance disagree on manifest bytes: host=%s accept=%s", artifact.Fingerprint, review.ManifestFingerprint)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("manifest is not observed by Desktop Host: %s", review.ManifestPath)
	}
	if !strings.Contains(renderReviewProjection(review), review.ManifestFingerprint) {
		t.Fatal("human review output hides the manifest binding")
	}
}

func appendManifestComment(t *testing.T, path string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("\n# external manifest byte update without semantic change\n"); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

package main

import (
	"strings"
	"testing"

	"tei/tools/internal/rmtevidence"
	"tei/tools/internal/rmtlink"
)

func TestRMTReviewedLinkRequiresAttachedEvidenceAndActualReviewedMeaning(t *testing.T) {
	dir := scratchWorkspace(t)
	base := unitPriceWithSelectionCandidate(revisionOfDir(t, dir))
	source := rmtEvidenceFixture(t, true)
	evidenceBytes, err := rmtevidence.CanonicalBytes(source)
	if err != nil {
		t.Fatal(err)
	}
	grounded, err := attachRMTTranscriptionEvidence(mustJSON(t, base), evidenceBytes)
	if err != nil {
		t.Fatal(err)
	}
	evaluated, err := evaluate(dir, mustJSON(t, grounded))
	if err != nil {
		t.Fatal(err)
	}
	target := rmtlink.SemanticTarget{
		Kind: "validation", ID: "sales.order_line.unit_price_positive", Change: "added",
	}
	link, context, err := prepareRMTReviewedLink(evaluated, source, target)
	if err != nil {
		t.Fatalf("actual Preview target and its own source Evidence were not bound: %v", err)
	}
	if err := rmtlink.Verify(link, context); err != nil {
		t.Fatal(err)
	}
	if link.Status != rmtlink.StatusProposed || link.MeaningStatus != rmtlink.MeaningUnresolved {
		t.Fatalf("candidate link claimed semantic Acceptance: %+v", link)
	}

	testCases := []struct {
		name string
		edit func(*evaluation, *rmtevidence.Evidence, *rmtlink.SemanticTarget)
	}{
		{"evidence removed from candidate", func(ev *evaluation, _ *rmtevidence.Evidence, _ *rmtlink.SemanticTarget) {
			ev.candidate.Evidence = nil
		}},
		{"different valid source token not in candidate", func(_ *evaluation, src *rmtevidence.Evidence, _ *rmtlink.SemanticTarget) {
			src.SourceToken = "sha256:" + strings.Repeat("a", 64)
			other, err := rmtevidence.Prepare(*src)
			if err != nil {
				t.Fatal(err)
			}
			*src = other
		}},
		{"candidate fingerprint mismatch", func(ev *evaluation, _ *rmtevidence.Evidence, _ *rmtlink.SemanticTarget) {
			ev.preview.CandidateFingerprint = "sha256:" + strings.Repeat("b", 64)
		}},
		{"candidate evidence fingerprint mismatch", func(ev *evaluation, _ *rmtevidence.Evidence, _ *rmtlink.SemanticTarget) {
			ev.preview.EvidenceFingerprint = "sha256:" + strings.Repeat("c", 64)
		}},
		{"base model changed", func(ev *evaluation, _ *rmtevidence.Evidence, _ *rmtlink.SemanticTarget) {
			ev.candidate.BaseModelRevision = "sha256:" + strings.Repeat("d", 64)
		}},
		{"workspace revision drifted", func(ev *evaluation, _ *rmtevidence.Evidence, _ *rmtlink.SemanticTarget) {
			ev.preview.CurrentModelRevision = "sha256:" + strings.Repeat("e", 64)
		}},
		{"stale candidate", func(ev *evaluation, _ *rmtevidence.Evidence, _ *rmtlink.SemanticTarget) {
			ev.preview.Stale = &Stale{}
		}},
		{"Preview rejected", func(ev *evaluation, _ *rmtevidence.Evidence, _ *rmtlink.SemanticTarget) {
			ev.preview.Acceptable = false
		}},
		{"missing semantic diff", func(ev *evaluation, _ *rmtevidence.Evidence, _ *rmtlink.SemanticTarget) {
			ev.preview.SemanticDiff = nil
		}},
		{"wrong semantic target", func(_ *evaluation, _ *rmtevidence.Evidence, target *rmtlink.SemanticTarget) {
			target.ID = "sales.order_line.quantity_positive"
		}},
		{"no source Evidence digest validation", func(_ *evaluation, src *rmtevidence.Evidence, _ *rmtlink.SemanticTarget) {
			src.SourceToken = "sha256:" + strings.Repeat("f", 64)
		}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			evCopy := evaluated
			sourceCopy := source
			targetCopy := target
			tc.edit(&evCopy, &sourceCopy, &targetCopy)
			if _, _, err := prepareRMTReviewedLink(evCopy, sourceCopy, targetCopy); err == nil {
				t.Fatal("an unreviewed or stale source-to-meaning link was allowed")
			}
		})
	}
}

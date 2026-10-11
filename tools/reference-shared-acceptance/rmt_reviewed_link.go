package main

import (
	"fmt"

	"tei/tools/internal/rmtevidence"
	"tei/tools/internal/rmtlink"
)

// prepareRMTReviewedLink builds a *candidate-only* source/meaning correlation
// from a real Shared Acceptance evaluation. Unlike rmtlink.Prepare, callers
// cannot invent the candidate / Review fingerprints or a target that is absent
// from the exact evaluated Semantic Diff.
//
// Even a successfully verified link does not establish that the source
// construct caused the semantic rule, prove human approval, or qualify the
// observer. These decisions are deliberately outside this function.
func prepareRMTReviewedLink(
	ev evaluation,
	source rmtevidence.Evidence,
	target rmtlink.SemanticTarget,
) (rmtlink.Link, rmtlink.Context, error) {
	fail := func(msg string) (rmtlink.Link, rmtlink.Context, error) {
		return rmtlink.Link{}, rmtlink.Context{}, fmt.Errorf("RMT Review-bound link: %s", msg)
	}
	if !ev.preview.Acceptable || ev.preview.Stale != nil {
		return fail("Preview is not acceptable or is stale")
	}
	candidate, fingerprint, err := normalizeCandidate(ev.candidate)
	if err != nil {
		return rmtlink.Link{}, rmtlink.Context{}, err
	}
	if ev.before.revision == "" ||
		candidate.BaseModelRevision != ev.before.revision ||
		ev.preview.BaseModelRevision != ev.before.revision ||
		ev.preview.CurrentModelRevision != ev.before.revision ||
		ev.preview.CandidateFingerprint != fingerprint ||
		ev.preview.EvidenceFingerprint != evidenceFingerprintOf(candidate.Evidence) {
		return fail("candidate, source revision or Evidence differs from current Preview")
	}
	review := buildReviewProjection(ev)
	if !review.PreviewAcceptable ||
		review.ReviewFingerprint == "" ||
		review.CandidateFingerprint != fingerprint ||
		review.EvidenceFingerprint != ev.preview.EvidenceFingerprint {
		return fail("Review does not bind the exact acceptable Preview")
	}

	matches := 0
	for _, change := range ev.preview.SemanticDiff {
		if change.Kind == target.Kind && change.ID == target.ID && change.Change == target.Change {
			matches++
		}
	}
	if matches != 1 {
		return fail("target is not a unique actual Semantic Diff of this Review")
	}

	validated, err := rmtevidence.Verify(source)
	if err != nil {
		return rmtlink.Link{}, rmtlink.Context{}, err
	}
	// Membership matters: an arbitrary source token must not be linked to a
	// reviewed candidate just because its independent checksum is well formed.
	sourceMatches := 0
	for _, e := range candidate.Evidence {
		existing, ok := rmtTranscriptionFromEvidence(e)
		if !ok {
			continue
		}
		if existing.EvidenceFingerprint == validated.EvidenceFingerprint {
			sourceMatches++
		}
	}
	if sourceMatches != 1 {
		return fail("source Evidence is not uniquely attached to the reviewed candidate")
	}

	context := rmtlink.Context{
		Source:               validated,
		CandidateFingerprint: fingerprint,
		BaseModelRevision:    candidate.BaseModelRevision,
		ReviewFingerprint:    review.ReviewFingerprint,
		Target:               target,
	}
	link, err := rmtlink.Prepare(context)
	if err != nil {
		return rmtlink.Link{}, rmtlink.Context{}, err
	}
	return link, context, nil
}

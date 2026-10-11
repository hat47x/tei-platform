package main

import (
	"strings"
	"testing"

	"tei/internal/knowledgecontext"
)

func TestBuildDataDesignCandidateReviewFromReferenceWorkspace(t *testing.T) {
	root := testRepositoryRoot(t)
	ctx, err := buildKnowledgeContext(
		root+"/reference/workspace",
		"semantic-data",
		"sales.order_line",
	)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.DataDesign == nil || len(ctx.DataDesign.BusinessIdentityCandidates) == 0 {
		t.Fatalf("reference Data Design candidate is missing: %+v", ctx.DataDesign)
	}
	candidate := ctx.DataDesign.BusinessIdentityCandidates[0]

	packet, err := buildDataDesignCandidateReview(
		root+"/reference/workspace",
		"semantic-data",
		"sales.order_line",
		knowledgecontext.DataDesignCandidateKindBusinessIdentity,
		candidate.Fingerprint,
		candidate.EvidenceRequirementFingerprint,
	)
	if err != nil {
		t.Fatal(err)
	}
	if packet.Subject.Kind != "semantic-data" ||
		packet.Subject.ID != "sales.order_line" ||
		packet.DataDesignFingerprint != ctx.DataDesign.Fingerprint ||
		packet.Candidate.Fingerprint != candidate.Fingerprint ||
		packet.Candidate.EvidenceRequirementFingerprint != candidate.EvidenceRequirementFingerprint {
		t.Fatalf("review packet did not bind current Reference candidate: %+v", packet)
	}
	if packet.SourceEvidenceStatus != "complete" ||
		packet.CandidateAmbiguityStatus != "clear" ||
		packet.BusinessEvidenceStatus != "unresolved" ||
		packet.ReviewStatus != "evidence-required" ||
		packet.PromotionStatus != "blocked" ||
		packet.FirstClassAuthoringContractStatus != "unresolved" {
		t.Fatalf(
			"source=%q ambiguity=%q business=%q review=%q promotion=%q authoring=%q",
			packet.SourceEvidenceStatus,
			packet.CandidateAmbiguityStatus,
			packet.BusinessEvidenceStatus,
			packet.ReviewStatus,
			packet.PromotionStatus,
			packet.FirstClassAuthoringContractStatus,
		)
	}
	for _, reason := range []string{
		"current-core-has-no-first-class-business-identity-authoring-contract",
		"semantic-extension-route-cannot-yet-losslessly-author-composite-identity-components-and-scope",
		"first-class-promotion-route-core-vs-semantic-extension-not-selected",
		"semantic-extension-route-is-not-accepted-production-authoring-contract",
		"authoritative-declaration-source-not-selected",
		"semantic-extension-namespace-authority-not-selected",
	} {
		if !containsDataDesignCandidateReviewString(packet.FirstClassAuthoringContractReasons, reason) {
			t.Fatalf("first-class authoring reason %q is missing: %+v", reason, packet.FirstClassAuthoringContractReasons)
		}
	}
	for _, requirement := range packet.BusinessEvidenceRequirements {
		if requirement.Status != "unresolved" {
			t.Fatalf("business Evidence was inferred from source trace: %+v", requirement)
		}
	}
}

func TestDataDesignCandidateReviewFingerprintGuardsCachedReviewState(t *testing.T) {
	root := testRepositoryRoot(t)
	ctx, err := buildKnowledgeContext(
		root+"/reference/workspace",
		"semantic-data",
		"sales.order_line",
	)
	if err != nil {
		t.Fatal(err)
	}
	candidate := ctx.DataDesign.BusinessIdentityCandidates[0]
	packet, err := buildDataDesignCandidateReview(
		root+"/reference/workspace",
		"semantic-data",
		"sales.order_line",
		knowledgecontext.DataDesignCandidateKindBusinessIdentity,
		candidate.Fingerprint,
		candidate.EvidenceRequirementFingerprint,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkDataDesignCandidateReviewFingerprint(packet, packet.Fingerprint); err != nil {
		t.Fatalf("current candidate review fingerprint was rejected: %v", err)
	}
	if err := checkDataDesignCandidateReviewFingerprint(
		packet,
		"sha256:"+strings.Repeat("0", 64),
	); err == nil || !strings.Contains(err.Error(), "candidate review fingerprint mismatch") {
		t.Fatalf("stale candidate review fingerprint was accepted: %v", err)
	}
}

func containsDataDesignCandidateReviewString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestDataDesignCandidateReviewRejectsStaleReferenceFingerprints(t *testing.T) {
	root := testRepositoryRoot(t)
	ctx, err := buildKnowledgeContext(
		root+"/reference/workspace",
		"semantic-data",
		"sales.order_line",
	)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.DataDesign == nil || len(ctx.DataDesign.BusinessIdentityCandidates) == 0 {
		t.Fatalf("reference candidate is missing: %+v", ctx.DataDesign)
	}
	candidate := ctx.DataDesign.BusinessIdentityCandidates[0]

	if err := checkDataDesignFingerprint(
		ctx,
		"sha256:"+strings.Repeat("0", 64),
	); err == nil || !strings.Contains(err.Error(), "fingerprint mismatch") {
		t.Fatalf("stale Data Design fingerprint was accepted: %v", err)
	}

	if _, err := buildDataDesignCandidateReview(
		root+"/reference/workspace",
		"semantic-data",
		"sales.order_line",
		knowledgecontext.DataDesignCandidateKindBusinessIdentity,
		"sha256:"+strings.Repeat("0", 64),
		candidate.EvidenceRequirementFingerprint,
	); err == nil || !strings.Contains(err.Error(), "not present in current Context") {
		t.Fatalf("stale candidate fingerprint was accepted: %v", err)
	}

	if _, err := buildDataDesignCandidateReview(
		root+"/reference/workspace",
		"semantic-data",
		"sales.order_line",
		knowledgecontext.DataDesignCandidateKindBusinessIdentity,
		candidate.Fingerprint,
		"sha256:"+strings.Repeat("0", 64),
	); err == nil || !strings.Contains(err.Error(), "evidence requirement fingerprint mismatch") {
		t.Fatalf("stale Evidence requirement contract was accepted: %v", err)
	}
}

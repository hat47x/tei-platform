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
	if packet.ReviewStatus != "evidence-required" ||
		packet.FirstClassAuthoringContractStatus != "unresolved" {
		t.Fatalf("review status=%q authoring=%q", packet.ReviewStatus, packet.FirstClassAuthoringContractStatus)
	}
	for _, requirement := range packet.BusinessEvidenceRequirements {
		if requirement.Status != "unresolved" {
			t.Fatalf("business Evidence was inferred from source trace: %+v", requirement)
		}
	}
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

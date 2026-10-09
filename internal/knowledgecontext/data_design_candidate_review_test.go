package knowledgecontext

import (
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/subject"
)

func TestDataDesignCandidateReviewBindsCurrentBusinessIdentityCandidate(t *testing.T) {
	ctx := dataDesignCandidateReviewContext(t)
	if ctx.DataDesign == nil || len(ctx.DataDesign.BusinessIdentityCandidates) != 1 {
		t.Fatalf("business identity candidate is missing: %+v", ctx.DataDesign)
	}
	candidate := ctx.DataDesign.BusinessIdentityCandidates[0]

	packet, err := BuildDataDesignCandidateReview(
		ctx,
		DataDesignCandidateKindBusinessIdentity,
		candidate.Fingerprint,
		candidate.EvidenceRequirementFingerprint,
	)
	if err != nil {
		t.Fatal(err)
	}
	if packet.ProjectionRevision != dataDesignCandidateReviewProjectionRevision ||
		packet.Fingerprint == "" ||
		packet.Subject != ctx.Subject ||
		packet.DataDesignFingerprint != ctx.DataDesign.Fingerprint {
		t.Fatalf("review identity is incomplete: %+v", packet)
	}
	if packet.Candidate.Kind != DataDesignCandidateKindBusinessIdentity ||
		packet.Candidate.Fingerprint != candidate.Fingerprint ||
		packet.Candidate.EvidenceRequirementFingerprint != candidate.EvidenceRequirementFingerprint ||
		packet.Candidate.BasisConstraint != candidate.BasisConstraint ||
		packet.Candidate.Relation != candidate.Relation ||
		packet.Candidate.TargetOccurrence != candidate.TargetOccurrence ||
		len(packet.Candidate.Terms) != len(candidate.Components) {
		t.Fatalf("candidate projection drifted: %+v want=%+v", packet.Candidate, candidate)
	}
	if packet.ReviewStatus != "evidence-required" ||
		packet.FirstClassAuthoringContractStatus != "unresolved" {
		t.Fatalf("review status=%q authoring=%q", packet.ReviewStatus, packet.FirstClassAuthoringContractStatus)
	}
	for _, reason := range []string{
		"current-core-has-no-first-class-business-identity-authoring-contract",
		"semantic-extension-v0-constraint-cannot-losslessly-encode-composite-identity-components-and-scope",
		"first-class-promotion-route-core-vs-semantic-extension-not-selected",
		"semantic-extension-v0-is-reference-not-production-authoring-contract",
		"authoritative-declaration-source-not-selected",
		"semantic-extension-namespace-authority-not-selected",
	} {
		if !containsDataDesignReviewString(packet.FirstClassAuthoringContractReasons, reason) {
			t.Fatalf("first-class authoring reason %q is missing: %+v", reason, packet.FirstClassAuthoringContractReasons)
		}
	}
	if len(packet.SourceEvidenceMissing) != 0 {
		t.Fatalf("candidate-specific source Evidence is missing: %+v", packet.SourceEvidenceMissing)
	}
	if len(packet.SourceEvidence) != 5 {
		t.Fatalf("candidate-specific source Evidence=%+v", packet.SourceEvidence)
	}
	if len(packet.BusinessEvidenceRequirements) != len(candidate.EvidenceNeeded) {
		t.Fatalf("requirements=%+v want=%+v", packet.BusinessEvidenceRequirements, candidate.EvidenceNeeded)
	}
	for _, requirement := range packet.BusinessEvidenceRequirements {
		if requirement.Status != "unresolved" {
			t.Fatalf("business Evidence was inferred as satisfied: %+v", requirement)
		}
	}
	for _, limit := range []string{
		"candidate-review-packet-is-derived-not-authority",
		"source-evidence-proves-traceability-not-business-meaning",
		"candidate-review-does-not-create-first-class-business-identity-or-functional-dependency",
		"semantic-extension-reference-is-not-current-lossless-authoring-contract",
	} {
		if !containsDataDesignReviewString(packet.AuthorityLimits, limit) {
			t.Fatalf("authority limit %q is missing: %+v", limit, packet.AuthorityLimits)
		}
	}

	again, err := BuildDataDesignCandidateReview(
		ctx,
		DataDesignCandidateKindBusinessIdentity,
		candidate.Fingerprint,
		candidate.EvidenceRequirementFingerprint,
	)
	if err != nil {
		t.Fatal(err)
	}
	if again.Fingerprint != packet.Fingerprint {
		t.Fatalf("review packet is not deterministic: %s/%s", packet.Fingerprint, again.Fingerprint)
	}
}

func TestDataDesignCandidateReviewBindsCurrentFunctionalDependencyCandidate(t *testing.T) {
	ctx := dataDesignCandidateReviewContext(t)
	if ctx.DataDesign == nil || len(ctx.DataDesign.FunctionalDependencyCandidates) != 1 {
		t.Fatalf("FD candidate is missing: %+v", ctx.DataDesign)
	}
	candidate := ctx.DataDesign.FunctionalDependencyCandidates[0]

	packet, err := BuildDataDesignCandidateReview(
		ctx,
		DataDesignCandidateKindFunctionalDependency,
		candidate.Fingerprint,
		candidate.EvidenceRequirementFingerprint,
	)
	if err != nil {
		t.Fatal(err)
	}
	if packet.Candidate.Kind != DataDesignCandidateKindFunctionalDependency ||
		packet.Candidate.TargetOccurrence != candidate.DependentOccurrence ||
		len(packet.Candidate.Terms) != len(candidate.Determinants) {
		t.Fatalf("FD candidate projection drifted: %+v want=%+v", packet.Candidate, candidate)
	}
	if packet.ReviewStatus != "evidence-required" ||
		packet.FirstClassAuthoringContractStatus != "unresolved" {
		t.Fatalf("review status=%q authoring=%q", packet.ReviewStatus, packet.FirstClassAuthoringContractStatus)
	}
	for _, reason := range []string{
		"current-core-has-no-first-class-functional-dependency-authoring-contract",
		"semantic-extension-v0-constraint-cannot-losslessly-encode-determinants-dependent-occurrence-and-scope",
		"semantic-extension-v0-is-reference-not-production-authoring-contract",
		"authoritative-declaration-source-and-namespace-authority-not-selected",
	} {
		if !containsDataDesignReviewString(packet.FirstClassAuthoringContractReasons, reason) {
			t.Fatalf("FD first-class authoring reason %q is missing: %+v", reason, packet.FirstClassAuthoringContractReasons)
		}
	}
	for _, requirement := range packet.BusinessEvidenceRequirements {
		if requirement.Status != "unresolved" {
			t.Fatalf("FD Evidence was inferred as satisfied: %+v", requirement)
		}
	}
}

func TestDataDesignCandidateReviewRejectsStaleCandidateAndEvidenceContract(t *testing.T) {
	ctx := dataDesignCandidateReviewContext(t)
	candidate := ctx.DataDesign.BusinessIdentityCandidates[0]

	if _, err := BuildDataDesignCandidateReview(
		ctx,
		DataDesignCandidateKindBusinessIdentity,
		"sha256:"+strings.Repeat("0", 64),
		candidate.EvidenceRequirementFingerprint,
	); err == nil || !strings.Contains(err.Error(), "not present in current Context") {
		t.Fatalf("stale candidate fingerprint was accepted: %v", err)
	}

	if _, err := BuildDataDesignCandidateReview(
		ctx,
		DataDesignCandidateKindBusinessIdentity,
		candidate.Fingerprint,
		"sha256:"+strings.Repeat("0", 64),
	); err == nil || !strings.Contains(err.Error(), "evidence requirement fingerprint mismatch") {
		t.Fatalf("stale Evidence requirement fingerprint was accepted: %v", err)
	}
}

func TestDataDesignCandidateReviewBlocksWhenRelevantSourceEvidenceIsMissing(t *testing.T) {
	w := dataDesignRelationUniqueWorld()
	target := subject.Subject{Kind: "semantic-data", ID: "sales.order_line"}
	sources := dataDesignCandidateReviewSources()
	sources.References = sources.References[:len(sources.References)-1]

	ctx := Build(w, sources, target)
	if ctx.DataDesign == nil || len(ctx.DataDesign.BusinessIdentityCandidates) != 1 {
		t.Fatalf("candidate is missing: %+v", ctx.DataDesign)
	}
	candidate := ctx.DataDesign.BusinessIdentityCandidates[0]

	packet, err := BuildDataDesignCandidateReview(
		ctx,
		DataDesignCandidateKindBusinessIdentity,
		candidate.Fingerprint,
		candidate.EvidenceRequirementFingerprint,
	)
	if err != nil {
		t.Fatal(err)
	}
	if packet.ReviewStatus != "blocked-context-unresolved" ||
		len(packet.SourceEvidenceMissing) != 1 ||
		packet.SourceEvidenceMissing[0].Kind != "semantic-validation" {
		t.Fatalf("missing relevant Evidence did not block review: %+v", packet)
	}
}

func dataDesignCandidateReviewContext(t *testing.T) Context {
	t.Helper()
	return Build(
		dataDesignRelationUniqueWorld(),
		dataDesignCandidateReviewSources(),
		subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
	)
}

func dataDesignCandidateReviewSources() sourceindex.Index {
	const path = "semantic/sales/order.tei"
	artifact := sourceindex.Artifact{
		Path:        path,
		Fingerprint: sourceindex.Fingerprint([]byte("candidate-review-source")),
	}
	subjects := []subject.Subject{
		{Kind: "semantic-data", ID: "sales.order"},
		{Kind: "semantic-data", ID: "sales.order_line"},
		{Kind: "semantic-attribute", ID: "sales.order_line.line_number"},
		{Kind: "semantic-relation", ID: "sales.order.lines"},
		{Kind: "semantic-validation", ID: "sales.order.lines.line_number_unique"},
	}
	references := make([]sourceindex.Reference, 0, len(subjects))
	for _, item := range subjects {
		references = append(references, sourceindex.Reference{
			Subject:      item,
			ArtifactPath: path,
			Role:         sourceindex.RoleDefinition,
		})
	}
	return sourceindex.Index{
		Artifacts:  []sourceindex.Artifact{artifact},
		References: references,
	}
}

func containsDataDesignReviewString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

package knowledgecontext

import (
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/subject"
)

func TestDataDesignCandidateReviewPacketIsNotAnAuthorityEnvelope(t *testing.T) {
	typ := reflect.TypeOf(DataDesignCandidateReviewPacket{})
	for _, forbidden := range []string{
		"Accepted",
		"Approved",
		"Authorized",
		"CanWrite",
		"CanExecute",
		"Truth",
		"Decision",
		"ResolvedBusinessEvidence",
		"CanonicalMeaning",
		"ProposalDelta",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("candidate review packet introduced forbidden authority field %q", forbidden)
		}
	}
}

func TestBuilderReviewFlowReachesDataDesignCandidateReview(t *testing.T) {
	ctx := dataDesignCandidateReviewContext(t)
	sources := dataDesignCandidateReviewSources()

	plan := BuildBuilderReviewPlan(ctx)
	if !containsDataDesignReviewString(
		plan.ReviewSequence,
		"review-data-design-candidates-by-stable-fingerprint-and-evidence-contract",
	) {
		t.Fatalf("Builder Review Plan does not route candidate review: %+v", plan.ReviewSequence)
	}

	handoff, err := BuildBuilderReviewHandoff(
		ctx,
		sources,
		QualificationRequirements{
			ExpectedKind: "semantic-data",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if handoff.Status != "qualified-for-disclosure" ||
		!containsDataDesignReviewString(handoff.SelectedSections, "data-design") {
		t.Fatalf("Builder Review Handoff did not select Data Design: %+v", handoff)
	}

	disclosure, err := BuildBuilderReviewDisclosure(ctx, sources, handoff)
	if err != nil {
		t.Fatal(err)
	}
	if disclosure.DataDesign == nil ||
		len(disclosure.DataDesign.BusinessIdentityCandidates) != 1 {
		t.Fatalf("qualified Disclosure lost Data Design candidate: %+v", disclosure.DataDesign)
	}
	candidate := disclosure.DataDesign.BusinessIdentityCandidates[0]

	packet, err := BuildDataDesignCandidateReview(
		ctx,
		DataDesignCandidateKindBusinessIdentity,
		candidate.Fingerprint,
		candidate.EvidenceRequirementFingerprint,
	)
	if err != nil {
		t.Fatal(err)
	}
	if packet.Candidate.Fingerprint != candidate.Fingerprint ||
		packet.DataDesignFingerprint != disclosure.DataDesign.Fingerprint ||
		packet.ReviewStatus != "evidence-required" {
		t.Fatalf("candidate review did not bind qualified Disclosure: %+v", packet)
	}
}

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
		"business-evidence-unresolved",
		"first-class-authoring-contract-unresolved",
	} {
		if !containsDataDesignReviewString(packet.PromotionBlockingReasons, reason) {
			t.Fatalf("promotion blocking reason %q is missing: %+v", reason, packet.PromotionBlockingReasons)
		}
	}
	for _, reason := range []string{
		"current-core-has-no-first-class-business-identity-authoring-contract",
		"semantic-extension-route-cannot-yet-losslessly-author-composite-identity-components-and-scope",
		"first-class-promotion-route-core-vs-semantic-extension-not-selected",
		"semantic-extension-route-is-not-accepted-production-authoring-contract",
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
	for _, tc := range []struct {
		requirement string
		subjects    []subject.Subject
	}{
		{
			requirement: "source-occurrence-business-identifier",
			subjects: []subject.Subject{
				{Kind: "semantic-data", ID: "sales.order"},
			},
		},
		{
			requirement: "relation-business-meaning",
			subjects: []subject.Subject{
				{Kind: "semantic-relation", ID: "sales.order.lines"},
			},
		},
		{
			requirement: "target-membership-exclusivity",
			subjects: []subject.Subject{
				{Kind: "semantic-data", ID: "sales.order_line"},
				{Kind: "semantic-relation", ID: "sales.order.lines"},
			},
		},
		{
			requirement: "identifier-stability",
			subjects: []subject.Subject{
				{Kind: "semantic-attribute", ID: "sales.order_line.line_number"},
				{Kind: "semantic-data", ID: "sales.order_line"},
			},
		},
	} {
		requirement, ok := findDataDesignReviewRequirement(
			packet.BusinessEvidenceRequirements,
			tc.requirement,
		)
		if !ok {
			t.Fatalf("business Evidence requirement %q is missing", tc.requirement)
		}
		if !reflect.DeepEqual(requirement.SuggestedSubjects, tc.subjects) {
			t.Fatalf(
				"search starts for %q=%+v want=%+v",
				tc.requirement,
				requirement.SuggestedSubjects,
				tc.subjects,
			)
		}
	}
	for _, step := range []string{
		"rebuild-current-data-design-context-and-verify-fingerprint",
		"review-first-class-authoring-contract-gaps",
		"select-core-or-semantic-extension-authoring-route-with-evidence",
		"require-lossless-authoring-contract-before-promoting-derived-candidate",
		"route-authoritative-change-through-shared-acceptance-only-after-authoring-contract-exists",
	} {
		if !containsDataDesignReviewString(packet.ReviewSequence, step) {
			t.Fatalf("review sequence step %q is missing: %+v", step, packet.ReviewSequence)
		}
	}
	for _, limit := range []string{
		"candidate-review-packet-is-derived-not-authority",
		"review-status-is-review-readiness-not-promotion-authority",
		"promotion-status-is-derived-readiness-not-acceptance-or-write-authority",
		"data-design-context-fingerprint-is-snapshot-identity-not-source-freshness-proof",
		"source-evidence-proves-traceability-not-business-meaning",
		"suggested-evidence-subject-is-search-start-not-evidence-or-authority",
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
	if packet.SourceEvidenceStatus != "complete" ||
		packet.CandidateAmbiguityStatus != "clear" ||
		packet.BusinessEvidenceStatus != "unresolved" ||
		packet.ReviewStatus != "evidence-required" ||
		packet.PromotionStatus != "blocked" ||
		packet.FirstClassAuthoringContractStatus != "unresolved" {
		t.Fatalf(
			"FD source=%q ambiguity=%q business=%q review=%q promotion=%q authoring=%q",
			packet.SourceEvidenceStatus,
			packet.CandidateAmbiguityStatus,
			packet.BusinessEvidenceStatus,
			packet.ReviewStatus,
			packet.PromotionStatus,
			packet.FirstClassAuthoringContractStatus,
		)
	}
	for _, reason := range []string{
		"current-core-has-no-first-class-functional-dependency-authoring-contract",
		"semantic-extension-route-cannot-yet-losslessly-author-determinants-dependent-occurrence-and-scope",
		"semantic-extension-route-is-not-accepted-production-authoring-contract",
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

func TestDataDesignEvidenceSearchHintsFailClosedForUnknownRequirement(t *testing.T) {
	ctx := dataDesignCandidateReviewContext(t)
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
	if len(packet.Candidate.Terms) == 0 {
		t.Fatal("candidate terms are missing")
	}
	if got := dataDesignEvidenceSearchSubjects(
		packet.Candidate,
		"future-unmodeled-business-evidence",
	); len(got) != 0 {
		t.Fatalf("unknown Evidence requirement received speculative search hints: %+v", got)
	}
}

func TestDataDesignCandidateReviewRejectsModifiedContextWithStaleFingerprint(t *testing.T) {
	ctx := dataDesignCandidateReviewContext(t)
	candidate := ctx.DataDesign.BusinessIdentityCandidates[0]
	ctx.DataDesign.AuthorityLimits = append(
		ctx.DataDesign.AuthorityLimits,
		"tampered-without-recomputing-context-fingerprint",
	)

	_, err := BuildDataDesignCandidateReview(
		ctx,
		DataDesignCandidateKindBusinessIdentity,
		candidate.Fingerprint,
		candidate.EvidenceRequirementFingerprint,
	)
	if err == nil || !strings.Contains(err.Error(), "unmodified current Data Design Context") {
		t.Fatalf("modified Context with stale fingerprint was accepted: %v", err)
	}
}

func TestDataDesignCandidateReviewRejectsModifiedCandidateBehindReFingerprintedContext(t *testing.T) {
	ctx := dataDesignCandidateReviewContext(t)
	candidate := ctx.DataDesign.BusinessIdentityCandidates[0]
	ctx.DataDesign.BusinessIdentityCandidates[0].Components[1].Attribute =
		"sales.order_line.changed_line_number"
	ctx.DataDesign.Fingerprint = dataDesignFingerprint(ctx.DataDesign)

	_, err := BuildDataDesignCandidateReview(
		ctx,
		DataDesignCandidateKindBusinessIdentity,
		candidate.Fingerprint,
		candidate.EvidenceRequirementFingerprint,
	)
	if err == nil || !strings.Contains(err.Error(), "candidate fingerprint does not match current candidate content") {
		t.Fatalf("modified candidate behind re-fingerprinted Context was accepted: %v", err)
	}
}

func TestDataDesignCandidateReviewRejectsModifiedEvidenceContractBehindReFingerprintedContext(t *testing.T) {
	ctx := dataDesignCandidateReviewContext(t)
	candidate := ctx.DataDesign.BusinessIdentityCandidates[0]
	ctx.DataDesign.BusinessIdentityCandidates[0].EvidenceNeeded = append(
		ctx.DataDesign.BusinessIdentityCandidates[0].EvidenceNeeded,
		"new-business-evidence-requirement",
	)
	ctx.DataDesign.Fingerprint = dataDesignFingerprint(ctx.DataDesign)

	_, err := BuildDataDesignCandidateReview(
		ctx,
		DataDesignCandidateKindBusinessIdentity,
		candidate.Fingerprint,
		candidate.EvidenceRequirementFingerprint,
	)
	if err == nil || !strings.Contains(err.Error(), "Evidence requirement fingerprint does not match current requirement content") {
		t.Fatalf("modified Evidence contract behind re-fingerprinted Context was accepted: %v", err)
	}
}

func TestDataDesignCandidateReviewRejectsModifiedFunctionalDependencyBehindReFingerprintedContext(t *testing.T) {
	ctx := dataDesignCandidateReviewContext(t)
	candidate := ctx.DataDesign.FunctionalDependencyCandidates[0]
	ctx.DataDesign.FunctionalDependencyCandidates[0].Determinants[1].Attribute =
		"sales.order_line.changed_line_number"
	ctx.DataDesign.Fingerprint = dataDesignFingerprint(ctx.DataDesign)

	_, err := BuildDataDesignCandidateReview(
		ctx,
		DataDesignCandidateKindFunctionalDependency,
		candidate.Fingerprint,
		candidate.EvidenceRequirementFingerprint,
	)
	if err == nil || !strings.Contains(err.Error(), "candidate fingerprint does not match current candidate content") {
		t.Fatalf("modified FD candidate behind re-fingerprinted Context was accepted: %v", err)
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

func TestDataDesignCandidateReviewBlocksOnlyOnCandidateLocalAmbiguity(t *testing.T) {
	ctx := dataDesignCandidateReviewContext(t)
	candidate := ctx.DataDesign.BusinessIdentityCandidates[0]
	ctx.DataDesign.ResolutionStatus = "ambiguous"
	ctx.DataDesign.Ambiguities = []DataDesignAmbiguity{
		{
			Kind:   "duplicate-persistence-mapping",
			ID:     "sales_core.order_line",
			Count:  2,
			Effect: "realization-suppressed",
		},
	}
	ctx.DataDesign.Fingerprint = dataDesignFingerprint(ctx.DataDesign)

	packet, err := BuildDataDesignCandidateReview(
		ctx,
		DataDesignCandidateKindBusinessIdentity,
		candidate.Fingerprint,
		candidate.EvidenceRequirementFingerprint,
	)
	if err != nil {
		t.Fatal(err)
	}
	if packet.DataDesignResolutionStatus != "ambiguous" ||
		packet.SourceEvidenceStatus != "complete" ||
		packet.CandidateAmbiguityStatus != "clear" ||
		packet.ReviewStatus != "evidence-required" ||
		packet.PromotionStatus != "blocked" ||
		len(packet.ReviewBlockingAmbiguities) != 0 {
		t.Fatalf("unrelated persistence ambiguity blocked candidate review: %+v", packet)
	}

	ctx.DataDesign.Ambiguities = append(
		ctx.DataDesign.Ambiguities,
		DataDesignAmbiguity{
			Kind:   "duplicate-semantic-relation",
			ID:     "sales.order.lines",
			Count:  2,
			Effect: "design-candidates-suppressed",
		},
	)
	ctx.DataDesign.Fingerprint = dataDesignFingerprint(ctx.DataDesign)
	packet, err = BuildDataDesignCandidateReview(
		ctx,
		DataDesignCandidateKindBusinessIdentity,
		candidate.Fingerprint,
		candidate.EvidenceRequirementFingerprint,
	)
	if err != nil {
		t.Fatal(err)
	}
	if packet.CandidateAmbiguityStatus != "blocked" ||
		packet.ReviewStatus != "blocked-context-unresolved" ||
		packet.PromotionStatus != "blocked" ||
		len(packet.ReviewBlockingAmbiguities) != 1 ||
		packet.ReviewBlockingAmbiguities[0].Kind != "duplicate-semantic-relation" ||
		!containsDataDesignReviewString(
			packet.PromotionBlockingReasons,
			"candidate-local-ambiguity-unresolved",
		) {
		t.Fatalf("candidate-local ambiguity did not block review: %+v", packet)
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
	if packet.SourceEvidenceStatus != "incomplete" ||
		packet.CandidateAmbiguityStatus != "clear" ||
		packet.ReviewStatus != "blocked-context-unresolved" ||
		packet.PromotionStatus != "blocked" ||
		len(packet.SourceEvidenceMissing) != 1 ||
		packet.SourceEvidenceMissing[0].Kind != "semantic-validation" ||
		!containsDataDesignReviewString(
			packet.PromotionBlockingReasons,
			"candidate-source-evidence-incomplete",
		) {
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

func findDataDesignReviewRequirement(
	values []DataDesignCandidateReviewRequirement,
	id string,
) (DataDesignCandidateReviewRequirement, bool) {
	for _, value := range values {
		if value.ID == id {
			return value, true
		}
	}
	return DataDesignCandidateReviewRequirement{}, false
}

func containsDataDesignReviewString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

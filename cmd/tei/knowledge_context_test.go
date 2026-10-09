package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/knowledgecontext"
)

func TestBuildKnowledgeDisclosureUsesCurrentContextWithoutLeakingUnselectedKnowledge(t *testing.T) {
	root := testRepositoryRoot(t)
	ctx, err := buildKnowledgeContext(
		filepath.Join(root, "reference", "workspace"),
		"semantic-data",
		"sales.order_line",
	)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.DataDesign == nil || ctx.DataDesign.Fingerprint == "" {
		t.Fatalf("current Data Design Context is missing: %+v", ctx.DataDesign)
	}

	disclosure, err := buildKnowledgeDisclosure(ctx, "source,data-design")
	if err != nil {
		t.Fatal(err)
	}
	if disclosure.Status != "complete" ||
		len(disclosure.DisclosedSections) != 2 ||
		disclosure.DisclosedSections[0] != "data-design" ||
		disclosure.DisclosedSections[1] != "source" {
		t.Fatalf("disclosure sections=%+v", disclosure)
	}
	if disclosure.DataDesign == nil ||
		disclosure.DataDesign.Fingerprint != ctx.DataDesign.Fingerprint {
		t.Fatalf("disclosure changed Data Design identity: disclosure=%+v context=%+v", disclosure.DataDesign, ctx.DataDesign)
	}
	if len(disclosure.Artifacts) == 0 || len(disclosure.References) == 0 {
		t.Fatalf("selected source knowledge is missing: artifacts=%+v references=%+v", disclosure.Artifacts, disclosure.References)
	}
	if disclosure.SemanticElement != nil ||
		disclosure.ApplicationPattern != nil ||
		len(disclosure.Outgoing) != 0 ||
		len(disclosure.Incoming) != 0 {
		t.Fatalf("unselected Knowledge leaked into disclosure: %+v", disclosure)
	}
	for _, omitted := range []string{"definition", "dependencies"} {
		if !containsKnowledgeContextString(disclosure.OmittedAvailableSections, omitted) {
			t.Fatalf("available but omitted section %q is not recorded: %+v", omitted, disclosure.OmittedAvailableSections)
		}
	}
	if !containsKnowledgeContextString(
		disclosure.AuthorityLimits,
		"omitted-knowledge-is-not-negative-evidence",
	) {
		t.Fatalf("selective disclosure authority boundary is missing: %+v", disclosure.AuthorityLimits)
	}
}

func TestSelectiveDisclosureDoesNotChangeKnowledgeQualification(t *testing.T) {
	root := testRepositoryRoot(t)
	workspacePath := filepath.Join(root, "reference", "workspace")
	requirements := knowledgecontext.QualificationRequirements{
		ExpectedKind: "application-pattern",
		Application:  "order_entry",
		RequiredDependencies: []knowledgecontext.DependencyRequirement{{
			Direction: "outgoing",
			Relation:  "contract-query",
			Kind:      "database-contract-query",
			ID:        "master_data.customer/find_by_id",
		}},
	}

	before, err := buildKnowledgeQualification(
		workspacePath,
		"application-pattern",
		"order_entry.customer_reference",
		requirements,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := buildKnowledgeContext(
		workspacePath,
		"application-pattern",
		"order_entry.customer_reference",
	)
	if err != nil {
		t.Fatal(err)
	}
	disclosure, err := buildKnowledgeDisclosure(ctx, "source,definition")
	if err != nil {
		t.Fatal(err)
	}
	if disclosure.Status != "complete" {
		t.Fatalf("selected Knowledge is unexpectedly partial: %+v", disclosure)
	}

	after, err := buildKnowledgeQualification(
		workspacePath,
		"application-pattern",
		"order_entry.customer_reference",
		requirements,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("selective disclosure changed qualification:\nbefore=%+v\nafter=%+v", before, after)
	}
	if before.Status != knowledgecontext.QualificationApplicable ||
		len(before.Evidence) == 0 ||
		len(before.DependencyEvidence) == 0 {
		t.Fatalf("qualification invariant test lost applicable Evidence: %+v", before)
	}
}

func TestCheckKnowledgeDisclosureFingerprint(t *testing.T) {
	root := testRepositoryRoot(t)
	ctx, err := buildKnowledgeContext(
		filepath.Join(root, "reference", "workspace"),
		"semantic-data",
		"sales.order_line",
	)
	if err != nil {
		t.Fatal(err)
	}
	disclosure, err := buildKnowledgeDisclosure(ctx, "source,data-design")
	if err != nil {
		t.Fatal(err)
	}
	if disclosure.Status != "complete" || disclosure.Fingerprint == "" {
		t.Fatalf("complete disclosure identity is missing: %+v", disclosure)
	}
	if err := checkKnowledgeDisclosureFingerprint(disclosure, ""); err != nil {
		t.Fatalf("empty expected disclosure fingerprint enabled the guard: %v", err)
	}
	if err := checkKnowledgeDisclosureFingerprint(disclosure, disclosure.Fingerprint); err != nil {
		t.Fatalf("current disclosure fingerprint was rejected: %v", err)
	}
	if err := checkKnowledgeDisclosureFingerprint(disclosure, "sha256:stale"); err == nil ||
		!strings.Contains(err.Error(), "fingerprint mismatch") {
		t.Fatalf("stale disclosure fingerprint was not rejected clearly: %v", err)
	}

	partial, err := buildKnowledgeDisclosure(ctx, "data-design,missing-source")
	if err == nil {
		t.Fatalf("unknown section must be rejected before disclosure guard: %+v", partial)
	}

	capabilityCtx, err := buildKnowledgeContext(
		filepath.Join(root, "reference", "workspace"),
		"semantic-capability",
		"sales.order.submit",
	)
	if err != nil {
		t.Fatal(err)
	}
	partial, err = buildKnowledgeDisclosure(capabilityCtx, "definition,data-design")
	if err != nil {
		t.Fatal(err)
	}
	if partial.Status != "partial" || !containsKnowledgeContextString(partial.UnavailableSections, "data-design") {
		t.Fatalf("expected partial capability disclosure: %+v", partial)
	}
	if err := checkKnowledgeDisclosureFingerprint(partial, partial.Fingerprint); err == nil ||
		!strings.Contains(err.Error(), "requires complete selected Knowledge") {
		t.Fatalf("partial disclosure allowed cached reuse: %v", err)
	}
}

func TestBuildBuilderReviewPlanRoutesCurrentKnowledge(t *testing.T) {
	root := testRepositoryRoot(t)
	plan, err := buildBuilderReviewPlan(
		filepath.Join(root, "reference", "workspace"),
		"semantic-data",
		"sales.order_line",
	)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ProjectionRevision != "tei.knowledge.builder-review-plan/v1" ||
		plan.Purpose != "design-change" ||
		plan.Fingerprint == "" {
		t.Fatalf("builder review plan identity is incomplete: %+v", plan)
	}
	for _, section := range []string{"source", "definition", "data-design", "dependencies"} {
		if !containsKnowledgeContextString(plan.RecommendedDisclosureSections, section) {
			t.Fatalf("builder review section %q is missing: %+v", section, plan.RecommendedDisclosureSections)
		}
	}
	if len(plan.ReviewSequence) == 0 ||
		plan.ReviewSequence[0] != "review-current-source-and-evidence" {
		t.Fatalf("builder review sequence does not start from current Evidence: %+v", plan.ReviewSequence)
	}
	if err := checkBuilderReviewPlanFingerprint(plan, ""); err != nil {
		t.Fatalf("empty expected review-plan fingerprint enabled the guard: %v", err)
	}
	if err := checkBuilderReviewPlanFingerprint(plan, plan.Fingerprint); err != nil {
		t.Fatalf("current review-plan fingerprint was rejected: %v", err)
	}
	if err := checkBuilderReviewPlanFingerprint(plan, "sha256:stale"); err == nil ||
		!strings.Contains(err.Error(), "fingerprint mismatch") {
		t.Fatalf("stale review-plan fingerprint was not rejected: %v", err)
	}
}

func TestDataDesignContextExposesOrderedAIReviewProtocol(t *testing.T) {
	root := testRepositoryRoot(t)
	ctx, err := buildKnowledgeContext(
		filepath.Join(root, "reference", "workspace"),
		"semantic-data",
		"sales.order_line",
	)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.DataDesign == nil {
		t.Fatal("Data Design Context is missing")
	}
	if ctx.DataDesign.ProjectionRevision != "tei.knowledge.data-design/v6" {
		t.Fatalf("data design projection revision=%q", ctx.DataDesign.ProjectionRevision)
	}
	want := []string{
		"resolve-current-semantic-data-and-source-evidence-before-design-change",
		"separate-business-occurrence-identity-from-persistence-runtime-and-ui-identifiers",
		"establish-business-identifier-candidate-key-scope-time-and-equivalence-from-evidence",
		"preserve-accepted-business-uniqueness-when-using-surrogate-identifiers",
		"establish-functional-dependency-as-business-semantic-claim-not-sample-data-correlation",
		"distinguish-functional-dependency-from-composite-domain-constraint",
		"treat-relation-keys-scoped-uniqueness-and-persistence-keys-as-candidates-not-authority",
		"preserve-unresolved-when-required-business-evidence-is-missing",
		"route-derived-identity-and-dependency-proposals-through-shared-acceptance",
	}
	if !reflect.DeepEqual(ctx.DataDesign.DesignReviewProtocol, want) {
		t.Fatalf("data design review protocol drifted:\n got=%+v\nwant=%+v", ctx.DataDesign.DesignReviewProtocol, want)
	}
	if !containsKnowledgeContextString(
		ctx.DataDesign.AuthorityLimits,
		"data-design-review-protocol-is-derived-guidance-not-semantic-authority",
	) {
		t.Fatalf("review protocol authority limit is missing: %+v", ctx.DataDesign.AuthorityLimits)
	}
	for _, limit := range []string{
		"current-data-uniqueness-is-not-functional-dependency-proof",
		"statistical-correlation-is-not-functional-dependency",
		"functional-dependency-direction-is-not-symmetric-by-default",
		"discovered-functional-dependency-is-proposal-not-accepted-business-semantics",
	} {
		if !containsKnowledgeContextString(ctx.DataDesign.AuthorityLimits, limit) {
			t.Fatalf("functional dependency authority limit %q is missing: %+v", limit, ctx.DataDesign.AuthorityLimits)
		}
	}
	if !containsKnowledgeContextString(
		ctx.DataDesign.FunctionalDependencyEvidenceQuestions,
		"is-the-dependency-an-accepted-business-rule-or-only-observed-in-current-data",
	) {
		t.Fatalf("functional dependency evidence question is missing: %+v", ctx.DataDesign.FunctionalDependencyEvidenceQuestions)
	}

	disclosure, err := buildKnowledgeDisclosure(ctx, "data-design")
	if err != nil {
		t.Fatal(err)
	}
	if disclosure.DataDesign == nil || !reflect.DeepEqual(disclosure.DataDesign.DesignReviewProtocol, want) {
		t.Fatalf("data-design disclosure lost review protocol: %+v", disclosure.DataDesign)
	}
}

func TestCheckDataDesignFingerprint(t *testing.T) {
	root := testRepositoryRoot(t)
	ctx, err := buildKnowledgeContext(
		filepath.Join(root, "reference", "workspace"),
		"semantic-data",
		"sales.order_line",
	)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.DataDesign == nil || ctx.DataDesign.Fingerprint == "" {
		t.Fatalf("Data Design Context fingerprint is missing: %+v", ctx.DataDesign)
	}
	if err := checkDataDesignFingerprint(ctx, ""); err != nil {
		t.Fatalf("empty expected fingerprint must not enable the guard: %v", err)
	}
	if err := checkDataDesignFingerprint(ctx, ctx.DataDesign.Fingerprint); err != nil {
		t.Fatalf("current Data Design fingerprint was rejected: %v", err)
	}
	if err := checkDataDesignFingerprint(ctx, "sha256:stale"); err == nil ||
		!strings.Contains(err.Error(), "fingerprint mismatch") {
		t.Fatalf("stale Data Design fingerprint was not rejected clearly: %v", err)
	}

	ambiguous := ctx
	ambiguous.DataDesign = &knowledgecontext.DataDesignContext{
		Fingerprint:            ctx.DataDesign.Fingerprint,
		ResolutionStatus:       "ambiguous",
		EvidenceCoverageStatus: "complete",
		Ambiguities: []knowledgecontext.DataDesignAmbiguity{
			{
				Kind:   "duplicate-semantic-data",
				ID:     "sales.order_line",
				Count:  2,
				Effect: "design-candidates-suppressed",
			},
		},
	}
	if err := checkDataDesignFingerprint(ambiguous, ctx.DataDesign.Fingerprint); err == nil ||
		!strings.Contains(err.Error(), "requires structurally resolved identity") {
		t.Fatalf("ambiguous Data Design Context allowed cached reuse: %v", err)
	}

	incomplete := ctx
	incomplete.DataDesign = &knowledgecontext.DataDesignContext{
		Fingerprint:            ctx.DataDesign.Fingerprint,
		ResolutionStatus:       "resolved",
		EvidenceCoverageStatus: "incomplete",
	}
	if err := checkDataDesignFingerprint(incomplete, ctx.DataDesign.Fingerprint); err == nil ||
		!strings.Contains(err.Error(), "requires complete Evidence coverage") {
		t.Fatalf("incomplete Evidence coverage allowed cached Context reuse: %v", err)
	}

	withoutDataDesign := knowledgecontext.Context{}
	if err := checkDataDesignFingerprint(withoutDataDesign, ctx.DataDesign.Fingerprint); err == nil ||
		!strings.Contains(err.Error(), "no Data Design Context") {
		t.Fatalf("missing Data Design Context did not fail closed: %v", err)
	}
}

func TestBuildKnowledgeContextReturnsScopedApplicationPatternContext(t *testing.T) {
	root := testRepositoryRoot(t)
	got, err := buildKnowledgeContext(
		filepath.Join(root, "reference", "workspace"),
		"application-pattern",
		"order_entry.customer_reference",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ApplicationPattern == nil {
		t.Fatalf("application pattern detail is missing: %+v", got)
	}
	if got.ApplicationPattern.Application != "order_entry" ||
		got.ApplicationPattern.Pattern != "order_entry.customer_reference" ||
		got.ApplicationPattern.Kind != "reference" ||
		len(got.ApplicationPattern.ContractQueryInputs) != 1 ||
		got.ApplicationPattern.ContractQueryInputs[0].TargetAttribute != "customer.customer.customer_id" ||
		got.ApplicationPattern.ContractQueryInputs[0].Input != "id" {
		t.Fatalf("application pattern detail drifted: %+v", got.ApplicationPattern)
	}
	if len(got.References) == 0 || len(got.Artifacts) == 0 {
		t.Fatalf("source trace is missing: references=%+v artifacts=%+v", got.References, got.Artifacts)
	}
	if len(got.Unresolved) != 0 {
		t.Fatalf("known application pattern has unresolved context: %+v", got.Unresolved)
	}

	for _, want := range []struct {
		direction string
		relation  string
		kind      string
		id        string
	}{
		{direction: "incoming", relation: "owns-pattern", kind: "application", id: "order_entry"},
		{direction: "outgoing", relation: "relation", kind: "semantic-relation", id: "sales.order.customer"},
		{direction: "outgoing", relation: "contract", kind: "database-contract", id: "master_data.customer"},
		{direction: "outgoing", relation: "contract-query", kind: "database-contract-query", id: "master_data.customer/find_by_id"},
		{direction: "outgoing", relation: "contract-query-input", kind: "semantic-attribute", id: "customer.customer.customer_id"},
	} {
		if !hasKnowledgeContextEdge(got, want.direction, want.relation, want.kind, want.id) {
			t.Fatalf("knowledge context edge is missing: %+v context=%+v", want, got)
		}
	}

	for _, limit := range []string{
		"derived-context-is-not-canonical-authority",
		"dependency-edge-is-not-semantic-applicability",
		"dependency-edge-is-not-write-or-runtime-authority",
		"projection-or-runtime-identity-is-not-business-data-identity",
	} {
		if !containsKnowledgeContextString(got.AuthorityLimits, limit) {
			t.Fatalf("authority limit %q is missing: %+v", limit, got.AuthorityLimits)
		}
	}
}

func TestContractQuerySQLRealizationReverseTracesByExplicitResultMappingOnly(t *testing.T) {
	root := testRepositoryRoot(t)
	const realizationID = "sales_core/sales_core.order_line/find_by_order/sql/postgres"
	got, err := buildKnowledgeContext(
		filepath.Join(root, "reference", "workspace"),
		"contract-query-sql-realization",
		realizationID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.DataDesign == nil ||
		got.DataDesign.Data != "sales.order_line" ||
		got.DataDesign.Focus.Kind != "contract-query-sql-realization" ||
		got.DataDesign.Focus.ID != realizationID {
		t.Fatalf("SQL realization did not reverse trace to Semantic Data: %+v", got.DataDesign)
	}
	if len(got.References) == 0 || len(got.Artifacts) == 0 {
		t.Fatalf("SQL realization source trace is missing: references=%+v artifacts=%+v", got.References, got.Artifacts)
	}
	if !hasDataDesignEvidence(
		got.DataDesign,
		"contract-query-sql-realization",
		realizationID,
		"databases/sales-core/query-realizations/order-line-find-by-order.tei",
	) {
		t.Fatalf("SQL realization authoring Evidence is missing: %+v", got.DataDesign.Evidence)
	}
	if got.DataDesign.BusinessOccurrenceIdentifier.Nature != "unresolved" ||
		got.DataDesign.CandidateKeys.Nature != "unresolved" ||
		got.DataDesign.FunctionalDependencies.Nature != "unresolved" {
		t.Fatalf("SQL realization identity was promoted to business identity: %+v", got.DataDesign)
	}
	for _, candidate := range got.DataDesign.BusinessIdentityCandidates {
		for _, component := range candidate.Components {
			if component.Attribute == "sales.order_line.order_line_id" {
				t.Fatalf("raw SQL result identity leaked into business identity candidate: %+v", candidate)
			}
		}
	}
	for _, limit := range []string{
		"contract-query-sql-realization-identity-is-not-business-data-identity",
		"raw-result-column-is-not-business-identity",
		"provider-specific-realization-is-not-semantic-authority",
	} {
		if !containsKnowledgeContextString(got.DataDesign.AuthorityLimits, limit) {
			t.Fatalf("SQL realization authority limit %q is missing: %+v", limit, got.DataDesign.AuthorityLimits)
		}
	}
}

func TestPersistenceMappingReverseTracesToSemanticDataWithoutInventingBusinessIdentity(t *testing.T) {
	root := testRepositoryRoot(t)
	got, err := buildKnowledgeContext(
		filepath.Join(root, "reference", "workspace"),
		"persistence",
		"sales_core.order_line",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.DataDesign == nil ||
		got.DataDesign.Data != "sales.order_line" ||
		got.DataDesign.Focus.Kind != "persistence" ||
		got.DataDesign.Focus.ID != "sales_core.order_line" {
		t.Fatalf("persistence reverse trace did not reach Semantic Data: %+v", got.DataDesign)
	}
	if len(got.References) == 0 || len(got.Artifacts) == 0 {
		t.Fatalf("persistence source trace is missing: references=%+v artifacts=%+v", got.References, got.Artifacts)
	}
	if len(got.DataDesign.PersistenceRealizations) != 1 {
		t.Fatalf("persistence realization is missing: %+v", got.DataDesign.PersistenceRealizations)
	}
	realization := got.DataDesign.PersistenceRealizations[0]
	if realization.Mapping != "sales_core.order_line" ||
		len(realization.Uniqueness) != 1 ||
		realization.Uniqueness[0].Kind != "primary-key" ||
		len(realization.Uniqueness[0].Attributes) != 1 ||
		realization.Uniqueness[0].Attributes[0] != "sales.order_line.order_line_id" {
		t.Fatalf("physical key realization drifted: %+v", realization)
	}
	if got.DataDesign.BusinessOccurrenceIdentifier.Nature != "unresolved" ||
		got.DataDesign.CandidateKeys.Nature != "unresolved" ||
		got.DataDesign.FunctionalDependencies.Nature != "unresolved" {
		t.Fatalf("persistence reverse trace invented logical identity: %+v", got.DataDesign)
	}
	if !hasDataDesignEvidence(
		got.DataDesign,
		"persistence",
		"sales_core.order_line",
		"databases/sales-core/persistence/order-line.tei",
	) {
		t.Fatalf("persistence mapping Evidence is missing: %+v", got.DataDesign.Evidence)
	}
	if !containsKnowledgeContextString(
		got.DataDesign.AuthorityLimits,
		"persistence-semantic-data-reference-is-reverse-trace-not-business-identity",
	) {
		t.Fatalf("persistence reverse-trace authority limit is missing: %+v", got.DataDesign.AuthorityLimits)
	}
}

func TestApplicationPatternIdentityDoesNotBecomeBusinessDataIdentity(t *testing.T) {
	root := testRepositoryRoot(t)
	got, err := buildKnowledgeContext(
		filepath.Join(root, "reference", "workspace"),
		"application-pattern",
		"order_entry.order_record",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ApplicationPattern == nil ||
		got.ApplicationPattern.Kind != "record" ||
		got.ApplicationPattern.Data != "sales.order" {
		t.Fatalf("record Pattern detail is missing: %+v", got.ApplicationPattern)
	}
	if got.DataDesign == nil ||
		got.DataDesign.Data != "sales.order" ||
		got.DataDesign.Focus.Kind != "application-pattern" ||
		got.DataDesign.Focus.ID != "order_entry.order_record" {
		t.Fatalf("record Pattern did not expose its explicit Semantic Data context: %+v", got.DataDesign)
	}
	if got.DataDesign.BusinessOccurrenceIdentifier.Nature != "unresolved" ||
		got.DataDesign.CandidateKeys.Nature != "unresolved" ||
		got.DataDesign.FunctionalDependencies.Nature != "unresolved" {
		t.Fatalf("Application Pattern identity was promoted to business data identity: %+v", got.DataDesign)
	}
	if got.DataDesign.EvidenceCoverageStatus != "complete" {
		t.Fatalf("record Pattern Data Design Evidence is incomplete: %+v", got.DataDesign)
	}
	if !hasDataDesignEvidence(
		got.DataDesign,
		"application-pattern",
		"order_entry.order_record",
		"applications/order-entry/patterns/order-record.tei",
	) {
		t.Fatalf("record Pattern focus Evidence is missing: %+v", got.DataDesign.Evidence)
	}
	for _, candidate := range got.DataDesign.BusinessIdentityCandidates {
		if candidate.TargetOccurrence == "sales.order" {
			t.Fatalf("record Pattern or physical key invented an order business identity candidate: %+v", candidate)
		}
	}
	if !hasKnowledgeContextEdge(got, "outgoing", "data", "semantic-data", "sales.order") {
		t.Fatalf("Pattern must reach Semantic Data through an explicit dependency edge: %+v", got.Outgoing)
	}
	if !containsKnowledgeContextString(
		got.AuthorityLimits,
		"projection-or-runtime-identity-is-not-business-data-identity",
	) {
		t.Fatalf("projection identity authority limit is missing: %+v", got.AuthorityLimits)
	}
	if !containsKnowledgeContextString(
		got.DataDesign.AuthorityLimits,
		"application-pattern-data-reference-is-context-not-business-identity",
	) {
		t.Fatalf("Pattern Data authority limit is missing: %+v", got.DataDesign.AuthorityLimits)
	}
}

func TestBuildKnowledgeContextReturnsSemanticCapabilityContext(t *testing.T) {
	root := testRepositoryRoot(t)
	got, err := buildKnowledgeContext(
		filepath.Join(root, "reference", "workspace"),
		"semantic-capability",
		"sales.order.submit",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.SemanticElement == nil ||
		got.SemanticElement.Kind != "capability" ||
		got.SemanticElement.ID != "sales.order.submit" {
		t.Fatalf("semantic capability detail is missing: %+v", got.SemanticElement)
	}
	if len(got.References) == 0 || len(got.Artifacts) == 0 {
		t.Fatalf("capability source trace is missing: references=%+v artifacts=%+v", got.References, got.Artifacts)
	}
	for _, relation := range []string{"input", "output"} {
		if !hasKnowledgeContextEdge(got, "outgoing", relation, "semantic-element", "sales.order") {
			t.Fatalf("capability %s dependency is missing: %+v", relation, got.Outgoing)
		}
	}
	if got.DataDesign != nil {
		t.Fatalf("capability identity was promoted to Data Design context: %+v", got.DataDesign)
	}
	if len(got.Unresolved) != 0 {
		t.Fatalf("known semantic capability has unresolved context: %+v", got.Unresolved)
	}
}

func TestBuildKnowledgeContextReturnsSemanticValidationContext(t *testing.T) {
	root := testRepositoryRoot(t)
	got, err := buildKnowledgeContext(
		filepath.Join(root, "reference", "workspace"),
		"semantic-validation",
		"sales.order_line.quantity_positive",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.SemanticElement == nil ||
		got.SemanticElement.Kind != "validation" ||
		got.SemanticElement.ID != "sales.order_line.quantity_positive" {
		t.Fatalf("semantic validation detail is missing: %+v", got)
	}
	if len(got.References) == 0 || len(got.Artifacts) == 0 {
		t.Fatalf("validation source trace is missing: references=%+v artifacts=%+v", got.References, got.Artifacts)
	}
	if len(got.Unresolved) != 0 {
		t.Fatalf("known validation has unresolved context: %+v", got.Unresolved)
	}
	if got.DataDesign == nil ||
		got.DataDesign.Data != "sales.order_line" ||
		got.DataDesign.Focus.Kind != "semantic-validation" ||
		got.DataDesign.Focus.ID != "sales.order_line.quantity_positive" {
		t.Fatalf("semantic validation did not resolve Data Design Context: %+v", got.DataDesign)
	}
	if !hasKnowledgeContextEdge(
		got,
		"outgoing",
		"attribute",
		"semantic-attribute",
		"sales.order_line.quantity",
	) {
		t.Fatalf("validation target attribute is missing: %+v", got.Outgoing)
	}
}

func TestRelationUniqueValidationResolvesScopedDataDesignWithoutInventingIdentity(t *testing.T) {
	root := testRepositoryRoot(t)
	got, err := buildKnowledgeContext(
		filepath.Join(root, "reference", "workspace"),
		"semantic-validation",
		"sales.order.lines.line_number_unique",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.DataDesign == nil ||
		got.DataDesign.Data != "sales.order_line" ||
		got.DataDesign.Focus.Kind != "semantic-validation" {
		t.Fatalf("relation-unique validation did not resolve target Data Design Context: %+v", got.DataDesign)
	}
	if len(got.DataDesign.ScopedUniqueness) != 1 ||
		got.DataDesign.ScopedUniqueness[0].Constraint != "sales.order.lines.line_number_unique" {
		t.Fatalf("scoped uniqueness is missing from validation context: %+v", got.DataDesign.ScopedUniqueness)
	}
	if len(got.DataDesign.FunctionalDependencyCandidates) != 1 {
		t.Fatalf("FD candidate is missing from validation context: %+v", got.DataDesign.FunctionalDependencyCandidates)
	}
	fdCandidate := got.DataDesign.FunctionalDependencyCandidates[0]
	if len(fdCandidate.Determinants) != 2 ||
		fdCandidate.Determinants[0].Kind != "relation-source-occurrence" ||
		fdCandidate.Determinants[0].Data != "sales.order" ||
		fdCandidate.Determinants[1].Kind != "attribute" ||
		fdCandidate.Determinants[1].Attribute != "sales.order_line.line_number" {
		t.Fatalf("validation context lost scoped determinant terms: %+v", fdCandidate)
	}
	if got.DataDesign.BusinessOccurrenceIdentifier.Nature != "unresolved" ||
		got.DataDesign.CandidateKeys.Nature != "unresolved" ||
		got.DataDesign.FunctionalDependencies.Nature != "unresolved" {
		t.Fatalf("validation context invented accepted identity or FD: %+v", got.DataDesign)
	}
}

func TestBuildKnowledgeContextReturnsDataDesignContextWithoutInventingIdentifier(t *testing.T) {
	root := testRepositoryRoot(t)
	got, err := buildKnowledgeContext(
		filepath.Join(root, "reference", "workspace"),
		"semantic-data",
		"sales.order_line",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.DataDesign == nil {
		t.Fatalf("data design context is missing: %+v", got)
	}
	dataDesign := got.DataDesign
	if dataDesign.Data != "sales.order_line" ||
		dataDesign.Focus.Kind != "semantic-data" ||
		dataDesign.Focus.ID != "sales.order_line" {
		t.Fatalf("data design scope drifted: %+v", dataDesign)
	}
	if dataDesign.ProjectionRevision != "tei.knowledge.data-design/v6" ||
		len(dataDesign.Fingerprint) <= len("sha256:") ||
		dataDesign.Fingerprint[:len("sha256:")] != "sha256:" ||
		dataDesign.ResolutionStatus != "resolved" ||
		dataDesign.EvidenceCoverageStatus != "complete" {
		t.Fatalf(
			"data design cache identity is incomplete: revision=%q fingerprint=%q resolutionStatus=%q evidenceCoverageStatus=%q",
			dataDesign.ProjectionRevision,
			dataDesign.Fingerprint,
			dataDesign.ResolutionStatus,
			dataDesign.EvidenceCoverageStatus,
		)
	}
	if len(dataDesign.Ambiguities) != 0 {
		t.Fatalf("valid Reference Data Design Context has semantic identity ambiguities: %+v", dataDesign.Ambiguities)
	}
	for name, state := range map[string]knowledgecontext.DataDesignKnowledgeState{
		"business identifier":         dataDesign.BusinessOccurrenceIdentifier,
		"candidate keys":              dataDesign.CandidateKeys,
		"functional dependency":       dataDesign.FunctionalDependencies,
		"composite domain constraint": dataDesign.CompositeDomainConstraints,
		"temporal/history conditions": dataDesign.TemporalHistoryConditions,
	} {
		if state.Nature != "unresolved" || state.Reason == "" {
			t.Fatalf("%s was invented or left unexplained: %+v", name, state)
		}
	}
	for _, question := range []string{
		"which-fields-identify-the-business-occurrence",
		"what-is-the-uniqueness-scope",
		"when-is-the-identifier-valid",
		"is-this-business-occurrence-identity-or-history-version-identity",
		"which-relations-or-constraints-depend-on-the-identity",
		"which-source-or-contract-owns-the-uniqueness-claim",
		"which-normalization-or-equivalence-policy-applies",
		"does-persistence-key-represent-business-identity-or-only-realization",
		"if-surrogate-key-is-used-which-accepted-uniqueness-preserves-business-identity",
	} {
		if !containsKnowledgeContextString(dataDesign.BusinessIdentityEvidenceQuestions, question) {
			t.Fatalf("business identity evidence question %q is missing: %+v", question, dataDesign.BusinessIdentityEvidenceQuestions)
		}
	}
	for _, question := range []string{
		"which-determinants-determine-which-dependents",
		"under-what-scope-and-time-does-the-dependency-hold",
		"which-source-or-evidence-establishes-the-dependency",
		"is-this-a-functional-dependency-or-a-composite-domain-constraint",
		"does-scoped-uniqueness-imply-business-functional-dependency-or-only-local-constraint",
	} {
		if !containsKnowledgeContextString(dataDesign.FunctionalDependencyEvidenceQuestions, question) {
			t.Fatalf("functional dependency evidence question %q is missing: %+v", question, dataDesign.FunctionalDependencyEvidenceQuestions)
		}
	}
	for _, question := range []string{
		"which-value-combinations-are-allowed-or-forbidden",
		"is-this-constraint-about-identity-functional-dependency-or-domain-membership",
		"under-what-scope-and-time-does-the-composite-domain-constraint-hold",
		"which-source-or-contract-establishes-the-allowed-combinations",
		"is-the-domain-static-or-derived-from-reference-data",
	} {
		if !containsKnowledgeContextString(dataDesign.CompositeDomainConstraintEvidenceQuestions, question) {
			t.Fatalf("composite domain constraint evidence question %q is missing: %+v", question, dataDesign.CompositeDomainConstraintEvidenceQuestions)
		}
	}
	for _, question := range []string{
		"when-is-the-business-occurrence-or-identity-valid",
		"can-the-business-identifier-be-reused-across-time",
		"does-the-history-record-represent-a-version-of-the-same-occurrence-or-a-new-occurrence",
		"may-validity-periods-overlap",
		"is-current-state-derived-from-validity-or-marked-by-a-latest-flag",
		"which-state-transitions-change-identity-membership-or-validity",
		"which-history-or-validity-period-must-be-preserved",
		"which-source-or-contract-establishes-the-temporal-semantics",
	} {
		if !containsKnowledgeContextString(dataDesign.TemporalHistoryEvidenceQuestions, question) {
			t.Fatalf("temporal/history evidence question %q is missing: %+v", question, dataDesign.TemporalHistoryEvidenceQuestions)
		}
	}
	for _, question := range []string{
		"what-business-meaning-does-the-relation-express",
		"is-the-relation-a-changeable-reference-parent-contained-repeat-or-independent-association",
		"what-cardinality-is-required",
		"is-the-relation-current-only-or-temporal",
		"what-history-or-validity-period-must-be-preserved",
		"which-source-or-contract-owns-the-relation",
	} {
		if !containsKnowledgeContextString(dataDesign.RelationEvidenceQuestions, question) {
			t.Fatalf("relation evidence question %q is missing: %+v", question, dataDesign.RelationEvidenceQuestions)
		}
	}

	var foundLineRelation bool
	for _, relation := range dataDesign.Relations {
		if relation.ID == "sales.order.lines" &&
			relation.Source == "sales.order" &&
			relation.Target == "sales.order_line" &&
			relation.Cardinality == "many" {
			foundLineRelation = true
			break
		}
	}
	if !foundLineRelation {
		t.Fatalf("incoming order-line relation is missing: %+v", dataDesign.Relations)
	}

	var foundLineUnique, foundQuantityPositive bool
	for _, constraint := range dataDesign.Constraints {
		switch constraint.ID {
		case "sales.order.lines.line_number_unique":
			foundLineUnique = constraint.Kind == "relation-unique" &&
				constraint.Relation == "sales.order.lines" &&
				constraint.Attribute == "sales.order_line.line_number"
		case "sales.order_line.quantity_positive":
			foundQuantityPositive = constraint.Kind == "positive" &&
				constraint.Attribute == "sales.order_line.quantity"
		}
	}
	if !foundLineUnique || !foundQuantityPositive {
		t.Fatalf("data design constraints are incomplete: %+v", dataDesign.Constraints)
	}

	if len(dataDesign.ScopedUniqueness) != 1 {
		t.Fatalf("scoped uniqueness=%+v", dataDesign.ScopedUniqueness)
	}
	uniqueness := dataDesign.ScopedUniqueness[0]
	if uniqueness.Constraint != "sales.order.lines.line_number_unique" ||
		uniqueness.Relation != "sales.order.lines" ||
		uniqueness.Source != "sales.order" ||
		uniqueness.Target != "sales.order_line" ||
		uniqueness.Attribute != "sales.order_line.line_number" ||
		uniqueness.Scope != "per-relation-source-occurrence" ||
		uniqueness.Nature != "accepted" {
		t.Fatalf("relation-unique meaning was widened or lost: %+v", uniqueness)
	}
	if dataDesign.CandidateKeys.Nature != "unresolved" ||
		dataDesign.BusinessOccurrenceIdentifier.Nature != "unresolved" {
		t.Fatalf("scoped uniqueness was promoted to a business key: %+v", dataDesign)
	}
	if len(dataDesign.BusinessIdentityCandidates) != 1 {
		t.Fatalf("business identity candidates=%+v", dataDesign.BusinessIdentityCandidates)
	}
	identityCandidate := dataDesign.BusinessIdentityCandidates[0]
	if identityCandidate.Fingerprint == "" ||
		identityCandidate.EvidenceRequirementFingerprint == "" {
		t.Fatalf("business identity candidate fingerprints are missing: %+v", identityCandidate)
	}
	if identityCandidate.Relation != "sales.order.lines" ||
		identityCandidate.TargetOccurrence != "sales.order_line" ||
		identityCandidate.BasisConstraint != "sales.order.lines.line_number_unique" ||
		identityCandidate.Scope != "per-relation-source-occurrence" ||
		identityCandidate.Nature != "candidate-from-accepted-scoped-uniqueness" {
		t.Fatalf("business identity candidate drifted: %+v", identityCandidate)
	}
	if len(identityCandidate.Components) != 2 ||
		identityCandidate.Components[0].Kind != "relation-source-occurrence" ||
		identityCandidate.Components[0].Data != "sales.order" ||
		identityCandidate.Components[1].Kind != "attribute" ||
		identityCandidate.Components[1].Data != "sales.order_line" ||
		identityCandidate.Components[1].Attribute != "sales.order_line.line_number" {
		t.Fatalf("business identity candidate lost composite occurrence terms: %+v", identityCandidate.Components)
	}
	for _, needed := range []string{
		"source-occurrence-business-identifier",
		"relation-business-meaning",
		"target-membership-exclusivity",
		"identifier-stability",
		"temporal-validity",
		"history-version-identity",
		"identifier-reuse-policy",
	} {
		if !containsKnowledgeContextString(identityCandidate.EvidenceNeeded, needed) {
			t.Fatalf("business identity Evidence need %q is missing: %+v", needed, identityCandidate.EvidenceNeeded)
		}
	}
	if dataDesign.BusinessOccurrenceIdentifier.Nature != "unresolved" ||
		dataDesign.CandidateKeys.Nature != "unresolved" {
		t.Fatalf("business identity candidate was promoted to accepted identity: %+v", dataDesign)
	}
	if len(dataDesign.FunctionalDependencyCandidates) != 1 {
		t.Fatalf("functional dependency candidates=%+v", dataDesign.FunctionalDependencyCandidates)
	}
	fdCandidate := dataDesign.FunctionalDependencyCandidates[0]
	if fdCandidate.Fingerprint == "" ||
		fdCandidate.EvidenceRequirementFingerprint == "" {
		t.Fatalf("functional dependency candidate fingerprints are missing: %+v", fdCandidate)
	}
	if fdCandidate.Relation != "sales.order.lines" ||
		fdCandidate.DependentOccurrence != "sales.order_line" ||
		fdCandidate.BasisConstraint != "sales.order.lines.line_number_unique" ||
		fdCandidate.Scope != "per-relation-source-occurrence" ||
		fdCandidate.Nature != "candidate-from-accepted-scoped-uniqueness" {
		t.Fatalf("functional dependency candidate drifted: %+v", fdCandidate)
	}
	if len(fdCandidate.Determinants) != 2 ||
		fdCandidate.Determinants[0].Kind != "relation-source-occurrence" ||
		fdCandidate.Determinants[0].Data != "sales.order" ||
		fdCandidate.Determinants[0].Attribute != "" ||
		fdCandidate.Determinants[1].Kind != "attribute" ||
		fdCandidate.Determinants[1].Data != "sales.order_line" ||
		fdCandidate.Determinants[1].Attribute != "sales.order_line.line_number" {
		t.Fatalf("functional dependency determinant terms lost scoped identity: %+v", fdCandidate.Determinants)
	}
	for _, needed := range []string{
		"source-occurrence-business-identifier",
		"relation-business-meaning",
		"target-membership-exclusivity",
		"temporal-validity",
		"history-version-identity",
		"temporal-dependency-scope",
	} {
		if !containsKnowledgeContextString(fdCandidate.EvidenceNeeded, needed) {
			t.Fatalf("functional dependency Evidence need %q is missing: %+v", needed, fdCandidate.EvidenceNeeded)
		}
	}
	if dataDesign.FunctionalDependencies.Nature != "unresolved" ||
		dataDesign.CompositeDomainConstraints.Nature != "unresolved" {
		t.Fatalf(
			"scoped uniqueness was promoted beyond its accepted local constraint: fd=%+v compositeDomain=%+v",
			dataDesign.FunctionalDependencies,
			dataDesign.CompositeDomainConstraints,
		)
	}

	if len(dataDesign.PersistenceRealizations) != 1 {
		t.Fatalf("persistence realizations=%+v", dataDesign.PersistenceRealizations)
	}
	realization := dataDesign.PersistenceRealizations[0]
	if realization.Mapping != "sales_core.order_line" ||
		realization.Database != "sales_core" ||
		realization.Table != "public.order_lines" ||
		realization.Authority != "database-owned-realization" {
		t.Fatalf("persistence realization drifted: %+v", realization)
	}
	if len(realization.Uniqueness) != 1 ||
		realization.Uniqueness[0].Kind != "primary-key" ||
		realization.Uniqueness[0].Nature != "accepted-realization" ||
		len(realization.Uniqueness[0].Attributes) != 1 ||
		realization.Uniqueness[0].Attributes[0] != "sales.order_line.order_line_id" {
		t.Fatalf("physical primary key realization is missing: %+v", realization.Uniqueness)
	}
	if dataDesign.BusinessOccurrenceIdentifier.Nature != "unresolved" ||
		dataDesign.CandidateKeys.Nature != "unresolved" ||
		dataDesign.FunctionalDependencies.Nature != "unresolved" ||
		dataDesign.CompositeDomainConstraints.Nature != "unresolved" {
		t.Fatalf("physical primary key was promoted to logical identity or composite domain semantics: %+v", dataDesign)
	}

	for _, want := range []struct {
		kind   string
		id     string
		source string
	}{
		{kind: "semantic-data", id: "sales.order_line", source: "semantic/sales/order.tei"},
		{kind: "semantic-data", id: "sales.order", source: "semantic/sales/order.tei"},
		{kind: "semantic-attribute", id: "sales.order_line.line_number", source: "semantic/sales/order.tei"},
		{kind: "semantic-relation", id: "sales.order.lines", source: "semantic/sales/order.tei"},
		{kind: "semantic-validation", id: "sales.order.lines.line_number_unique", source: "semantic/sales/validation.tei"},
		{kind: "persistence", id: "sales_core.order_line", source: "databases/sales-core/persistence/order-line.tei"},
	} {
		if !hasDataDesignEvidence(dataDesign, want.kind, want.id, want.source) {
			t.Fatalf("data design Evidence is missing: %+v evidence=%+v", want, dataDesign.Evidence)
		}
	}
	if len(dataDesign.EvidenceMissing) != 0 {
		t.Fatalf("known data design subjects lack exact source Evidence: %+v", dataDesign.EvidenceMissing)
	}

	for _, limit := range []string{
		"data-design-context-is-derived-not-authority",
		"data-design-fingerprint-is-cache-invalidation-not-semantic-proof",
		"data-design-evidence-coverage-is-not-source-freshness-proof",
		"incomplete-data-design-evidence-blocks-complete-cache-reuse",
		"absence-of-authored-business-identifier-is-not-no-identifier",
		"business-identity-candidate-is-not-business-identifier-or-candidate-key",
		"derived-design-candidate-fingerprint-is-identity-not-acceptance",
		"derived-design-candidate-evidence-requirement-fingerprint-is-review-contract-not-acceptance",
		"absence-of-authored-functional-dependency-is-not-no-dependency",
		"functional-dependency-candidate-is-not-accepted-functional-dependency",
		"absence-of-authored-composite-domain-constraint-is-not-no-composite-domain-constraint",
		"relation-shape-is-not-composite-domain-constraint-evidence",
		"scoped-uniqueness-is-not-composite-domain-constraint",
		"persistence-unique-constraint-is-not-composite-domain-constraint",
		"absence-of-authored-temporal-history-condition-is-not-timeless-semantics",
		"datetime-or-timestamp-attribute-is-not-temporal-semantics",
		"persistence-history-column-is-not-business-temporal-semantics",
		"effective-date-or-history-number-is-not-business-occurrence-identity-by-default",
		"latest-flag-is-not-business-identity-evidence",
		"history-row-identity-is-not-business-occurrence-identity-by-default",
		"temporal-history-model-is-opt-in-not-universal",
		"ambiguous-semantic-identity-does-not-produce-design-candidates",
		"ambiguous-persistence-identity-does-not-produce-realization-facts",
		"relation-shape-does-not-establish-business-meaning",
		"relation-shape-does-not-establish-temporal-semantics",
		"relation-key-is-not-business-identifier",
		"scoped-uniqueness-is-not-business-identifier-or-candidate-key",
		"persistence-primary-key-is-not-business-identifier-or-candidate-key",
		"persistence-unique-constraint-is-not-business-identifier-or-candidate-key",
		"identifier-like-attribute-is-not-business-identity",
		"identifier-value-presence-is-not-uniqueness-evidence",
		"normalized-string-equality-is-not-business-identity-evidence",
		"application-pattern-ui-or-runtime-identity-is-not-data-identity",
	} {
		if !containsKnowledgeContextString(dataDesign.AuthorityLimits, limit) {
			t.Fatalf("data design authority limit %q is missing: %+v", limit, dataDesign.AuthorityLimits)
		}
	}
}

func TestBuildKnowledgeContextExposesRelationKeyWithoutPromotingItToBusinessIdentifier(t *testing.T) {
	root := testRepositoryRoot(t)
	got, err := buildKnowledgeContext(
		filepath.Join(root, "reference", "workspace"),
		"semantic-relation",
		"sales.order.customer",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.DataDesign == nil || got.DataDesign.Data != "sales.order" {
		t.Fatalf("relation focus did not resolve parent data design context: %+v", got.DataDesign)
	}

	var found bool
	for _, relation := range got.DataDesign.Relations {
		if relation.ID != "sales.order.customer" {
			continue
		}
		if relation.Source != "sales.order" || relation.Target != "customer.customer" || relation.Cardinality != "one" {
			t.Fatalf("relation topology drifted: %+v", relation)
		}
		if len(relation.Keys) != 1 ||
			relation.Keys[0].SourceAttribute != "sales.order.customer_id" ||
			relation.Keys[0].TargetAttribute != "customer.customer.customer_id" {
			t.Fatalf("relation key mapping is incomplete: %+v", relation.Keys)
		}
		if relation.BusinessMeaning.Nature != "unresolved" ||
			relation.BusinessMeaning.Reason == "" ||
			relation.TemporalSemantics.Nature != "unresolved" ||
			relation.TemporalSemantics.Reason == "" {
			t.Fatalf("relation shape was promoted to business or temporal meaning: %+v", relation)
		}
		found = true
		break
	}
	if !found {
		t.Fatalf("focused relation is missing from data design context: %+v", got.DataDesign.Relations)
	}
	if got.DataDesign.BusinessOccurrenceIdentifier.Nature != "unresolved" ||
		got.DataDesign.FunctionalDependencies.Nature != "unresolved" {
		t.Fatalf("relation key was promoted to business identity or FD: %+v", got.DataDesign)
	}
	for _, want := range []struct {
		kind   string
		id     string
		source string
	}{
		{kind: "semantic-relation", id: "sales.order.customer", source: "semantic/sales/order.tei"},
		{kind: "semantic-attribute", id: "sales.order.customer_id", source: "semantic/sales/order.tei"},
		{kind: "semantic-attribute", id: "customer.customer.customer_id", source: "semantic/customer/customer.tei"},
		{kind: "semantic-data", id: "customer.customer", source: "semantic/customer/customer.tei"},
	} {
		if !hasDataDesignEvidence(got.DataDesign, want.kind, want.id, want.source) {
			t.Fatalf("relation key Evidence is missing: %+v evidence=%+v", want, got.DataDesign.Evidence)
		}
	}
}

func TestBuildKnowledgeContextPreservesUnknownSubjectAsUnresolved(t *testing.T) {
	root := testRepositoryRoot(t)
	got, err := buildKnowledgeContext(
		filepath.Join(root, "reference", "workspace"),
		"semantic-data",
		"missing.example",
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"semantic-element-not-found-in-derived-index",
		"source-reference-unavailable",
	} {
		if !containsKnowledgeContextString(got.Unresolved, want) {
			t.Fatalf("unresolved %q is missing: %+v", want, got.Unresolved)
		}
	}
	if got.SemanticElement != nil || got.ApplicationPattern != nil || got.DataDesign != nil {
		t.Fatalf("unknown subject produced authoritative or derived design detail: %+v", got)
	}
}

func TestBuildKnowledgeContextRequiresWorkspaceAndSubject(t *testing.T) {
	if _, err := buildKnowledgeContext("", "application-pattern", "order_entry.customer_reference"); err == nil {
		t.Fatal("empty workspace was accepted")
	}
	if _, err := buildKnowledgeContext("unused", "", "order_entry.customer_reference"); err == nil {
		t.Fatal("empty subject kind was accepted")
	}
	if _, err := buildKnowledgeContext("unused", "application-pattern", ""); err == nil {
		t.Fatal("empty subject id was accepted")
	}
}

func hasKnowledgeContextEdge(ctx knowledgecontext.Context, direction, relation, otherKind, otherID string) bool {
	edges := ctx.Outgoing
	if direction == "incoming" {
		edges = ctx.Incoming
	}
	for _, edge := range edges {
		if edge.Relation != relation {
			continue
		}
		if direction == "incoming" && edge.From.Kind == otherKind && edge.From.ID == otherID {
			return true
		}
		if direction == "outgoing" && edge.To.Kind == otherKind && edge.To.ID == otherID {
			return true
		}
	}
	return false
}

func hasDataDesignEvidence(ctx *knowledgecontext.DataDesignContext, kind, id, source string) bool {
	for _, evidence := range ctx.Evidence {
		if evidence.Subject.Kind == kind &&
			evidence.Subject.ID == id &&
			evidence.Source == source &&
			evidence.SourceFingerprint != "" &&
			evidence.Range.Start.Offset <= evidence.Range.End.Offset {
			return true
		}
	}
	return false
}

func containsKnowledgeContextString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestBuildKnowledgeQualificationClassifiesActualPatternApplicable(t *testing.T) {
	root := testRepositoryRoot(t)
	got, err := buildKnowledgeQualification(
		filepath.Join(root, "reference", "workspace"),
		"application-pattern",
		"order_entry.customer_reference",
		knowledgecontext.QualificationRequirements{
			ExpectedKind: "application-pattern",
			Application:  "order_entry",
			RequiredDependencies: []knowledgecontext.DependencyRequirement{{
				Direction: "outgoing",
				Relation:  "contract-query",
				Kind:      "database-contract-query",
				ID:        "master_data.customer/find_by_id",
			}},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != knowledgecontext.QualificationApplicable {
		t.Fatalf("status=%s reasons=%+v", got.Status, got.Reasons)
	}
	if len(got.Evidence) == 0 {
		t.Fatalf("applicable qualification lacks exact Evidence: %+v", got)
	}
	for _, evidence := range got.Evidence {
		if evidence.SubjectKind != "application-pattern" ||
			evidence.Subject != "order_entry.customer_reference" ||
			evidence.Source == "" ||
			evidence.SourceFingerprint == "" {
			t.Fatalf("qualification Evidence is incomplete: %+v", evidence)
		}
	}
	if len(got.DependencyEvidence) == 0 {
		t.Fatalf("applicable qualification lacks required dependency Evidence: %+v", got)
	}
	var foundDependency bool
	for _, evidence := range got.DependencyEvidence {
		if evidence.SubjectKind == "database-contract-query" &&
			evidence.Subject == "master_data.customer/find_by_id" &&
			evidence.Source != "" &&
			evidence.SourceFingerprint != "" {
			foundDependency = true
			break
		}
	}
	if !foundDependency {
		t.Fatalf("contract query dependency Evidence is missing: %+v", got.DependencyEvidence)
	}
}

func TestBuildKnowledgeQualificationRejectsApplicationOwnedNearMatch(t *testing.T) {
	root := testRepositoryRoot(t)
	got, err := buildKnowledgeQualification(
		filepath.Join(root, "reference", "workspace"),
		"application-pattern",
		"order_entry.customer_reference",
		knowledgecontext.QualificationRequirements{
			ExpectedKind: "application-pattern",
			Application:  "repair_intake",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != knowledgecontext.QualificationIncompatible {
		t.Fatalf("status=%s reasons=%+v", got.Status, got.Reasons)
	}
}

func TestBuildKnowledgeQualificationPreservesUnknownSubject(t *testing.T) {
	root := testRepositoryRoot(t)
	got, err := buildKnowledgeQualification(
		filepath.Join(root, "reference", "workspace"),
		"semantic-data",
		"missing.example",
		knowledgecontext.QualificationRequirements{ExpectedKind: "semantic-data"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != knowledgecontext.QualificationUnknown {
		t.Fatalf("status=%s reasons=%+v", got.Status, got.Reasons)
	}
}

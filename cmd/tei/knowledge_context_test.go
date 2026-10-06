package main

import (
	"path/filepath"
	"testing"

	"tei/internal/knowledgecontext"
)

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
	} {
		if !containsKnowledgeContextString(got.AuthorityLimits, limit) {
			t.Fatalf("authority limit %q is missing: %+v", limit, got.AuthorityLimits)
		}
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
	for name, state := range map[string]knowledgecontext.DataDesignKnowledgeState{
		"business identifier":   dataDesign.BusinessOccurrenceIdentifier,
		"candidate keys":        dataDesign.CandidateKeys,
		"functional dependency": dataDesign.FunctionalDependencies,
	} {
		if state.Nature != "unresolved" || state.Reason == "" {
			t.Fatalf("%s was invented or left unexplained: %+v", name, state)
		}
	}
	for _, question := range []string{
		"which-fields-identify-the-business-occurrence",
		"what-is-the-uniqueness-scope",
		"when-is-the-identifier-valid",
		"which-relations-or-constraints-depend-on-the-identity",
		"which-source-or-contract-owns-the-uniqueness-claim",
	} {
		if !containsKnowledgeContextString(dataDesign.BusinessIdentityEvidenceQuestions, question) {
			t.Fatalf("business identity evidence question %q is missing: %+v", question, dataDesign.BusinessIdentityEvidenceQuestions)
		}
	}
	for _, question := range []string{
		"which-determinants-determine-which-dependents",
		"under-what-scope-and-time-does-the-dependency-hold",
		"which-source-or-evidence-establishes-the-dependency",
		"is-this-a-functional-dependency-or-a-domain-constraint",
	} {
		if !containsKnowledgeContextString(dataDesign.FunctionalDependencyEvidenceQuestions, question) {
			t.Fatalf("functional dependency evidence question %q is missing: %+v", question, dataDesign.FunctionalDependencyEvidenceQuestions)
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

	for _, limit := range []string{
		"absence-of-authored-business-identifier-is-not-no-identifier",
		"absence-of-authored-functional-dependency-is-not-no-dependency",
		"relation-key-is-not-business-identifier",
		"identifier-like-attribute-is-not-business-identity",
		"identifier-value-presence-is-not-uniqueness-evidence",
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

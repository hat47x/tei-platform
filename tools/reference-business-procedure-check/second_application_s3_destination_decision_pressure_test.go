package main

import (
	"strings"
	"testing"
)

type secondApplicationS3DecisionPacket struct {
	GateCEntryState              string
	S1S2State                    string
	ScenarioLabel                string
	ApplicationIDCandidate       string
	ApplicationDecisionBasis     string
	ApplicationDecisionStatus    string
	BusinessContext              string
	BusinessRelationCandidate    string
	SemanticNamespaceCandidate   string
	SemanticDataCandidate        string
	CustomerIDAttributeCandidate string
	CustomerIDAttributeType      string
	BusinessRelationTarget       string
	BusinessRelationCardinality  string
	BusinessRelationKeySource    string
	BusinessRelationKeyTarget    string
	PatternCandidate             string
	DatasourceAliasCandidate     string
	AuthorizationRequirement     string
	AuthorizationPrincipalState  string
	AuthorizationResourceState   string
	BusinessProcedureDecision    string
	DevelopmentBindingPolicy     string
	ProductionBindingPolicy      string
	EnvironmentBindingSelected   bool
	FirstTargetScope             string
	AllowedOperation             string
	SourceAuthoringAttempted     bool
	AcceptanceAttempted          bool
	EnvironmentBindingCreated    bool
	TargetImplementationStarted  bool
	SharedAuthoritySubjects      []string
	OriginProvenanceSubjects     []string
	AuthorityLimits              []string
}

func TestSecondApplicationS3MakesExplicitDestinationDecisions(t *testing.T) {
	got := secondApplicationS3Decision()

	if got.GateCEntryState != "entry-evidence-complete" || got.S1S2State != "source-defined" {
		t.Fatalf("S3 started without completed prior stages: %+v", got)
	}
	if got.ScenarioLabel != "repair_intake-test-only" {
		t.Fatalf("scenario label drifted: %q", got.ScenarioLabel)
	}
	if got.ApplicationIDCandidate != "repair_intake" ||
		got.ApplicationDecisionBasis != "explicit-s3-destination-decision" ||
		got.ApplicationDecisionStatus != "proposed-for-shared-acceptance" {
		t.Fatalf("Application identity was not an explicit S3 proposal: %+v", got)
	}
	if got.ApplicationIDCandidate == got.ScenarioLabel {
		t.Fatalf("test-only scenario label was promoted directly to Application identity: %+v", got)
	}
	if got.BusinessContext != "repair-intake" ||
		got.BusinessRelationCandidate != "repair_intake.intake.customer" ||
		got.PatternCandidate != "repair_intake.customer_reference" ||
		got.DatasourceAliasCandidate != "customer_master" {
		t.Fatalf("destination-owned decisions are incomplete: %+v", got)
	}
}

func TestSecondApplicationS3DecidesTheDestinationSemanticExplicitly(t *testing.T) {
	got := secondApplicationS3Decision()

	if got.SemanticNamespaceCandidate != "repair_intake" ||
		got.SemanticDataCandidate != "repair_intake.intake" ||
		got.CustomerIDAttributeCandidate != "repair_intake.intake.customer_id" ||
		got.CustomerIDAttributeType != "customer.customer_id" ||
		got.BusinessRelationTarget != "customer.customer" ||
		got.BusinessRelationCardinality != "one" ||
		got.BusinessRelationKeySource != "repair_intake.intake.customer_id" ||
		got.BusinessRelationKeyTarget != "customer.customer.customer_id" {
		t.Fatalf("destination Semantic decisions are incomplete or drifted: %+v", got)
	}
	for _, id := range []string{
		got.SemanticDataCandidate,
		got.CustomerIDAttributeCandidate,
		got.BusinessRelationCandidate,
		got.BusinessRelationKeySource,
	} {
		if id != got.SemanticNamespaceCandidate && !strings.HasPrefix(id, got.SemanticNamespaceCandidate+".") {
			t.Fatalf("destination-owned identity %q is outside namespace %q", id, got.SemanticNamespaceCandidate)
		}
		if strings.HasPrefix(id, "sales.") {
			t.Fatalf("origin sales.* identity leaked into the destination decision: %q", id)
		}
	}
	if got.BusinessRelationTarget != "customer.customer" ||
		got.CustomerIDAttributeType != "customer.customer_id" ||
		got.BusinessRelationKeyTarget != "customer.customer.customer_id" {
		t.Fatalf("shared Customer authority must stay a link, not a copy: %+v", got)
	}
}

func TestSecondApplicationS3DoesNotCopyOriginApplicationMeaning(t *testing.T) {
	got := secondApplicationS3Decision()

	for _, forbidden := range []string{
		"sales.order.customer",
		"order_entry.customer_reference",
		"master",
	} {
		switch forbidden {
		case got.BusinessRelationCandidate, got.PatternCandidate, got.DatasourceAliasCandidate:
			t.Fatalf("origin-owned meaning leaked into destination decision: %q %+v", forbidden, got)
		}
	}
	if len(got.OriginProvenanceSubjects) != 1 ||
		got.OriginProvenanceSubjects[0] != "application-pattern:order_entry.customer_reference" {
		t.Fatalf("origin Pattern should remain provenance only: %+v", got.OriginProvenanceSubjects)
	}
}

func TestSecondApplicationS3KeepsSharedCustomerAuthorityLinked(t *testing.T) {
	got := secondApplicationS3Decision()
	for _, want := range []string{
		"semantic-data:customer.customer",
		"semantic-type:customer.customer_id",
		"database-contract:master_data.customer",
		"database-contract-query:master_data.customer/find_by_id",
	} {
		found := false
		for _, subject := range got.SharedAuthoritySubjects {
			if subject == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("S3 lost shared Customer authority %q: %+v", want, got.SharedAuthoritySubjects)
		}
	}
}

func TestSecondApplicationS3SeparatesAuthorizationRequirementFromGrant(t *testing.T) {
	got := secondApplicationS3Decision()

	if got.AuthorizationRequirement != "required-current-authority-check" {
		t.Fatalf("read-only lookup was treated as implicitly public: %+v", got)
	}
	if got.AuthorizationPrincipalState != "unresolved" ||
		got.AuthorizationResourceState != "unresolved" {
		t.Fatalf("S3 invented execution authority identities: %+v", got)
	}
}

func TestSecondApplicationS3ResolvesBindingPolicyWithoutSelectingBinding(t *testing.T) {
	got := secondApplicationS3Decision()

	if got.DevelopmentBindingPolicy != "explicit-destination-binding-after-acceptance" ||
		got.ProductionBindingPolicy != "explicit-destination-binding-after-acceptance" {
		t.Fatalf("Environment binding policy is not explicit: %+v", got)
	}
	if got.EnvironmentBindingSelected || got.EnvironmentBindingCreated {
		t.Fatalf("S3 crossed into Environment authority: %+v", got)
	}
}

func TestSecondApplicationS3KeepsFirstTargetReadOnlyAndAvoidsPrematureProcedure(t *testing.T) {
	got := secondApplicationS3Decision()

	if got.BusinessProcedureDecision != "not-required-for-first-reference" {
		t.Fatalf("S3 introduced a Business Procedure without a step/handoff need: %+v", got)
	}
	if got.FirstTargetScope != "customer-id-lookup-result" ||
		got.AllowedOperation != "read-only-customer-find-by-id" {
		t.Fatalf("first Target scope drifted beyond lookup: %+v", got)
	}
	if got.SourceAuthoringAttempted ||
		got.AcceptanceAttempted ||
		got.EnvironmentBindingCreated ||
		got.TargetImplementationStarted {
		t.Fatalf("S3 crossed a later-stage authority boundary: %+v", got)
	}
}

func secondApplicationS3Decision() secondApplicationS3DecisionPacket {
	return secondApplicationS3DecisionPacket{
		GateCEntryState:              "entry-evidence-complete",
		S1S2State:                    "source-defined",
		ScenarioLabel:                "repair_intake-test-only",
		ApplicationIDCandidate:       "repair_intake",
		ApplicationDecisionBasis:     "explicit-s3-destination-decision",
		ApplicationDecisionStatus:    "proposed-for-shared-acceptance",
		BusinessContext:              "repair-intake",
		BusinessRelationCandidate:    "repair_intake.intake.customer",
		SemanticNamespaceCandidate:   "repair_intake",
		SemanticDataCandidate:        "repair_intake.intake",
		CustomerIDAttributeCandidate: "repair_intake.intake.customer_id",
		CustomerIDAttributeType:      "customer.customer_id",
		BusinessRelationTarget:       "customer.customer",
		BusinessRelationCardinality:  "one",
		BusinessRelationKeySource:    "repair_intake.intake.customer_id",
		BusinessRelationKeyTarget:    "customer.customer.customer_id",
		PatternCandidate:             "repair_intake.customer_reference",
		DatasourceAliasCandidate:     "customer_master",
		AuthorizationRequirement:     "required-current-authority-check",
		AuthorizationPrincipalState:  "unresolved",
		AuthorizationResourceState:   "unresolved",
		BusinessProcedureDecision:    "not-required-for-first-reference",
		DevelopmentBindingPolicy:     "explicit-destination-binding-after-acceptance",
		ProductionBindingPolicy:      "explicit-destination-binding-after-acceptance",
		EnvironmentBindingSelected:   false,
		FirstTargetScope:             "customer-id-lookup-result",
		AllowedOperation:             "read-only-customer-find-by-id",
		SourceAuthoringAttempted:     false,
		AcceptanceAttempted:          false,
		EnvironmentBindingCreated:    false,
		TargetImplementationStarted:  false,
		SharedAuthoritySubjects: []string{
			"semantic-data:customer.customer",
			"semantic-type:customer.customer_id",
			"database-contract:master_data.customer",
			"database-contract-query:master_data.customer/find_by_id",
		},
		OriginProvenanceSubjects: []string{
			"application-pattern:order_entry.customer_reference",
		},
		AuthorityLimits: []string{
			"scenario-label-is-not-application-authority",
			"s3-decision-is-not-shared-acceptance",
			"read-contract-is-not-authorization-grant",
			"binding-policy-is-not-environment-binding",
			"target-scope-is-not-target-implementation",
		},
	}
}

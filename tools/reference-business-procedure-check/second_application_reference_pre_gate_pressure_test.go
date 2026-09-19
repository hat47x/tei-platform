package main

import (
	"reflect"
	"testing"

	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

type secondApplicationPreGateProbe struct {
	ScenarioLabel                    string   `json:"scenarioLabel"`
	RequiredGate                     string   `json:"requiredGate"`
	CanonicalApplicationCreated      bool     `json:"canonicalApplicationCreated"`
	SourceAuthoringAttempted         bool     `json:"sourceAuthoringAttempted"`
	AcceptanceAttempted              bool     `json:"acceptanceAttempted"`
	EnvironmentBindingCreated        bool     `json:"environmentBindingCreated"`
	TargetImplementationStarted      bool     `json:"targetImplementationStarted"`
	SharedAuthoritySubjects          []string `json:"sharedAuthoritySubjects"`
	DestinationOwnedCandidates       []string `json:"destinationOwnedCandidates"`
	EnvironmentOwnedResponsibilities []string `json:"environmentOwnedResponsibilities"`
	OriginProvenanceSubjects         []string `json:"originProvenanceSubjects"`
	Unresolved                       []string `json:"unresolved"`
	AuthorityLimits                  []string `json:"authorityLimits"`
}

func TestSecondApplicationReferencePreGateKeepsGateBoundaryAndThreeOwnershipLayers(t *testing.T) {
	root := repoPath()
	ws := root + "/reference/workspace"
	loaded, err := workspace.Load(ws)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := workspace.BuildSourceIndex(ws, loaded)
	if err != nil {
		t.Fatal(err)
	}

	destination := knowledgePortabilityR0DestinationProbe{
		Label:              "repair_intake-test-only",
		CustomerSemanticID: "customer.customer",
		DatabaseTarget:     "master_data",
		ContractID:         "master_data.customer",
		ContractMajor:      2,
		Access:             database.AccessRead,
		DatasourceAlias:    "customer_master",
	}
	got, err := deriveSecondApplicationPreGateProbe(loaded.World, idx, destination)
	if err != nil {
		t.Fatal(err)
	}

	if got.RequiredGate != "gate-c-ready" {
		t.Fatalf("pre-Gate contract lost Gate C prerequisite: %+v", got)
	}
	if got.CanonicalApplicationCreated ||
		got.SourceAuthoringAttempted ||
		got.AcceptanceAttempted ||
		got.EnvironmentBindingCreated ||
		got.TargetImplementationStarted {
		t.Fatalf("pre-Gate contract crossed an implementation/acceptance boundary: %+v", got)
	}

	for _, want := range []string{
		"semantic-data:customer.customer",
		"database-contract:master_data.customer",
	} {
		if !containsPortabilityR0(got.SharedAuthoritySubjects, want) {
			t.Fatalf("shared authority subject %q is missing: %+v", want, got.SharedAuthoritySubjects)
		}
	}
	if containsPortabilityR0(got.SharedAuthoritySubjects, "application-pattern:order_entry.customer_reference") {
		t.Fatalf("origin Application Pattern was promoted to shared authority: %+v", got.SharedAuthoritySubjects)
	}

	for _, want := range []string{
		"application-identity:<unresolved>",
		"application-pattern:<destination>.customer_reference",
		"datasource-alias:customer_master",
		"business-relation:<unresolved>",
	} {
		if !containsPortabilityR0(got.DestinationOwnedCandidates, want) {
			t.Fatalf("destination-owned candidate %q is missing: %+v", want, got.DestinationOwnedCandidates)
		}
	}

	for _, want := range []string{
		"development-database-binding:<unresolved>",
		"production-database-binding:<unresolved>",
	} {
		if !containsPortabilityR0(got.EnvironmentOwnedResponsibilities, want) {
			t.Fatalf("Environment-owned responsibility %q is missing: %+v", want, got.EnvironmentOwnedResponsibilities)
		}
	}

	if !containsPortabilityR0(got.OriginProvenanceSubjects, "application-pattern:order_entry.customer_reference") {
		t.Fatalf("origin Pattern provenance was lost: %+v", got.OriginProvenanceSubjects)
	}
	for _, want := range []string{
		"canonical-application-id",
		"business-context-relation",
		"business-procedure",
		"development-environment-binding",
		"production-environment-binding",
		"first-web-target-surface",
		"authorization-requirement",
	} {
		if !containsPortabilityR0(got.Unresolved, want) {
			t.Fatalf("pre-Gate contract collapsed unresolved destination decision %q: %+v", want, got.Unresolved)
		}
	}

	assertSecondApplicationPreGateNoShortcut(t, reflect.TypeOf(secondApplicationPreGateProbe{}))
}

func TestSecondApplicationReferencePreGateKeepsQualificationAxesIndependent(t *testing.T) {
	root := repoPath()
	ws := root + "/reference/workspace"
	loaded, err := workspace.Load(ws)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := workspace.BuildSourceIndex(ws, loaded)
	if err != nil {
		t.Fatal(err)
	}

	unknown := knowledgePortabilityR0DestinationProbe{
		Label:              "repair_intake-test-only",
		CustomerSemanticID: "customer.customer",
		Access:             database.AccessRead,
	}
	semantic, err := deriveKnowledgePortabilityR0SemanticCandidate(idx, unknown)
	if err != nil {
		t.Fatal(err)
	}
	contract, err := deriveKnowledgePortabilityR0ContractCandidate(loaded.World, idx, unknown)
	if err != nil {
		t.Fatal(err)
	}
	if semantic.Qualification != "applicable" {
		t.Fatalf("shared Customer Semantic should remain independently applicable: %+v", semantic)
	}
	if contract.Qualification != "unknown" {
		t.Fatalf("missing destination Database/Contract choice was inherited from origin: %+v", contract)
	}

	incompatible := unknown
	incompatible.DatabaseTarget = "master_data"
	incompatible.ContractID = "master_data.customer"
	incompatible.ContractMajor = 99
	incompatible.DatasourceAlias = "customer_master"
	contract, err = deriveKnowledgePortabilityR0ContractCandidate(loaded.World, idx, incompatible)
	if err != nil {
		t.Fatal(err)
	}
	if contract.Qualification != "incompatible" {
		t.Fatalf("incompatible Contract major was not rejected: %+v", contract)
	}
	semantic, err = deriveKnowledgePortabilityR0SemanticCandidate(idx, incompatible)
	if err != nil {
		t.Fatal(err)
	}
	if semantic.Qualification != "applicable" {
		t.Fatalf("Contract incompatibility incorrectly invalidated the shared Semantic identity: %+v", semantic)
	}
}

func TestSecondApplicationReferencePreGateFixesExpectedChangePropagationWithoutImplementingImpactEngine(t *testing.T) {
	cases := map[string]string{
		"semantic-data:customer.customer":                      "dependency-requalification",
		"database-contract:master_data.customer":               "dependency-requalification",
		"application-pattern:order_entry.customer_reference":   "provenance-drift",
		"application:order_entry":                              "unaffected",
		"environment-binding:development/order_entry/master":   "unaffected",
		"environment-binding:production/order_entry/master":    "unaffected",
		"application-pattern:<destination>.customer_reference": "destination-local-change",
	}
	for subject, want := range cases {
		if got := secondApplicationPreGateExpectedChangeEffect(subject); got != want {
			t.Fatalf("change effect for %q=%q want=%q", subject, got, want)
		}
	}
	if got := secondApplicationPreGateExpectedChangeEffect("application-pattern:order_entry.order_record"); got != "unaffected" {
		t.Fatalf("unrelated origin Pattern stale-all'ed destination: %q", got)
	}
	if got := secondApplicationPreGateExpectedChangeEffect("unknown:future-subject"); got != "unresolved" {
		t.Fatalf("unsupported change was silently classified as safe: %q", got)
	}
}

func deriveSecondApplicationPreGateProbe(
	world ir.World,
	idx sourceindex.Index,
	destination knowledgePortabilityR0DestinationProbe,
) (secondApplicationPreGateProbe, error) {
	semanticCandidate, err := deriveKnowledgePortabilityR0SemanticCandidate(idx, destination)
	if err != nil {
		return secondApplicationPreGateProbe{}, err
	}
	contractCandidate, err := deriveKnowledgePortabilityR0ContractCandidate(world, idx, destination)
	if err != nil {
		return secondApplicationPreGateProbe{}, err
	}
	patternCandidate, err := deriveKnowledgePortabilityR0PatternCandidate(idx, "repair_intake.customer_reference")
	if err != nil {
		return secondApplicationPreGateProbe{}, err
	}
	developmentBinding, err := deriveKnowledgePortabilityR0EnvironmentProbe(world, "development")
	if err != nil {
		return secondApplicationPreGateProbe{}, err
	}
	productionBinding, err := deriveKnowledgePortabilityR0EnvironmentProbe(world, "production")
	if err != nil {
		return secondApplicationPreGateProbe{}, err
	}

	if semanticCandidate.Qualification != "applicable" {
		return secondApplicationPreGateProbe{}, errPortabilityR0("shared Customer Semantic is not applicable for pre-Gate destination profile")
	}
	if contractCandidate.Qualification != "applicable" {
		return secondApplicationPreGateProbe{}, errPortabilityR0("Customer Contract is not applicable for pre-Gate destination profile")
	}
	if patternCandidate.ReuseRelation != "adopted-local-variation" {
		return secondApplicationPreGateProbe{}, errPortabilityR0("origin Application Pattern was not preserved as local-variation provenance")
	}
	if developmentBinding.ReuseRelation != "rebind-required" || productionBinding.ReuseRelation != "rebind-required" {
		return secondApplicationPreGateProbe{}, errPortabilityR0("origin Environment bindings were not preserved as rebind-required")
	}

	return secondApplicationPreGateProbe{
		ScenarioLabel:               destination.Label,
		RequiredGate:                "gate-c-ready",
		CanonicalApplicationCreated: false,
		SourceAuthoringAttempted:    false,
		AcceptanceAttempted:         false,
		EnvironmentBindingCreated:   false,
		TargetImplementationStarted: false,
		SharedAuthoritySubjects: normalizePortabilityR0([]string{
			semanticCandidate.AuthoritySubject,
			contractCandidate.AuthoritySubject,
		}),
		DestinationOwnedCandidates: normalizePortabilityR0([]string{
			"application-identity:<unresolved>",
			"application-pattern:<destination>.customer_reference",
			"datasource-alias:" + destination.DatasourceAlias,
			"business-relation:<unresolved>",
		}),
		EnvironmentOwnedResponsibilities: []string{
			"development-database-binding:<unresolved>",
			"production-database-binding:<unresolved>",
		},
		OriginProvenanceSubjects: normalizePortabilityR0(append(
			append([]string(nil), semanticCandidate.ProvenanceSubjects...),
			patternCandidate.ProvenanceSubjects...,
		)),
		Unresolved: []string{
			"authorization-requirement",
			"business-context-relation",
			"business-procedure",
			"canonical-application-id",
			"development-environment-binding",
			"first-web-target-surface",
			"production-environment-binding",
		},
		AuthorityLimits: []string{
			"pre-gate-contract-is-not-application-source",
			"qualified-reuse-does-not-equal-shared-acceptance",
			"origin-pattern-is-not-destination-authority",
			"origin-environment-binding-is-not-destination-binding",
			"change-expectation-is-not-impact-engine-evidence",
		},
	}, nil
}

func secondApplicationPreGateExpectedChangeEffect(subject string) string {
	switch subject {
	case "semantic-data:customer.customer",
		"database-contract:master_data.customer":
		return "dependency-requalification"
	case "application-pattern:order_entry.customer_reference":
		return "provenance-drift"
	case "application-pattern:<destination>.customer_reference":
		return "destination-local-change"
	case "application:order_entry",
		"environment-binding:development/order_entry/master",
		"environment-binding:production/order_entry/master",
		"application-pattern:order_entry.order_record":
		return "unaffected"
	default:
		return "unresolved"
	}
}

func assertSecondApplicationPreGateNoShortcut(t *testing.T, typ reflect.Type) {
	t.Helper()
	for _, forbidden := range []string{
		"Ready",
		"Portable",
		"AutoCreate",
		"AutoApply",
		"AutoSync",
		"Accepted",
		"Approved",
		"Preferred",
		"Winner",
		"Safe",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("second Application pre-Gate probe introduced unsupported shortcut field %q", forbidden)
		}
	}
}

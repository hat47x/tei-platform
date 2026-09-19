package main

import (
	"testing"

	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

type secondApplicationS1TypeCandidate struct {
	AuthoritySubject     string
	Qualification        string
	ReuseRelation        string
	DependencySubject    string
	AuthoritySource      string
	AuthorityFingerprint string
	Reasons              []string
	AuthorityLimits      []string
}

type secondApplicationS1QueryCandidate struct {
	AuthoritySubject     string
	Qualification        string
	ReuseRelation        string
	DependencySubject    string
	AuthoritySource      string
	AuthorityFingerprint string
	Result               string
	ResultCardinality    string
	Reasons              []string
	AuthorityLimits      []string
}

type secondApplicationS1S2Packet struct {
	GateCEntryState                string
	ScenarioLabel                  string
	Intent                         string
	SourceAuthoringAttempted       bool
	AcceptanceAttempted            bool
	EnvironmentBindingCreated      bool
	TargetImplementationStarted    bool
	SemanticCandidate              knowledgePortabilityR0CandidateProbe
	CustomerIDTypeCandidate        secondApplicationS1TypeCandidate
	ContractCandidate              knowledgePortabilityR0CandidateProbe
	ContractQueryCandidate         secondApplicationS1QueryCandidate
	PatternProvenanceCandidate     knowledgePortabilityR0CandidateProbe
	DevelopmentBindingEvidence     knowledgePortabilityR0EnvironmentProbe
	ProductionBindingEvidence      knowledgePortabilityR0EnvironmentProbe
	SharedAuthoritySubjects        []string
	LinkedDependencySubjects       []string
	ProvenanceSubjects             []string
	UnresolvedDestinationDecisions []string
	GateCCarryForward              []string
	AuthorityLimits                []string
}

func TestSecondApplicationS1S2ActualPacketStartsFromActualGateCEntry(t *testing.T) {
	got, err := deriveSecondApplicationS1S2Packet(repoPath())
	if err != nil {
		t.Fatal(err)
	}
	if got.GateCEntryState != "entry-evidence-complete" {
		t.Fatalf("S1/S2 started without complete Gate C entry: %+v", got)
	}
	if got.ScenarioLabel != "repair_intake-test-only" {
		t.Fatalf("synthetic scenario label drifted: %q", got.ScenarioLabel)
	}
	if got.SourceAuthoringAttempted ||
		got.AcceptanceAttempted ||
		got.EnvironmentBindingCreated ||
		got.TargetImplementationStarted {
		t.Fatalf("S1/S2 crossed an authoring or acceptance boundary: %+v", got)
	}
}

func TestSecondApplicationS1S2QualifiesSharedCustomerAuthorities(t *testing.T) {
	got, err := deriveSecondApplicationS1S2Packet(repoPath())
	if err != nil {
		t.Fatal(err)
	}

	if got.SemanticCandidate.AuthoritySubject != "semantic-data:customer.customer" ||
		got.SemanticCandidate.Qualification != "applicable" ||
		got.SemanticCandidate.ReuseRelation != "linked-reuse" {
		t.Fatalf("Customer Semantic qualification=%+v", got.SemanticCandidate)
	}
	if got.CustomerIDTypeCandidate.AuthoritySubject != "semantic-type:customer.customer_id" ||
		got.CustomerIDTypeCandidate.Qualification != "applicable" ||
		got.CustomerIDTypeCandidate.ReuseRelation != "linked-reuse" {
		t.Fatalf("Customer ID Type qualification=%+v", got.CustomerIDTypeCandidate)
	}
	if got.ContractCandidate.AuthoritySubject != "database-contract:master_data.customer" ||
		got.ContractCandidate.Qualification != "applicable" ||
		got.ContractCandidate.ReuseRelation != "conditional-linked-reuse" {
		t.Fatalf("Customer Contract qualification=%+v", got.ContractCandidate)
	}
	if got.ContractCandidate.DestinationAlias != "" {
		t.Fatalf("S1/S2 silently selected a destination datasource alias: %+v", got.ContractCandidate)
	}
	if got.ContractQueryCandidate.AuthoritySubject != "database-contract-query:master_data.customer/find_by_id" ||
		got.ContractQueryCandidate.Qualification != "applicable" ||
		got.ContractQueryCandidate.ReuseRelation != "conditional-linked-reuse" ||
		got.ContractQueryCandidate.Result != "customer.customer" ||
		got.ContractQueryCandidate.ResultCardinality != "zero-or-one" {
		t.Fatalf("Customer Contract Query qualification=%+v", got.ContractQueryCandidate)
	}

	for _, want := range []string{
		"semantic-data:customer.customer",
		"semantic-type:customer.customer_id",
		"database-contract:master_data.customer",
		"database-contract-query:master_data.customer/find_by_id",
	} {
		if !containsPortabilityR0(got.SharedAuthoritySubjects, want) {
			t.Fatalf("S1/S2 shared authority lost %q: %+v", want, got.SharedAuthoritySubjects)
		}
		if !containsPortabilityR0(got.LinkedDependencySubjects, want) {
			t.Fatalf("S1/S2 linked dependency lost %q: %+v", want, got.LinkedDependencySubjects)
		}
	}
}

func TestSecondApplicationS1S2KeepsOriginPatternAsProvenanceAndEnvironmentAsRebind(t *testing.T) {
	got, err := deriveSecondApplicationS1S2Packet(repoPath())
	if err != nil {
		t.Fatal(err)
	}
	pattern := got.PatternProvenanceCandidate
	if pattern.AuthoritySubject != "application-pattern:order_entry.customer_reference" ||
		pattern.Qualification != "applicable-as-provenance" ||
		pattern.ReuseRelation != "adopted-local-variation" ||
		pattern.DestinationSubject != "application-pattern:<destination>.customer_reference" {
		t.Fatalf("origin Pattern provenance qualification=%+v", pattern)
	}
	if containsPortabilityR0(got.LinkedDependencySubjects, "application-pattern:order_entry.customer_reference") {
		t.Fatalf("origin Pattern leaked into destination linked dependencies: %+v", got.LinkedDependencySubjects)
	}
	if !containsPortabilityR0(got.ProvenanceSubjects, "application-pattern:order_entry.customer_reference") {
		t.Fatalf("origin Pattern provenance was lost: %+v", got.ProvenanceSubjects)
	}

	for _, binding := range []knowledgePortabilityR0EnvironmentProbe{
		got.DevelopmentBindingEvidence,
		got.ProductionBindingEvidence,
	} {
		if binding.ReuseRelation != "rebind-required" ||
			binding.Qualification != "destination-owned" ||
			binding.DestinationBinding != "" {
			t.Fatalf("origin Environment binding was treated as destination configuration: %+v", binding)
		}
		if binding.OriginApplication != "order_entry" || binding.OriginAlias != "master" {
			t.Fatalf("Environment provenance drifted: %+v", binding)
		}
	}
}

func TestSecondApplicationS1S2KeepsDestinationIdentityAndAuthorityDecisionsUnresolved(t *testing.T) {
	got, err := deriveSecondApplicationS1S2Packet(repoPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"canonical-application-id",
		"business-context",
		"business-relation",
		"destination-pattern-id",
		"datasource-alias",
		"authorization-requirement",
		"business-procedure",
		"development-environment-binding",
		"production-environment-binding",
		"first-web-target-surface",
	} {
		if !containsPortabilityR0(got.UnresolvedDestinationDecisions, want) {
			t.Fatalf("S1/S2 silently resolved destination Decision %q: %+v", want, got.UnresolvedDestinationDecisions)
		}
	}
	for _, forbidden := range []string{
		"application:repair_intake",
		"semantic-relation:repair_intake.intake.customer",
		"datasource-alias:customer_master",
		"environment-binding:development/repair_intake/customer_master",
		"environment-binding:production/repair_intake/customer_master",
	} {
		if containsPortabilityR0(got.SharedAuthoritySubjects, forbidden) ||
			containsPortabilityR0(got.LinkedDependencySubjects, forbidden) ||
			containsPortabilityR0(got.ProvenanceSubjects, forbidden) {
			t.Fatalf("synthetic destination choice %q was promoted before S3: %+v", forbidden, got)
		}
	}
}

func TestSecondApplicationS1S2DoesNotTurnGateCTargetDifferencesIntoSemanticDependencies(t *testing.T) {
	got, err := deriveSecondApplicationS1S2Packet(repoPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, carry := range got.GateCCarryForward {
		if containsPortabilityR0(got.LinkedDependencySubjects, carry) ||
			containsPortabilityR0(got.SharedAuthoritySubjects, carry) {
			t.Fatalf("Gate C Target/Runtime carry-forward became semantic reuse authority: %q %+v", carry, got)
		}
	}
}

func deriveSecondApplicationS1S2Packet(root string) (secondApplicationS1S2Packet, error) {
	entry := qualifyGateCSecondApplicationEntry(actualSecondApplicationGateCEntryInput())
	if entry.State != "entry-evidence-complete" {
		return secondApplicationS1S2Packet{}, errPortabilityR0("actual Gate C entry is not complete")
	}

	ws := root + "/reference/workspace"
	loaded, err := workspace.Load(ws)
	if err != nil {
		return secondApplicationS1S2Packet{}, err
	}
	idx, err := workspace.BuildSourceIndex(ws, loaded)
	if err != nil {
		return secondApplicationS1S2Packet{}, err
	}

	requirementProbe := knowledgePortabilityR0DestinationProbe{
		Label:              "repair_intake-test-only",
		CustomerSemanticID: "customer.customer",
		DatabaseTarget:     "master_data",
		ContractID:         "master_data.customer",
		ContractMajor:      2,
		Access:             database.AccessRead,
		DatasourceAlias:    "",
	}
	semanticCandidate, err := deriveKnowledgePortabilityR0SemanticCandidate(idx, requirementProbe)
	if err != nil {
		return secondApplicationS1S2Packet{}, err
	}
	typeCandidate, err := deriveSecondApplicationCustomerIDTypeCandidate(loaded.World, idx)
	if err != nil {
		return secondApplicationS1S2Packet{}, err
	}
	contractCandidate, err := deriveKnowledgePortabilityR0ContractCandidate(loaded.World, idx, requirementProbe)
	if err != nil {
		return secondApplicationS1S2Packet{}, err
	}
	queryCandidate, err := deriveSecondApplicationCustomerQueryCandidate(loaded.World, idx, contractCandidate)
	if err != nil {
		return secondApplicationS1S2Packet{}, err
	}
	patternCandidate, err := deriveKnowledgePortabilityR0PatternCandidate(idx, "<destination>.customer_reference")
	if err != nil {
		return secondApplicationS1S2Packet{}, err
	}
	developmentBinding, err := deriveKnowledgePortabilityR0EnvironmentProbe(loaded.World, "development")
	if err != nil {
		return secondApplicationS1S2Packet{}, err
	}
	productionBinding, err := deriveKnowledgePortabilityR0EnvironmentProbe(loaded.World, "production")
	if err != nil {
		return secondApplicationS1S2Packet{}, err
	}

	if semanticCandidate.Qualification != "applicable" ||
		typeCandidate.Qualification != "applicable" ||
		contractCandidate.Qualification != "applicable" ||
		queryCandidate.Qualification != "applicable" {
		return secondApplicationS1S2Packet{}, errPortabilityR0("S1 shared Customer candidate set is not fully applicable")
	}

	shared := normalizePortabilityR0([]string{
		semanticCandidate.AuthoritySubject,
		typeCandidate.AuthoritySubject,
		contractCandidate.AuthoritySubject,
		queryCandidate.AuthoritySubject,
	})
	dependencies := normalizePortabilityR0([]string{
		semanticCandidate.AuthoritySubject,
		typeCandidate.DependencySubject,
		contractCandidate.AuthoritySubject,
		queryCandidate.DependencySubject,
	})
	provenance := normalizePortabilityR0(patternCandidate.ProvenanceSubjects)

	return secondApplicationS1S2Packet{
		GateCEntryState:             entry.State,
		ScenarioLabel:               requirementProbe.Label,
		Intent:                      "destination Business contextからCustomerをIDで参照したい",
		SourceAuthoringAttempted:    false,
		AcceptanceAttempted:         false,
		EnvironmentBindingCreated:   false,
		TargetImplementationStarted: false,
		SemanticCandidate:           semanticCandidate,
		CustomerIDTypeCandidate:     typeCandidate,
		ContractCandidate:           contractCandidate,
		ContractQueryCandidate:      queryCandidate,
		PatternProvenanceCandidate:  patternCandidate,
		DevelopmentBindingEvidence:  developmentBinding,
		ProductionBindingEvidence:   productionBinding,
		SharedAuthoritySubjects:     shared,
		LinkedDependencySubjects:    dependencies,
		ProvenanceSubjects:          provenance,
		UnresolvedDestinationDecisions: []string{
			"authorization-requirement",
			"business-context",
			"business-procedure",
			"business-relation",
			"canonical-application-id",
			"datasource-alias",
			"destination-pattern-id",
			"development-environment-binding",
			"first-web-target-surface",
			"production-environment-binding",
		},
		GateCCarryForward: append([]string(nil), actualSecondApplicationGateCEntryInput().Unresolved...),
		AuthorityLimits: []string{
			"gate-c-entry-does-not-select-destination-meaning",
			"requirement-probe-is-not-accepted-destination-source",
			"qualification-is-not-shared-acceptance",
			"portability-relation-is-not-shared-acceptance",
			"origin-pattern-is-provenance-not-destination-authority",
			"origin-environment-binding-is-evidence-not-destination-configuration",
			"target-difference-handoff-is-not-semantic-reuse-authority",
		},
	}, nil
}

func deriveSecondApplicationCustomerIDTypeCandidate(
	world ir.World,
	idx sourceindex.Index,
) (secondApplicationS1TypeCandidate, error) {
	subject := sourceindex.Subject{Kind: "semantic-type", ID: "customer.customer_id"}
	source, fingerprint, err := knowledgePortabilityR0Source(idx, subject)
	if err != nil {
		return secondApplicationS1TypeCandidate{}, err
	}

	typeFound := false
	attributeFound := false
	for _, namespace := range world.Semantics {
		for _, semanticType := range namespace.Types {
			if string(semanticType.ID) == "customer.customer_id" {
				typeFound = true
			}
		}
		for _, data := range namespace.Data {
			if string(data.ID) != "customer.customer" {
				continue
			}
			for _, attribute := range data.Attributes {
				if string(attribute.ID) == "customer.customer.customer_id" &&
					string(attribute.TypeRef) == "customer.customer_id" {
					attributeFound = true
				}
			}
		}
	}
	if !typeFound || !attributeFound {
		return secondApplicationS1TypeCandidate{}, errPortabilityR0("Customer ID Type is not the current Customer identifier type")
	}

	key := portabilityR0SubjectKey(subject)
	return secondApplicationS1TypeCandidate{
		AuthoritySubject:     key,
		Qualification:        "applicable",
		ReuseRelation:        "linked-reuse",
		DependencySubject:    key,
		AuthoritySource:      source,
		AuthorityFingerprint: fingerprint,
		Reasons: []string{
			"customer.customer.customer_id explicitly uses customer.customer_id",
			"destination requirement is Customer lookup by Customer ID",
		},
		AuthorityLimits: []string{
			"type-identity-does-not-select-destination-application-identity",
			"type-reuse-does-not-author-destination-input-surface",
		},
	}, nil
}

func deriveSecondApplicationCustomerQueryCandidate(
	world ir.World,
	idx sourceindex.Index,
	contractCandidate knowledgePortabilityR0CandidateProbe,
) (secondApplicationS1QueryCandidate, error) {
	if contractCandidate.Qualification != "applicable" {
		return secondApplicationS1QueryCandidate{}, errPortabilityR0("Contract must be applicable before its Query can be qualified")
	}
	subject := sourceindex.Subject{
		Kind: "database-contract-query",
		ID:   "master_data.customer/find_by_id",
	}
	source, fingerprint, err := knowledgePortabilityR0Source(idx, subject)
	if err != nil {
		return secondApplicationS1QueryCandidate{}, err
	}

	var found *database.QueryContract
	for _, db := range world.Databases {
		if string(db.ID) != "master_data" {
			continue
		}
		for _, contract := range db.Contracts {
			if string(contract.ID) != "master_data.customer" || contract.Version.Major != 2 {
				continue
			}
			for i := range contract.Queries {
				if contract.Queries[i].Name == "find_by_id" {
					query := contract.Queries[i]
					found = &query
				}
			}
		}
	}
	if found == nil {
		return secondApplicationS1QueryCandidate{}, errPortabilityR0("master_data.customer/find_by_id Query did not resolve")
	}
	if len(found.Inputs) != 1 ||
		string(found.Inputs[0].Name) != "id" ||
		string(found.Inputs[0].Type) != "customer.customer_id" ||
		string(found.Result) != "customer.customer" ||
		string(found.ResultCardinality) != "zero-or-one" {
		return secondApplicationS1QueryCandidate{}, errPortabilityR0("Customer find_by_id Query signature does not match lookup intent")
	}

	key := portabilityR0SubjectKey(subject)
	return secondApplicationS1QueryCandidate{
		AuthoritySubject:     key,
		Qualification:        "applicable",
		ReuseRelation:        "conditional-linked-reuse",
		DependencySubject:    key,
		AuthoritySource:      source,
		AuthorityFingerprint: fingerprint,
		Result:               string(found.Result),
		ResultCardinality:    string(found.ResultCardinality),
		Reasons: []string{
			"Query belongs to the explicitly qualified compatible Customer Contract",
			"Query accepts customer.customer_id and returns customer.customer",
		},
		AuthorityLimits: []string{
			"query-reuse-does-not-copy-origin-application-pattern",
			"query-reuse-does-not-select-destination-datasource-alias",
		},
	}, nil
}

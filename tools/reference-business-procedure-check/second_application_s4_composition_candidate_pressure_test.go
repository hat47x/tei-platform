package main

import (
	"fmt"
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/semantic"
	"tei/internal/syntax"
)

type secondApplicationS4CompositionCandidate struct {
	Status                       string
	SharedAcceptanceStatus       string
	SemanticSource               string
	ApplicationSource            string
	PatternSource                string
	AuthorizationRequirement     string
	EnvironmentBindingIncluded   bool
	TargetImplementationIncluded bool
	BusinessProcedureIncluded    bool
	SourceAuthoringAttempted     bool
}

func TestSecondApplicationS4ComposesSemanticFromS3Decisions(t *testing.T) {
	s3 := secondApplicationS3Decision()
	got := secondApplicationS4Composition(s3)

	if got.Status != "proposal" || got.SharedAcceptanceStatus != "not-attempted" {
		t.Fatalf("S4 crossed the Shared Acceptance boundary: %+v", got)
	}

	ns, err := syntax.NewParser(got.SemanticSource).ParseSemanticFile()
	if err != nil {
		t.Fatalf("S4 Semantic candidate did not parse with existing grammar: %v", err)
	}
	if string(ns.ID) != s3.SemanticNamespaceCandidate || len(ns.Data) != 1 {
		t.Fatalf("destination Semantic drifted: %+v", ns)
	}
	data := ns.Data[0]
	if string(data.ID) != s3.SemanticDataCandidate ||
		len(data.Attributes) != 1 ||
		string(data.Attributes[0].ID) != s3.CustomerIDAttributeCandidate ||
		string(data.Attributes[0].TypeRef) != s3.CustomerIDAttributeType {
		t.Fatalf("destination Data/attribute drifted: %+v", data)
	}
	if len(data.Relations) != 1 {
		t.Fatalf("destination Relation count drifted: %+v", data.Relations)
	}
	relation := data.Relations[0]
	if string(relation.ID) != s3.BusinessRelationCandidate ||
		string(relation.Target) != s3.BusinessRelationTarget ||
		string(relation.Cardinality) != s3.BusinessRelationCardinality ||
		relation.Cardinality != semantic.CardinalityOne ||
		len(relation.Keys) != 1 ||
		string(relation.Keys[0].SourceAttribute) != s3.BusinessRelationKeySource ||
		string(relation.Keys[0].TargetAttribute) != s3.BusinessRelationKeyTarget {
		t.Fatalf("destination Relation/key drifted: %+v", relation)
	}
	if relation.ID == "sales.order.customer" || data.ID == "sales.order" {
		t.Fatalf("origin-owned meaning leaked into S4 Semantic: %+v", data)
	}
}

func TestSecondApplicationS4ComposesApplicationFromS3Decisions(t *testing.T) {
	s3 := secondApplicationS3Decision()
	got := secondApplicationS4Composition(s3)

	app, err := syntax.NewParser(got.ApplicationSource).ParseApplication()
	if err != nil {
		t.Fatalf("S4 Application candidate did not parse with existing grammar: %v", err)
	}
	if string(app.ID) != s3.ApplicationIDCandidate {
		t.Fatalf("Application ID drifted from S3: got=%q want=%q", app.ID, s3.ApplicationIDCandidate)
	}
	if len(app.SemanticUses) != 2 ||
		string(app.SemanticUses[0]) != s3.SemanticNamespaceCandidate ||
		app.SemanticUses[1] != "customer" {
		t.Fatalf("Application semantic uses drifted (destination namespace, then shared Customer): %+v", app.SemanticUses)
	}
	if len(app.Databases) != 1 {
		t.Fatalf("database requirements=%+v", app.Databases)
	}
	db := app.Databases[0]
	if string(db.Alias) != s3.DatasourceAliasCandidate ||
		db.TargetDatabase != "master_data" ||
		db.Access != database.AccessRead {
		t.Fatalf("Application database requirement drifted from S3/S1-S2: %+v", db)
	}
	if len(db.RequiredContracts) != 1 ||
		db.RequiredContracts[0].ContractID != "master_data.customer" ||
		db.RequiredContracts[0].CompatibleMajor != 2 {
		t.Fatalf("shared Customer Contract requirement drifted: %+v", db.RequiredContracts)
	}
}

func TestSecondApplicationS4ComposesPatternFromS3Decisions(t *testing.T) {
	s3 := secondApplicationS3Decision()
	got := secondApplicationS4Composition(s3)

	pattern, err := syntax.NewParser(got.PatternSource).ParseApplicationPattern()
	if err != nil {
		t.Fatalf("S4 Pattern candidate did not parse with existing grammar: %v", err)
	}
	if string(pattern.ID) != s3.PatternCandidate ||
		pattern.Kind != application.PatternReference ||
		string(pattern.Relation) != s3.BusinessRelationCandidate ||
		string(pattern.Datasource) != s3.DatasourceAliasCandidate {
		t.Fatalf("Pattern drifted from S3 decisions: %+v", pattern)
	}
	if pattern.Relation == "sales.order.customer" || pattern.Datasource == "master" {
		t.Fatalf("origin-owned meaning leaked into S4 Pattern: %+v", pattern)
	}
	if pattern.ContractQuery == nil ||
		pattern.ContractQuery.Contract != "master_data.customer" ||
		pattern.ContractQuery.Query != "find_by_id" {
		t.Fatalf("shared Contract Query drifted: %+v", pattern.ContractQuery)
	}
	if len(pattern.ContractQuery.Inputs) != 1 ||
		pattern.ContractQuery.Inputs[0].TargetAttribute != "customer.customer.customer_id" ||
		pattern.ContractQuery.Inputs[0].Input != "id" {
		t.Fatalf("shared Customer ID mapping drifted: %+v", pattern.ContractQuery.Inputs)
	}
}

func TestSecondApplicationS4DoesNotCrossLaterAuthorityBoundaries(t *testing.T) {
	s3 := secondApplicationS3Decision()
	got := secondApplicationS4Composition(s3)

	if got.AuthorizationRequirement != "required-current-authority-check" {
		t.Fatalf("S4 lost S3 current-authority requirement: %+v", got)
	}
	if got.EnvironmentBindingIncluded ||
		got.TargetImplementationIncluded ||
		got.BusinessProcedureIncluded ||
		got.SourceAuthoringAttempted {
		t.Fatalf("S4 included a later-stage concern: %+v", got)
	}
	if got.SharedAcceptanceStatus != "not-attempted" {
		t.Fatalf("S4 self-accepted its Proposal: %+v", got)
	}
}

func secondApplicationS4Composition(s3 secondApplicationS3DecisionPacket) secondApplicationS4CompositionCandidate {
	semanticSrc := fmt.Sprintf(`namespace %s

data Intake {
    attribute customerId type %s
    relation customer to %s %s
        key %s -> %s
}`, s3.SemanticNamespaceCandidate, s3.CustomerIDAttributeType,
		s3.BusinessRelationTarget, s3.BusinessRelationCardinality,
		s3.BusinessRelationKeySource, s3.BusinessRelationKeyTarget)

	appSrc := fmt.Sprintf(`application %s {
    uses semantic %s
    uses semantic customer

    database %s {
        target master_data
        access read
        requires {
            contract master_data.customer ^2
        }
    }
}`, s3.ApplicationIDCandidate, s3.SemanticNamespaceCandidate, s3.DatasourceAliasCandidate)

	patternSrc := fmt.Sprintf(`pattern %s reference {
    relation %s
    datasource %s
    contract-query {
        contract master_data.customer
        query find_by_id
        input customer.customer.customer_id -> id
    }
}`, s3.PatternCandidate, s3.BusinessRelationCandidate, s3.DatasourceAliasCandidate)

	return secondApplicationS4CompositionCandidate{
		Status:                       "proposal",
		SharedAcceptanceStatus:       "not-attempted",
		SemanticSource:               semanticSrc,
		ApplicationSource:            appSrc,
		PatternSource:                patternSrc,
		AuthorizationRequirement:     s3.AuthorizationRequirement,
		EnvironmentBindingIncluded:   false,
		TargetImplementationIncluded: false,
		BusinessProcedureIncluded:    false,
		SourceAuthoringAttempted:     false,
	}
}

package main

import (
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/syntax"
)

func TestSecondApplicationSchemaSufficiencyUsesExistingApplicationAndPatternGrammar(t *testing.T) {
	appSrc := `application repair_intake {
    uses semantic customer

    database customer_master {
        target master_data
        access read
        requires {
            contract master_data.customer ^2
        }
    }
}`

	app, err := syntax.NewParser(appSrc).ParseApplication()
	if err != nil {
		t.Fatalf("existing Application grammar could not express the pre-Gate destination shape: %v", err)
	}
	if app.ID != "repair_intake" {
		t.Fatalf("Application ID=%q", app.ID)
	}
	if len(app.SemanticUses) != 1 || app.SemanticUses[0] != "customer" {
		t.Fatalf("shared Customer Semantic was not expressed explicitly: %+v", app.SemanticUses)
	}
	if len(app.Databases) != 1 {
		t.Fatalf("database requirements=%+v", app.Databases)
	}
	db := app.Databases[0]
	if db.Alias != "customer_master" {
		t.Fatalf("destination-owned datasource alias=%q", db.Alias)
	}
	if db.Alias == "master" {
		t.Fatal("origin order_entry datasource alias leaked into destination shape")
	}
	if db.TargetDatabase != "master_data" || db.Access != database.AccessRead {
		t.Fatalf("destination Database requirement=%+v", db)
	}
	if len(db.RequiredContracts) != 1 ||
		db.RequiredContracts[0].ContractID != "master_data.customer" ||
		db.RequiredContracts[0].CompatibleMajor != 2 {
		t.Fatalf("shared Contract requirement=%+v", db.RequiredContracts)
	}

	patternSrc := `pattern repair_intake.customer_reference reference {
    relation repair_intake.intake.customer
    datasource customer_master
    contract-query {
        contract master_data.customer
        query find_by_id
        input customer.customer.customer_id -> id
    }
}`

	pattern, err := syntax.NewParser(patternSrc).ParseApplicationPattern()
	if err != nil {
		t.Fatalf("existing Application Pattern grammar could not express the destination reference shape: %v", err)
	}
	if pattern.ID != "repair_intake.customer_reference" || pattern.Kind != application.PatternReference {
		t.Fatalf("destination Pattern identity/kind=%+v", pattern)
	}
	if pattern.Relation != "repair_intake.intake.customer" {
		t.Fatalf("destination-owned Relation=%q", pattern.Relation)
	}
	if pattern.Relation == "sales.order.customer" {
		t.Fatal("origin sales.order.customer Relation leaked into destination Pattern")
	}
	if pattern.Datasource != "customer_master" || pattern.Datasource == "master" {
		t.Fatalf("destination Pattern datasource=%q", pattern.Datasource)
	}
	if pattern.ContractQuery == nil {
		t.Fatal("destination Pattern lost shared Contract Query")
	}
	if pattern.ContractQuery.Contract != "master_data.customer" ||
		pattern.ContractQuery.Query != "find_by_id" {
		t.Fatalf("destination Contract Query=%+v", pattern.ContractQuery)
	}
	if len(pattern.ContractQuery.Inputs) != 1 ||
		pattern.ContractQuery.Inputs[0].TargetAttribute != "customer.customer.customer_id" ||
		pattern.ContractQuery.Inputs[0].Input != "id" {
		t.Fatalf("destination Contract Query input binding=%+v", pattern.ContractQuery.Inputs)
	}
}

func TestSecondApplicationSchemaSufficiencyUsesExistingEnvironmentBindingGrammar(t *testing.T) {
	envSrc := `environment development {
    database-instance master_data.dev {
        definition master_data
        provider oracle
    }

    bind repair_intake.customer_master -> master_data.dev
}`

	env, err := syntax.NewParser(envSrc).ParseEnvironment()
	if err != nil {
		t.Fatalf("existing Environment grammar could not express destination rebinding: %v", err)
	}
	if env.ID != "development" {
		t.Fatalf("Environment ID=%q", env.ID)
	}
	if len(env.Instances) != 1 ||
		env.Instances[0].ID != "master_data.dev" ||
		env.Instances[0].Definition != "master_data" ||
		env.Instances[0].Provider != "oracle" {
		t.Fatalf("Database instance=%+v", env.Instances)
	}
	if len(env.Bindings) != 1 {
		t.Fatalf("Environment bindings=%+v", env.Bindings)
	}
	binding := env.Bindings[0]
	if binding.Application != "repair_intake" ||
		binding.Alias != "customer_master" ||
		binding.Instance != "master_data.dev" {
		t.Fatalf("destination Environment binding=%+v", binding)
	}
	if binding.Application == "order_entry" || binding.Alias == "master" {
		t.Fatalf("origin Environment binding leaked into destination grammar probe: %+v", binding)
	}
}

func TestSecondApplicationSchemaSufficiencyDoesNotRequirePortabilitySyntax(t *testing.T) {
	// The pre-Gate destination shape is expressible with existing Canonical
	// declaration kinds. Portability remains planning / qualification context,
	// and must not silently become Application authoring syntax.
	for _, forbidden := range []string{
		"knowledge-package",
		"portable",
		"inherit",
		"override",
		"adopt-from",
		"sync-from",
	} {
		src := "application repair_intake {\n" +
			"    uses semantic customer\n" +
			"    " + forbidden + " customer.customer\n" +
			"}\n"
		if _, err := syntax.NewParser(src).ParseApplication(); err == nil {
			t.Fatalf("unsupported portability directive %q was accepted as Canonical Application syntax", forbidden)
		}
	}
}

package main

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

func referenceOrderApplication(t *testing.T, w ir.World) application.Application {
	t.Helper()
	for _, app := range w.Applications {
		if app.ID == "order_entry" {
			return app
		}
	}
	t.Fatal("order_entry Application is missing")
	return application.Application{}
}

func TestDeclaredPersistenceMatchesOnlyOneWritableApplicationTarget(t *testing.T) {
	w := referenceWorkspace(t)
	app := referenceOrderApplication(t, w)
	found, err := MatchDeclaredWriteMapping(w, app, "sales.order")
	if err != nil {
		t.Fatal(err)
	}
	if found == nil || found.Mapping.ID != "sales_core.order" ||
		found.Mapping.Database != "sales_core" ||
		found.Mapping.SemanticData != "sales.order" ||
		found.Datasource.Alias != "sales" ||
		found.Datasource.Access != database.AccessReadWrite {
		t.Fatalf("selected incorrect declared Mapping: %+v", found)
	}

	readOnly, err := MatchDeclaredWriteMapping(w, app, "customer.customer")
	if err != nil || readOnly != nil {
		t.Fatalf("read-only requirement must not satisfy write preflight: %+v, %v", readOnly, err)
	}
}

func TestDeclaredPersistenceMissingAndAmbiguousPathsFailClosed(t *testing.T) {
	w := referenceWorkspace(t)
	app := referenceOrderApplication(t, w)
	missing := w
	missing.Persistence = nil
	selected, err := MatchDeclaredWriteMapping(missing, app, "sales.order")
	if err != nil || selected != nil {
		t.Fatalf("missing Mapping was inferred: %+v, %v", selected, err)
	}

	duplicate := referenceWorkspace(t)
	for _, m := range duplicate.Persistence {
		if m.SemanticData == "sales.order" {
			duplicate.Persistence = append(duplicate.Persistence, m)
			break
		}
	}
	if _, err := MatchDeclaredWriteMapping(duplicate, app, "sales.order"); err == nil || !strings.Contains(err.Error(), "explicit target") {
		t.Fatalf("ambiguous Mapping did not fail closed: %v", err)
	}

	duplicateRequirement := referenceOrderApplication(t, w)
	for _, requirement := range duplicateRequirement.Databases {
		if requirement.Alias == "sales" {
			duplicateRequirement.Databases = append(duplicateRequirement.Databases, requirement)
			break
		}
	}
	if _, err := MatchDeclaredWriteMapping(w, duplicateRequirement, "sales.order"); err == nil {
		t.Fatal("duplicate writable requirement was accepted")
	}

	duplicateDatabase := referenceWorkspace(t)
	for _, db := range duplicateDatabase.Databases {
		if db.ID == "sales_core" {
			duplicateDatabase.Databases = append(duplicateDatabase.Databases, db)
			break
		}
	}
	if _, err := MatchDeclaredWriteMapping(duplicateDatabase, app, "sales.order"); err == nil {
		t.Fatal("duplicate Canonical Database was accepted")
	}
}

func TestCommandDeclaredPersistenceDoesNotClaimExecutionOrWriteAuthority(t *testing.T) {
	w := referenceWorkspace(t)
	for i := range w.Semantics {
		for j := range w.Semantics[i].Capabilities {
			capability := &w.Semantics[i].Capabilities[j]
			if capability.ID == "sales.order.submit" {
				capability.Effect = &semantic.DataEffect{
					Kind: semantic.DataEffectCreate, Data: "sales.order",
				}
			}
		}
	}
	report, err := InspectExplicitPattern(w, "order_entry", "order_entry.submit_order")
	if err != nil {
		t.Fatal(err)
	}
	if report.WriteDatabase != "sales_core" ||
		report.PersistenceMapping != "sales_core.order" ||
		report.WriteDatasource != "sales" ||
		!includes(report.Resolved, "writable-persistence-declarations") {
		t.Fatalf("existing Mapping and Requirement were not resolved: %+v", report)
	}
	if report.Executable || !includes(report.Unverified, "current-authority") ||
		!includes(report.Unverified, "persistence") ||
		!includes(report.Unverified, "independent-database-read") {
		t.Fatalf("declaration was promoted to an actual write: %+v", report)
	}

	loaded, err := workspace.Load("../../examples/first-capability-effect")
	if err != nil {
		t.Fatal(err)
	}
	declared, err := InspectExplicitPattern(loaded.World, "request_log", "request_log.create_request")
	if err != nil {
		t.Fatal(err)
	}
	if declared.DeclaredData != "intake.request" || declared.DataOperation != "create" ||
		declared.PersistenceMapping != "" || declared.WriteDatabase != "" ||
		declared.WriteDatasource != "" || declared.Executable {
		t.Fatalf("data intent without persistence invented a Provider: %+v", declared)
	}
}

package validation

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/semantic"
)

// unboundWorld builds a workspace with one query whose alias is not bound in any
// environment. The instances are supplied by the caller so each test controls
// what the hint can legitimately offer.
func unboundWorld(instances []environment.DatabaseInstance) ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{ID: core.CanonicalID("sales")}},
		Databases: []database.Database{{
			ID: "sales_core", Kind: "relational", Ownership: database.OwnershipManagedHere,
		}},
		Applications: []application.Application{{
			ID:           core.CanonicalID("returns_intake"),
			SemanticUses: []core.CanonicalID{"sales"},
			Databases: []application.DatabaseRequirement{{
				Alias: "sales", TargetDatabase: "sales_core", Access: database.AccessRead,
			}},
		}},
		Queries: []query.EmbeddedSQL{{
			ID:          core.CanonicalID("returns_intake.open_returns"),
			Application: core.CanonicalID("returns_intake"),
			Datasource:  "sales",
			Dialect:     "postgres",
		}},
		Environments: []environment.Environment{{
			ID:        core.CanonicalID("development"),
			Instances: instances,
		}},
	}
}

// An unbound datasource says what is missing but not what to write, and the
// binding lives in a different file from the query (#4169).
func TestUnboundDatasourceNamesTheBindingToAdd(t *testing.T) {
	// No instance for sales_core exists, so the hint may only describe the shape.
	message := diagnosticMessage(t, unboundWorld([]environment.DatabaseInstance{{
		ID: "master_data.dev", Definition: "master_data", Provider: "oracle",
	}}), "UNBOUND_DATASOURCE")
	for _, want := range []string{
		"bind returns_intake.sales",
		"inside an environment block",
		"<database-instance>",
	} {
		if !strings.Contains(message, want) {
			t.Errorf("message does not contain %q: %s", want, message)
		}
	}
}

// When an instance for the targeted database is already declared, the hint names
// it instead of a placeholder.
func TestUnboundDatasourcePrefersADeclaredInstance(t *testing.T) {
	message := diagnosticMessage(t, unboundWorld([]environment.DatabaseInstance{{
		ID: "sales_core.dev", Definition: "sales_core", Provider: "postgres",
	}}), "UNBOUND_DATASOURCE")
	if !strings.Contains(message, "sales_core.dev") {
		t.Errorf("message does not name the declared instance: %s", message)
	}
	if strings.Contains(message, "<database-instance>") {
		t.Errorf("message still shows a placeholder although an instance exists: %s", message)
	}
}

// Fail closed: an instance for a different database must never be offered, because
// following that hint would bind the alias to the wrong database.
func TestUnboundDatasourceDoesNotBorrowAnotherDatabasesInstance(t *testing.T) {
	message := diagnosticMessage(t, unboundWorld([]environment.DatabaseInstance{{
		ID: "master_data.dev", Definition: "master_data", Provider: "oracle",
	}}), "UNBOUND_DATASOURCE")
	if strings.Contains(message, "master_data.dev") {
		t.Errorf("message names another database's instance: %s", message)
	}
	if !strings.Contains(message, "<database-instance>") {
		t.Errorf("message does not fall back to the shape: %s", message)
	}
}

func TestUnboundDatasourceDoesNotSuggestIncompatibleProvider(t *testing.T) {
	w := unboundWorld([]environment.DatabaseInstance{
		{ID: "sales_core.oracle", Definition: "sales_core", Provider: "oracle"},
		{ID: "sales_core.pg", Definition: "sales_core", Provider: "postgres"},
	})
	message := diagnosticMessage(t, w, "UNBOUND_DATASOURCE")
	if strings.Contains(message, "sales_core.oracle") || !strings.Contains(message, "sales_core.pg") {
		t.Fatalf("hint must select a dialect-compatible instance: %s", message)
	}
	w.Environments[0].Instances = w.Environments[0].Instances[:1]
	message = diagnosticMessage(t, w, "UNBOUND_DATASOURCE")
	if strings.Contains(message, "sales_core.oracle") || !strings.Contains(message, "<database-instance>") {
		t.Fatalf("incompatible instance must leave the binding unresolved: %s", message)
	}
}

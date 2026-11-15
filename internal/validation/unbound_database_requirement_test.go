package validation

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/environment"
	"tei/internal/ir"
)

// An application's database block is a requirement for a datasource alias. If an
// environment exists and nothing binds that alias, the requirement is unbound and
// the application cannot reach its database (#4462).
func TestUnboundDatabaseRequirementIsReported(t *testing.T) {
	world := ir.World{
		Applications: []application.Application{{
			ID: "reporting",
			Databases: []application.DatabaseRequirement{{
				Alias: "sales", TargetDatabase: "sales_core", Access: database.AccessRead,
			}},
		}},
		Databases: []database.Database{{ID: "sales_core", Kind: "relational", Ownership: database.OwnershipManagedHere}},
		Environments: []environment.Environment{{
			ID: "development",
			Instances: []environment.DatabaseInstance{{
				ID: "sales_core.dev", Definition: "sales_core", Provider: environment.ProviderPostgres,
			}},
		}},
	}
	diagnostics := Validate(world)
	var found bool
	for _, d := range diagnostics {
		if d.Code == "UNBOUND_DATABASE_REQUIREMENT" {
			found = true
			if d.Severity != Warning {
				t.Errorf("severity = %q, want warning", d.Severity)
			}
			if !strings.Contains(d.Message, "bind reporting.sales -> sales_core.dev") {
				t.Errorf("message does not name the bind to write: %s", d.Message)
			}
		}
	}
	if !found {
		t.Fatalf("an unbound application database was not reported: %v", diagnostics)
	}
}

// Bound is bound: the warning must not fire once an environment binds the alias.
func TestBoundDatabaseRequirementIsNotReported(t *testing.T) {
	world := ir.World{
		Applications: []application.Application{{
			ID:        "reporting",
			Databases: []application.DatabaseRequirement{{Alias: "sales", TargetDatabase: "sales_core", Access: database.AccessRead}},
		}},
		Databases: []database.Database{{ID: "sales_core", Kind: "relational", Ownership: database.OwnershipManagedHere}},
		Environments: []environment.Environment{{
			ID:        "development",
			Instances: []environment.DatabaseInstance{{ID: "sales_core.dev", Definition: "sales_core", Provider: environment.ProviderPostgres}},
			Bindings: []environment.ApplicationDatabaseBinding{{
				Application: "reporting", Alias: "sales", Instance: "sales_core.dev",
			}},
		}},
	}
	for _, d := range Validate(world) {
		if d.Code == "UNBOUND_DATABASE_REQUIREMENT" {
			t.Fatalf("a bound requirement was reported: %s", d.Message)
		}
	}
}

// A workspace with no environment cannot bind anything yet, so it must stay
// quiet: the five partial example workspaces rely on that.
func TestUnboundDatabaseRequirementStaysQuietWithoutEnvironments(t *testing.T) {
	world := ir.World{
		Applications: []application.Application{{
			ID:        "reporting",
			Databases: []application.DatabaseRequirement{{Alias: "sales", TargetDatabase: "sales_core", Access: database.AccessRead}},
		}},
		Databases: []database.Database{{ID: "sales_core", Kind: "relational", Ownership: database.OwnershipManagedHere}},
	}
	for _, d := range Validate(world) {
		if d.Code == "UNBOUND_DATABASE_REQUIREMENT" {
			t.Fatalf("warned in a workspace with no environments: %s", d.Message)
		}
	}
}

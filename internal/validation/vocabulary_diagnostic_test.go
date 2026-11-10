package validation

import (
	"strings"
	"testing"

	"tei/internal/application"
	apprelease "tei/internal/application/release"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/migration"
	"tei/internal/database/persistence"
	"tei/internal/database/release"
	"tei/internal/ir"
	"tei/internal/releaseplan"
)

// diagnosticMessage returns the message of the first diagnostic with the code.
func diagnosticMessage(t *testing.T, world ir.World, code string) string {
	t.Helper()
	for _, d := range Validate(world) {
		if string(d.Code) == code {
			return d.Message
		}
	}
	t.Fatalf("missing diagnostic %s", code)
	return ""
}

// An author who writes an unknown storage type, nullability or access mode is
// told only "invalid" unless the message names the alternatives. Each closed
// vocabulary now lists itself, and this pins that the list in the message is the
// list the validator accepts (#2094 round 54).
func TestClosedVocabularyDiagnosticsNameTheirValues(t *testing.T) {
	cases := []struct {
		name   string
		world  ir.World
		code   string
		values []string
	}{
		{
			name: "access",
			world: ir.World{
				Applications: []application.Application{{
					ID: "order_entry",
					Databases: []application.DatabaseRequirement{{
						Alias:  core.LocalName("sales"),
						Access: database.Access("nothing"),
					}},
				}},
			},
			code:   "INVALID_DATABASE_ACCESS",
			values: []string{"read", "write", "read-write"},
		},
		{
			name: "storage type",
			world: ir.World{
				Databases: []database.Database{{ID: "sales_core", Kind: "relational"}},
				Persistence: []persistence.Mapping{{
					ID:           "sales_core.order",
					Database:     "sales_core",
					SemanticData: "sales.order",
					Columns: []persistence.ColumnMapping{{
						Attribute: "sales.order.order_id", Column: "order_id",
						Storage: persistence.StorageType("NUMBERISH"), Nullability: persistence.Required,
					}},
				}},
			},
			code:   "INVALID_PERSISTENCE_STORAGE_TYPE",
			values: []string{"int64", "varchar", "text", "datetime-tz", "decimal", "json"},
		},
		{
			name: "nullability",
			world: ir.World{
				Databases: []database.Database{{ID: "sales_core", Kind: "relational"}},
				Persistence: []persistence.Mapping{{
					ID:           "sales_core.order",
					Database:     "sales_core",
					SemanticData: "sales.order",
					Columns: []persistence.ColumnMapping{{
						Attribute: "sales.order.order_id", Column: "order_id",
						Storage: persistence.StorageInt64, Nullability: persistence.Nullability("maybe"),
					}},
				}},
			},
			code:   "INVALID_PERSISTENCE_NULLABILITY",
			values: []string{"required", "optional"},
		},
		{
			name: "database ownership",
			world: ir.World{
				Databases: []database.Database{{ID: "sales_core", Kind: "relational", Ownership: database.Ownership("managed")}},
			},
			code:   "INVALID_DATABASE_OWNERSHIP",
			values: []string{"managed-here", "external-observed"},
		},
		{
			name: "migration recovery",
			world: ir.World{
				Databases:  []database.Database{{ID: "sales_core", Kind: "relational", Ownership: database.OwnershipManagedHere}},
				Migrations: []migration.Definition{{ID: "m1", Database: "sales_core", Recovery: migration.RecoveryStrategy("rollback")}},
			},
			code:   "INVALID_MIGRATION_RECOVERY",
			values: []string{"forward-fix", "explicit-rollback", "not-reversible"},
		},
		{
			name: "migration step kind",
			world: ir.World{
				Databases: []database.Database{{ID: "sales_core", Kind: "relational", Ownership: database.OwnershipManagedHere}},
				Migrations: []migration.Definition{{
					ID: "m1", Database: "sales_core", Recovery: migration.RecoveryForwardFix,
					Steps: []migration.Step{{Kind: migration.StepKind("check")}},
				}},
			},
			code:   "INVALID_MIGRATION_STEP_KIND",
			values: []string{"schema", "data", "verify"},
		},
		{
			name: "application release access",
			world: ir.World{
				ApplicationReleases: []apprelease.Definition{{
					Application: "order_entry", Version: "1.0.0",
					Databases: []application.DatabaseRequirement{{Alias: "sales", Access: database.Access("nothing")}},
				}},
			},
			code:   "INVALID_APPLICATION_RELEASE_DATABASE_ACCESS",
			values: []string{"read", "write", "read-write"},
		},
		{
			name: "database release data readiness",
			world: ir.World{
				Databases:        []database.Database{{ID: "sales_core", Kind: "relational", Ownership: database.OwnershipManagedHere}},
				DatabaseReleases: []release.Definition{{Database: "sales_core", Version: "1.0.0", DataReadiness: release.DataReadiness("partial")}},
			},
			code:   "INVALID_DATABASE_RELEASE_DATA_READINESS",
			values: []string{"ready", "preparing", "not-ready", "unknown"},
		},
		{
			name: "solution release step kind",
			world: ir.World{
				SolutionReleases: []releaseplan.Definition{{
					ID: "release", Version: "1.0.0",
					Steps: []releaseplan.Step{{ID: "s1", Kind: releaseplan.StepKind("database"), Phase: releaseplan.PhaseExpand}},
				}},
			},
			code:   "INVALID_SOLUTION_RELEASE_STEP_KIND",
			values: []string{"database-release", "application-release", "application-retire", "database-gate"},
		},
		{
			name: "solution release phase",
			world: ir.World{
				SolutionReleases: []releaseplan.Definition{{
					ID: "release", Version: "1.0.0",
					Steps: []releaseplan.Step{{ID: "s1", Kind: releaseplan.StepDatabaseRelease, Phase: releaseplan.Phase("cutover")}},
				}},
			},
			code:   "INVALID_SOLUTION_RELEASE_PHASE",
			values: []string{"expand", "populate", "transition", "rollout", "contract", "verify", "other"},
		},
		{
			name: "rollout strategy",
			world: ir.World{
				SolutionReleases: []releaseplan.Definition{{
					ID: "release", Version: "1.0.0",
					Steps: []releaseplan.Step{{
						ID: "s1", Kind: releaseplan.StepDatabaseRelease, Phase: releaseplan.PhaseExpand,
						RolloutStrategy: releaseplan.RolloutStrategy("rolling"),
					}},
				}},
			},
			code:   "INVALID_ROLLOUT_STRATEGY",
			values: []string{"direct", "canary", "blue-green", "gradual"},
		},
	}
	for _, c := range cases {
		message := diagnosticMessage(t, c.world, c.code)
		for _, value := range c.values {
			if !strings.Contains(message, value) {
				t.Errorf("%s: message does not name %q: %s", c.name, value, message)
			}
		}
		if !strings.Contains(message, "valid values are") {
			t.Errorf("%s: message does not introduce the list: %s", c.name, message)
		}
	}
}

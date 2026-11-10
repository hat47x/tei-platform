package validation

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/migration"
	"tei/internal/database/release"
	"tei/internal/ir"
	"tei/internal/releaseplan"
)

// The migration and release-plan vocabulary batch (#4094).
func TestMigrationAndReleaseVocabularyDiagnosticsNameTheirValues(t *testing.T) {
	cases := []struct {
		name   string
		world  ir.World
		code   string
		values []string
	}{
		{
			name: "migration recovery",
			world: ir.World{
				Databases: []database.Database{{ID: "sales_core", Kind: "relational", Ownership: database.OwnershipManagedHere}},
				Migrations: []migration.Definition{{
					ID: "probe", Database: "sales_core", Recovery: migration.RecoveryStrategy("gibberish"),
				}},
			},
			code:   "INVALID_MIGRATION_RECOVERY",
			values: []string{"forward-fix", "explicit-rollback", "not-reversible"},
		},
		{
			name: "migration step kind",
			world: ir.World{
				Databases: []database.Database{{ID: "sales_core", Kind: "relational", Ownership: database.OwnershipManagedHere}},
				Migrations: []migration.Definition{{
					ID: "probe", Database: "sales_core", Recovery: migration.RecoveryForwardFix,
					Steps: []migration.Step{{Kind: migration.StepKind("gibberish")}},
				}},
			},
			code:   "INVALID_MIGRATION_STEP_KIND",
			values: []string{"schema", "data", "verify"},
		},
		{
			name: "data readiness",
			world: ir.World{
				Databases: []database.Database{{ID: "sales_core", Kind: "relational", Ownership: database.OwnershipManagedHere}},
				DatabaseReleases: []release.Definition{{
					Database: "sales_core", Version: "9.9.9",
					DataReadiness: release.DataReadiness("gibberish"),
				}},
			},
			code:   "INVALID_DATABASE_RELEASE_DATA_READINESS",
			values: []string{"ready", "preparing", "not-ready", "unknown"},
		},
		{
			name: "solution release step kind",
			world: ir.World{SolutionReleases: []releaseplan.Definition{{
				ID: "probe", Version: "1.0.0",
				Steps: []releaseplan.Step{{
					ID: core.LocalName("one"), Kind: releaseplan.StepKind("gibberish"), Phase: releaseplan.PhaseExpand,
				}},
			}}},
			code:   "INVALID_SOLUTION_RELEASE_STEP_KIND",
			values: []string{"database-release", "application-release", "application-retire", "database-gate"},
		},
		{
			name: "solution release phase",
			world: ir.World{SolutionReleases: []releaseplan.Definition{{
				ID: "probe", Version: "1.0.0",
				Steps: []releaseplan.Step{{
					ID: core.LocalName("one"), Kind: releaseplan.StepDatabaseGate, Phase: releaseplan.Phase("gibberish"),
				}},
			}}},
			code:   "INVALID_SOLUTION_RELEASE_PHASE",
			values: []string{"expand", "populate", "transition", "rollout", "contract", "verify", "other"},
		},
		{
			name: "rollout strategy",
			world: ir.World{SolutionReleases: []releaseplan.Definition{{
				ID: "probe", Version: "1.0.0",
				Steps: []releaseplan.Step{{
					ID: core.LocalName("one"), Kind: releaseplan.StepDatabaseGate, Phase: releaseplan.PhaseExpand,
					RolloutStrategy: releaseplan.RolloutStrategy("gibberish"),
				}},
			}}},
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

// The empty rollout strategy is accepted, so the message must not imply that only
// the listed values work.
func TestRolloutStrategyDiagnosticSaysTheStrategyMayBeOmitted(t *testing.T) {
	message := diagnosticMessage(t, ir.World{SolutionReleases: []releaseplan.Definition{{
		ID: "probe", Version: "1.0.0",
		Steps: []releaseplan.Step{{
			ID: core.LocalName("one"), Kind: releaseplan.StepDatabaseGate, Phase: releaseplan.PhaseExpand,
			RolloutStrategy: releaseplan.RolloutStrategy("gibberish"),
		}},
	}}}, "INVALID_ROLLOUT_STRATEGY")
	if !strings.Contains(message, "omit") {
		t.Errorf("message does not say the strategy may be omitted, although the empty strategy is accepted: %s", message)
	}
}

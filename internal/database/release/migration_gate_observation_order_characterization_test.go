package release_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	dbrelease "tei/internal/database/release"
)

// MG-20/MG-29 characterize another evidence ceiling: a structurally
// compatible "external-observed" snapshot does not certify a fresh observation
// or independently verified migration history. These fields are not proof of
// actual data migration, current grants, or client compatibility.
func TestMG29ObservedAtDoesNotQualifyCurrentGateEvidence(t *testing.T) {
	dbID := core.MustCanonicalID("sales_core")
	contractID := core.MustCanonicalID("sales_core.order")
	app := application.Application{
		ID: core.MustCanonicalID("order_entry"),
		Databases: []application.DatabaseRequirement{{
			Alias:          core.LocalName("sales"),
			TargetDatabase: dbID,
			Access:         database.AccessRead,
			RequiredContracts: []database.ContractRequirement{{
				ContractID: contractID, CompatibleMajor: 5,
			}},
		}},
	}
	observed := dbrelease.StateSnapshot{
		Database:      dbID,
		Source:        dbrelease.StateFromObserved,
		DataReadiness: dbrelease.DataReady,
		Provides: []dbrelease.ProvidedContract{{
			ID: contractID, Version: database.ContractVersion{Major: 5},
		}},
	}
	for _, stamp := range []string{"", "2020-01-01T00:00:00Z"} {
		t.Run("observed-"+stamp, func(t *testing.T) {
			state := observed
			state.ObservedAt = stamp
			got := dbrelease.CheckApplication(app, []dbrelease.StateSnapshot{state})
			if got.Overall != dbrelease.Compatible {
				t.Fatalf("expected to characterize contract-only result for ObservedAt=%q, got %s", stamp, got.Overall)
			}
		})
	}
}

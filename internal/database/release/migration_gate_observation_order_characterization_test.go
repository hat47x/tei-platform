package release_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	dbrelease "tei/internal/database/release"
)

// MG-29 documents a CURRENT GAP, not an acceptable migration-gate behavior.
// CheckApplication indexes snapshots by Database ID and silently retains the
// final duplicate. Merely reordering old/new observations can reverse the
// compatibility result. A future owner-approved fix should make this result
// unambiguous rather than pinning this ordering behavior.
func TestMG29DuplicateDatabaseObservationsAreOrderDependent(t *testing.T) {
	dbID := core.MustCanonicalID("sales_core")
	contractID := core.MustCanonicalID("sales_core.order")
	app := application.Application{
		ID: core.MustCanonicalID("order_entry"),
		Databases: []application.DatabaseRequirement{{
			Alias: core.LocalName("sales"),
			TargetDatabase: dbID,
			Access: database.AccessReadWrite,
			RequiredContracts: []database.ContractRequirement{{
				ContractID: contractID, CompatibleMajor: 5,
			}},
		}},
	}
	oldObserved := dbrelease.StateSnapshot{
		Database: dbID,
		Source: dbrelease.StateFromObserved,
		ObservedAt: "2026-11-05T09:00:00Z",
		DataReadiness: dbrelease.DataReady,
		Provides: []dbrelease.ProvidedContract{{
			ID: contractID, Version: database.ContractVersion{Major: 4},
		}},
	}
	newObserved := dbrelease.StateSnapshot{
		Database: dbID,
		Source: dbrelease.StateFromObserved,
		ObservedAt: "2026-11-05T10:00:00Z",
		DataReadiness: dbrelease.DataReady,
		Provides: []dbrelease.ProvidedContract{{
			ID: contractID, Version: database.ContractVersion{Major: 5},
		}},
	}

	// Same observations; only slice order differs.
	compatible := dbrelease.CheckApplication(app, []dbrelease.StateSnapshot{oldObserved, newObserved})
	blocked := dbrelease.CheckApplication(app, []dbrelease.StateSnapshot{newObserved, oldObserved})
	if compatible.Overall != dbrelease.Compatible || blocked.Overall != dbrelease.Blocked {
		t.Fatalf("expected to characterize unsafe last-observation-wins behavior: old-new=%s new-old=%s", compatible.Overall, blocked.Overall)
	}
}

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
			Alias: core.LocalName("sales"),
			TargetDatabase: dbID,
			Access: database.AccessRead,
			RequiredContracts: []database.ContractRequirement{{
				ContractID: contractID, CompatibleMajor: 5,
			}},
		}},
	}
	observed := dbrelease.StateSnapshot{
		Database: dbID,
		Source: dbrelease.StateFromObserved,
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

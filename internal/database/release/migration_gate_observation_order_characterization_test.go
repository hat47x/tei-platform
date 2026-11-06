package release_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	dbrelease "tei/internal/database/release"
)

// MG-29a safety regression: duplicate states for one DB must be rejected
// independent of observation order; an observed timestamp is not authority.
func TestMG29DuplicateDatabaseObservationsAreRejectedRegardlessOfOrder(t *testing.T) {
	dbID := core.MustCanonicalID("sales_core")
	contractID := core.MustCanonicalID("sales_core.order")
	app := application.Application{
		ID: core.MustCanonicalID("order_entry"),
		Databases: []application.DatabaseRequirement{{
			Alias:          core.LocalName("sales"),
			TargetDatabase: dbID,
			Access:         database.AccessReadWrite,
			RequiredContracts: []database.ContractRequirement{{
				ContractID: contractID, CompatibleMajor: 5,
			}},
		}},
	}
	oldObserved := dbrelease.StateSnapshot{
		Database:      dbID,
		Source:        dbrelease.StateFromObserved,
		ObservedAt:    "2026-11-05T09:00:00Z",
		DataReadiness: dbrelease.DataReady,
		Provides: []dbrelease.ProvidedContract{{
			ID: contractID, Version: database.ContractVersion{Major: 4},
		}},
	}
	newObserved := dbrelease.StateSnapshot{
		Database:      dbID,
		Source:        dbrelease.StateFromObserved,
		ObservedAt:    "2026-11-05T10:00:00Z",
		DataReadiness: dbrelease.DataReady,
		Provides: []dbrelease.ProvidedContract{{
			ID: contractID, Version: database.ContractVersion{Major: 5},
		}},
	}

	// Same observations; only slice order differs.
	compatible := dbrelease.CheckApplication(app, []dbrelease.StateSnapshot{oldObserved, newObserved})
	blocked := dbrelease.CheckApplication(app, []dbrelease.StateSnapshot{newObserved, oldObserved})
	if compatible.Overall != dbrelease.Unknown || blocked.Overall != dbrelease.Unknown ||
		len(compatible.Checks) != 1 || len(blocked.Checks) != 1 ||
		compatible.Checks[0].State != dbrelease.Unknown || blocked.Checks[0].State != dbrelease.Unknown {
		t.Fatalf("duplicate database states must be rejected regardless of order: old-new=%#v new-old=%#v", compatible, blocked)
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

// MG-29b safety regression: duplicate contract declarations for one DB
// must be rejected rather than choosing the final declaration by input order.
func TestMG29DuplicateProvidedContractVersionsAreRejectedRegardlessOfOrder(t *testing.T) {
	dbID := core.MustCanonicalID("sales_core")
	contractID := core.MustCanonicalID("sales_core.order")
	app := application.Application{
		ID: core.MustCanonicalID("order_entry"),
		Databases: []application.DatabaseRequirement{{
			Alias: core.LocalName("sales"), TargetDatabase: dbID,
			Access: database.AccessRead,
			RequiredContracts: []database.ContractRequirement{{
				ContractID: contractID, CompatibleMajor: 5,
			}},
		}},
	}
	older := dbrelease.ProvidedContract{ID: contractID, Version: database.ContractVersion{Major: 4}}
	newer := dbrelease.ProvidedContract{ID: contractID, Version: database.ContractVersion{Major: 5}}
	base := dbrelease.StateSnapshot{
		Database: dbID, Source: dbrelease.StateFromObserved,
		DataReadiness: dbrelease.DataReady,
	}

	first := base
	first.Provides = []dbrelease.ProvidedContract{older, newer}
	second := base
	second.Provides = []dbrelease.ProvidedContract{newer, older}

	compatible := dbrelease.CheckApplication(app, []dbrelease.StateSnapshot{first})
	blocked := dbrelease.CheckApplication(app, []dbrelease.StateSnapshot{second})
	if compatible.Overall != dbrelease.Unknown || blocked.Overall != dbrelease.Unknown ||
		len(compatible.Checks) != 1 || len(blocked.Checks) != 1 ||
		compatible.Checks[0].State != dbrelease.Unknown || blocked.Checks[0].State != dbrelease.Unknown {
		t.Fatalf("duplicate provided contracts must be rejected regardless of order: old-new=%#v new-old=%#v",
			compatible, blocked)
	}
}

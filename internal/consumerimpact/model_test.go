package consumerimpact

import (
	"testing"

	"tei/internal/application"
	apprelease "tei/internal/application/release"
	"tei/internal/core"
	"tei/internal/database"
	dbrelease "tei/internal/database/release"
	"tei/internal/deployment"
)

func TestAnalyzeTargetBlocksActiveConsumerWhenContractWithdrawn(t *testing.T) {
	legacy := core.MustCanonicalID("sales_core.order_legacy")
	modern := core.MustCanonicalID("sales_core.order")
	target := dbrelease.Definition{
		Database: core.MustCanonicalID("sales_core"), Version: "4.2.0", DataReadiness: dbrelease.DataReady,
		Provides: []dbrelease.ProvidedContract{{ID: modern, Version: database.ContractVersion{Major: 4, Minor: 2}}},
	}
	state := deployment.Snapshot{
		Environment:        core.MustCanonicalID("production"),
		ActiveApplications: []deployment.ApplicationReleaseRef{{Application: core.MustCanonicalID("order_entry"), Version: "3.7.0"}},
		Databases: []dbrelease.StateSnapshot{{Database: core.MustCanonicalID("sales_core"), Source: dbrelease.StateFromRelease, Release: "4.1.0", DataReadiness: dbrelease.DataReady, Provides: []dbrelease.ProvidedContract{
			{ID: legacy, Version: database.ContractVersion{Major: 1, Minor: 0}},
			{ID: modern, Version: database.ContractVersion{Major: 4, Minor: 1}},
		}}},
	}
	apps := []apprelease.Definition{{Application: core.MustCanonicalID("order_entry"), Version: "3.7.0", Databases: []application.DatabaseRequirement{{Alias: core.LocalName("sales"), TargetDatabase: core.MustCanonicalID("sales_core"), Access: database.AccessRead, RequiredContracts: []database.ContractRequirement{{ContractID: legacy, CompatibleMajor: 1}}}}}}

	got := AnalyzeTarget(target, state, apps)
	if got.Safe {
		t.Fatal("expected unsafe target")
	}
	if len(got.Consumers) != 1 || got.Consumers[0].Contract != legacy {
		t.Fatalf("unexpected consumers: %+v", got.Consumers)
	}
	if len(got.ContractChanges) != 2 {
		t.Fatalf("expected legacy withdrawal and modern minor change, got %+v", got.ContractChanges)
	}
}

func TestAnalyzeTargetDoesNotTreatNextMajorAsCompatible(t *testing.T) {
	c := core.MustCanonicalID("sales_core.order")
	target := dbrelease.Definition{Database: core.MustCanonicalID("sales_core"), Version: "5.0.0", DataReadiness: dbrelease.DataReady, Provides: []dbrelease.ProvidedContract{{ID: c, Version: database.ContractVersion{Major: 5}}}}
	state := deployment.Snapshot{Environment: core.MustCanonicalID("production"), ActiveApplications: []deployment.ApplicationReleaseRef{{Application: core.MustCanonicalID("order_entry"), Version: "3.8.0"}}}
	apps := []apprelease.Definition{{Application: core.MustCanonicalID("order_entry"), Version: "3.8.0", Databases: []application.DatabaseRequirement{{Alias: core.LocalName("sales"), TargetDatabase: core.MustCanonicalID("sales_core"), RequiredContracts: []database.ContractRequirement{{ContractID: c, CompatibleMajor: 4}}}}}}
	got := AnalyzeTarget(target, state, apps)
	if got.Safe || len(got.Consumers) != 1 {
		t.Fatalf("expected incompatible major consumer, got %+v", got)
	}
}

package releasegraph

import (
	"testing"

	"tei/internal/application"
	apprelease "tei/internal/application/release"
	"tei/internal/core"
	"tei/internal/database"
	dbrelease "tei/internal/database/release"
)

func TestBuildRequiresMultipleDatabaseStatesBeforeApplicationRelease(t *testing.T) {
	ar := apprelease.Definition{Application: core.MustCanonicalID("order_entry"), Version: "3.8.0", Databases: []application.DatabaseRequirement{
		{Alias: core.LocalName("sales"), TargetDatabase: core.MustCanonicalID("sales_core"), Access: database.AccessReadWrite, RequiredContracts: []database.ContractRequirement{{ContractID: core.MustCanonicalID("sales_core.order"), CompatibleMajor: 4}}},
		{Alias: core.LocalName("master"), TargetDatabase: core.MustCanonicalID("master_data"), Access: database.AccessRead, RequiredContracts: []database.ContractRequirement{{ContractID: core.MustCanonicalID("master_data.customer"), CompatibleMajor: 2}}},
	}}
	states := []dbrelease.StateSnapshot{
		{Database: core.MustCanonicalID("sales_core"), Source: dbrelease.StateFromRelease, Release: "4.2.0", DataReadiness: dbrelease.DataReady, Provides: []dbrelease.ProvidedContract{{ID: core.MustCanonicalID("sales_core.order"), Version: database.ContractVersion{Major: 4, Minor: 2}}}},
		{Database: core.MustCanonicalID("master_data"), Source: dbrelease.StateFromObserved, DataReadiness: dbrelease.DataReady, Provides: []dbrelease.ProvidedContract{{ID: core.MustCanonicalID("master_data.customer"), Version: database.ContractVersion{Major: 2, Minor: 4}}}},
	}
	g := Build(ar, states)
	if !g.Deployable {
		t.Fatalf("expected deployable graph, reasons=%v", g.Reasons)
	}
	contractEdges := 0
	for _, e := range g.Edges {
		if e.Relation == "requires-contract" {
			contractEdges++
		}
	}
	if contractEdges != 2 {
		t.Fatalf("expected two independent database prerequisites, got %d", contractEdges)
	}
}

func TestBuildBlocksWhenOneDatabaseIsNotReady(t *testing.T) {
	ar := apprelease.Definition{Application: core.MustCanonicalID("a"), Version: "1.0.0", Databases: []application.DatabaseRequirement{{Alias: core.LocalName("db"), TargetDatabase: core.MustCanonicalID("d"), Access: database.AccessRead, RequiredContracts: []database.ContractRequirement{{ContractID: core.MustCanonicalID("d.c"), CompatibleMajor: 1}}}}}
	states := []dbrelease.StateSnapshot{{Database: core.MustCanonicalID("d"), Source: dbrelease.StateFromRelease, Release: "1.0.0", DataReadiness: dbrelease.DataPreparing, Provides: []dbrelease.ProvidedContract{{ID: core.MustCanonicalID("d.c"), Version: database.ContractVersion{Major: 1}}}}}
	g := Build(ar, states)
	if g.Deployable {
		t.Fatal("expected non-ready database to block application release")
	}
}

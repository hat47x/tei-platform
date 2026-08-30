package releaseplan

import (
	"testing"

	"tei/internal/application"
	apprelease "tei/internal/application/release"
	"tei/internal/core"
	"tei/internal/database"
	dbrelease "tei/internal/database/release"
	"tei/internal/deployment"
)

func referenceReleases() ([]apprelease.Definition, []dbrelease.Definition) {
	apps := []apprelease.Definition{{Application: core.MustCanonicalID("order_entry"), Version: "3.8.0", Databases: []application.DatabaseRequirement{
		{Alias: core.LocalName("sales"), TargetDatabase: core.MustCanonicalID("sales_core"), Access: database.AccessReadWrite, RequiredContracts: []database.ContractRequirement{{ContractID: core.MustCanonicalID("sales_core.order"), CompatibleMajor: 4}}},
		{Alias: core.LocalName("master"), TargetDatabase: core.MustCanonicalID("master_data"), Access: database.AccessRead, RequiredContracts: []database.ContractRequirement{{ContractID: core.MustCanonicalID("master_data.customer"), CompatibleMajor: 2}}},
	}}}
	dbs := []dbrelease.Definition{{Database: core.MustCanonicalID("sales_core"), Version: "4.2.0", DataReadiness: dbrelease.DataReady, Provides: []dbrelease.ProvidedContract{{ID: core.MustCanonicalID("sales_core.order"), Version: database.ContractVersion{Major: 4, Minor: 2}}}}}
	return apps, dbs
}

func TestCompileAddsContractPrerequisites(t *testing.T) {
	apps, dbs := referenceReleases()
	def := Definition{ID: core.MustCanonicalID("rollout"), Version: "1.0.0", Steps: []Step{
		{ID: core.LocalName("sales"), Kind: StepDatabaseRelease, Phase: PhaseExpand, Database: core.MustCanonicalID("sales_core"), Version: "4.2.0"},
		{ID: core.LocalName("master"), Kind: StepDatabaseGate, Phase: PhaseTransition, Database: core.MustCanonicalID("master_data"), Contract: core.MustCanonicalID("master_data.customer"), CompatibleMajor: 2},
		{ID: core.LocalName("app"), Kind: StepApplicationRelease, Phase: PhaseRollout, Application: core.MustCanonicalID("order_entry"), Version: "3.8.0"},
	}}
	got := Compile(def, apps, dbs)
	if !got.Executable {
		t.Fatalf("expected executable, findings=%v", got.Findings)
	}
	var contractEdges int
	for _, e := range got.Edges {
		if e.Relation == "requires-contract" {
			contractEdges++
		}
	}
	if contractEdges != 2 {
		t.Fatalf("expected 2 contract prerequisite edges, got %d", contractEdges)
	}
}

func TestCompileRejectsAmbiguousContractProviders(t *testing.T) {
	apps, dbs := referenceReleases()
	dbs = append(dbs, dbrelease.Definition{Database: core.MustCanonicalID("sales_core"), Version: "4.3.0", DataReadiness: dbrelease.DataReady, Provides: []dbrelease.ProvidedContract{{ID: core.MustCanonicalID("sales_core.order"), Version: database.ContractVersion{Major: 4, Minor: 3}}}})
	def := Definition{ID: core.MustCanonicalID("rollout"), Version: "1.0.0", Steps: []Step{
		{ID: core.LocalName("sales42"), Kind: StepDatabaseRelease, Phase: PhaseExpand, Database: core.MustCanonicalID("sales_core"), Version: "4.2.0"},
		{ID: core.LocalName("sales43"), Kind: StepDatabaseRelease, Phase: PhaseExpand, Database: core.MustCanonicalID("sales_core"), Version: "4.3.0"},
		{ID: core.LocalName("master"), Kind: StepDatabaseGate, Phase: PhaseTransition, Database: core.MustCanonicalID("master_data"), Contract: core.MustCanonicalID("master_data.customer"), CompatibleMajor: 2},
		{ID: core.LocalName("app"), Kind: StepApplicationRelease, Phase: PhaseRollout, Application: core.MustCanonicalID("order_entry"), Version: "3.8.0"},
	}}
	got := Compile(def, apps, dbs)
	if got.Executable {
		t.Fatal("expected ambiguous providers to require explicit disambiguation")
	}
}

func TestCompileRejectsCycle(t *testing.T) {
	def := Definition{ID: core.MustCanonicalID("cycle"), Version: "1.0.0", Steps: []Step{
		{ID: core.LocalName("a"), Kind: StepDatabaseGate, Phase: PhaseOther, Database: core.MustCanonicalID("d"), Contract: core.MustCanonicalID("d.c"), CompatibleMajor: 1},
		{ID: core.LocalName("b"), Kind: StepDatabaseGate, Phase: PhaseOther, Database: core.MustCanonicalID("e"), Contract: core.MustCanonicalID("e.c"), CompatibleMajor: 1},
	}, After: []Dependency{{Step: core.LocalName("a"), After: core.LocalName("b")}, {Step: core.LocalName("b"), After: core.LocalName("a")}}}
	got := Compile(def, nil, nil)
	if got.Executable {
		t.Fatal("expected cycle to block release plan")
	}
}

func TestCompileWithDeploymentAddsRetirementGateBeforeContractWithdrawal(t *testing.T) {
	apps := []apprelease.Definition{
		{Application: core.MustCanonicalID("order_entry"), Version: "3.7.0", Databases: []application.DatabaseRequirement{
			{Alias: core.LocalName("sales"), TargetDatabase: core.MustCanonicalID("sales_core"), RequiredContracts: []database.ContractRequirement{{ContractID: core.MustCanonicalID("sales_core.order_legacy"), CompatibleMajor: 1}}},
			{Alias: core.LocalName("master"), TargetDatabase: core.MustCanonicalID("master_data"), RequiredContracts: []database.ContractRequirement{{ContractID: core.MustCanonicalID("master_data.customer"), CompatibleMajor: 2}}},
		}},
		{Application: core.MustCanonicalID("order_entry"), Version: "3.8.0", Databases: []application.DatabaseRequirement{
			{Alias: core.LocalName("sales"), TargetDatabase: core.MustCanonicalID("sales_core"), RequiredContracts: []database.ContractRequirement{{ContractID: core.MustCanonicalID("sales_core.order"), CompatibleMajor: 4}}},
			{Alias: core.LocalName("master"), TargetDatabase: core.MustCanonicalID("master_data"), RequiredContracts: []database.ContractRequirement{{ContractID: core.MustCanonicalID("master_data.customer"), CompatibleMajor: 2}}},
		}},
	}
	dbs := []dbrelease.Definition{
		{Database: core.MustCanonicalID("sales_core"), Version: "4.1.0", DataReadiness: dbrelease.DataReady, Provides: []dbrelease.ProvidedContract{
			{ID: core.MustCanonicalID("sales_core.order_legacy"), Version: database.ContractVersion{Major: 1}},
			{ID: core.MustCanonicalID("sales_core.order"), Version: database.ContractVersion{Major: 4, Minor: 1}},
		}},
		{Database: core.MustCanonicalID("sales_core"), Version: "4.2.0", DataReadiness: dbrelease.DataReady, Provides: []dbrelease.ProvidedContract{
			{ID: core.MustCanonicalID("sales_core.order"), Version: database.ContractVersion{Major: 4, Minor: 2}},
		}},
	}
	def := Definition{ID: core.MustCanonicalID("order_rollout"), Version: "2026.8.27", Steps: []Step{
		{ID: core.LocalName("sales_expand"), Kind: StepDatabaseRelease, Phase: PhaseExpand, Database: core.MustCanonicalID("sales_core"), Version: "4.1.0"},
		{ID: core.LocalName("master_ready"), Kind: StepDatabaseGate, Phase: PhaseTransition, Database: core.MustCanonicalID("master_data"), Contract: core.MustCanonicalID("master_data.customer"), CompatibleMajor: 2},
		{ID: core.LocalName("order_app"), Kind: StepApplicationRelease, Phase: PhaseRollout, Application: core.MustCanonicalID("order_entry"), Version: "3.8.0"},
		{ID: core.LocalName("retire_order_37"), Kind: StepApplicationRetire, Phase: PhaseTransition, Application: core.MustCanonicalID("order_entry"), Version: "3.7.0"},
		{ID: core.LocalName("sales_contract"), Kind: StepDatabaseRelease, Phase: PhaseContract, Database: core.MustCanonicalID("sales_core"), Version: "4.2.0"},
	}, After: []Dependency{
		{Step: core.LocalName("order_app"), After: core.LocalName("sales_expand")},
		{Step: core.LocalName("retire_order_37"), After: core.LocalName("order_app")},
	}}
	state := deployment.Snapshot{Environment: core.MustCanonicalID("production"), ActiveApplications: []deployment.ApplicationReleaseRef{{Application: core.MustCanonicalID("order_entry"), Version: "3.7.0"}}, Databases: []dbrelease.StateSnapshot{
		{Database: core.MustCanonicalID("sales_core"), Source: dbrelease.StateFromRelease, Release: "4.0.0", DataReadiness: dbrelease.DataReady, Provides: []dbrelease.ProvidedContract{{ID: core.MustCanonicalID("sales_core.order_legacy"), Version: database.ContractVersion{Major: 1}}}},
		{Database: core.MustCanonicalID("master_data"), Source: dbrelease.StateFromObserved, DataReadiness: dbrelease.DataReady, Provides: []dbrelease.ProvidedContract{{ID: core.MustCanonicalID("master_data.customer"), Version: database.ContractVersion{Major: 2, Minor: 4}}}},
	}}

	got := CompileWithDeployment(def, apps, dbs, state)
	if !got.Executable {
		t.Fatalf("expected executable plan, findings=%v", got.Findings)
	}
	foundGate := false
	for _, e := range got.Edges {
		if e.From == core.LocalName("retire_order_37") && e.To == core.LocalName("sales_contract") && e.Relation == "retires-incompatible-consumer" {
			foundGate = true
		}
	}
	if !foundGate {
		t.Fatalf("expected retirement gate edge, got %+v", got.Edges)
	}
	if len(got.ConsumerGates) != 2 {
		t.Fatalf("expected gates for two DB release steps, got %+v", got.ConsumerGates)
	}
	for _, g := range got.ConsumerGates {
		if !g.Resolved {
			t.Fatalf("expected resolved gate for %s: %+v", g.Step, g)
		}
	}
}

func TestCompileWithDeploymentBlocksWithdrawalWithoutRetirement(t *testing.T) {
	app := apprelease.Definition{Application: core.MustCanonicalID("a"), Version: "1.0.0", Databases: []application.DatabaseRequirement{{Alias: core.LocalName("db"), TargetDatabase: core.MustCanonicalID("d"), RequiredContracts: []database.ContractRequirement{{ContractID: core.MustCanonicalID("d.legacy"), CompatibleMajor: 1}}}}}
	db := dbrelease.Definition{Database: core.MustCanonicalID("d"), Version: "2.0.0", DataReadiness: dbrelease.DataReady}
	def := Definition{ID: core.MustCanonicalID("s"), Version: "1.0.0", Steps: []Step{{ID: core.LocalName("contract"), Kind: StepDatabaseRelease, Phase: PhaseContract, Database: core.MustCanonicalID("d"), Version: "2.0.0"}}}
	state := deployment.Snapshot{Environment: core.MustCanonicalID("prod"), ActiveApplications: []deployment.ApplicationReleaseRef{{Application: core.MustCanonicalID("a"), Version: "1.0.0"}}, Databases: []dbrelease.StateSnapshot{{Database: core.MustCanonicalID("d"), Source: dbrelease.StateFromRelease, Release: "1.0.0", DataReadiness: dbrelease.DataReady, Provides: []dbrelease.ProvidedContract{{ID: core.MustCanonicalID("d.legacy"), Version: database.ContractVersion{Major: 1}}}}}}
	got := CompileWithDeployment(def, []apprelease.Definition{app}, []dbrelease.Definition{db}, state)
	if got.Executable {
		t.Fatalf("expected withdrawal to be blocked: %+v", got)
	}
}

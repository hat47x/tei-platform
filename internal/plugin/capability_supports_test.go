package plugin_test

import (
	"testing"

	"tei/internal/database/migration"
	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/plugin"
)

// The Supports* family answers whether a declared capability names a given
// action, observation, evidence kind, database provider, migration tool or
// editor feature. internal/validation branches on these answers to decide
// whether a binding is satisfied, so a wrong answer silently accepts or
// rejects a workspace. None of the six was named by a test.
//
// They all share one shape — membership in a slice — so the failure they are
// prone to is the same: an empty declaration answering true, or a near-match
// answering true because the comparison is not exact.
func TestCapabilitySupportsAnswersMembershipExactly(t *testing.T) {
	capability := plugin.Capability{
		Actions:           []deployment.ActionKind{deployment.ActionDatabaseRelease},
		Observations:      []deployment.ObservationKind{deployment.ObservationDatabaseState},
		Evidence:          []deployment.EvidenceKind{deployment.EvidenceRolloutHealth},
		DatabaseProviders: []environment.DatabaseProvider{environment.ProviderPostgres},
		MigrationTools:    []migration.Tool{migration.ToolAtlas},
		Features:          []environment.Feature{environment.FeatureSQLAnalysis},
	}

	t.Run("declared values are supported", func(t *testing.T) {
		if !capability.SupportsAction(deployment.ActionDatabaseRelease) {
			t.Error("declared action")
		}
		if !capability.SupportsObservation(deployment.ObservationDatabaseState) {
			t.Error("declared observation")
		}
		if !capability.SupportsEvidence(deployment.EvidenceRolloutHealth) {
			t.Error("declared evidence")
		}
		if !capability.SupportsDatabaseProvider(environment.ProviderPostgres) {
			t.Error("declared database provider")
		}
		if !capability.SupportsMigrationTool(migration.ToolAtlas) {
			t.Error("declared migration tool")
		}
		if !capability.SupportsFeature(environment.FeatureSQLAnalysis) {
			t.Error("declared feature")
		}
	})

	t.Run("undeclared values are not supported", func(t *testing.T) {
		if capability.SupportsAction(deployment.ActionTrafficShift) {
			t.Error("undeclared action")
		}
		if capability.SupportsObservation(deployment.ObservationApplicationRuntime) {
			t.Error("undeclared observation")
		}
		// The only other evidence constant, so this checks the comparison is
		// not answering true for any evidence at all.
		if capability.SupportsEvidence(deployment.EvidenceKind("other")) {
			t.Error("undeclared evidence")
		}
		if capability.SupportsDatabaseProvider(environment.ProviderMySQL) {
			t.Error("undeclared database provider")
		}
		if capability.SupportsMigrationTool(migration.ToolFlyway) {
			t.Error("undeclared migration tool")
		}
		if capability.SupportsFeature(environment.FeatureDatabaseWorkbench) {
			t.Error("undeclared feature")
		}
	})

	t.Run("an empty declaration supports nothing", func(t *testing.T) {
		var empty plugin.Capability
		if empty.SupportsAction(deployment.ActionDatabaseRelease) ||
			empty.SupportsObservation(deployment.ObservationDatabaseState) ||
			empty.SupportsEvidence(deployment.EvidenceRolloutHealth) ||
			empty.SupportsDatabaseProvider(environment.ProviderPostgres) ||
			empty.SupportsMigrationTool(migration.ToolAtlas) ||
			empty.SupportsFeature(environment.FeatureSQLAnalysis) {
			t.Error("an empty capability must support nothing")
		}
	})

	t.Run("a near-match is not a match", func(t *testing.T) {
		// Exact comparison, not a prefix or case-insensitive one. A value that
		// differs only in spelling must not be reported as supported.
		near := plugin.Capability{
			Actions:           []deployment.ActionKind{deployment.ActionKind("database-release-2")},
			Observations:      []deployment.ObservationKind{deployment.ObservationKind("Database-State")},
			DatabaseProviders: []environment.DatabaseProvider{environment.DatabaseProvider("postgresql")},
			Features:          []environment.Feature{environment.Feature("editor.sql_analysi")},
		}
		if near.SupportsAction(deployment.ActionDatabaseRelease) {
			t.Error("a suffix-extended action matched")
		}
		if near.SupportsObservation(deployment.ObservationDatabaseState) {
			t.Error("a differently-cased observation matched")
		}
		if near.SupportsDatabaseProvider(environment.ProviderPostgres) {
			t.Error("an extended provider name matched")
		}
		if near.SupportsFeature(environment.FeatureSQLAnalysis) {
			t.Error("a truncated feature name matched")
		}
	})
}

// The observation predicate is the one internal/plugin/runtime uses to gate a
// builtin observation request, so an empty observation list must refuse rather
// than fall through to an allow.
func TestAnEmptyObservationListRefusesEveryObservation(t *testing.T) {
	capability := plugin.Capability{Kind: plugin.CapabilityObservationProvider}
	for _, kind := range []deployment.ObservationKind{
		deployment.ObservationApplicationRuntime,
		deployment.ObservationDatabaseState,
	} {
		if capability.SupportsObservation(kind) {
			t.Fatalf("an observation provider that declares none must refuse %q", kind)
		}
	}

	// A kind outside the vocabulary is not supported either, so a caller cannot
	// reach a match by widening the request.
	capability.Observations = []deployment.ObservationKind{deployment.ObservationDatabaseState}
	if capability.SupportsObservation(deployment.ObservationKind("unknown")) {
		t.Fatal("an unknown observation kind must not match a declared one")
	}
}

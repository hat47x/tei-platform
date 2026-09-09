package migration

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

func Plan(id string, database core.CanonicalID, actualFingerprint string, diff schemadiff.Diff, provider environment.DatabaseProvider) (Definition, error) {
	return planWithRenderer(id, database, actualFingerprint, diff, provider, func() ([]string, error) {
		return Render(provider, diff)
	})
}

// PlanPostgresWithObservation is the PostgreSQL planning entry point for a
// diff that may contain destructive table removal. The provider observation is
// required to prove partition readiness before DROP TABLE reaches SQL
// rendering. Non-destructive callers may continue to use Plan.
func PlanPostgresWithObservation(id string, database core.CanonicalID, actualFingerprint string, diff schemadiff.Diff, observation postgres.SchemaObservation) (Definition, error) {
	return planWithRenderer(id, database, actualFingerprint, diff, environment.ProviderPostgres, func() ([]string, error) {
		return postgres.RenderMigrationSQLWithObservation(diff, observation)
	})
}

func planWithRenderer(id string, database core.CanonicalID, actualFingerprint string, diff schemadiff.Diff, provider environment.DatabaseProvider, render func() ([]string, error)) (Definition, error) {
	if strings.TrimSpace(id) == "" {
		return Definition{}, fmt.Errorf("migration id is required")
	}
	if diff.Database != string(database) {
		return Definition{}, fmt.Errorf("migration database %s differs from diff database %s", database, diff.Database)
	}
	m := Definition{ID: id, Database: database, BasedOnActualFingerprint: actualFingerprint, Recovery: RecoveryForwardFix}
	for i := range diff.Changes {
		c := diff.Changes[i]
		m.Steps = append(m.Steps, Step{Kind: StepSchema, Change: &c})
	}
	stmts, err := render()
	if err != nil {
		return Definition{}, err
	}
	m.Realizations = []Realization{{Provider: provider, Statements: stmts}}
	m.Steps = append(m.Steps, Step{Kind: StepVerify, Description: "re-introspect database and compare with desired physical schema"})
	return m, nil
}

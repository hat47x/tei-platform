package migration

import (
	"context"
	"fmt"
	"slices"

	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

// PostgresObservationFunc obtains a fresh PostgreSQL schema observation at the
// apply boundary. Callers bind the concrete Database instance and catalog
// access outside the migration model.
type PostgresObservationFunc func(context.Context) (postgres.SchemaObservation, error)

// PostgresStatementExecutor executes one already-planned PostgreSQL statement
// sequence. ApplyPostgresWithFreshObservation calls it at most once.
type PostgresStatementExecutor func(context.Context, []string) error

// ApplyPostgresWithFreshObservation revalidates PostgreSQL destructive
// readiness immediately before executing a migration Definition. The fresh
// observation is authoritative for the apply precondition; planning-time
// partition absence is never reused as proof that the relation is still safe to
// drop.
//
// The function reconstructs the schema diff from the Definition, renders it
// again through PostgreSQL's observation-aware renderer, and requires the fresh
// rendering to match the planned realization exactly. A stale or incomplete
// observation, a newly attached partition, or a changed realization therefore
// stops before the executor is called.
func ApplyPostgresWithFreshObservation(ctx context.Context, definition Definition, observe PostgresObservationFunc, execute PostgresStatementExecutor) error {
	if observe == nil {
		return fmt.Errorf("postgres migration apply observation is nil")
	}
	if execute == nil {
		return fmt.Errorf("postgres migration apply executor is nil")
	}

	planned, err := postgresRealization(definition)
	if err != nil {
		return err
	}
	diff, err := schemaDiffFromDefinition(definition)
	if err != nil {
		return err
	}

	observation, err := observe(ctx)
	if err != nil {
		return fmt.Errorf("postgres migration apply observation: %w", err)
	}
	freshStatements, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	if err != nil {
		return fmt.Errorf("postgres migration apply precondition: %w", err)
	}
	if !slices.Equal(freshStatements, planned.Statements) {
		return fmt.Errorf("postgres migration apply precondition: planned statements differ from fresh observation-aware rendering")
	}

	statements := append([]string(nil), planned.Statements...)
	if err := execute(ctx, statements); err != nil {
		return fmt.Errorf("postgres migration execution: %w", err)
	}
	return nil
}

func postgresRealization(definition Definition) (Realization, error) {
	if len(definition.Realizations) != 1 {
		return Realization{}, fmt.Errorf("postgres migration apply requires exactly one realization, got %d", len(definition.Realizations))
	}
	planned := definition.Realizations[0]
	if planned.Provider != environment.ProviderPostgres {
		return Realization{}, fmt.Errorf("postgres migration apply requires postgres realization, got %q", planned.Provider)
	}
	return planned, nil
}

func schemaDiffFromDefinition(definition Definition) (schemadiff.Diff, error) {
	diff := schemadiff.Diff{Database: string(definition.Database)}
	for i, step := range definition.Steps {
		if step.Kind != StepSchema {
			continue
		}
		if step.Change == nil {
			return schemadiff.Diff{}, fmt.Errorf("postgres migration apply schema step %d has no change", i+1)
		}
		diff.Changes = append(diff.Changes, *step.Change)
	}
	return diff, nil
}

package migration_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database/migration"
	"tei/internal/database/physical"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

func TestApplyPostgresWithFreshObservationBlocksNewPartitionBeforeExecution(t *testing.T) {
	definition := plannedOrdinaryPostgresDrop(t)
	executeCalls := 0

	err := migration.ApplyPostgresWithFreshObservation(
		context.Background(),
		definition,
		func(context.Context) (postgres.SchemaObservation, error) {
			return postgres.SchemaObservation{
				Snapshot:                     physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core")},
				PartitionObservationComplete: true,
				Partitions: []postgres.PartitionRelation{{
					Child:  postgres.RelationIdentity{Schema: "public", Name: "audit_log_2026"},
					Parent: postgres.RelationIdentity{Schema: "public", Name: "audit_log"},
				}},
			}, nil
		},
		func(context.Context, []string) error {
			executeCalls++
			return nil
		},
	)
	if err == nil {
		t.Fatal("expected fresh partition observation to block apply")
	}
	var blocked *postgres.DestructiveReadinessError
	if !errors.As(err, &blocked) {
		t.Fatalf("err type=%T err=%v", err, err)
	}
	if executeCalls != 0 {
		t.Fatalf("executor calls=%d", executeCalls)
	}
}

func TestApplyPostgresWithFreshObservationBlocksObservationFailure(t *testing.T) {
	definition := plannedOrdinaryPostgresDrop(t)
	executeCalls := 0

	err := migration.ApplyPostgresWithFreshObservation(
		context.Background(),
		definition,
		func(context.Context) (postgres.SchemaObservation, error) {
			return postgres.SchemaObservation{}, errors.New("catalog unavailable")
		},
		func(context.Context, []string) error {
			executeCalls++
			return nil
		},
	)
	if err == nil || !strings.Contains(err.Error(), "apply observation") {
		t.Fatalf("err=%v", err)
	}
	if executeCalls != 0 {
		t.Fatalf("executor calls=%d", executeCalls)
	}
}

func TestApplyPostgresWithFreshObservationBlocksIncompleteObservation(t *testing.T) {
	definition := plannedOrdinaryPostgresDrop(t)
	executeCalls := 0

	err := migration.ApplyPostgresWithFreshObservation(
		context.Background(),
		definition,
		func(context.Context) (postgres.SchemaObservation, error) {
			return postgres.SchemaObservation{
				Snapshot: physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core")},
			}, nil
		},
		func(context.Context, []string) error {
			executeCalls++
			return nil
		},
	)
	if err == nil {
		t.Fatal("expected incomplete partition observation to block apply")
	}
	var blocked *postgres.DestructiveReadinessError
	if !errors.As(err, &blocked) {
		t.Fatalf("err type=%T err=%v", err, err)
	}
	if executeCalls != 0 {
		t.Fatalf("executor calls=%d", executeCalls)
	}
}

func TestApplyPostgresWithFreshObservationExecutesOrdinaryDropOnce(t *testing.T) {
	definition := plannedOrdinaryPostgresDrop(t)
	executeCalls := 0
	var executed []string

	err := migration.ApplyPostgresWithFreshObservation(
		context.Background(),
		definition,
		freshCompletePostgresObservation,
		func(_ context.Context, statements []string) error {
			executeCalls++
			executed = append([]string(nil), statements...)
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if executeCalls != 1 {
		t.Fatalf("executor calls=%d", executeCalls)
	}
	if len(executed) != 1 || executed[0] != `DROP TABLE "public"."audit_log";` {
		t.Fatalf("executed=%#v", executed)
	}
	if strings.Contains(strings.ToUpper(executed[0]), "CASCADE") {
		t.Fatalf("statement must not contain CASCADE: %s", executed[0])
	}
}

func TestApplyPostgresWithFreshObservationDoesNotRetryExecutionErrorWithCascade(t *testing.T) {
	definition := plannedOrdinaryPostgresDrop(t)
	executeCalls := 0
	var executed []string

	err := migration.ApplyPostgresWithFreshObservation(
		context.Background(),
		definition,
		freshCompletePostgresObservation,
		func(_ context.Context, statements []string) error {
			executeCalls++
			executed = append([]string(nil), statements...)
			return errors.New("dependent objects still exist")
		},
	)
	if err == nil || !strings.Contains(err.Error(), "postgres migration execution") || !strings.Contains(err.Error(), "dependent objects still exist") {
		t.Fatalf("err=%v", err)
	}
	if executeCalls != 1 {
		t.Fatalf("executor calls=%d", executeCalls)
	}
	if len(executed) != 1 {
		t.Fatalf("executed=%#v", executed)
	}
	for _, statement := range executed {
		if strings.Contains(strings.ToUpper(statement), "CASCADE") {
			t.Fatalf("execution error must not trigger CASCADE retry: %s", statement)
		}
	}
}

func TestApplyPostgresWithFreshObservationRejectsChangedPlannedStatements(t *testing.T) {
	definition := plannedOrdinaryPostgresDrop(t)
	definition.Realizations[0].Statements[0] = `DROP TABLE "public"."other_table";`
	executeCalls := 0

	err := migration.ApplyPostgresWithFreshObservation(
		context.Background(),
		definition,
		freshCompletePostgresObservation,
		func(context.Context, []string) error {
			executeCalls++
			return nil
		},
	)
	if err == nil || !strings.Contains(err.Error(), "planned statements differ") {
		t.Fatalf("err=%v", err)
	}
	if executeCalls != 0 {
		t.Fatalf("executor calls=%d", executeCalls)
	}
}

func freshCompletePostgresObservation(context.Context) (postgres.SchemaObservation, error) {
	return postgres.SchemaObservation{
		Snapshot:                     physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core")},
		PartitionObservationComplete: true,
	}, nil
}

func plannedOrdinaryPostgresDrop(t *testing.T) migration.Definition {
	t.Helper()
	definition, err := migration.PlanPostgresWithObservation(
		"20260910_001",
		core.MustCanonicalID("sales_core"),
		"fp-planning",
		postgresDropDiff("public.audit_log"),
		postgres.SchemaObservation{
			Snapshot:                     physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core")},
			PartitionObservationComplete: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return definition
}

// TestMA01MA02NonDestructiveApplyCharacterization exposes the narrow boundary
// of ApplyPostgresWithFreshObservation. It does NOT establish safe admission.
// Independent expected at the actual Apply Caller: refuse both inputs before
// executing statements unless base fingerprint and concrete target are proven.
func TestMA01MA02NonDestructiveApplyCharacterization(t *testing.T) {
	db := core.MustCanonicalID("sales_core")
	column := physical.Column{Name: "audited", PhysicalType: "pg_catalog.bool", Nullable: true}
	diff := schemadiff.Diff{
		Database: string(db),
		Changes: []schemadiff.Change{{
			Kind: schemadiff.AddColumn, Table: "public.orders", Column: &column,
		}},
	}
	definition, err := migration.Plan(
		"ma01_02", db, "base-schema-A", diff, environment.ProviderPostgres,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(definition.Realizations) != 1 || len(definition.Realizations[0].Statements) != 1 {
		t.Fatalf("invalid fixture: %#v", definition.Realizations)
	}

	tests := []struct {
		name     string
		snapshot physical.SchemaSnapshot
	}{
		{
			name: "MA-01 actual fingerprint drift but SQL unchanged",
			snapshot: physical.SchemaSnapshot{
				Database: db, Fingerprint: "actual-schema-B",
			},
		},
		{
			name: "MA-02 observer database differs from plan",
			snapshot: physical.SchemaSnapshot{
				Database:    core.MustCanonicalID("other_core"),
				Fingerprint: "base-schema-A",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executeCalls := 0
			err := migration.ApplyPostgresWithFreshObservation(
				context.Background(),
				definition,
				func(context.Context) (postgres.SchemaObservation, error) {
					return postgres.SchemaObservation{Snapshot: tt.snapshot}, nil
				},
				func(_ context.Context, statements []string) error {
					executeCalls++
					if len(statements) != 1 || statements[0] != definition.Realizations[0].Statements[0] {
						t.Fatalf("unexpected executed statements: %#v", statements)
					}
					return nil
				},
			)
			if err != nil {
				t.Fatalf("characterization expected present callback behavior, got: %v", err)
			}
			if executeCalls != 1 {
				t.Fatalf("expected current unqualified callback, got calls=%d", executeCalls)
			}
			// Required future Apply Admission: executeCalls=0.
			// This passing test is evidence of a gap, not a migration safety PASS.
		})
	}
}

// TestMA11ObservationCallbackDoesNotFenceExecution is a characterization of
// the callback seam, not evidence of a real PostgreSQL race or safe admission.
// The independent expected for a production Apply boundary is a verified
// target-bound lock/fence (or an abort), not merely a preceding observation.
func TestMA11ObservationCallbackDoesNotFenceExecution(t *testing.T) {
	db := core.MustCanonicalID("sales_core")
	column := physical.Column{Name: "audited", PhysicalType: "pg_catalog.bool", Nullable: true}
	diff := schemadiff.Diff{
		Database: string(db),
		Changes: []schemadiff.Change{{
			Kind: schemadiff.AddColumn, Table: "public.orders", Column: &column,
		}},
	}
	definition, err := migration.Plan("ma11", db, "fp-before", diff, environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	currentFingerprint := "fp-before"
	observationCalls, executeCalls := 0, 0
	err = migration.ApplyPostgresWithFreshObservation(
		context.Background(),
		definition,
		func(context.Context) (postgres.SchemaObservation, error) {
			observationCalls++
			return postgres.SchemaObservation{Snapshot: physical.SchemaSnapshot{
				Database: db, Fingerprint: currentFingerprint,
			}}, nil
		},
		func(context.Context, []string) error {
			// Simulate drift at the executor callback after the observation
			// but before any real SQL effect would be committed.
			currentFingerprint = "fp-after"
			executeCalls++
			return nil
		},
	)
	if err != nil {
		t.Fatalf("current callback contract unexpectedly rejected simulated drift: %v", err)
	}
	if observationCalls != 1 || executeCalls != 1 || currentFingerprint != "fp-after" {
		t.Fatalf("expected one observe, one execute, and drift without fencing: observed=%d executed=%d current=%q", observationCalls, executeCalls, currentFingerprint)
	}
	// Required production behavior: executeCalls=0 unless the same target's
	// preconditions are still protected at the actual execution boundary.
}

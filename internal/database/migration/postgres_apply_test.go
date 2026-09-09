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

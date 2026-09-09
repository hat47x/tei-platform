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

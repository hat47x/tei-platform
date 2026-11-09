package command

import (
	"context"
	"errors"
	"testing"

	"strings"
	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// readOnlyDatasourceWorld declares the database that owns the data as
// read-only for the application, so a writing command cannot be run against it.
func readOnlyDatasourceWorld(access database.Access) ir.World {
	world := commandWorld()
	world.Databases = []database.Database{{
		ID:           "sales_core",
		Kind:         "relational",
		Ownership:    database.OwnershipManagedHere,
		SemanticUses: []core.CanonicalID{"sales"},
	}}
	world.Applications[0].Databases = []application.DatabaseRequirement{{
		Alias:          "sales",
		TargetDatabase: "sales_core",
		Access:         access,
	}}
	return world
}

// TestInvokerRefusesWriteThroughReadOnlyDatasource checks that a declared
// read-only datasource refuses a writing command before the executor runs. The
// declaration is the authority here; the database is not contacted.
func TestInvokerRefusesWriteThroughReadOnlyDatasource(t *testing.T) {
	executor := &datasourceRecordingExecutor{}
	invoker := Invoker{
		World:      readOnlyDatasourceWorld(database.AccessRead),
		Executor:   executor,
		UnitOfWork: &MemoryUnitOfWork{},
		Authorizer: ReferenceAuthorizer{},
	}
	_, err := invoker.Invoke(context.Background(), request())
	var executionError *ExecutionError
	if !errors.As(err, &executionError) || executionError.Code != "COMMAND_DATASOURCE_READ_ONLY" {
		t.Fatalf("err=%v", err)
	}
	if executor.called {
		t.Fatal("read-only datasource reached the executor")
	}
}

// TestInvokerAllowsWriteThroughReadWriteDatasource checks that the check is
// about the declaration, not about the command: the same writing command runs
// when the datasource is declared read-write.
func TestInvokerAllowsWriteThroughReadWriteDatasource(t *testing.T) {
	executor := &datasourceRecordingExecutor{}
	invoker := Invoker{
		World:      readOnlyDatasourceWorld(database.AccessReadWrite),
		Executor:   executor,
		UnitOfWork: &MemoryUnitOfWork{},
		Authorizer: ReferenceAuthorizer{},
	}
	if _, err := invoker.Invoke(context.Background(), request()); err != nil {
		t.Fatalf("err=%v", err)
	}
	if !executor.called {
		t.Fatal("read-write datasource did not reach the executor")
	}
}

// TestInvokerDoesNotGuessADatasource checks that a data set no datasource covers
// is not refused by invention. The model has no binding between the data and a
// database here, so the check stays silent rather than guessing one.
func TestInvokerDoesNotGuessADatasource(t *testing.T) {
	world := commandWorld()
	world.Databases = []database.Database{{
		ID:           "sales_core",
		Kind:         "relational",
		Ownership:    database.OwnershipManagedHere,
		SemanticUses: []core.CanonicalID{"customer"},
	}}
	world.Applications[0].Databases = []application.DatabaseRequirement{{
		Alias:          "customer",
		TargetDatabase: "sales_core",
		Access:         database.AccessRead,
	}}
	invoker := Invoker{
		World:      world,
		Executor:   &datasourceRecordingExecutor{},
		UnitOfWork: &MemoryUnitOfWork{},
		Authorizer: ReferenceAuthorizer{},
	}
	if _, err := invoker.Invoke(context.Background(), request()); err != nil {
		t.Fatalf("err=%v", err)
	}
}

// TestInvokerReportsReadOnlyDatasourceWithTheDeclaration checks that the
// refusal names the datasource alias the Application declared, so a reader can
// find the declaration that refused the write.
func TestInvokerReportsReadOnlyDatasourceWithTheDeclaration(t *testing.T) {
	invoker := Invoker{
		World:      readOnlyDatasourceWorld(database.AccessRead),
		Executor:   &datasourceRecordingExecutor{},
		UnitOfWork: &MemoryUnitOfWork{},
		Authorizer: ReferenceAuthorizer{},
	}
	_, err := invoker.Invoke(context.Background(), request())
	if err == nil {
		t.Fatal("expected a refusal")
	}
	for _, want := range []string{"sales", "sales_core", "sales.order"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal %q does not name %q", err.Error(), want)
		}
	}
}

// TestInvokerDatasourceRefusalDoesNotDependOnDeclarationOrder checks that two
// datasources covering the same namespace give the same result in either
// declaration order. The request names no datasource, so a writable declaration
// must not hide a read-only one that covers the same data (#2611).
func TestInvokerDatasourceRefusalDoesNotDependOnDeclarationOrder(t *testing.T) {
	archive := application.DatabaseRequirement{Alias: "archive", TargetDatabase: "archive_db", Access: database.AccessRead}
	primary := application.DatabaseRequirement{Alias: "primary", TargetDatabase: "primary_db", Access: database.AccessReadWrite}
	orders := map[string][]application.DatabaseRequirement{
		"archive-first": {archive, primary},
		"primary-first": {primary, archive},
	}
	for name, requirements := range orders {
		t.Run(name, func(t *testing.T) {
			world := commandWorld()
			world.Databases = []database.Database{
				{ID: "archive_db", Kind: "relational", Ownership: database.OwnershipManagedHere, SemanticUses: []core.CanonicalID{"sales"}},
				{ID: "primary_db", Kind: "relational", Ownership: database.OwnershipManagedHere, SemanticUses: []core.CanonicalID{"sales"}},
			}
			world.Applications[0].Databases = requirements
			executor := &datasourceRecordingExecutor{}
			invoker := Invoker{
				World:      world,
				Executor:   executor,
				UnitOfWork: &MemoryUnitOfWork{},
				Authorizer: ReferenceAuthorizer{},
			}
			_, err := invoker.Invoke(context.Background(), request())
			var executionError *ExecutionError
			if !errors.As(err, &executionError) || executionError.Code != "COMMAND_DATASOURCE_READ_ONLY" {
				t.Fatalf("err=%v", err)
			}
			if executor.called {
				t.Fatal("read-only datasource reached the executor")
			}
			if !strings.Contains(err.Error(), "archive_db") {
				t.Fatalf("refusal %q does not name the read-only datasource", err.Error())
			}
		})
	}
}

type datasourceRecordingExecutor struct {
	called bool
}

func (e *datasourceRecordingExecutor) Execute(context.Context, Request) (Result, error) {
	e.called = true
	return Result{Effect: &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"}}, nil
}

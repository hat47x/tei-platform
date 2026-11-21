//go:build !tei_order_only && !tei_repair_only

package main

import (
	"database/sql"
	"strings"
	"testing"

	"tei/internal/driver"
	"tei/internal/realization"
	"tei/internal/workspace"
)

// A host that asks for a database-backed read must not get a page served from
// the fixture instead. Construction is deliberately lazy (it opens no
// connection at startup), so the refusal appears at read time and must be an
// error rather than a silent fixture fallback.
func TestPostgresReadRunnerFailsClosedWhenTheDatabaseIsUnreachable(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("pgx", "host=127.0.0.1 port=1 user=none dbname=none sslmode=disable")
	if err != nil {
		t.Fatalf("opening a broken connection should be lazy: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	runner, err := newPostgresReadRunner(loaded.World, db)
	if err != nil {
		// Refusing at construction is also fail-closed.
		return
	}
	if _, err := runner.Run(t.Context(), readScenarioInput{From: "2026-09-01T00:00:00Z", RowIndex: 0}); err == nil {
		t.Fatal("an unreachable database produced a page instead of an error")
	}
}

// No connection at all is refused outright: the fixture must be chosen
// explicitly, never by absence.
func TestPostgresReadRunnerRequiresAnExplicitConnection(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newPostgresReadRunner(loaded.World, nil); err == nil {
		t.Fatal("a nil connection was accepted; the read would fall back to the fixture")
	}
}

// The Workspace's own acceptance is what makes a database read possible. A
// Workspace whose binding names no PostgreSQL executor must be refused rather
// than served by guessing a table.
func TestPostgresReadRunnerRequiresADeclaredExecutorBinding(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	stripped := loaded.World
	stripped.ContractQueryExecutorBindings = nil
	db, err := sql.Open("pgx", "host=127.0.0.1 port=1 user=none dbname=none sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, buildErr := newPostgresReadRunner(stripped, db)
	if buildErr == nil {
		t.Fatal("a Workspace without a declared PostgreSQL binding was accepted")
	}
	if !strings.Contains(buildErr.Error(), "binding") {
		t.Fatalf("the refusal does not name the missing binding: %q", buildErr)
	}
}

// A binding for some PostgreSQL Query cannot authorize the selected
// order-line Query or its Database instance. The previous constructor
// accepted any nonempty PostgreSQL binding set, postponing the mismatch
// until after Browse had already started reading the database.
func TestPostgresReadRunnerRefusesUnrelatedExecutorBindingBeforeBrowse(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := driver.NewResolver(loaded.World).ResolveHeaderDetailContractQuery(
		"order_entry", "order_entry.order_lines",
	)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("pgx", "host=127.0.0.1 port=1 user=none dbname=none sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, tc := range []struct {
		name   string
		change func(*realization.ContractQueryExecutorBinding)
	}{
		{"other target", func(b *realization.ContractQueryExecutorBinding) { b.Target.Query = "unrelated_query" }},
		{"other Environment", func(b *realization.ContractQueryExecutorBinding) { b.Environment = "other_environment" }},
		{"other Database instance", func(b *realization.ContractQueryExecutorBinding) { b.Instance = "other_instance" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			world := loaded.World
			world.ContractQueryExecutorBindings = append(
				[]realization.ContractQueryExecutorBinding(nil),
				loaded.World.ContractQueryExecutorBindings...,
			)
			changed := false
			postgresStillDeclared := false
			for i := range world.ContractQueryExecutorBindings {
				binding := &world.ContractQueryExecutorBindings[i]
				if binding.Environment == "development" &&
					binding.Target == resolved.Binding.Target &&
					binding.Executor == "database.query.postgres" {
					tc.change(binding)
					changed = true
				}
				if binding.Executor == "database.query.postgres" {
					postgresStillDeclared = true
				}
			}
			if !changed || !postgresStillDeclared {
				t.Fatal("fixture did not preserve a declared PostgreSQL executor while breaking selected binding")
			}
			_, buildErr := newPostgresReadRunner(world, db)
			if buildErr == nil ||
				!strings.Contains(buildErr.Error(), "order_entry.order_lines") ||
				!strings.Contains(buildErr.Error(), "binding") {
				t.Fatalf("unrelated PostgreSQL binding qualified the selected read: %v", buildErr)
			}
		})
	}
}

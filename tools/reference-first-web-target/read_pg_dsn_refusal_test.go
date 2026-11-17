//go:build !tei_order_only && !tei_repair_only

package main

import (
	"strings"
	"testing"

	"tei/tools/internal/postgresdb"
)

// TestReadPGDSNFlagRefusesAHostSuppliedDatabase pins the refusal that keeps the
// flag from replacing a Host's database. A Host that injected a connection has
// already chosen what the page reads; a command-line flag must not silently
// point the page at a different database.
func TestReadPGDSNFlagRefusesAHostSuppliedDatabase(t *testing.T) {
	// A handle the driver never dialed is enough: the refusal happens before the
	// flag's own DSN is opened, so nothing external runs in this test.
	hostDB, err := postgresdb.OpenSQL("postgres://host@127.0.0.1:1/host?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = hostDB.Close() }()

	err = runCombinedWebTargetWithHost(t.Context(), nil, nil,
		webHostWiring{readDatabase: hostDB},
		[]string{
			"-workspace", referenceWorkspacePath(t), "-application", "order_entry",
			"-baseline-id", "B", "-baseline-revision", testBaselineRevision,
			"-target-revision", testTargetRevision, "-listen", "127.0.0.1:0",
			"-legacy-users", legacyFixturePath(t, "users.json"),
			"-legacy-authorization", legacyFixturePath(t, "authorization.json"),
			"-diagnostic-unaccepted-source",
			"-read-pg-dsn", "postgres://someone@127.0.0.1:1/none?sslmode=disable",
		})
	if err == nil {
		t.Fatal("a supplied Host database did not conflict with the flag")
	}
	if !strings.Contains(err.Error(), "WEB_READ_DATABASE_CONFLICT") {
		t.Fatalf("error=%q", err)
	}
	if strings.Contains(err.Error(), "127.0.0.1") || strings.Contains(err.Error(), "postgres://") {
		t.Fatalf("the conflict diagnostic leaked connection text: %q", err)
	}
}

// TestReadPGDSNUnusableDoesNotLeakTheDSN checks the open-failure path: a DSN the
// driver cannot parse must produce a diagnostic that carries no connection text,
// since a DSN may hold a password.
func TestReadPGDSNUnusableDoesNotLeakTheDSN(t *testing.T) {
	const secret = "postgres://user:must-not-appear@host.invalid:notaport/none"
	err := runCombinedWebTargetWithHost(t.Context(), nil, nil, webHostWiring{},
		[]string{
			"-workspace", referenceWorkspacePath(t), "-application", "order_entry",
			"-baseline-id", "B", "-baseline-revision", testBaselineRevision,
			"-target-revision", testTargetRevision, "-listen", "127.0.0.1:0",
			"-legacy-users", legacyFixturePath(t, "users.json"),
			"-legacy-authorization", legacyFixturePath(t, "authorization.json"),
			"-diagnostic-unaccepted-source",
			"-read-pg-dsn", secret,
		})
	if err == nil {
		t.Skip("the driver accepted this DSN; the leak check needs one it rejects")
	}
	if !strings.Contains(err.Error(), "WEB_READ_DATABASE_UNUSABLE") {
		t.Fatalf("error=%q", err)
	}
	for _, fragment := range []string{"must-not-appear", "host.invalid", "postgres://"} {
		if strings.Contains(err.Error(), fragment) {
			t.Fatalf("the open failure leaked %q: %v", fragment, err)
		}
	}
}

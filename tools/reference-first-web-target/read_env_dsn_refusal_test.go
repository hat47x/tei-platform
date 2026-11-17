//go:build !tei_order_only && !tei_repair_only

package main

import (
	"strings"
	"testing"

	"tei/tools/internal/postgresdb"
)

// TestServeReadFromDBRefusesAHostSuppliedDatabase pins the refusal that keeps
// -serve-read-from-db from replacing a Host's database. A Host that injected a
// connection has already chosen what the page reads; a command-line flag must
// not silently point the page at a different database.
func TestServeReadFromDBRefusesAHostSuppliedDatabase(t *testing.T) {
	// A handle the driver never dialed is enough: the refusal happens before the
	// environment DSN is opened, so nothing external runs in this test.
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
			"-serve-read-from-db",
		})
	if err == nil {
		t.Fatal("a supplied Host database did not conflict with -serve-read-from-db")
	}
	if !strings.Contains(err.Error(), "WEB_READ_DATABASE_CONFLICT") {
		t.Fatalf("error=%q", err)
	}
}

// TestServeReadFromDBRequiresTheEnvironmentDSN pins the unconfigured case: the
// flag selects the environment variable, so asking for it without one must
// refuse and name the variable rather than fall back to the fixture rows.
func TestServeReadFromDBRequiresTheEnvironmentDSN(t *testing.T) {
	t.Setenv(readDSNEnv, "")
	err := runCombinedWebTargetWithHost(t.Context(), nil, nil, webHostWiring{},
		[]string{
			"-workspace", referenceWorkspacePath(t), "-application", "order_entry",
			"-baseline-id", "B", "-baseline-revision", testBaselineRevision,
			"-target-revision", testTargetRevision, "-listen", "127.0.0.1:0",
			"-legacy-users", legacyFixturePath(t, "users.json"),
			"-legacy-authorization", legacyFixturePath(t, "authorization.json"),
			"-diagnostic-unaccepted-source",
			"-serve-read-from-db",
		})
	if err == nil {
		t.Fatal("the flag was accepted without the environment DSN")
	}
	if !strings.Contains(err.Error(), "WEB_READ_DATABASE_UNCONFIGURED") ||
		!strings.Contains(err.Error(), readDSNEnv) {
		t.Fatalf("error=%q", err)
	}
}

// TestServeReadFromDBUnusableDoesNotLeakTheDSN checks the open-failure path: a
// DSN the driver cannot parse must produce a diagnostic that carries no
// connection text, since the environment variable may hold a password.
func TestServeReadFromDBUnusableDoesNotLeakTheDSN(t *testing.T) {
	const secret = "postgres://user:must-not-appear@host.invalid:notaport/none"
	t.Setenv(readDSNEnv, secret)
	err := runCombinedWebTargetWithHost(t.Context(), nil, nil, webHostWiring{},
		[]string{
			"-workspace", referenceWorkspacePath(t), "-application", "order_entry",
			"-baseline-id", "B", "-baseline-revision", testBaselineRevision,
			"-target-revision", testTargetRevision, "-listen", "127.0.0.1:0",
			"-legacy-users", legacyFixturePath(t, "users.json"),
			"-legacy-authorization", legacyFixturePath(t, "authorization.json"),
			"-diagnostic-unaccepted-source",
			"-serve-read-from-db",
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

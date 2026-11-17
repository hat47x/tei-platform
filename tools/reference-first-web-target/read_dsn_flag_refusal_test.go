//go:build !tei_order_only && !tei_repair_only

package main

import (
	"strings"
	"testing"

	"tei/tools/internal/postgresdb"
)

// TestReadDSNFlagIsRefusedWhenTheHostAlreadySuppliedOne pins the refusal that
// keeps the flag from replacing a Host's database. A Host that injected a
// connection has already chosen what the page reads; a command-line flag must
// not silently point the page at a different database.
func TestReadDSNFlagIsRefusedWhenTheHostAlreadySuppliedOne(t *testing.T) {
	// The Host supplies a connection handle. Opening it is not needed: the
	// refusal must happen before the flag's DSN is parsed or dialed, so a
	// handle the driver never dialed still proves the conflict.
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
			"-diagnostic-unaccepted-source",
			"-read-dsn", "postgres://someone@127.0.0.1:1/none?sslmode=disable",
		})
	if err == nil {
		t.Fatal("a supplied Host database did not conflict with the flag")
	}
	if !strings.Contains(err.Error(), "READ_DATABASE_CONFLICT") {
		t.Fatalf("error=%q", err)
	}
	if strings.Contains(err.Error(), "127.0.0.1") || strings.Contains(err.Error(), "postgres://") {
		t.Fatalf("the conflict diagnostic leaked connection text: %q", err)
	}
}

// TestReadDSNOpenFailureDoesNotLeakTheDSN checks the open failure path: a DSN
// the driver cannot parse must produce a diagnostic that carries no connection
// text, since a DSN may hold a password.
func TestReadDSNOpenFailureDoesNotLeakTheDSN(t *testing.T) {
	const secret = "postgres://user:must-not-appear@host.invalid:notaport/none"
	err := runCombinedWebTargetWithHost(t.Context(), nil, nil, webHostWiring{},
		[]string{
			"-workspace", referenceWorkspacePath(t), "-application", "order_entry",
			"-baseline-id", "B", "-baseline-revision", testBaselineRevision,
			"-target-revision", testTargetRevision, "-listen", "127.0.0.1:0",
			"-diagnostic-unaccepted-source",
			"-legacy-users", legacyFixturePath(t, "users.json"),
			"-legacy-authorization", legacyFixturePath(t, "authorization.json"),
			"-read-dsn", secret,
		})
	if err == nil {
		t.Skip("the driver accepted this DSN; the leak check needs one it rejects")
	}
	if !strings.Contains(err.Error(), "READ_DATABASE_OPEN_FAILED") {
		t.Fatalf("error=%q", err)
	}
	for _, fragment := range []string{"must-not-appear", "host.invalid", "postgres://"} {
		if strings.Contains(err.Error(), fragment) {
			t.Fatalf("the open failure leaked %q: %v", fragment, err)
		}
	}
}

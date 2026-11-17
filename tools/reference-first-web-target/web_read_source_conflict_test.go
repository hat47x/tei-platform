//go:build !tei_repair_only

package main

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"
)

// A page selects one order from browse and then shows that order's detail rows.
// Serving the two halves from different sources would let a fixture order carry
// database detail rows, so the host must refuse the combination instead of
// letting whichever field was set last win.
//
// The connection is lazy and never used: the conflict is decided before any
// query runs, which is the point of checking it at wiring time.
func TestWebHostRefusesADatabaseReadWithASeparateBrowse(t *testing.T) {
	db, err := sql.Open("pgx", "host=127.0.0.1 port=1 user=none dbname=none sslmode=disable")
	if err != nil {
		t.Fatalf("opening a lazy connection should not fail: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	err = runCombinedWebTargetWithHost(t.Context(), nil, nil,
		webHostWiring{
			readDatabase: db,
			browse:       referenceBrowseExecutor{},
		},
		[]string{
			"-workspace", referenceWorkspacePath(t), "-application", "order_entry",
			"-baseline-id", "B", "-baseline-revision", testBaselineRevision,
			"-target-revision", testTargetRevision, "-listen", "127.0.0.1:0",
			"-legacy-users", legacyFixturePath(t, "users.json"),
			"-legacy-authorization", legacyFixturePath(t, "authorization.json"),
			"-diagnostic-unaccepted-source",
		})
	if err == nil {
		t.Fatal("a database read combined with a separate browse executor was accepted")
	}
	if !strings.Contains(err.Error(), "WEB_READ_BROWSE_SOURCE_CONFLICT") {
		t.Fatalf("the refusal does not name the conflict: %v", err)
	}
}

// A browse executor without a database read is still allowed: that is how a
// Host supplies a real browse while the detail read stays on the fixture, and
// the existing wiring tests rely on it. Publishing HTTP blocks, so the start is
// run under a cancellable context and only the wiring decision is judged.
func TestWebHostKeepsASeparateBrowseWithoutADatabaseRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runCombinedWebTargetWithHost(ctx, nil, nil,
			webHostWiring{browse: referenceBrowseExecutor{}},
			[]string{
				"-workspace", referenceWorkspacePath(t), "-application", "order_entry",
				"-baseline-id", "B", "-baseline-revision", testBaselineRevision,
				"-target-revision", testTargetRevision, "-listen", "127.0.0.1:0",
				"-legacy-users", legacyFixturePath(t, "users.json"),
				"-legacy-authorization", legacyFixturePath(t, "authorization.json"),
				"-diagnostic-unaccepted-source",
			})
	}()
	t.Cleanup(cancel)

	select {
	case err := <-done:
		if err != nil && strings.Contains(err.Error(), "WEB_READ_BROWSE_SOURCE_CONFLICT") {
			t.Fatalf("a browse executor alone was refused: %v", err)
		}
	case <-time.After(2 * time.Second):
		// Still serving, which means the wiring was accepted.
	}
}

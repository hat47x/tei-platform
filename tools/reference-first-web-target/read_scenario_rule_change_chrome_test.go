//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tei/tools/internal/postgresdb"
)

// TestFirstWebRuleChangeReachesActualChromeDOM is the last piece plan/NOW.md
// road item 2 lists as unimplemented: receive the first Web Target's
// PostgreSQL-backed HTTP response in a real Chrome DOM and compare the answers
// before and after a rule edit through that same path.
//
// Two halves already exist and are reused rather than duplicated:
//   - TestFirstWebAcceptsARuleChangeEndToEnd compares before/after answers in the
//     HTTP body;
//   - TestShippedWebPageShowsRealPostgresRowAnswers receives one revision in a
//     real Chrome DOM.
//
// What was missing is the combination: both revisions' DOM, compared against
// the same independently written expectations (expected-before.json and
// expected-after.json, keyed by order_line_id, never by display position).
func TestFirstWebRuleChangeReachesActualChromeDOM(t *testing.T) {
	if os.Getenv("TEI_VIEW_BROWSER_BINARY") == "" {
		t.Skip("TEI_VIEW_BROWSER_BINARY is unset; real Chrome DOM not run")
	}
	dsn := webReadPostgresDSN(t)
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := requireIsolatedWebReadDatabase(context.Background(), db, readPostgresDatabase); err != nil {
		t.Fatal(err)
	}
	if err := seedReadFixture(t, db); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec("drop table if exists public.order_lines") })
	t.Setenv(readDSNEnv, dsn)

	fixtures := destinationFixtureRoot(t)
	ruleDocument, err := os.ReadFile(filepath.Join(fixtures, "edit", "unit-price.tei.md"))
	if err != nil {
		t.Fatal(err)
	}

	for _, stage := range []struct {
		name     string
		expected string
		edited   bool
	}{
		{name: "before", expected: "expected-before.json"},
		{name: "after", expected: "expected-after.json", edited: true},
	} {
		t.Run(stage.name, func(t *testing.T) {
			scratch := t.TempDir()
			copyWorkspaceForRuleChange(t, referenceWorkspacePath(t), scratch)
			if stage.edited {
				applyUnitPriceRule(t, scratch, string(ruleDocument))
			}

			address := startFirstWebAddress(t, scratch)
			dom := actualWebReadChromeDOM(t, address)

			if strings.Contains(dom, dsn) {
				t.Fatal("the Chrome DOM carried the database connection string")
			}
			lines := sectionBetween(dom, `id="order-lines"`, `id="read-identity"`)
			if lines == "" {
				t.Fatal("the Chrome DOM did not contain the order-line read")
			}
			want := expectedAnswersForOrder(t, filepath.Join(fixtures, stage.expected), "1001")
			requirePageAnswersMatchExpectation(t, lines, want)

			// The edit is a rule, not a row: the same seeded values and the same
			// witnessed duplicate must survive both stages.
			for _, seeded := range []string{"2.000", "0.000", "5.000"} {
				if !strings.Contains(lines, seeded) {
					t.Fatalf("stage=%s: the Chrome DOM lost the seeded quantity %q", stage.name, seeded)
				}
			}
			if !strings.Contains(dom, "sales.order.lines.line_number_unique not-satisfied") {
				t.Fatalf("stage=%s: the Chrome DOM lost the witnessed duplicate's relation decision", stage.name)
			}
			t.Logf("actual Chrome DOM verified for order 1001 (stage %s)", stage.name)
		})
	}
}

// startFirstWebAddress starts the shipped entry point on one Workspace over the
// read database and returns the address it published on. It differs from
// runFirstWebAndReadOrder only in returning the address instead of the page,
// because the Chrome DOM has to be obtained by navigating there.
func startFirstWebAddress(t *testing.T, workspaceRoot string) string {
	t.Helper()
	address := freeLoopbackAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runCombinedWebTargetWithHost(ctx, nil, nil, webHostWiring{},
			[]string{
				"-workspace", workspaceRoot, "-application", "order_entry",
				"-baseline-id", "B", "-baseline-revision", testBaselineRevision,
				"-target-revision", testTargetRevision, "-listen", address,
				"-legacy-users", legacyFixturePath(t, "users.json"),
				"-legacy-authorization", legacyFixturePath(t, "authorization.json"),
				"-diagnostic-unaccepted-source",
				"-serve-read-from-db",
			})
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("the web target did not stop with its context")
		}
	})
	waitForHealth(t, address, done)
	return address
}

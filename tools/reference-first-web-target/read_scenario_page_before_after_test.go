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

// TestShippedWebPageShowsTheRuleEditThroughChrome is the browser receipt for
// plan/NOW.md road item 2: a real browser must receive the rule change through
// the first Web Target's HTTP route, before and after, and preserve the answers
// in its DOM.
//
// TestFirstWebAcceptsARuleChangeEndToEnd already runs both stages through that
// route and checks the HTTP body. It stops at the body because a browser receipt
// is a different qualification; this test supplies only that half and reuses the
// same Workspace copy, rule edit, expectations and page checks, so the two do not
// judge different things.
func TestShippedWebPageShowsTheRuleEditThroughChrome(t *testing.T) {
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
	t.Cleanup(func() {
		_, _ = db.Exec("drop table if exists public.order_lines")
		_, _ = db.Exec("drop table if exists public.orders")
	})
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

			address := startWebTargetForRuleChange(t, scratch)
			want := expectedAnswersForOrder(t, filepath.Join(fixtures, stage.expected), "1001")

			dom := actualWebReadChromeDOM(t, address)
			if !strings.Contains(dom, `id="read-result"`) || strings.Contains(dom, dsn) {
				t.Fatal("Chrome did not receive the PostgreSQL-backed read or exposed the DSN")
			}
			if !strings.Contains(dom, "sales.order.lines.line_number_unique not-satisfied") {
				t.Fatal("Chrome DOM lost the witnessed duplicate's relation decision")
			}
			// The change is a rule, not a row: the same three seeded quantities
			// must survive both stages, so a page that differs only by losing
			// rows cannot pass as the rule change.
			for _, seeded := range []string{"2.000", "0.000", "5.000"} {
				if !strings.Contains(dom, seeded) {
					t.Fatalf("Chrome DOM lost the seeded quantity %q", seeded)
				}
			}
			lines := sectionBetween(dom, `id="order-lines"`, `id="read-identity"`)
			if lines == "" {
				t.Fatal("Chrome DOM did not contain the order-line read")
			}
			requirePageAnswersMatchExpectation(t, lines, want)
			t.Logf("actual first Web HTTP Chrome DOM verified for the %s stage of order 1001", stage.name)
		})
	}
}

// startWebTargetForRuleChange runs the shipped entry point on one Workspace and
// returns its address. It differs from the body-only helper in returning the
// address, because a browser needs to open the form itself.
func startWebTargetForRuleChange(t *testing.T, workspaceRoot string) string {
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

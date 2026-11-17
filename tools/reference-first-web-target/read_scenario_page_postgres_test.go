//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"tei/tools/internal/postgresdb"
)

// TestShippedWebPageShowsRealPostgresRowAnswers is plan/NOW.md road item 2's
// remaining half: the page a person opens in a browser carries the per-row rule
// answers of a read of a real PostgreSQL database.
//
// It drives the shipped entry point (runCombinedWebTargetWithHost with the
// empty Host wiring, exactly as the binary does) and supplies the read database
// through the CLI flag `-serve-read-from-db` and the connection string in
// TEI_WEB_READ_PG_DSN (main). The expectations are written from the
// fixture's seed and the Application's declared rules, not copied from the
// response: order 1001 holds quantities 2, 0 and 5, and its lines carry
// line numbers 1, 2 and 2.
func TestShippedWebPageShowsRealPostgresRowAnswers(t *testing.T) {
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
	// The page must show the three seeded lines, not the two fixture rows, so
	// the table stays until the process has served its response.
	t.Cleanup(func() { _, _ = db.Exec("drop table if exists public.order_lines") })

	address := freeLoopbackAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runCombinedWebTargetWithHost(ctx, nil, nil, webHostWiring{},
			[]string{
				"-workspace", referenceWorkspacePath(t), "-application", "order_entry",
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

	page := postCustomSubmitPage(t, address, url.Values{
		"loginId":    {"legacy-order-editor"},
		"credential": {"editor-secret"},
		"from":       {"2026-09-01T00:00:00Z"},
		"row":        {"0"},
	})

	if strings.Contains(page, dsn) {
		t.Fatal("the page carried the database connection string")
	}

	// The read section exists and names the read; without it the answers below
	// would be unattributable to this database read.
	if !strings.Contains(page, `id="read-result"`) || !strings.Contains(page, `id="order-lines"`) {
		t.Fatalf("the page did not render the read section")
	}
	if !strings.Contains(page, "Selected order") || !strings.Contains(page, "Row count") {
		t.Fatalf("the page does not name the read its answers belong to")
	}
	if strings.Contains(page, "No order-line rows.") {
		t.Fatalf("the page shows no order lines although the database holds three")
	}

	// The three seeded quantities appear. A fixture read would show only
	// quantities 2 and 1, so these values discriminate the database from it.
	for _, want := range []string{"2.000", "0.000", "5.000"} {
		if !strings.Contains(page, want) {
			t.Fatalf("the page does not show the seeded quantity %q", want)
		}
	}

	// Every line is answered for both row rules.
	for _, rule := range []string{
		"sales.order_line.quantity_positive",
		"sales.order_line.line_number_positive",
	} {
		if got := strings.Count(page, rule); got < 3 {
			t.Fatalf("rule %s is answered %d times, want one per seeded line (3)", rule, got)
		}
	}
	// Exactly one line fails the positive-quantity rule: the one with 0.
	// Count inside the order-lines article only: the page also prints answer
	// pills for the Command's own Canonical Validations, and the styling class
	// names contain "not-satisfied" as a substring.
	lines := sectionBetween(page, `id="order-lines"`, `id="read-identity"`)
	if lines == "" {
		t.Fatalf("the order-lines section could not be isolated")
	}
	if got := strings.Count(lines, `<span class="pill">not-satisfied</span>`); got != 1 {
		t.Fatalf("not-satisfied answer pills in the order-lines section=%d, want exactly the one seeded zero quantity", got)
	}
	if got := strings.Count(lines, `<span class="pill">satisfied</span>`); got != 5 {
		t.Fatalf("satisfied answer pills in the order-lines section=%d, want 5 (two rules on three lines, one failing)", got)
	}

	// The relation-wide rule is refuted by the witnessed duplicate and its
	// answer reaches the page, not only the JSON endpoint.
	if !strings.Contains(page, "sales.order.lines.line_number_unique") {
		t.Fatalf("the page does not show the relation-wide uniqueness answer")
	}
	if !strings.Contains(page, "sales.order.lines.line_number_unique not-satisfied") {
		t.Fatalf("the witnessed duplicate did not refute uniqueness on the page")
	}
}

// webReadPostgresDSN reads the DSN the Web read tests use and refuses to run
// without the explicit disposable-database confirmation. It is the same
// environment contract the runner-level read test uses, so one CI job can
// serve both without a second disposable database.
func webReadPostgresDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv(readPostgresDSNEnv)
	if dsn == "" {
		t.Skip(readPostgresDSNEnv + " is not set; real PostgreSQL Web page not run")
	}
	if os.Getenv(readPostgresNameEnv) != readPostgresDatabase {
		t.Fatalf("refusing to modify %q without %s=%s", readPostgresDatabase, readPostgresNameEnv, readPostgresDatabase)
	}
	return dsn
}

// freeLoopbackAddress reserves and releases a port, so the server binds the
// address the client was told about without racing another process for it.
func freeLoopbackAddress(t *testing.T) string {
	t.Helper()
	reservation, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := reservation.Addr().String()
	if err := reservation.Close(); err != nil {
		t.Fatal(err)
	}
	return address
}

// waitForHealth waits until the target publishes HTTP, and fails with the
// server's own error if it stops first.
func waitForHealth(t *testing.T, address string, done <-chan error) {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	deadline := time.Now().Add(20 * time.Second)
	for {
		response, err := client.Get("http://" + address + "/healthz")
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		select {
		case err := <-done:
			t.Fatalf("the web target stopped before publishing HTTP: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("the web target did not publish HTTP")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// postCustomSubmitPage posts the page's own form and returns the rendered page.
func postCustomSubmitPage(t *testing.T, address string, form url.Values) string {
	t.Helper()
	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.PostForm("http://"+address+"/custom-submit", form)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.StatusCode, raw)
	}
	return string(raw)
}

// sectionBetween isolates one rendered section, so a count inside it does not
// include answers the page prints elsewhere.
func sectionBetween(page, start, end string) string {
	from := strings.Index(page, start)
	if from < 0 {
		return ""
	}
	to := strings.Index(page[from:], end)
	if to <= 0 {
		if rest := strings.Index(page[from:], "</section>"); rest > 0 {
			return page[from : from+rest]
		}
		return page[from:]
	}
	return page[from : from+to]
}

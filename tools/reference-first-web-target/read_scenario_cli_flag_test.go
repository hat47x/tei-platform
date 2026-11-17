//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"tei/internal/workspace"
	"tei/tools/internal/postgresdb"
)

// TestShippedCLIReadsTheRealDatabaseForTheWeb is the first Web Target's read
// half over HTTP with the database supplied the way a user supplies it: the
// `-read-dsn` flag, no embedding Host.
//
// Before this flag existed the read database could only be injected by a Go
// caller (webHostWiring.readDatabase). The running binary therefore had no way
// to serve the order-line read from PostgreSQL, so the HTTP page could show
// only fixture rows while the CLI read showed the database.
//
// The expectation is written from the fixture's seed and its declared rules, not
// copied from the response: order 1001 holds quantities 2, 0 and 5, so exactly
// one line fails the positive rule, and line_number 2 appears twice, so the
// relation-wide uniqueness is refuted even under a partial read.
func TestShippedCLIReadsTheRealDatabaseForTheWeb(t *testing.T) {
	dsn := readPostgresDSN(t)

	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
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
	_ = loaded

	reservation, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := reservation.Addr().String()
	if err := reservation.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		// No webHostWiring: the database arrives only through the flag, which is
		// what a user running the binary has.
		done <- runCombinedWebTargetWithHost(ctx, nil, nil, webHostWiring{},
			[]string{
				"-workspace", referenceWorkspacePath(t), "-application", "order_entry",
				"-baseline-id", "B", "-baseline-revision", testBaselineRevision,
				"-target-revision", testTargetRevision, "-listen", address,
				"-legacy-users", legacyFixturePath(t, "users.json"),
				"-legacy-authorization", legacyFixturePath(t, "authorization.json"),
				"-diagnostic-unaccepted-source",
				"-read-dsn", dsn,
			})
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("web target did not stop with its context")
		}
	})

	client := &http.Client{Timeout: 10 * time.Second}
	base := "http://" + address
	for deadline := time.Now().Add(15 * time.Second); ; {
		response, err := client.Get(base + "/healthz")
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				break
			}
		}
		select {
		case err := <-done:
			t.Fatalf("web target stopped before publishing HTTP: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("web target did not publish HTTP")
		}
		time.Sleep(100 * time.Millisecond)
	}

	response, err := client.Get(base + "/scenario/read?from=2026-09-01T00:00:00Z&row=0")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.StatusCode, raw)
	}
	if strings.Contains(string(raw), dsn) {
		t.Fatal("the HTTP response carried the database connection string")
	}
	var envelope webReadScenarioResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("response is not a read result: %v\n%s", err, raw)
	}
	read := envelope.Result
	if len(read.OrderLineRows) != 3 {
		t.Fatalf("order 1001 holds 3 seeded lines, the HTTP read reported %d: %s",
			len(read.OrderLineRows), raw)
	}
	if len(read.RowAnswers) != len(read.OrderLineRows) {
		t.Fatalf("row answers=%d for %d rows", len(read.RowAnswers), len(read.OrderLineRows))
	}

	// Exactly one not-satisfied answer, on the row that itself holds a zero
	// quantity. The answer is compared against the row at the same position, so
	// a shifted answer list fails here.
	unsatisfied := 0
	for index, row := range read.RowAnswers {
		for _, answer := range row.Answers {
			if answer.Result != "not-satisfied" {
				continue
			}
			unsatisfied++
			if answer.Rule != "sales.order_line.quantity_positive" {
				t.Fatalf("row %d: the failing rule is %q, expected only the positive one", index, answer.Rule)
			}
			quantity, ok := orderLineAttribute(read.OrderLineRows[index], "sales.order_line.quantity")
			if !ok {
				t.Fatalf("row %d is answered but holds no quantity", index)
			}
			if !isZeroQuantity(quantity) {
				t.Fatalf("row %d is not-satisfied but holds quantity %q", index, quantity)
			}
		}
	}
	if unsatisfied != 1 {
		t.Fatalf("not-satisfied answers=%d, the seed puts exactly one zero quantity", unsatisfied)
	}

	// The relation-wide rule is refuted by the witnessed duplicate, which a
	// partial read can still prove.
	const wantDuplicate = "sales.order.lines.line_number_unique not-satisfied"
	found := false
	for _, line := range read.RelationAnswers {
		if !strings.Contains(line, "sales.order.lines.line_number_unique") {
			continue
		}
		found = true
		if line != wantDuplicate {
			t.Fatalf("relation answer=%q, want %q", line, wantDuplicate)
		}
	}
	if !found {
		t.Fatal("no relation-wide uniqueness answer came back over HTTP")
	}
}

// readPostgresDSN reads the DSN the Web read tests use, and refuses to run
// without the explicit disposable-database confirmation.
func readPostgresDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv(readPostgresDSNEnv)
	if dsn == "" {
		t.Skip(readPostgresDSNEnv + " is not set; real PostgreSQL Web read not run")
	}
	if os.Getenv(readPostgresNameEnv) != readPostgresDatabase {
		t.Fatalf("refusing to modify %q without %s=%s", readPostgresDatabase, readPostgresNameEnv, readPostgresDatabase)
	}
	return dsn
}

// isZeroQuantity reports whether PostgreSQL's numeric text denotes zero,
// whatever scale it was returned with ("0", "0.000").
func isZeroQuantity(text string) bool {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "-") {
		text = text[1:]
	}
	text = strings.ReplaceAll(text, ".", "")
	if text == "" {
		return false
	}
	for _, digit := range text {
		if digit != '0' {
			return false
		}
	}
	return true
}

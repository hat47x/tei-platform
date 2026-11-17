//go:build !tei_order_only && !tei_repair_only

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/workspace"
	"tei/tools/internal/postgresdb"
)

// The page must carry the answers of the read the HTTP route actually made.
//
// NOW.md 道の2 asks that the row answers and the detail rows be bound to the
// same real read, the same parent scope and the same Query input. This test
// drives the shipped HTTP route (/scenario/read) with the real database the
// HTTP host selected, and compares the answers to the seed's own values, not to
// the product's JSON.
//
// The browser half of 道の2 (a real Chrome DOM) is covered separately for the
// product CLI; the Web Target's HTTP route is what this test covers.
func TestReadScenarioHTTPAnswersComeFromTheRealDatabaseRead(t *testing.T) {
	dsn := os.Getenv(readPostgresDSNEnv)
	if dsn == "" {
		t.Skip(readPostgresDSNEnv + " is not set; the HTTP route was not read from PostgreSQL")
	}
	if os.Getenv(readPostgresNameEnv) != readPostgresDatabase {
		t.Fatalf("refusing to modify %q without %s=%s", readPostgresDatabase, readPostgresNameEnv, readPostgresDatabase)
	}
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := seedReadFixture(t, db); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec("drop table if exists public.order_lines"); err != nil {
			t.Errorf("cannot clean up disposable order lines: %v", err)
		}
	})

	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	reader, err := newPostgresReadRunner(loaded.World, db)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := buildWebTargetProjection(loaded.World, projectionConfig{
		HandoverBaselineID:         testBaselineID,
		BaselineRepositoryRevision: testBaselineRevision,
		TargetBuildRevision:        testTargetRevision,
		ApplicationID:              core.MustCanonicalID("order_entry"),
	})
	if err != nil {
		t.Fatal(err)
	}
	server, err := newWebTargetServer(projection, reader, nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(
		http.MethodGet, "/scenario/read?from=2026-09-01T00%3A00%3A00Z&row=0", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP read status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	var response webReadScenarioResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	read := response.Result
	if len(read.OrderLineRows) == 0 {
		t.Fatal("the HTTP read reported no order lines although the database holds them")
	}

	// Rows and answers must be one-for-one, and each answer must name the
	// position of the row it is shown with.
	if len(read.RowAnswers) != len(read.OrderLineRows) {
		t.Fatalf("row answers=%d for %d rows", len(read.RowAnswers), len(read.OrderLineRows))
	}
	for index, row := range read.RowAnswers {
		if row.RowIndex != index {
			t.Fatalf("answer %d names row %d", index, row.RowIndex)
		}
	}

	// The read must name itself, and the name must be this HTTP read: the
	// selected order, the executed Contract Query, and the row count.
	if read.ReadIdentity.SelectedOrderKey != "1001" {
		t.Fatalf("read identity names order %q, want the selected one", read.ReadIdentity.SelectedOrderKey)
	}
	if read.ReadIdentity.Database != "sales_core" ||
		read.ReadIdentity.Contract != "sales_core.order_line" ||
		read.ReadIdentity.Query != "find_by_order" {
		t.Fatalf("read identity does not name the executed Contract Query: %+v", read.ReadIdentity)
	}
	if read.ReadIdentity.RowCount != len(read.OrderLineRows) {
		t.Fatalf("read identity states %d rows but the page shows %d", read.ReadIdentity.RowCount, len(read.OrderLineRows))
	}

	// The seed puts quantity 0 on exactly one line of order 1001. Its
	// positive-quantity rule must be the only not-satisfied answer, and it must
	// sit on the row that actually holds 0.
	unsatisfied := 0
	for index, row := range read.RowAnswers {
		for _, answer := range row.Answers {
			if answer.Result != "not-satisfied" {
				if answer.Result != "satisfied" && answer.Result != "undecided" {
					t.Fatalf("row %d rule %s has unknown answer %q", index, answer.Rule, answer.Result)
				}
				continue
			}
			unsatisfied++
			if answer.Rule != "sales.order_line.quantity_positive" {
				t.Fatalf("row %d: unexpected rule %q failed", index, answer.Rule)
			}
			quantity, ok := orderLineAttribute(read.OrderLineRows[index], "sales.order_line.quantity")
			if !ok {
				t.Fatalf("row %d has no quantity although it was answered", index)
			}
			if strings.TrimSpace(quantity) != "0" {
				t.Fatalf("row %d is marked not-satisfied but holds quantity %q; the seed puts 0 on exactly one line",
					index, quantity)
			}
		}
	}
	if unsatisfied != 1 {
		t.Fatalf("not-satisfied answers=%d, want exactly the one seeded zero quantity", unsatisfied)
	}

	// A rule about the whole relation stays undecided while the read cannot
	// certify it saw every row, and it must say why.
	for _, line := range read.RelationAnswers {
		if strings.Contains(line, "line_number_unique") && !strings.Contains(line, "undecided") {
			t.Fatalf("relation-wide rule did not stay undecided: %q", line)
		}
	}
}

//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/workspace"
)

// An order whose Query returned no detail lines must reach the page as two
// separate statements, not as one verdict:
//
//   - the relation-wide uniqueness answer is "satisfied", because there is no
//     duplicate among zero rows, and
//   - the record-level rules could not be evaluated at all, which is an
//     Execution Error rather than a Validation pass.
//
// docs/first-change.md tells a reader that `Selected row` 2 (order 1003, no
// lines) shows "No order-line rows." together with an Execution Error. This
// fixes that HTTP outcome without a database: the read below is the real
// reference projection with its rows removed, so the rendering and the status
// mapping are the shipped ones.
//
// The expected values are written here, not read from the response: the page
// must say exactly these things, and a page that silently reported "every line
// passed" would fail every assertion below.
func TestShippedFormRendersAnEmptyOrderAsAnExecutionError(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
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

	// Start from the real projection, then remove the rows. Taking the rows
	// away from a read that DID return them is what makes the emptiness the
	// only difference from the ordinary page.
	base, err := newReferenceReadScenarioRunner(loaded.World).Run(
		context.Background(),
		readScenarioInput{From: "2026-09-01T00:00:00Z", RowIndex: 0},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(base.OrderLineRows) < 2 {
		t.Fatalf("test setup: the reference read returned %d lines, so removing them would prove nothing",
			len(base.OrderLineRows))
	}
	base.OrderLineRows = nil
	base.RowAnswers = nil
	// The identity must describe the read that was actually answered. Leaving
	// RowCount at its old value would be a different fault (a page whose
	// answers do not match its own identity) and is already refused elsewhere.
	base.ReadIdentity.RowCount = 0
	base.RelationAnswers = []string{"sales.order.lines.line_number_unique satisfied"}

	read := staticReadScenarioRunner{result: base}
	commandRunner, err := newReferenceCommandScenarioRunner(
		loaded.World,
		legacyFixturePath(t, "users.json"),
		legacyFixturePath(t, "authorization.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	commandRunner.read = read
	server, err := newWebTargetServer(projection, read, commandRunner)
	if err != nil {
		t.Fatal(err)
	}

	recorder := postCustomSubmit(t, server.Handler(), url.Values{
		"loginId":    {"legacy-order-editor"},
		"credential": {"editor-secret"},
		"from":       {"2026-09-01T00:00:00Z"},
		"row":        {"2"},
	})
	body := recorder.Body.String()

	// 422 is the status the shipped handler uses for an Execution Error, and
	// an empty collection is one. A 200 here would tell the reader the order
	// was judged.
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("an order with no lines returned status %d, want 422 (body: %s)",
			recorder.Code, body)
	}

	for _, want := range []string{
		// The two statements the guide tells a reader to read together.
		"No order-line rows.",
		"sales.order.lines.line_number_unique satisfied",
		// The refusal itself, naming what could not be evaluated.
		"Execution error",
		"cannot evaluate positive validation sales.order_line.quantity_positive over an empty order-line collection",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("the page does not say %q, so a reader cannot tell an empty order from a satisfied one", want)
		}
	}

	// The opposite statement must be absent. A page that says all lines passed
	// is exactly the reading this test exists to prevent.
	for _, forbidden := range []string{
		"validation-not-satisfied",
		"All lines satisfied",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("an empty order was rendered as %q", forbidden)
		}
	}
}

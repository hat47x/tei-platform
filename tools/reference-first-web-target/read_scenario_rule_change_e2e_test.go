//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tei/tools/internal/postgresdb"
)

// TestFirstWebAcceptsARuleChangeEndToEnd is plan/NOW.md road item 2's remaining
// acceptance: one rule change reaches the first Web Target's HTTP page, and the
// answers the page shows change with it while the database rows stay the same.
//
// The two runs use the same seeded database and the same binary. Only the
// Workspace differs: the second run has the unit-price rule added, exactly as
// testdata/destination-order-change/edit/unit-price.tei.md writes it. That is
// the change docs/first-change.md tells a reader to make.
//
// The expected answers are read from expected-before.json and
// expected-after.json, which were authored independently of both entries. What
// this test adds is the browser-facing half: the page a person opens carries the
// before answers, and after the edit it carries the after answers, without
// touching the database.
func TestFirstWebAcceptsARuleChangeEndToEnd(t *testing.T) {
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
			page := runFirstWebAndReadOrder(t, scratch)
			if strings.Contains(page, dsn) {
				t.Fatal("the page carried the database connection string")
			}

			want := expectedAnswersForOrder(t, filepath.Join(fixtures, stage.expected), "1001")
			lines := sectionBetween(page, `id="order-lines"`, `id="read-identity"`)
			if lines == "" {
				t.Fatal("the page did not render the order-line read")
			}
			requirePageAnswersMatchExpectation(t, lines, want)

			// The change is a rule, not a row: the same three primary keys and
			// the same values must appear in both stages.
			for _, seeded := range []string{"2.000", "0.000", "5.000"} {
				if !strings.Contains(lines, seeded) {
					t.Fatalf("stage=%s lost the seeded quantity %q, so the rows changed too", stage.name, seeded)
				}
			}
			if !strings.Contains(page, "sales.order.lines.line_number_unique not-satisfied") {
				t.Fatalf("stage=%s lost the witnessed duplicate's relation decision", stage.name)
			}

			// The HTTP body is not the only statement: a real browser must
			// receive this page and keep the same answers in its DOM. Each stage
			// gets its own run and receipt, so one page cannot stand in for both.
			t.Run("chrome-dom", func(t *testing.T) {
				browser := os.Getenv("TEI_VIEW_BROWSER_BINARY")
				if browser == "" {
					t.Skip("TEI_VIEW_BROWSER_BINARY is unset; real Chrome DOM not run")
				}
				address := freeLoopbackAddress(t)
				ctx, cancel := context.WithCancel(context.Background())
				stopped := make(chan error, 1)
				go func() {
					stopped <- runCombinedWebTargetWithHost(ctx, nil, nil, webHostWiring{},
						[]string{
							"-workspace", scratch, "-application", "order_entry",
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
					case <-stopped:
					case <-time.After(10 * time.Second):
						t.Error("the web target did not stop with its context")
					}
				})
				waitForHealth(t, address, stopped)

				dom := chromeDOMOfSubmittedOrder(t, browser, address)
				if strings.Contains(dom, dsn) {
					t.Fatal("the Chrome DOM carried the database connection string")
				}
				chromeLines := sectionBetween(dom, `id="order-lines"`, `id="read-identity"`)
				if chromeLines == "" {
					t.Fatal("the Chrome DOM did not contain the order-line read")
				}
				requirePageAnswersMatchExpectation(t, chromeLines, want)
				t.Logf("actual rule-change Chrome DOM verified for order 1001 (stage %s)", stage.name)
			})
		})
	}
}

// runFirstWebAndReadOrder starts the shipped entry point on one Workspace and
// returns the page the Web Target serves for order 1001.
func runFirstWebAndReadOrder(t *testing.T, workspaceRoot string) string {
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
	return postCustomSubmitPage(t, address, url.Values{
		"loginId":    {"legacy-order-editor"},
		"credential": {"editor-secret"},
		"from":       {"2026-09-01T00:00:00Z"},
		"row":        {"0"},
	})
}

// expectedAnswersForOrder reads one order's per-line answers from an
// independently authored expectation file. The key of perLine is the detail's
// order_line_id, not a line number or a position.
func expectedAnswersForOrder(t *testing.T, path, orderID string) map[string]map[string]string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var expected struct {
		Orders map[string]struct {
			PerLine map[string]map[string]string `json:"perLine"`
		} `json:"orders"`
	}
	if err := json.Unmarshal(raw, &expected); err != nil {
		t.Fatalf("%s is not the expected shape: %v", path, err)
	}
	order, ok := expected.Orders[orderID]
	if !ok {
		t.Fatalf("%s has no entry for order %s", path, orderID)
	}
	return order.PerLine
}

// requirePageAnswersMatchExpectation checks each rendered order-line block
// against the independent expectation, joined by the block's declared
// primary-key attribute.
func requirePageAnswersMatchExpectation(t *testing.T, lines string, want map[string]map[string]string) {
	t.Helper()
	blocks := strings.Split(lines, "<p><strong>Line ")
	if got := len(blocks) - 1; got != len(want) {
		t.Fatalf("rendered %d order-line blocks, the expectation lists %d rows", got, len(want))
	}
	valueOf := func(block, attribute string) (string, bool) {
		marker := "<code>" + attribute + "</code></th><td>"
		start := strings.Index(block, marker)
		if start < 0 {
			return "", false
		}
		tail := block[start+len(marker):]
		end := strings.Index(tail, "</td>")
		if end < 0 {
			return "", false
		}
		return strings.TrimSpace(tail[:end]), true
	}
	seen := map[string]bool{}
	for _, block := range blocks[1:] {
		key, ok := valueOf(block, "sales.order_line.order_line_id")
		if !ok {
			t.Fatal("a rendered line lacks its declared primary-key attribute")
		}
		answers, exists := want[key]
		if !exists {
			t.Fatalf("the page rendered order_line_id %q, which the expectation does not list", key)
		}
		if seen[key] {
			t.Fatalf("order_line_id %q was rendered twice", key)
		}
		seen[key] = true

		rendered := 0
		for rule, answer := range answers {
			fragment := "<code>" + rule + "</code> <span class=\"pill\">" + answer + "</span>"
			if !strings.Contains(block, fragment) {
				t.Fatalf("order_line_id %s does not show %s=%s", key, rule, answer)
			}
			rendered++
		}
		if count := strings.Count(block, "<p class=\"status-line\">"); count != rendered {
			t.Fatalf("order_line_id %s shows %d rule answers, the expectation lists %d", key, count, rendered)
		}
	}
}

// copyWorkspaceForRuleChange copies the Workspace into a scratch directory.
// Line endings are normalised for the Source types the loader fingerprints, the
// same reconstruction copyPracticalBWorkspace performs for its historical
// fixture, so a CRLF checkout cannot change the rule text.
func copyWorkspaceForRuleChange(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		switch filepath.Ext(rel) {
		case ".tei", ".sql", ".md", ".yaml":
			data = []byte(strings.ReplaceAll(string(data), "\r\n", "\n"))
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("cannot copy the Workspace for the rule change: %v", err)
	}
}

// applyUnitPriceRule makes the same edit docs/first-change.md describes: select
// the authoring language, add the rule's document, and declare the rule in the
// Application. It refuses if the fixture no longer matches what it expects, so a
// silent no-op cannot make this test pass.
func applyUnitPriceRule(t *testing.T, root, ruleDocument string) {
	t.Helper()
	manifest := filepath.Join(root, "tei.workspace.yaml")
	content, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "languages:") {
		t.Fatalf("the Workspace already selects a language; the edit this test performs would be a no-op")
	}
	content = append(content, []byte("languages:\n  - tei.authoring/v1\n")...)
	if err := os.WriteFile(manifest, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "unit-price.tei.md"), []byte(ruleDocument), 0o644); err != nil {
		t.Fatal(err)
	}
	application := filepath.Join(root, "applications", "order-entry", "application.tei")
	content, err = os.ReadFile(application)
	if err != nil {
		t.Fatal(err)
	}
	const anchor = "    uses validation sales.order_line.quantity_positive\n"
	if strings.Count(string(content), anchor) != 1 {
		t.Fatalf("application.tei does not hold %q exactly once", anchor)
	}
	edited := strings.Replace(string(content), anchor,
		anchor+"    uses validation sales.order_line.unit_price_positive\n", 1)
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
}

// destinationFixtureRoot locates the independent fixtures from this source file
// rather than from the process's working directory.
func destinationFixtureRoot(t *testing.T) string {
	t.Helper()
	for _, candidate := range []string{
		filepath.Join("..", "..", "testdata", "destination-order-change"),
		filepath.Join("testdata", "destination-order-change"),
	} {
		if _, err := os.Stat(filepath.Join(candidate, "expected-before.json")); err == nil {
			return candidate
		}
	}
	t.Fatal("cannot locate testdata/destination-order-change from this package")
	return ""
}

// chromeDOMOfSubmittedOrder opens the running target's form in headless Chrome.
//
// The bootstrap page only submits the form; the final document is served by the
// running first Web Target, backed by the real PostgreSQL read. A browser
// opening a static file would not exercise this HTTP boundary.
func chromeDOMOfSubmittedOrder(t *testing.T, browser, address string) string {
	t.Helper()
	return chromeDOMOfSelectedOrder(t, browser, address, 0)
}

// chromeDOMOfSelectedOrder submits the same reference authentication and
// explicitly selected browse row. This still uses a test-only bootstrap form,
// not the shipped form's individual DOM controls.
func chromeDOMOfSelectedOrder(t *testing.T, browser, address string, row int) string {
	t.Helper()
	if row != 0 && row != 1 {
		t.Fatalf("the published first-change guide selects only rows 0 and 1, got %d", row)
	}
	bootstrap := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, fmt.Sprintf(`<!doctype html><html><body>
<form id="bootstrap-post" method="post" action="http://%s/custom-submit">
<input type="hidden" name="loginId" value="legacy-order-editor">
<input type="hidden" name="credential" value="editor-secret">
<input type="hidden" name="from" value="2026-09-01T00:00:00Z">
<input type="hidden" name="row" value="%d">
</form><script>document.getElementById("bootstrap-post").submit()</script></body></html>`, address, row))
	}))
	defer bootstrap.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, browser,
		"--headless", "--no-sandbox", "--disable-gpu", "--disable-dev-shm-usage",
		"--no-first-run", "--virtual-time-budget=7500",
		"--user-data-dir="+t.TempDir(), "--dump-dom", bootstrap.URL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Chrome failed to render the first Web Target HTTP page: %v", err)
	}
	dom := string(output)
	if strings.Contains(dom, `id="bootstrap-post"`) ||
		!strings.Contains(dom, `id="read-result"`) {
		t.Fatal("Chrome did not navigate from the submission bootstrap to the Web Target result")
	}
	return dom
}

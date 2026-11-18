//go:build !tei_repair_only

package main

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tei/tools/internal/postgresdb"
)

// copyWorkspaceBytes copies a Workspace without rewriting any file. Source
// fingerprints are recorded over the bytes that were accepted, so a copy that
// normalized line endings would no longer be the accepted revision.
func copyWorkspaceBytes(t *testing.T, src, dst string) {
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
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// applyRuleEdit performs the change plan/NOW.md step 3 describes: add one rule
// document, select the authoring language, and use the new rule. It edits the
// Workspace the Web Target will load, so the answers the page shows have to
// change with it.
func applyRuleEdit(t *testing.T, root, ruleDocument string) {
	t.Helper()
	manifest := filepath.Join(root, "tei.workspace.yaml")
	content, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "languages:") {
		t.Fatalf("the copied workspace already selects a language")
	}
	if err := os.WriteFile(manifest, append(content, []byte("languages:\n  - tei.authoring/v1\n")...), 0o644); err != nil {
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

// startWebTargetFor runs the shipped entry point against one Workspace and
// returns its address. The context cancels the server.
func startWebTargetFor(t *testing.T, ctx context.Context, workspace string) string {
	t.Helper()
	address := freeLoopbackAddress(t)
	done := make(chan error, 1)
	go func() {
		done <- runCombinedWebTargetWithHost(ctx, nil, nil, webHostWiring{},
			[]string{
				"-workspace", workspace, "-application", "order_entry",
				"-baseline-id", "B", "-baseline-revision", testBaselineRevision,
				"-target-revision", testTargetRevision, "-listen", address,
				"-legacy-users", legacyFixturePath(t, "users.json"),
				"-legacy-authorization", legacyFixturePath(t, "authorization.json"),
				"-diagnostic-unaccepted-source",
				"-serve-read-from-db",
			})
	}()
	t.Cleanup(func() {
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("the web target did not stop with its context")
		}
	})
	waitForHealth(t, address, done)
	return address
}

// perLineExpectation reads orders -> order -> perLine from the fixture's hand
// written expectation file.
func perLineExpectation(t *testing.T, path, order string) map[string]map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Orders map[string]struct {
			PerLine map[string]map[string]string `json:"perLine"`
		} `json:"orders"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("cannot read %s: %v", path, err)
	}
	entry, ok := file.Orders[order]
	if !ok {
		t.Fatalf("%s has no order %s", path, order)
	}
	return entry.PerLine
}

// requireAnswersFromExpectation checks each rendered order-line block against
// the expectation for its declared primary key. A block is found by that key,
// never by its position on the page.
func requireAnswersFromExpectation(t *testing.T, lines string, expected map[string]map[string]string, wantRule string) {
	t.Helper()
	parts := strings.Split(lines, "<p><strong>Line ")
	if got := len(parts) - 1; got != len(expected) {
		t.Fatalf("rendered %d order-line blocks, want %d", got, len(expected))
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
	for _, block := range parts[1:] {
		key, ok := valueOf(block, "sales.order_line.order_line_id")
		if !ok {
			t.Fatal("a rendered line has no declared primary-key attribute")
		}
		want, declared := expected[key]
		if !declared || seen[key] {
			t.Fatalf("rendered an unknown or repeated order-line primary key %q", key)
		}
		seen[key] = true
		for rule, answer := range want {
			if !strings.Contains(block, "<code>"+rule+"</code> <span class=\"pill\">"+answer+"</span>") {
				t.Fatalf("order line %s does not carry the expected answer %s=%s", key, rule, answer)
			}
		}
		// The point of the before/after pair: the new rule is absent before the
		// edit and present after it. A page that always shows the same rules
		// would pass neither stage's expectation.
		hasNewRule := strings.Contains(block, "sales.order_line.unit_price_positive")
		switch {
		case wantRule == "unit_price_positive" && !hasNewRule:
			t.Fatalf("order line %s does not show the rule the edit added", key)
		case wantRule == "" && hasNewRule:
			t.Fatalf("order line %s shows the edited rule before the edit", key)
		}
	}
}

// TestShippedWebPageShowsTheRuleEditThroughChrome is plan/NOW.md road item 2's
// remaining half: the rule change in step 3 is visible in the Chrome DOM of the
// page the first Web Target serves from the real database, through the same
// route both before and after.
//
// The expected answers come from testdata/destination-order-change's
// expected-before.json and expected-after.json, which were written by hand from
// the rules and the seed. They are not copied from the response. Running both
// stages through one route is the point: same database, same seed, same page,
// same read, one rule added.
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

	fixtures := fixtureRoot(t)
	editedRule, err := os.ReadFile(filepath.Join(fixtures, "edit", "unit-price.tei.md"))
	if err != nil {
		t.Fatal(err)
	}

	for _, stage := range []struct {
		name       string
		expected   string
		edited     bool
		wantedrnew string
	}{
		{name: "before", expected: "expected-before.json"},
		{name: "after", expected: "expected-after.json", edited: true, wantedrnew: "unit_price_positive"},
	} {
		t.Run(stage.name, func(t *testing.T) {
			root := t.TempDir()
			copyWorkspaceBytes(t, referenceWorkspacePath(t), root)
			if stage.edited {
				applyRuleEdit(t, root, string(editedRule))
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			address := startWebTargetFor(t, ctx, root)

			form := url.Values{
				"loginId":    {"legacy-order-editor"},
				"credential": {"editor-secret"},
				"from":       {"2026-09-01T00:00:00Z"},
				"row":        {"0"},
			}
			page := postCustomSubmitPage(t, address, form)
			if strings.Contains(page, dsn) {
				t.Fatal("the page carried the database connection string")
			}
			lines := sectionBetween(page, `id="order-lines"`, `id="read-identity"`)
			if lines == "" {
				t.Fatal("the served page did not render the order-line read")
			}
			requireAnswersFromExpectation(t, lines,
				perLineExpectation(t, filepath.Join(fixtures, stage.expected), "1001"), stage.wantedrnew)

			// The same route, opened by a real browser. A browser opening a
			// static CLI file would not exercise this HTTP boundary, so this is
			// a separate receipt from the body assertion above.
			t.Run("chrome-dom", func(t *testing.T) {
				dom := actualWebReadChromeDOM(t, address)
				if !strings.Contains(dom, `id="read-result"`) || strings.Contains(dom, dsn) {
					t.Fatal("Chrome did not receive the PostgreSQL-backed read or exposed the DSN")
				}
				if !strings.Contains(dom, "sales.order.lines.line_number_unique not-satisfied") {
					t.Fatal("Chrome DOM lost the witnessed duplicate's Relation decision")
				}
				domLines := sectionBetween(dom, `id="order-lines"`, `id="read-identity"`)
				if domLines == "" {
					t.Fatal("Chrome DOM did not contain the order-line read")
				}
				requireAnswersFromExpectation(t, domLines,
					perLineExpectation(t, filepath.Join(fixtures, stage.expected), "1001"), stage.wantedrnew)
				t.Logf("actual first Web HTTP Chrome DOM verified for the %s stage of order 1001", stage.name)
			})
		})
	}
}

//go:build !tei_order_only && !tei_repair_only

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"tei/tools/internal/postgresdb"
)

// TestFirstChangeGuideRunsEndToEnd walks docs/first-change.md's five steps in
// order, as written, against one real PostgreSQL database.
//
// The guide is the acceptance document for plan/NOW.md's destination, so it has
// to be checked as a sequence: step 2's answers, step 3's three-file edit,
// step 4's validate and re-read, and step 5's page in a real browser. Every
// existing test checks one slice; nothing ran the guide start to finish.
//
// It builds the CLI from this module and runs it as a separate process, the way
// a reader would, rather than calling the handlers in-process. Expected answers
// come from testdata/destination-order-change/expected-before.json and
// expected-after.json, which were authored independently of the entries.
func TestFirstChangeGuideRunsEndToEnd(t *testing.T) {
	dsn := webReadPostgresDSN(t)
	binary := buildCLIForGuide(t)

	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := requireIsolatedWebReadDatabase(context.Background(), db, readPostgresDatabase); err != nil {
		t.Fatal(err)
	}

	fixtures := destinationFixtureRoot(t)
	ruleDocument, err := os.ReadFile(filepath.Join(fixtures, "edit", "unit-price.tei.md"))
	if err != nil {
		t.Fatal(err)
	}

	// Step 0: the tables come from the definitions, then the guide's seed.
	root := t.TempDir()
	copyWorkspaceForRuleChange(t, referenceWorkspacePath(t), root)
	applySchemaFromDefinitions(t, binary, root, db)
	if err := seedReadFixture(t, db); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec("drop table if exists public.order_lines") })
	t.Cleanup(func() { _, _ = db.Exec("drop table if exists public.orders") })

	// Step 2: read the detail and see the answers.
	before := readOrderThroughCLI(t, binary, root, dsn, "1001")
	requireCLIAnswersMatchExpectation(t, before, expectedAnswersForOrder(t, filepath.Join(fixtures, "expected-before.json"), "1001"))

	// Step 3: add the unit-price rule, exactly the three edits the guide writes.
	applyUnitPriceRule(t, root, string(ruleDocument))

	// Step 4a: validate. The edit must not be rejected.
	if validated := runGuideCLI(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("step 4: validate rejected the edited Workspace: code=%d stdout=%s stderr=%s",
			validated.code, validated.stdout, validated.stderr)
	}
	// The guide says an example that contradicts its rule is a diagnostic. If
	// that were not so, the validate above would prove nothing about examples.
	contradicted := contradictTheRuleExample(t, root)
	if validated := runGuideCLI(t, binary, "validate", "-workspace", contradicted); validated.code == 0 {
		t.Fatal("step 4: an example contradicting its rule passed validation")
	} else if !strings.Contains(validated.stdout+validated.stderr, "RULE_EXAMPLE_MISMATCH") {
		t.Fatalf("step 4: the example mismatch was not the diagnostic: %s", validated.stdout+validated.stderr)
	}

	// Step 4b: read again. The new rule's answer appears and the rows do not change.
	after := readOrderThroughCLI(t, binary, root, dsn, "1001")
	requireCLIAnswersMatchExpectation(t, after, expectedAnswersForOrder(t, filepath.Join(fixtures, "expected-after.json"), "1001"))
	for _, key := range []string{"1", "2", "3"} {
		if _, ok := after.Rows[key]; !ok {
			t.Fatalf("step 4: the re-read lost order-line %s, so the rows changed too", key)
		}
	}
	beforeAnswer, _ := before.Answers["3"]["sales.order_line.quantity_positive"]
	afterAnswer, ok := after.Answers["3"]["sales.order_line.unit_price_positive"]
	if !ok || afterAnswer != "not-satisfied" {
		t.Fatalf("step 4: the edited rule did not appear as not-satisfied for the zero-price line: %+v", after.Answers["3"])
	}
	if beforeAnswer == "" {
		t.Fatal("step 2: the pre-edit read answered no quantity rule for the same line")
	}

	// Step 5: the same Workspace in the first Web Target, in a real browser.
	browser := os.Getenv("TEI_VIEW_BROWSER_BINARY")
	if browser == "" {
		t.Skip("TEI_VIEW_BROWSER_BINARY is unset; step 5's browser receipt not run")
	}
	t.Setenv(readDSNEnv, dsn)
	address := freeLoopbackAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runCombinedWebTargetWithHost(ctx, nil, nil, webHostWiring{},
			[]string{
				"-workspace", root, "-application", "order_entry",
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
	dom := chromeDOMOfSubmittedOrder(t, browser, address)
	if strings.Contains(dom, dsn) {
		t.Fatal("step 5: the page carried the database connection string")
	}
	pageLines := sectionBetween(dom, `id="order-lines"`, `id="read-identity"`)
	if pageLines == "" {
		t.Fatal("step 5: the page did not render the order-line read")
	}
	// The page must carry the after answers, because the guide passes the edited
	// Workspace. Handing it the original Workspace would show the pre-edit rules.
	requirePageAnswersMatchExpectation(t, pageLines, expectedAnswersForOrder(t, filepath.Join(fixtures, "expected-after.json"), "1001"))
	t.Log("first-change guide steps 0-5 verified end to end against real PostgreSQL and Chrome")
}

// cliOrderRead is the part of `tei view -environment` this acceptance reads.
type cliOrderRead struct {
	Rows    map[string]bool
	Answers map[string]map[string]string
}

// readOrderThroughCLI runs the guide's step-2 command as a separate process.
func readOrderThroughCLI(t *testing.T, binary, workspaceRoot, dsn, orderID string) cliOrderRead {
	t.Helper()
	got := runGuideCLI(t, binary,
		"view", "-workspace", workspaceRoot, "-application", "order_entry",
		"-pattern", "order_entry.order_lines", "-environment", "development",
		"-input", "order_id="+orderID, "-format", "json")
	if got.code != 0 {
		t.Fatalf("view failed: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
	}
	if strings.Contains(got.stdout, dsn) {
		t.Fatal("the CLI JSON carried the database connection string")
	}
	var report struct {
		OK     bool `json:"ok"`
		Result struct {
			Rows []struct {
				Key string `json:"key"`
			} `json:"rows"`
			PerRow map[string]map[string]string `json:"perRow"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &report); err != nil {
		t.Fatalf("the CLI output is not the documented JSON: %v", err)
	}
	if !report.OK {
		t.Fatalf("view reported failure: %s", got.stdout)
	}
	// Relation completeness belongs to the separate Query/coverage qualification
	// tests. This five-step acceptance checks the same concrete row answers even
	// when the reference binding gains a justified complete-coverage claim.
	read := cliOrderRead{Rows: map[string]bool{}, Answers: report.Result.PerRow}
	for _, row := range report.Result.Rows {
		read.Rows[row.Key] = true
	}
	return read
}

// requireCLIAnswersMatchExpectation checks the CLI's per-row answers against the
// independent expectation, joined by the row key the guide names.
func requireCLIAnswersMatchExpectation(t *testing.T, got cliOrderRead, want map[string]map[string]string) {
	t.Helper()
	if len(got.Rows) != len(want) {
		t.Fatalf("the read returned %d rows, the expectation lists %d", len(got.Rows), len(want))
	}
	if len(got.Answers) != len(want) {
		t.Fatalf("the read returned %d row-answer groups, the expectation lists %d", len(got.Answers), len(want))
	}
	for key, wantRules := range want {
		if !got.Rows[key] {
			t.Fatalf("order-line %s has answers but is absent from the actual read rows", key)
		}
		answers, ok := got.Answers[key]
		if !ok {
			t.Fatalf("the read did not answer order-line %s", key)
		}
		if len(answers) != len(wantRules) {
			t.Fatalf("order-line %s: %d answers, the expectation lists %d rules", key, len(answers), len(wantRules))
		}
		for rule, wantAnswer := range wantRules {
			if got := answers[rule]; got != wantAnswer {
				t.Fatalf("order-line %s rule %s: got %q, the independent expectation says %q", key, rule, got, wantAnswer)
			}
		}
	}
}

// applySchemaFromDefinitions runs the guide's step 0: the Create statements the
// CLI prints are applied to the database, so the table the guide describes is
// the table this acceptance reads.
func applySchemaFromDefinitions(t *testing.T, binary, workspaceRoot string, db *sql.DB) {
	t.Helper()
	printed := runGuideCLI(t, binary,
		"desired-schema", "-workspace", workspaceRoot,
		"-database", "sales_core", "-provider", "postgres", "-format", "sql")
	if printed.code != 0 {
		t.Fatalf("step 0: desired-schema failed: code=%d stderr=%s", printed.code, printed.stderr)
	}
	// Start from an empty schema so an earlier run cannot stand in for these
	// statements.
	for _, table := range []string{"public.order_lines", "public.orders"} {
		if _, err := db.Exec("drop table if exists " + table + " cascade"); err != nil {
			t.Fatalf("step 0: cannot clear %s: %v", table, err)
		}
	}
	applied := 0
	for _, statement := range strings.Split(printed.stdout, ";") {
		trimmed := strings.TrimSpace(statement)
		if trimmed == "" {
			continue
		}
		if _, err := db.Exec(trimmed); err != nil {
			t.Fatalf("step 0: a printed statement did not apply: %v\nstatement:\n%s", err, trimmed)
		}
		applied++
	}
	if applied == 0 {
		t.Fatal("step 0: desired-schema printed no statements")
	}
}

// contradictTheRuleExample makes the copy the guide describes: it rewrites the
// example's not-satisfied cell to satisfied, which must then be a diagnostic.
func contradictTheRuleExample(t *testing.T, root string) string {
	t.Helper()
	scratch := t.TempDir()
	copyWorkspaceForRuleChange(t, root, scratch)
	document := filepath.Join(scratch, "semantic", "sales", "unit-price.tei.md")
	content, err := os.ReadFile(document)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(content), "| 0 | not-satisfied |", "| 0 | satisfied |", 1)
	if edited == string(content) {
		t.Fatal("the example table does not hold the row the guide names, so this check would be vacuous")
	}
	if err := os.WriteFile(document, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	return scratch
}

// buildCLIForGuide builds the CLI from this module, once per test run, the way
// the guide's first command does.
func buildCLIForGuide(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	var stderr bytes.Buffer
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/tei")
	cmd.Dir = repoRootForGuide(t)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("cannot build the CLI: %v: %s", err, stderr.String())
	}
	return binary
}

// repoRootForGuide resolves the module root from this source file.
func repoRootForGuide(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve the test source path")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// guideCLIResult keeps stdout and stderr apart, so a diagnostic on one stream is
// not read as a JSON document on the other.
type guideCLIResult struct {
	stdout string
	stderr string
	code   int
}

// runGuideCLI runs one CLI command the way a reader does: as a separate process.
func runGuideCLI(t *testing.T, binary string, args ...string) guideCLIResult {
	t.Helper()
	var stdout, stderr bytes.Buffer
	command := exec.Command(binary, args...)
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("cannot run the CLI: %v", err)
		}
		code = exit.ExitCode()
	}
	return guideCLIResult{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

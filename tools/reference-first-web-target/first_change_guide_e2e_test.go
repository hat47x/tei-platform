//go:build !tei_order_only && !tei_repair_only

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/tools/internal/postgresdb"
)

// TestFirstChangeGuideRunsEndToEnd checks the handbook's five semantic steps
// against one real PostgreSQL database. Setup, Workspace copy and rule editing
// execute the actual POSIX shell blocks from the handbook in one attempt.
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

	fixtures, err := filepath.Abs(destinationFixtureRoot(t))
	if err != nil {
		t.Fatal("cannot resolve the published fixture directory")
	}
	guide, err := os.ReadFile(filepath.Join(repoRootForGuide(t), "docs", "first-change.md"))
	if err != nil {
		t.Fatal(err)
	}
	handbook := string(guide)
	if runtime.GOOS == "windows" {
		t.Skip("the documented POSIX shell and symlinks are tested by the PostgreSQL Linux CI")
	}

	// Step 0: execute the handbook's actual ./tei + psql -1 script. The
	// earlier test generated and seeded the database through Go helpers instead.
	// A disposable directory preserves the handbook's relative CLI and seed
	// paths without modifying the repository or introducing a second fixture.
	attempt := t.TempDir()
	for _, dir := range []string{"reference", "testdata"} {
		if err := os.MkdirAll(filepath.Join(attempt, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for target, source := range map[string]string{
		"tei":                               binary,
		"reference/workspace":               filepath.Join(repoRootForGuide(t), "reference", "workspace"),
		"testdata/destination-order-change": fixtures,
	} {
		if err := os.Symlink(source, filepath.Join(attempt, target)); err != nil {
			t.Fatal(err)
		}
	}
	var preexistingParents, preexistingDetails bool
	if err := db.QueryRowContext(t.Context(),
		"SELECT to_regclass('public.orders') IS NOT NULL, to_regclass('public.order_lines') IS NOT NULL").
		Scan(&preexistingParents, &preexistingDetails); err != nil {
		t.Fatal("cannot verify the disposable database starts empty")
	}
	if preexistingParents || preexistingDetails {
		// The dedicated CI database is shared with earlier tests, which may
		// leave their order fixtures behind. Reset only the two known order
		// tables, and only after verifying the ACTUAL isolated database name.
		// This is test setup, not a hidden replacement for the guide's DDL.
		// Do not cascade into objects owned by another fixture, even in this DB.
		for _, table := range []string{"public.order_lines", "public.orders"} {
			if _, err := db.Exec("DROP TABLE IF EXISTS " + table); err != nil {
				t.Fatal("cannot reset the isolated order tables before step 0")
			}
		}
	}
	if err := db.QueryRowContext(t.Context(),
		"SELECT to_regclass('public.orders') IS NOT NULL, to_regclass('public.order_lines') IS NOT NULL").
		Scan(&preexistingParents, &preexistingDetails); err != nil || preexistingParents || preexistingDetails {
		t.Fatal("step 0 did not start without parent/detail tables in the isolated database")
	}
	// Register destructive cleanup only after the isolated database guard.
	// This cleanup is never registered against an arbitrary user database.
	t.Cleanup(func() {
		for _, table := range []string{"public.order_lines", "public.orders"} {
			if _, err := db.Exec("drop table if exists " + table); err != nil {
				t.Errorf("cannot clean up generated %s table: %v", table, err)
			}
		}
	})
	setup := exec.Command("sh", "-eu", "-c", documentedGuideShellStep(t, handbook,
		"## 0. 表を作り、明細を入れる", "## 1. 作業領域の写しを作る"))
	setup.Dir = attempt
	setup.Env = append(os.Environ(), "TEI_VIEW_PG_DSN="+dsn, "TMPDIR="+attempt)
	if _, err := setup.CombinedOutput(); err != nil {
		// psql output may include a connection URL. Never include its raw
		// stderr or the environment in the CI failure record.
		t.Fatalf("step 0's documented CLI and atomic psql script failed: %v (output withheld)", err)
	}
	var ordersInBrowse, linesInRead int
	if err := db.QueryRow("SELECT count(*) FROM public.orders WHERE order_id IN (1001, 1002, 1003)").Scan(&ordersInBrowse); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM public.order_lines").Scan(&linesInRead); err != nil {
		t.Fatal(err)
	}
	if ordersInBrowse != 3 || linesInRead != 5 {
		t.Fatalf("step 0's actual shell committed %d parent orders and %d detail rows, want 3 and 5",
			ordersInBrowse, linesInRead)
	}

	// Step 1: execute the published mktemp/cp commands. Capture only the
	// scratch path; later steps use this exact Workspace, not a Go copy.
	receipt := filepath.Join(attempt, "guide-workspace.path")
	copyScript := documentedGuideShellStep(t, handbook,
		"## 1. 作業領域の写しを作る", "## 2. 明細を読み、規則の答えを見る")
	copyScript += "\nprintf '%s\\n' \"$TEI_FIRST_WORKSPACE\" > \"$TEI_GUIDE_RECEIPT\"\n"
	copyCmd := exec.Command("sh", "-eu", "-c", copyScript)
	copyCmd.Dir = repoRootForGuide(t)
	copyCmd.Env = append(os.Environ(), "TMPDIR="+attempt, "TEI_GUIDE_RECEIPT="+receipt)
	if output, err := copyCmd.CombinedOutput(); err != nil {
		t.Fatalf("step 1's literal shell copy failed: %v\n%s", err, output)
	}
	workspaceReceipt, err := os.ReadFile(receipt)
	if err != nil {
		t.Fatal(err)
	}
	root := strings.TrimSpace(string(workspaceReceipt))
	if !strings.HasPrefix(root, attempt+string(os.PathSeparator)) {
		t.Fatalf("the guide copy did not make a fresh isolated Workspace: %q", root)
	}

	// Step 2: run the exact published CLI command against this Workspace.
	viewScript := documentedGuideShellStep(t, handbook,
		"## 2. 明細を読み、規則の答えを見る", "## 3. 規則を一つ足す")
	before := readOrderThroughDocumentedShell(t, attempt, root, dsn, viewScript)
	requireCLIAnswersMatchExpectation(t, before, expectedAnswersForOrder(t, filepath.Join(fixtures, "expected-before.json"), "1001"))

	// Step 3: run the actual handbook commands in the same Workspace the
	// preceding CLI read. Check the copied Rule against its independent fixture.
	editScript := documentedGuideShellStep(t, handbook,
		"## 3. 規則を一つ足す", "## 4. 確かめて、読み直す")
	edit := exec.Command("sh", "-eu", "-c", editScript)
	edit.Dir = repoRootForGuide(t)
	edit.Env = append(os.Environ(), "TEI_FIRST_WORKSPACE="+root)
	if output, err := edit.CombinedOutput(); err != nil {
		t.Fatalf("step 3's documented shell edits failed: %v\n%s", err, output)
	}
	wantRule, err := os.ReadFile(filepath.Join(fixtures, "edit", "unit-price.tei.md"))
	if err != nil {
		t.Fatal(err)
	}
	gotRule, err := os.ReadFile(filepath.Join(root, "semantic", "sales", "unit-price.tei.md"))
	if err != nil || !bytes.Equal(gotRule, wantRule) {
		t.Fatal("step 3 did not place the exact documented rule and example")
	}
	application, err := os.ReadFile(filepath.Join(root, "applications", "order-entry", "application.tei"))
	if err != nil || strings.Count(string(application), "uses validation sales.order_line.unit_price_positive") != 1 {
		t.Fatal("step 3 did not register the rule exactly once in the Application")
	}
	workspaceSettings, err := os.ReadFile(filepath.Join(root, "tei.workspace.yaml"))
	if err != nil || !strings.Contains(string(workspaceSettings), "  - tei.authoring/v1") {
		t.Fatal("step 3 did not register the authoring language")
	}

	// Step 4a: run the published validate shell, without a test-only CLI alias.
	validateScript := documentedGuideShellStep(t, handbook,
		"## 4. 確かめて、読み直す", "## 5. Webの画面で同じ答えを見る")
	validate := exec.Command("sh", "-eu", "-c", validateScript)
	validate.Dir = attempt
	validate.Env = append(os.Environ(), "TEI_FIRST_WORKSPACE="+root)
	if _, err := validate.CombinedOutput(); err != nil {
		t.Fatalf("step 4: documented validation rejected the edited Workspace: %v (output withheld)", err)
	}
	// The guide says an example that contradicts its rule is a diagnostic. If
	// that were not so, the validate above would prove nothing about examples.
	contradicted := contradictTheRuleExample(t, root)
	if validated := runGuideCLI(t, binary, "validate", "-workspace", contradicted); validated.code == 0 {
		t.Fatal("step 4: an example contradicting its rule passed validation")
	} else if !strings.Contains(validated.stdout+validated.stderr, "RULE_EXAMPLE_MISMATCH") {
		t.Fatalf("step 4: the example mismatch was not the diagnostic: %s", validated.stdout+validated.stderr)
	}

	// Step 4b: repeat step 2 literally, as the guide instructs.
	after := readOrderThroughDocumentedShell(t, attempt, root, dsn, viewScript)
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

	// Step 5: execute the handbook's literal go-run shell block, against
	// the same isolated PostgreSQL database and edited Workspace as steps 0-4.
	// Do not mistake a separately built binary for the published entry point.
	browser := os.Getenv("TEI_VIEW_BROWSER_BINARY")
	if browser == "" {
		t.Skip("TEI_VIEW_BROWSER_BINARY is unset; step 5's browser receipt not run")
	}
	const address = "127.0.0.1:8080"
	available, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatalf("step 5: the published Web port is occupied: %v", err)
	}
	if err := available.Close(); err != nil {
		t.Fatal("step 5: cannot release the published Web port")
	}
	webScript := documentedGuideShellStep(t, handbook,
		"## 5. Webの画面で同じ答えを見る", "## まだできないこと")
	if !strings.Contains(webScript, "export TEI_WEB_READ_PG_DSN=\"$TEI_VIEW_PG_DSN\"") ||
		!strings.Contains(webScript, "go run ./tools/reference-first-web-target") {
		t.Fatal("step 5 must contain the published DSN export and go-run entry")
	}
	// The POSIX subprocess helper kills the whole process group, including the
	// compiled child of go run, so the server cannot survive the test.
	webDone := startDocumentedGuideWeb(t, webScript, repoRootForGuide(t),
		append(os.Environ(), "TEI_VIEW_PG_DSN="+dsn, "TEI_FIRST_WORKSPACE="+root))
	waitForHealth(t, address, webDone)

	// The published step 5 chooses row 0 (order 1001), then row 1
	// (order 1002). Both selections must reach the SAME shipped Web process,
	// edited Source and PostgreSQL database. This still automates POST using
	// the existing test bootstrap; actual shipped-form field entry remains open.
	for _, choice := range []struct {
		row            int
		orderID        string
		relationAnswer string
	}{
		{row: 0, orderID: "1001", relationAnswer: "not-satisfied"},
		{row: 1, orderID: "1002", relationAnswer: "satisfied"},
	} {
		t.Run("selected-order-"+choice.orderID, func(t *testing.T) {
			dom := chromeDOMOfSelectedOrder(t, browser, address, choice.row)
			if strings.Contains(dom, dsn) {
				t.Fatal("step 5: the Chrome page carried the database connection string")
			}
			pageLines := sectionBetween(dom, `id="order-lines"`, `id="read-identity"`)
			if pageLines == "" {
				t.Fatal("step 5: the page did not render the order-line read")
			}
			// The independently authored expected-after.json has disjoint row
			// keys for orders 1001 and 1002. Comparing by primary key rejects
			// a wrong browse selection even if both pages have similar answers.
			requirePageAnswersMatchExpectation(t, pageLines,
				expectedAnswersForOrder(t, filepath.Join(fixtures, "expected-after.json"), choice.orderID))
			relation := "sales.order.lines.line_number_unique " + choice.relationAnswer
			if !strings.Contains(dom, relation) {
				t.Fatalf("step 5: order %s does not show the independently expected Relation answer %s",
					choice.orderID, choice.relationAnswer)
			}
		})
	}
	// Keep the established CI receipt prefix. Step 5 runs the published go-run
	// command; Chrome still uses test bootstrap POST, not shipped-form entry.
	t.Log("first-change guide steps 0-5 verified end to end against real PostgreSQL and Chrome: steps 0-5 published shell including actual go run; bootstrap POST for orders 1001 and 1002 (not shipped-form input or human completion)")
}

// documentedGuideShellStep returns the commands from the published POSIX
// handbook, without replacing them with equivalent Go test helpers.
func documentedGuideShellStep(t *testing.T, handbook, from, to string) string {
	t.Helper()
	section := sectionBetween(handbook, from, to)
	fence := strings.Repeat(string(rune(96)), 3)
	start := strings.Index(section, fence+"bash\n")
	if start < 0 {
		t.Fatalf("documented section %q has no bash block", from)
	}
	body := section[start+len(fence+"bash\n"):]
	end := strings.Index(body, fence)
	if end < 0 {
		t.Fatalf("documented section %q has an unterminated bash block", from)
	}
	return body[:end]
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
	return decodeGuideOrderRead(t, got.stdout, dsn)
}

// readOrderThroughDocumentedShell executes the published step-2 shell block
// against the exact Workspace copied by step 1, before and after the edit.
func readOrderThroughDocumentedShell(t *testing.T, attempt, workspaceRoot, dsn, script string) cliOrderRead {
	t.Helper()
	command := exec.Command("sh", "-eu", "-c", script)
	command.Dir = attempt
	command.Env = append(os.Environ(),
		"TEI_FIRST_WORKSPACE="+workspaceRoot, "TEI_VIEW_PG_DSN="+dsn)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("step 2/4 documented view command failed: %v (output withheld)", err)
	}
	return decodeGuideOrderRead(t, stdout.String(), dsn)
}

// decodeGuideOrderRead checks the returned rows and rule answers; both the
// documented shell path and the auxiliary CLI path use the same parser.
func decodeGuideOrderRead(t *testing.T, payload, dsn string) cliOrderRead {
	t.Helper()
	if strings.Contains(payload, dsn) {
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
	if err := json.Unmarshal([]byte(payload), &report); err != nil {
		t.Fatalf("the CLI output is not the documented JSON: %v", err)
	}
	if !report.OK {
		t.Fatal("the documented view reported failure")
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
	// statements. Do not cascade into unrelated dependent fixtures.
	for _, table := range []string{"public.order_lines", "public.orders"} {
		if _, err := db.Exec("drop table if exists " + table); err != nil {
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

// TestFirstChangeGuideExecutesLiteralRuleEdit runs the actual POSIX shell
// blocks for the Workspace copy and the Rule edit in docs/first-change.md.
// Existing E2E tests cover the PostgreSQL read and Chrome; this specifically
// checks that the documented commands work without replacing them with a Go
// helper or hard-coding a previously edited Workspace directory.
func TestFirstChangeGuideExecutesLiteralRuleEdit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the handbook requires a POSIX shell, mktemp and awk")
	}
	guide, err := os.ReadFile(filepath.Join(repoRootForGuide(t), "docs", "first-change.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(guide)
	step0 := sectionBetween(text, "## 0. 表を作り、明細を入れる", "## 1. 作業領域の写しを作る")
	if !strings.Contains(step0, "\n./tei desired-schema -workspace ") {
		t.Fatal("the handbook did not invoke the locally built ./tei binary")
	}

	block := func(startHeading, endHeading string) string {
		t.Helper()
		section := sectionBetween(text, startHeading, endHeading)
		const fence = "```bash\n"
		start := strings.Index(section, fence)
		if start < 0 {
			t.Fatalf("no shell block under %s", startHeading)
		}
		body := section[start+len(fence):]
		end := strings.Index(body, "```")
		if end < 0 {
			t.Fatalf("unterminated shell block under %s", startHeading)
		}
		return body[:end]
	}

	copyScript := block("## 1. 作業領域の写しを作る", "## 2. 明細を読み、規則の答えを見る")
	if !strings.Contains(copyScript, "mktemp -d") ||
		!strings.Contains(copyScript, "cp -R reference/workspace/.") ||
		strings.Contains(copyScript, "/tmp/my-workspace") {
		t.Fatal("step 1 does not allocate and populate a fresh Workspace")
	}
	sandbox := t.TempDir()
	receipt := filepath.Join(sandbox, "workspace.path")
	copyWorkspace := func() string {
		t.Helper()
		// The receipt merely reports the shell-created directory to the test.
		// The commands from the handbook are not changed or replaced.
		script := copyScript + "\nprintf '%s\\n' \"$TEI_FIRST_WORKSPACE\" > \"$TEI_GUIDE_RECEIPT\"\n"
		cmd := exec.Command("sh", "-eu", "-c", script)
		cmd.Dir = repoRootForGuide(t)
		cmd.Env = append(os.Environ(), "TMPDIR="+sandbox, "TEI_GUIDE_RECEIPT="+receipt)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("step 1 failed in the actual POSIX shell: %v\n%s", err, output)
		}
		output, err := os.ReadFile(receipt)
		if err != nil {
			t.Fatal(err)
		}
		root := strings.TrimSpace(string(output))
		if !strings.HasPrefix(root, sandbox+string(os.PathSeparator)) {
			t.Fatalf("step 1 copied outside the isolated test directory: %q", root)
		}
		return root
	}
	previous := copyWorkspace()
	root := copyWorkspace()
	if previous == root {
		t.Fatal("step 1 reused a Workspace from the previous attempt")
	}

	editScript := block("## 3. 規則を一つ足す", "## 4. 確かめて、読み直す")
	if !strings.Contains(editScript, "$TEI_FIRST_WORKSPACE") || strings.Contains(editScript, "/tmp/my-workspace") {
		t.Fatal("step 3 did not use the Workspace created in step 1")
	}
	edit := exec.Command("sh", "-eu", "-c", editScript)
	edit.Dir = repoRootForGuide(t)
	edit.Env = append(os.Environ(), "TEI_FIRST_WORKSPACE="+root)
	if output, err := edit.CombinedOutput(); err != nil {
		t.Fatalf("step 3 failed in the actual POSIX shell: %v\n%s", err, output)
	}

	settings, err := os.ReadFile(filepath.Join(root, "tei.workspace.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(settings), "  - tei.authoring/v1") {
		t.Fatal("the edit did not register tei.authoring/v1")
	}
	rule, err := os.ReadFile(filepath.Join(root, "semantic", "sales", "unit-price.tei.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rule), "rule sales.order_line.unit_price_positive:") ||
		!strings.Contains(string(rule), "| 0 | not-satisfied |") {
		t.Fatal("the edit did not place the rule together with its example")
	}
	application, err := os.ReadFile(filepath.Join(root, "applications", "order-entry", "application.tei"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(application), "uses validation sales.order_line.unit_price_positive") != 1 {
		t.Fatal("the awk insertion did not register the new rule exactly once")
	}
	if _, err := os.Stat(filepath.Join(root, "applications", "order-entry", "application.tei.next")); !os.IsNotExist(err) {
		t.Fatalf("the mv command left an intermediate source behind: %v", err)
	}
	if _, err := os.Stat(filepath.Join(previous, "semantic", "sales", "unit-price.tei.md")); !os.IsNotExist(err) {
		t.Fatalf("the previous unedited Workspace was contaminated: %v", err)
	}
	binary := buildCLIForGuide(t)
	if validated := runGuideCLI(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the actual shell edit did not validate: code=%d stdout=%s stderr=%s",
			validated.code, validated.stdout, validated.stderr)
	}
}

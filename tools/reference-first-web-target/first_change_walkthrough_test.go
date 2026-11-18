//go:build !tei_order_only && !tei_repair_only

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The human walkthrough must display the Rule added in step 3. Reading the
// untouched reference Workspace in step 5 used to show the old answers even
// though both commands were individually valid. Keep the instructions aligned
// with the same edited Workspace and PostgreSQL-backed Web entry as the
// before/after HTTP + Chrome acceptance test.
func TestFirstChangeWalkthroughWebUsesEditedWorkspace(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "first-change.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)
	if !strings.Contains(doc, "cp -r reference/workspace /tmp/my-workspace") ||
		!strings.Contains(doc, "./tei validate -workspace /tmp/my-workspace") {
		t.Fatal("the walkthrough no longer creates and validates the edited Workspace")
	}
	parts := strings.SplitN(doc, "## 5. Webの画面で同じ答えを見る", 2)
	if len(parts) != 2 {
		t.Fatal("Web walkthrough step 5 was not found")
	}
	webStep := strings.SplitN(parts[1], "## まだできないこと", 2)[0]
	for _, required := range []string{
		"export TEI_WEB_READ_PG_DSN=",
		"-serve-read-from-db",
		"-diagnostic-unaccepted-source",
		"-workspace /tmp/my-workspace -application order_entry",
		"sales.order_line.unit_price_positive",
		"sales.order.lines.line_number_unique",
		"`not-satisfied`",
		"`satisfied`",
	} {
		if !strings.Contains(webStep, required) {
			t.Errorf("the Web walkthrough omitted %q", required)
		}
	}
	if strings.Contains(webStep, "-workspace reference/workspace") {
		t.Fatal("step 5 would read the unchanged Workspace instead of the added Rule")
	}
	if strings.Contains(webStep, "-read-pg-dsn") {
		t.Fatal("step 5 documents a removed DB connection flag")
	}
}

// TestFirstChangeWalkthroughUsesTheLocallyBuiltCLI checks the literal command
// boundary that the five-step Go acceptance does not cover. A POSIX shell
// normally does not search its current directory for a bare "tei" command;
// the guide builds ./tei and must execute that file when creating the schema.
func TestFirstChangeWalkthroughUsesTheLocallyBuiltCLI(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "first-change.md"))
	if err != nil {
		t.Fatal(err)
	}
	guide := string(raw)
	if !strings.Contains(guide, "go build -o tei ./cmd/tei") {
		t.Fatal("the walkthrough no longer builds its local CLI")
	}
	after, ok := strings.Cut(guide, "## 0. 表を作り、明細を入れる")
	if !ok {
		t.Fatal("the schema setup step is absent")
	}
	step0, ok := strings.Cut(after, "## 1. 作業領域の写しを作る")
	if !ok {
		t.Fatal("the schema setup step has no boundary")
	}
	if !strings.Contains(step0, "./tei desired-schema -workspace reference/workspace") {
		t.Fatal("the DDL step does not invoke the CLI built by this walkthrough")
	}
	if strings.Contains(step0, "\ntei desired-schema ") {
		t.Fatal("the DDL step relies on the current directory being on PATH")
	}
	if !strings.Contains(step0, `psql "$TEI_VIEW_PG_DSN" -v ON_ERROR_STOP=1 -f testdata/destination-order-change/seed.sql`) {
		t.Fatal("the seed step can report success after a SQL error")
	}
}

// The browser form requires all four inputs. The Go-driven Chrome acceptance
// submits them programmatically, so it cannot reveal a missing human step.
func TestFirstChangeGuideExplainsRequiredFormInputs(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "first-change.md"))
	if err != nil {
		t.Fatal(err)
	}
	after, ok := strings.Cut(string(raw), "## 5. Webの画面で同じ答えを見る")
	if !ok {
		t.Fatal("the browser step is absent")
	}
	step5, _, _ := strings.Cut(after, "## まだできないこと")
	for _, required := range []string{
		"`Login ID` に`legacy-order-editor`",
		"`Credential` に`editor-secret`",
		"`Browse from` に`2026-09-01T00:00:00Z`",
		"`Selected row` に`0`",
		"参照実装のテスト専用",
	} {
		if !strings.Contains(step5, required) {
			t.Errorf("the guide does not explain the required reference form input %q", required)
		}
	}
}

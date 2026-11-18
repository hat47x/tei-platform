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
	// A fresh Workspace must be created for each attempt: copying into an
	// existing fixed directory silently nests the source and can make a later
	// test read a previously edited Rule rather than the selected Source.
	for _, required := range []string{
		`export TEI_FIRST_WORKSPACE="$(mktemp -d)"`,
		`cp -R reference/workspace/. "$TEI_FIRST_WORKSPACE/"`,
		`./tei view -workspace "$TEI_FIRST_WORKSPACE" -application order_entry`,
		`>> "$TEI_FIRST_WORKSPACE/tei.workspace.yaml"`,
		`"$TEI_FIRST_WORKSPACE/semantic/sales/"`,
		`"$TEI_FIRST_WORKSPACE/applications/order-entry/application.tei"`,
		`./tei validate -workspace "$TEI_FIRST_WORKSPACE"`,
	} {
		if !strings.Contains(doc, required) {
			t.Errorf("the walkthrough lost the same fresh Workspace at %q", required)
		}
	}
	if strings.Contains(doc, "/tmp/my-workspace") {
		t.Fatal("the walkthrough reused a fixed path that can contain a previous attempt")
	}
	// The guide builds ./tei, not an executable installed in the user's PATH.
	// A bare 'tei desired-schema' used to fail for a first-time reader even
	// when the Go integration test (which calls the binary by full path) passed.
	setup := strings.SplitN(doc, "## 1. 作業領域の写しを作る", 2)[0]
	if !strings.Contains(setup, "go build -o tei ./cmd/tei") ||
		!strings.Contains(setup, "リポジトリのルートディレクトリ") ||
		!strings.Contains(setup, "./tei desired-schema -workspace reference/workspace") ||
		strings.Contains(setup, "\ntei desired-schema ") {
		t.Fatal("step 0 did not use the locally built ./tei executable")
	}
	// psql normally continues after an error in a SQL file unless
	// ON_ERROR_STOP is set. Accept either a standalone fail-fast seed
	// invocation or the stricter single transaction that applies both the
	// generated schema and seed together. The latter also prevents a failed
	// generator from being hidden by the final process of a shell pipeline.
	separateSeed := strings.Contains(setup, `psql "$TEI_VIEW_PG_DSN" -v ON_ERROR_STOP=1 -f testdata/destination-order-change/seed.sql`)
	atomicSeed := strings.Contains(setup, `psql "$TEI_VIEW_PG_DSN" -v ON_ERROR_STOP=1 -1`) &&
		strings.Contains(setup, `-f "$schema_sql" -f testdata/destination-order-change/seed.sql`)
	if !separateSeed && !atomicSeed {
		t.Fatal("step 0 can continue after a seed SQL failure instead of refusing the walkthrough")
	}
	parts := strings.SplitN(doc, "## 5. Webの画面で同じ答えを見る", 2)
	if len(parts) != 2 {
		t.Fatal("Web walkthrough step 5 was not found")
	}
	webStep := strings.SplitN(parts[1], "## まだできないこと", 2)[0]
	for _, required := range []string{
		"export TEI_WEB_READ_PG_DSN=",
		`-workspace "$TEI_FIRST_WORKSPACE" -application order_entry`,
		"-serve-read-from-db",
		"-diagnostic-unaccepted-source",
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
	for _, required := range []string{
		"`Login ID`", "`legacy-order-editor`",
		"`Credential`", "`editor-secret`",
		"`Browse from`", "`2026-09-01T00:00:00Z`",
		"`Selected row`", "`0`",
		"本番用の認証情報ではない",
	} {
		if !strings.Contains(webStep, required) {
			t.Errorf("the browser instructions omit %q", required)
		}
	}
}

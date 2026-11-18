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

//go:build !tei_order_only && !tei_repair_only

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The end-to-end test exercises a built CLI and a browser, but not the shell
// commands and form instructions literally printed in the first-use guide.
// Guard the two prerequisites a reader needs before that E2E can be repeated.
func TestFirstChangeGuideNamesTheBuiltCLIAndRequiredBrowserFields(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "first-change.md"))
	if err != nil {
		t.Fatal(err)
	}
	guide := string(raw)

	if !strings.Contains(guide, "go build -o tei ./cmd/tei") {
		t.Fatal("the guide no longer builds the local tei executable")
	}
	stepZero := strings.SplitN(guide, "## 1. 作業領域", 2)[0]
	if !strings.Contains(stepZero, "./tei desired-schema -workspace reference/workspace") {
		t.Fatal("step 0 must invoke the local executable that the guide just built")
	}
	if strings.Contains(stepZero, "\ntei desired-schema") {
		t.Fatal("step 0 incorrectly assumes that the local executable is already on PATH")
	}

	formStep := strings.SplitN(guide, "## 5. Webの画面", 2)
	if len(formStep) != 2 {
		t.Fatal("the guide has no browser step")
	}
	// These are the four required fields in custom_ui.go, with the values used
	// by the real-browser E2E. They are local synthetic fixture credentials.
	for _, required := range []string{
		"`Login ID`", "`legacy-order-editor`",
		"`Credential`", "`editor-secret`",
		"`Browse from`", "`2026-09-01T00:00:00Z`",
		"`Selected row`", "`0`",
	} {
		if !strings.Contains(formStep[1], required) {
			t.Errorf("the browser step omits %s", required)
		}
	}
	if !strings.Contains(formStep[1], "本番用の認証情報ではない") {
		t.Error("the guide does not distinguish synthetic fixture credentials from production credentials")
	}
}

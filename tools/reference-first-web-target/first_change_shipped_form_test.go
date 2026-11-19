//go:build !tei_order_only && !tei_repair_only

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// verifyFirstChangeShippedForm uses the product's GET /custom-submit form and
// its real input fields, not the historical hidden-input POST bootstrap. It
// reuses the same PostgreSQL-backed Web process and edited Source as the guide.
func verifyFirstChangeShippedForm(t *testing.T, browser, address, fixtures string) {
	t.Helper()
	playwrightRoot := os.Getenv("TEI_FIRST_CHANGE_PLAYWRIGHT_ROOT")
	if playwrightRoot == "" {
		t.Fatal("real product-form Chrome test requires TEI_FIRST_CHANGE_PLAYWRIGHT_ROOT")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal("the real product-form test requires Node")
	}
	browserPath, err := exec.LookPath(browser)
	if err != nil {
		t.Fatal("the configured Chrome binary is unavailable")
	}
	script := filepath.Join(repoRootForGuide(t), "tools", "reference-first-web-target",
		"first_change_shipped_form_browser.mjs")
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, script,
		"http://"+address+"/custom-submit", playwrightRoot)
	cmd.Dir = repoRootForGuide(t)
	cmd.Env = append(os.Environ(), "TEI_VIEW_BROWSER_BINARY="+browserPath)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = io.Discard // Do not expose database or browser input in CI logs.
	if err := cmd.Run(); err != nil {
		t.Fatalf("the real Chrome product-form entry did not complete: %v (output withheld)", err)
	}
	var receipt struct {
		Transport string `json:"transport"`
		Cases     []struct {
			Row        int    `json:"row"`
			GetStatus  int    `json:"getStatus"`
			PostStatus int    `json:"postStatus"`
			Posts      int    `json:"posts"`
			HTML       string `json:"html"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &receipt); err != nil {
		t.Fatal("the real product form returned an invalid browser observation")
	}
	if receipt.Transport != "product-get-form-playwright" || len(receipt.Cases) != 2 {
		t.Fatal("the browser did not report two independently submitted real form views")
	}
	seen := map[int]bool{}
	for _, observation := range receipt.Cases {
		if (observation.Row != 0 && observation.Row != 1) || seen[observation.Row] {
			t.Fatal("the browser repeated or changed a selected form row")
		}
		seen[observation.Row] = true
		if observation.GetStatus != 200 || observation.PostStatus != 200 || observation.Posts != 1 {
			t.Fatal("the product GET form and its single POST did not complete")
		}
		orderID, relationAnswer := "1001", "not-satisfied"
		if observation.Row == 1 {
			orderID, relationAnswer = "1002", "satisfied"
		}
		if strings.Contains(observation.HTML, "editor-secret") ||
			!strings.Contains(observation.HTML, `id="read-result"`) {
			t.Fatal("the returned product page leaked a credential or omitted the real result")
		}
		rows := sectionBetween(observation.HTML, `id="order-lines"`, `id="read-identity"`)
		if rows == "" {
			t.Fatalf("the actual form for order %s did not render its read", orderID)
		}
		requirePageAnswersMatchExpectation(t, rows,
			expectedAnswersForOrder(t, filepath.Join(fixtures, "expected-after.json"), orderID))
		if !strings.Contains(observation.HTML, "sales.order.lines.line_number_unique "+relationAnswer) {
			t.Fatalf("the actual form for order %s did not show its expected relation answer", orderID)
		}
	}
	t.Log("real Chrome product GET form fields and Submit -> one HTTP POST each -> PostgreSQL row-key Expected for orders 1001 and 1002 (not human first-use acceptance)")
}

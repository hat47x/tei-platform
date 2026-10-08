package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/workspace"
)

// This test owns only a scratch copy. It does not accept a business decision
// or change the checked-in Reference workspace.
func workspaceWithUnitPriceRule(t *testing.T) string {
	t.Helper()
	scratch := filepath.Join(t.TempDir(), "workspace")
	if err := os.CopyFS(scratch, os.DirFS(referenceWorkspacePath(t))); err != nil {
		t.Fatal(err)
	}
	appendSource := func(rel string, change func(string) (string, bool)) {
		t.Helper()
		path := filepath.Join(scratch, filepath.FromSlash(rel))
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		updated, ok := change(string(original))
		if !ok {
			t.Fatalf("scratch fixture source %s does not have the expected shape", rel)
		}
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	appendSource("semantic/sales/validation.tei", func(s string) (string, bool) {
		if strings.Contains(s, "sales.order_line.unit_price_positive") {
			return "", false
		}
		return strings.TrimRight(s, "\r\n") +
			"\n\nvalidation sales.order_line.unit_price_positive positive {\n" +
			"    attribute sales.order_line.unit_price\n}\n", true
	})
	appendSource("applications/order-entry/application.tei", func(s string) (string, bool) {
		const anchor = "    uses validation sales.order.lines.line_number_unique\n"
		if !strings.Contains(s, anchor) ||
			strings.Contains(s, "uses validation sales.order_line.unit_price_positive") {
			return "", false
		}
		return strings.Replace(s, anchor, anchor+
			"    uses validation sales.order_line.unit_price_positive\n", 1), true
	})
	return scratch
}

func TestPracticalBusinessRuleChangeReachesDriverAndWebWithoutTargetRuleList(t *testing.T) {
	base, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	read, err := newReferenceReadScenarioRunner(base.World).Run(
		context.Background(),
		readScenarioInput{From: "2026-09-18T08:30:00Z", RowIndex: 0},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(read.OrderLineRows) == 0 {
		t.Fatal("order line fixture is empty")
	}
	// Two existing rules are satisfied; a zero price exercises only the new rule.
	if !setSemanticRowValue(&read.OrderLineRows[0], "sales.order_line.unit_price", "0") {
		t.Fatal("unit price not present in read result")
	}
	input := commandScenarioInput{
		LoginID: "legacy-order-editor", Credential: "editor-secret",
		From: "2026-09-18T08:30:00Z", RowIndex: 0,
	}

	before, err := newReferenceCommandScenarioRunner(base.World,
		legacyFixturePath(t, "users.json"), legacyFixturePath(t, "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	before.read = staticReadScenarioRunner{result: read}
	prior, err := before.RunCommand(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if prior.Stage != "completed" || len(prior.Validations) != 2 {
		t.Fatalf("before change stage=%s validations=%+v", prior.Stage, prior.Validations)
	}

	changed, err := workspace.Load(workspaceWithUnitPriceRule(t))
	if err != nil {
		t.Fatal(err)
	}
	after, err := newReferenceCommandScenarioRunner(changed.World,
		legacyFixturePath(t, "users.json"), legacyFixturePath(t, "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	after.read = staticReadScenarioRunner{result: read}
	rejected, err := after.RunCommand(context.Background(), input)
	if err != nil {
		t.Fatalf("business validation failure became execution error: %v", err)
	}
	if rejected.Stage != "validation-not-satisfied" ||
		len(rejected.Validations) != 3 ||
		rejected.Validations[2].Validation != "sales.order_line.unit_price_positive" ||
		rejected.Validations[2].Result != "not-satisfied" {
		t.Fatalf("new rule not evaluated via Canonical selection: %+v", rejected)
	}
	if rejected.Authorization != nil || rejected.Command != nil || rejected.Observation != nil {
		t.Fatalf("rejected command must not have downstream effects: %+v", rejected)
	}

	// The HTTP endpoint uses the same runner: Web may not keep a second rule list.
	projection, err := buildWebTargetProjection(changed.World, projectionConfig{
		HandoverBaselineID: testBaselineID,
		BaselineRepositoryRevision: testBaselineRevision,
		TargetBuildRevision: testTargetRevision,
		ApplicationID: core.MustCanonicalID("order_entry"),
	})
	if err != nil {
		t.Fatal(err)
	}
	server, err := newWebTargetServer(projection,
		newReferenceReadScenarioRunner(changed.World), after)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/scenario/submit", strings.NewReader(
		`{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:30:00Z","row":0}`,
	))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response webCommandScenarioResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Result.Stage != "validation-not-satisfied" ||
		len(response.Result.Validations) != 3 ||
		response.Result.Command != nil || response.Result.Observation != nil {
		t.Fatalf("Web did not preserve the Driver validation outcome: %+v", response.Result)
	}

	// The accepted rule must not reject a valid amount.
	if !setSemanticRowValue(&read.OrderLineRows[0], "sales.order_line.unit_price", "4000.00") {
		t.Fatal("unit price no longer present")
	}
	after.read = staticReadScenarioRunner{result: read}
	positive, err := after.RunCommand(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if positive.Stage != "completed" || len(positive.Validations) != 3 {
		t.Fatalf("valid price was not accepted: %+v", positive)
	}
}

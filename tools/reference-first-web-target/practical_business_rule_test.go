package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/workspace"
)

// This is a test-only acceptance on a scratch workspace. It uses the real
// Shared Acceptance CLI, but does NOT represent a human decision to adopt
// this rule in the checked-in Reference or production.
func workspaceWithUnitPriceRule(t *testing.T) string {
	t.Helper()
	scratch := filepath.Join(t.TempDir(), "workspace")
	if err := os.CopyFS(scratch, os.DirFS(referenceWorkspacePath(t))); err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(referenceWorkspacePath(t), "..", ".."))
	runAcceptance := func(args ...string) []byte {
		t.Helper()
		cmd := exec.Command("go", append([]string{"run", "./tools/reference-shared-acceptance"}, args...)...)
		cmd.Dir = repoRoot
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Shared Acceptance CLI args=%v error=%v output=%s", args, err, output)
		}
		return output
	}
	var revision struct {
		ModelRevision string `json:"modelRevision"`
	}
	if err := json.Unmarshal(runAcceptance("revision", "-workspace", scratch), &revision); err != nil {
		t.Fatal(err)
	}
	if revision.ModelRevision == "" {
		t.Fatal("Shared Acceptance CLI returned empty model revision")
	}
	candidate := map[string]any{
		"schema":            "tei.reference.change-candidate/v0",
		"id":                "cand-practical-business-rule-test",
		"baseModelRevision": revision.ModelRevision,
		"origin":            map[string]string{"kind": "cli", "actor": "test"},
		"intent":            "受注明細の単価制約と利用宣言を一つの変更として試す",
		"evidence": []map[string]string{
			{"source": "semantic/sales/validation.tei"},
			{"source": "applications/order-entry/application.tei"},
		},
		"delta": []map[string]any{
			{
				"op":        "add-validation",
				"namespace": "sales",
				"id":        "sales.order_line.unit_price_positive",
				"kind":      "positive",
				"attribute": "sales.order_line.unit_price",
			},
			{
				"op":        "add-application-validation-use",
				"placement": "applications/order-entry/application.tei",
				"applicationValidationUse": map[string]string{
					"application": "order_entry",
					"validation":  "sales.order_line.unit_price_positive",
				},
			},
		},
	}
	data, err := json.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	candidatePath := filepath.Join(t.TempDir(), "candidate.json")
	if err := os.WriteFile(candidatePath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	// Preview is read-only; the two authoritative files may change only on Accept.
	validationPath := filepath.Join(scratch, "semantic", "sales", "validation.tei")
	applicationPath := filepath.Join(scratch, "applications", "order-entry", "application.tei")
	beforeValidation, err := os.ReadFile(validationPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeApplication, err := os.ReadFile(applicationPath)
	if err != nil {
		t.Fatal(err)
	}
	var preview struct {
		Acceptable bool `json:"acceptable"`
		SourceEdits []struct {
			Path string `json:"path"`
		} `json:"sourceEdits"`
	}
	if err := json.Unmarshal(runAcceptance("preview", "-workspace", scratch, "-candidate", candidatePath), &preview); err != nil {
		t.Fatal(err)
	}
	if !preview.Acceptable || len(preview.SourceEdits) != 2 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	afterPreviewValidation, err := os.ReadFile(validationPath)
	if err != nil {
		t.Fatal(err)
	}
	afterPreviewApplication, err := os.ReadFile(applicationPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(beforeValidation, afterPreviewValidation) || !bytes.Equal(beforeApplication, afterPreviewApplication) {
		t.Fatal("Preview mutated authoritative source")
	}
	var review struct {
		PreviewAcceptable bool   `json:"previewAcceptable"`
		ReviewFingerprint string `json:"reviewFingerprint"`
	}
	if err := json.Unmarshal(runAcceptance("review", "-workspace", scratch, "-candidate", candidatePath, "-format", "json"), &review); err != nil {
		t.Fatal(err)
	}
	if !review.PreviewAcceptable || review.ReviewFingerprint == "" {
		t.Fatalf("unexpected review: %+v", review)
	}
	var decision struct {
		Decision      string `json:"decision"`
		SourceChanged bool   `json:"sourceChanged"`
		ReviewBinding *struct {
			Fingerprint string `json:"fingerprint"`
		} `json:"reviewBinding"`
	}
	if err := json.Unmarshal(runAcceptance("accept", "-workspace", scratch,
		"-candidate", candidatePath, "-expected-review-fingerprint", review.ReviewFingerprint),
		&decision); err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "accepted" || !decision.SourceChanged ||
		decision.ReviewBinding == nil ||
		decision.ReviewBinding.Fingerprint != review.ReviewFingerprint {
		t.Fatalf("test-only acceptance is not review-bound: %+v", decision)
	}
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
	if len(read.OrderLineRows) < 2 {
		t.Fatal("order line fixture is empty")
	}
	// Two existing rules are satisfied; a zero price in the second line exercises only the new rule.
	if !setSemanticRowValue(&read.OrderLineRows[1], "sales.order_line.unit_price", "0") {
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
	if !setSemanticRowValue(&read.OrderLineRows[1], "sales.order_line.unit_price", "4500.00") {
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

func TestReferenceCommandScenarioEvaluatesPositiveRuleOnEveryOrderLine(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	read, err := newReferenceReadScenarioRunner(loaded.World).Run(
		context.Background(),
		readScenarioInput{From: "2026-09-18T08:30:00Z", RowIndex: 0},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(read.OrderLineRows) < 2 {
		t.Fatalf("fixture has only %d order lines", len(read.OrderLineRows))
	}
	if !setSemanticRowValue(&read.OrderLineRows[1], "sales.order_line.quantity", "0") {
		t.Fatal("second order line has no quantity")
	}
	runner, err := newReferenceCommandScenarioRunner(loaded.World,
		legacyFixturePath(t, "users.json"), legacyFixturePath(t, "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	runner.read = staticReadScenarioRunner{result: read}
	result, err := runner.RunCommand(context.Background(), commandScenarioInput{
		LoginID: "legacy-order-editor", Credential: "editor-secret",
		From: "2026-09-18T08:30:00Z", RowIndex: 0,
	})
	if err != nil {
		t.Fatalf("nonpositive quantity was an execution error: %v", err)
	}
	if result.Stage != "validation-not-satisfied" || len(result.Validations) != 1 ||
		result.Validations[0].Validation != "sales.order_line.quantity_positive" ||
		result.Validations[0].Result != "not-satisfied" {
		t.Fatalf("second line was not validated: %+v", result)
	}
	if result.Command != nil || result.Observation != nil {
		t.Fatalf("invalid second line reached downstream: %+v", result)
	}
}

func TestReferenceCommandScenarioRejectsMissingValuesAcrossAllLines(t *testing.T) {
	dir := workspaceWithUnitPriceRule(t)
	loaded, err := workspace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	read, err := newReferenceReadScenarioRunner(loaded.World).Run(
		context.Background(), readScenarioInput{From: "2026-09-18T08:30:00Z", RowIndex: 0},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(read.OrderLineRows) < 2 {
		t.Fatal("scratch fixture does not have at least two order lines")
	}
	// Remove the new rule's attribute from the second line. Unknown data must
	// be an execution error, never a successful validation or business denial.
	var kept []semanticValueProjection
	for _, attr := range read.OrderLineRows[1].Attributes {
		if attr.Attribute != "sales.order_line.unit_price" {
			kept = append(kept, attr)
		}
	}
	read.OrderLineRows[1].Attributes = kept
	runner, err := newReferenceCommandScenarioRunner(loaded.World,
		legacyFixturePath(t, "users.json"), legacyFixturePath(t, "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	runner.read = staticReadScenarioRunner{result: read}
	result, err := runner.RunCommand(context.Background(), commandScenarioInput{
		LoginID: "legacy-order-editor", Credential: "editor-secret",
		From: "2026-09-18T08:30:00Z", RowIndex: 0,
	})
	if err == nil || !strings.Contains(err.Error(), "order line 1") {
		t.Fatalf("missing second-line price did not stop evaluation: stage=%s error=%v", result.Stage, err)
	}
	if result.Command != nil || result.Observation != nil {
		t.Fatalf("missing price reached downstream side effects: %+v", result)
	}
}

func TestReferenceCommandScenarioFailsClosedForEmptyOrderLineCollection(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	read, err := newReferenceReadScenarioRunner(loaded.World).Run(
		context.Background(), readScenarioInput{From: "2026-09-18T08:30:00Z", RowIndex: 0},
	)
	if err != nil {
		t.Fatal(err)
	}
	read.OrderLineRows = nil
	runner, err := newReferenceCommandScenarioRunner(loaded.World,
		legacyFixturePath(t, "users.json"), legacyFixturePath(t, "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	runner.read = staticReadScenarioRunner{result: read}
	result, err := runner.RunCommand(context.Background(), commandScenarioInput{
		LoginID: "legacy-order-editor", Credential: "editor-secret",
		From: "2026-09-18T08:30:00Z", RowIndex: 0,
	})
	if err == nil || !strings.Contains(err.Error(), "empty order-line collection") {
		t.Fatalf("empty detail did not fail closed: stage=%s error=%v", result.Stage, err)
	}
	if result.Command != nil || result.Observation != nil {
		t.Fatalf("empty detail reached downstream side effects: %+v", result)
	}
}

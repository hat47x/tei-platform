package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReviewPacketClassifiesAmbiguityConflictAndStalePremise(t *testing.T) {
	dir := t.TempDir()
	knowledgePath := writeReviewJSON(t, dir, "knowledge.json", map[string]any{
		"subject": map[string]any{
			"kind": "application-pattern",
			"id":   "order_entry.customer_reference",
		},
		"unresolved": []string{"write-capability-not-established"},
	})
	crossViewPath := writeReviewJSON(t, dir, "cross-view.json", map[string]any{
		"procedure": "order-entry.order-handling",
		"diagnostics": []map[string]any{
			{
				"code":     "BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT",
				"severity": "architecture-policy",
				"step":     "BP-02",
				"message":  "accepted read access conflicts with requested read-write",
			},
		},
	})
	acceptancePath := writeReviewJSON(t, dir, "acceptance.json", map[string]any{
		"decision": "rejected",
		"stale": map[string]any{
			"baseModelRevision":    "sha256:old",
			"currentModelRevision": "sha256:current",
		},
	})

	got, err := buildReviewPacket("ref-spec-01", knowledgePath, crossViewPath, acceptancePath)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "review-needed" ||
		!got.Coverage.KnowledgeContext ||
		!got.Coverage.CrossView ||
		!got.Coverage.SharedAcceptance ||
		len(got.MissingInputs) != 0 {
		t.Fatalf("review packet status/coverage drifted: %+v", got)
	}
	wantClasses := []string{"ambiguity", "inconsistency", "stale-premise"}
	var classes []string
	for _, item := range got.Findings {
		classes = append(classes, item.Class)
	}
	if !reflect.DeepEqual(classes, wantClasses) {
		t.Fatalf("finding classes=%v want=%v findings=%+v", classes, wantClasses, got.Findings)
	}
	if got.Findings[0].Code != "KNOWLEDGE_UNRESOLVED" ||
		got.Findings[1].Code != "BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT" ||
		got.Findings[2].Code != "STALE_BASE_MODEL_REVISION" {
		t.Fatalf("finding codes drifted: %+v", got.Findings)
	}
}

func TestReviewPacketConsumesValidationScopeReviewWithoutOwningItsSemantics(t *testing.T) {
	dir := t.TempDir()
	crossViewPath := writeReviewJSON(t, dir, "cross-view.json", map[string]any{
		"procedure":   "order-entry.order-handling",
		"diagnostics": []any{},
		"validationScopeReview": []map[string]any{
			{
				"step":            "BP-05",
				"status":          "review-needed",
				"reason":          "Application全体の宣言と業務手順の明示参照に差があります",
				"applicationOnly": []string{"sales.order_line.line_number_positive"},
			},
		},
	})

	got, err := buildReviewPacket("validation-scope", "", crossViewPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "review-needed" ||
		got.Coverage.KnowledgeContext ||
		!got.Coverage.CrossView ||
		got.Coverage.SharedAcceptance {
		t.Fatalf("unexpected packet: %+v", got)
	}
	if !reflect.DeepEqual(got.MissingInputs, []string{"knowledge-context", "shared-acceptance"}) {
		t.Fatalf("missing inputs=%v", got.MissingInputs)
	}
	if len(got.Findings) != 1 ||
		got.Findings[0].Class != "cross-view-difference" ||
		got.Findings[0].Code != "VALIDATION_SCOPE_REVIEW_NEEDED" ||
		got.Findings[0].Subject != "BP-05" {
		t.Fatalf("validation scope finding drifted: %+v", got.Findings)
	}
}

func TestReviewPacketTreatsMissingInputsAsCoverageGapNotNegativeEvidence(t *testing.T) {
	dir := t.TempDir()
	knowledgePath := writeReviewJSON(t, dir, "knowledge.json", map[string]any{
		"subject": map[string]any{
			"kind": "application-pattern",
			"id":   "order_entry.customer_reference",
		},
	})

	got, err := buildReviewPacket("partial", knowledgePath, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "partial-no-finding" || len(got.Findings) != 0 {
		t.Fatalf("missing inputs became a finding or complete result: %+v", got)
	}
	if !reflect.DeepEqual(got.MissingInputs, []string{"cross-view", "shared-acceptance"}) {
		t.Fatalf("missing inputs=%v", got.MissingInputs)
	}
	if !containsReviewString(got.AuthorityLimits, "missing-input-is-not-negative-evidence") ||
		!containsReviewString(got.AuthorityLimits, "no-finding-is-not-proof-of-specification-completeness") {
		t.Fatalf("authority limits are incomplete: %+v", got.AuthorityLimits)
	}
}

func TestReviewPacketKeepsUnknownCrossViewDiagnosticSourceOwned(t *testing.T) {
	dir := t.TempDir()
	crossViewPath := writeReviewJSON(t, dir, "cross-view.json", map[string]any{
		"diagnostics": []map[string]any{
			{
				"code":     "BUSINESS_CUSTOM_POLICY_REVIEW",
				"severity": "architecture-policy",
				"step":     "BP-09",
				"message":  "source tool owns this diagnostic meaning",
			},
		},
	})

	got, err := buildReviewPacket("source-owned", "", crossViewPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Findings) != 1 ||
		got.Findings[0].Class != "source-diagnostic" ||
		got.Findings[0].Code != "BUSINESS_CUSTOM_POLICY_REVIEW" {
		t.Fatalf("unknown diagnostic was reinterpreted: %+v", got.Findings)
	}
}

func TestReviewPacketRejectsMissingAllInputsAndTrailingJSON(t *testing.T) {
	if _, err := buildReviewPacket("", "", "", ""); err == nil {
		t.Fatal("empty review input was accepted")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte(`{"subject":{"kind":"semantic-data","id":"sales.order"}} {"extra":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := buildReviewPacket("bad", path, "", ""); err == nil {
		t.Fatal("trailing JSON value was accepted")
	}
}

func TestREFSPEC01CustomerEditUsesExistingKnowledgeAndCheckerOutputs(t *testing.T) {
	root := repositoryRoot(t)
	temp := t.TempDir()

	knowledgeOut := runReviewCommand(
		t,
		root,
		"go", "run", "./cmd/tei", "knowledge-context",
		"-workspace", filepath.Join(root, "reference", "workspace"),
		"-subject-kind", "application-pattern",
		"-subject-id", "order_entry.customer_reference",
	)
	knowledgePath := filepath.Join(temp, "knowledge.json")
	if err := os.WriteFile(knowledgePath, knowledgeOut, 0o644); err != nil {
		t.Fatal(err)
	}

	crossViewOut := runReviewCommand(
		t,
		root,
		"go", "run", "./tools/reference-business-procedure-check",
		"-workspace", filepath.Join(root, "reference", "workspace"),
		"-fixture", filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"),
		"-probe", filepath.Join(root, "reference", "business-system-design", "order-entry-customer-edit-conflict.json"),
		"-expect-diagnostic", "BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT",
	)
	crossViewPath := filepath.Join(temp, "cross-view.json")
	if err := os.WriteFile(crossViewPath, crossViewOut, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := buildReviewPacket("ref-spec-01-customer-edit", knowledgePath, crossViewPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "review-needed" ||
		!got.Coverage.KnowledgeContext ||
		!got.Coverage.CrossView ||
		got.Coverage.SharedAcceptance {
		t.Fatalf("REF-SPEC-01 packet coverage drifted: %+v", got)
	}
	var conflict *finding
	for i := range got.Findings {
		if got.Findings[i].Code == "BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT" {
			conflict = &got.Findings[i]
			break
		}
	}
	if conflict == nil ||
		conflict.Class != "inconsistency" ||
		conflict.Subject != "BP-02" ||
		!strings.Contains(conflict.Message, "master") ||
		!strings.Contains(conflict.Message, "read-write") {
		t.Fatalf("customer edit inconsistency is not reviewable: %+v", got.Findings)
	}
	if !reflect.DeepEqual(got.MissingInputs, []string{"shared-acceptance"}) {
		t.Fatalf("missing acceptance evidence was hidden: %v", got.MissingInputs)
	}
}

func writeReviewJSON(t *testing.T, dir, name string, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func runReviewCommand(t *testing.T, dir, name string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %v failed: %v\\nstderr=%s", name, args, err, stderr.String())
	}
	return stdout.Bytes()
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("repository root is not reachable from test package: %v", err)
	}
	return root
}

func containsReviewString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

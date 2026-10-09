package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
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
		"artifacts": []map[string]any{
			{
				"path":        "applications/order-entry/patterns/customer-reference.tei",
				"fingerprint": "sha256:pattern",
			},
		},
		"references": []map[string]any{
			{
				"subject": map[string]any{
					"kind": "application-pattern",
					"id":   "order_entry.customer_reference",
				},
				"artifactPath": "applications/order-entry/patterns/customer-reference.tei",
				"role":         "definition",
			},
			{
				"subject": map[string]any{
					"kind": "application-pattern",
					"id":   "order_entry.customer_reference",
				},
				"artifactPath": "applications/order-entry/patterns/customer-reference.tei",
				"role":         "definition",
			},
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
		"schema":   "tei.reference.change-decision/v0",
		"decision": "rejected",
		"stale": map[string]any{
			"baseModelRevision":    "sha256:old",
			"currentModelRevision": "sha256:current",
		},
		"evidence": []map[string]any{
			{
				"subjectKind":       "semantic-data",
				"subject":           "sales.order",
				"source":            "semantic/sales/order.tei",
				"sourceFingerprint": "sha256:order",
			},
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
	if len(got.Evidence) != 2 {
		t.Fatalf("review Evidence was not deduplicated or preserved: %+v", got.Evidence)
	}
	if got.Evidence[0].Subject != "application-pattern:order_entry.customer_reference" ||
		got.Evidence[0].Source != "applications/order-entry/patterns/customer-reference.tei" ||
		got.Evidence[0].Fingerprint != "sha256:pattern" ||
		got.Evidence[0].Role != "definition" {
		t.Fatalf("knowledge Evidence drifted: %+v", got.Evidence[0])
	}
	if got.Evidence[1].Subject != "semantic-data:sales.order" ||
		got.Evidence[1].Source != "semantic/sales/order.tei" ||
		got.Evidence[1].Fingerprint != "sha256:order" ||
		got.Evidence[1].Role != "candidate-evidence" {
		t.Fatalf("acceptance Evidence drifted: %+v", got.Evidence[1])
	}
}

func TestReviewPacketKeepsSharedAcceptanceUnresolvedAsAmbiguity(t *testing.T) {
	dir := t.TempDir()
	acceptancePath := writeReviewJSON(t, dir, "acceptance.json", map[string]any{
		"schema":   "tei.reference.change-preview/v0",
		"decision": "preview",
		"unresolved": []map[string]any{
			{
				"code":    "DRIVER_SCENARIO_NOT_RUN",
				"message": "解析はDriver scenarioを実行しない",
			},
		},
	})

	got, err := buildReviewPacket("shared-unresolved", "", "", acceptancePath)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "review-needed" ||
		len(got.Findings) != 1 ||
		got.Findings[0].Class != "ambiguity" ||
		got.Findings[0].Source != "shared-acceptance" ||
		got.Findings[0].Code != "DRIVER_SCENARIO_NOT_RUN" ||
		got.Findings[0].Message != "解析はDriver scenarioを実行しない" {
		t.Fatalf("Shared Acceptance unresolved item drifted: %+v", got)
	}
	if !reflect.DeepEqual(got.MissingInputs, []string{"cross-view", "knowledge-context"}) {
		t.Fatalf("missing inputs=%v", got.MissingInputs)
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
		"procedure": "order-entry.order-handling",
		"diagnostics": []map[string]any{
			{
				"code":     "BUSINESS_APPLICATION_PATTERN_REF_NOT_FOUND",
				"severity": "semantic-error",
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
		got.Findings[0].Code != "BUSINESS_APPLICATION_PATTERN_REF_NOT_FOUND" {
		t.Fatalf("unknown diagnostic was reinterpreted: %+v", got.Findings)
	}
}

func TestReviewPacketRejectsMismatchedInputContracts(t *testing.T) {
	dir := t.TempDir()

	badKnowledge := writeReviewJSON(t, dir, "bad-knowledge.json", map[string]any{
		"subject": map[string]any{"kind": "", "id": ""},
	})
	if _, err := buildReviewPacket("bad-knowledge", badKnowledge, "", ""); err == nil {
		t.Fatal("knowledge input without subject identity was accepted")
	}

	badCrossView := writeReviewJSON(t, dir, "bad-cross-view.json", map[string]any{
		"diagnostics": []any{},
	})
	if _, err := buildReviewPacket("bad-cross-view", "", badCrossView, ""); err == nil {
		t.Fatal("cross-view input without procedure identity was accepted")
	}

	badAcceptance := writeReviewJSON(t, dir, "bad-acceptance.json", map[string]any{
		"schema": "tei.reference.decision-index/v0",
	})
	if _, err := buildReviewPacket("bad-acceptance", "", "", badAcceptance); err == nil {
		t.Fatal("unsupported Shared Acceptance schema was accepted")
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
	applicationKnowledgeOut := runReviewCommand(
		t,
		root,
		"go", "run", "./cmd/tei", "knowledge-context",
		"-workspace", filepath.Join(root, "reference", "workspace"),
		"-subject-kind", "application",
		"-subject-id", "order_entry",
	)
	applicationKnowledgePath := filepath.Join(temp, "application-knowledge.json")
	if err := os.WriteFile(applicationKnowledgePath, applicationKnowledgeOut, 0o644); err != nil {
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

	got, err := buildReviewPacketWithKnowledge(
		"ref-spec-01-customer-edit",
		[]string{knowledgePath, applicationKnowledgePath},
		crossViewPath,
		"",
	)
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
	if !hasReviewEvidence(
		got.Evidence,
		"application-pattern:order_entry.customer_reference",
		"applications/order-entry/patterns/customer-reference.tei",
	) {
		t.Fatalf("customer reference Source Evidence is not reachable from the packet: %+v", got.Evidence)
	}
	if !hasReviewEvidence(
		got.Evidence,
		"application:order_entry",
		"applications/order-entry/application.tei",
	) {
		t.Fatalf("owning Application Source Evidence is not reachable from the packet: %+v", got.Evidence)
	}
}

func TestREFSPEC01MissingDefinitionUsesActualKnowledgeContext(t *testing.T) {
	root := repositoryRoot(t)
	temp := t.TempDir()

	knowledgeOut := runReviewCommand(
		t,
		root,
		"go", "run", "./cmd/tei", "knowledge-context",
		"-workspace", filepath.Join(root, "reference", "workspace"),
		"-subject-kind", "semantic-data",
		"-subject-id", "missing.example",
	)
	knowledgePath := filepath.Join(temp, "missing-knowledge.json")
	if err := os.WriteFile(knowledgePath, knowledgeOut, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := buildReviewPacket("ref-spec-01-missing-definition", knowledgePath, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "review-needed" ||
		!got.Coverage.KnowledgeContext ||
		got.Coverage.CrossView ||
		got.Coverage.SharedAcceptance {
		t.Fatalf("missing-definition packet coverage/status drifted: %+v", got)
	}
	if !reflect.DeepEqual(got.MissingInputs, []string{"cross-view", "shared-acceptance"}) {
		t.Fatalf("missing-definition packet missing inputs=%v", got.MissingInputs)
	}
	var reasons []string
	for _, item := range got.Findings {
		if item.Source != "knowledge-context" ||
			item.Class != "ambiguity" ||
			item.Code != "KNOWLEDGE_UNRESOLVED" ||
			item.Subject != "semantic-data:missing.example" {
			t.Fatalf("missing-definition finding drifted: %+v", item)
		}
		reasons = append(reasons, item.Message)
	}
	sort.Strings(reasons)
	want := []string{
		"semantic-element-not-found-in-derived-index",
		"source-reference-unavailable",
	}
	if !reflect.DeepEqual(reasons, want) {
		t.Fatalf("missing-definition reasons=%v want=%v", reasons, want)
	}
}

func TestREFSPEC01StalePremiseUsesActualSharedAcceptancePreview(t *testing.T) {
	root := repositoryRoot(t)
	temp := t.TempDir()

	baseCandidate, err := os.ReadFile(filepath.Join(root, "plan", "issue", "000075-practical-b", "candidate.json"))
	if err != nil {
		t.Fatal(err)
	}
	var candidate map[string]any
	if err := json.Unmarshal(baseCandidate, &candidate); err != nil {
		t.Fatal(err)
	}
	candidate["baseModelRevision"] = "sha256:" + strings.Repeat("0", 64)
	candidatePath := writeReviewJSON(t, temp, "stale-candidate.json", candidate)

	previewOut, previewErr, code := runReviewCommandWithExit(
		t,
		root,
		"go", "run", "./tools/reference-shared-acceptance", "preview",
		"-workspace", filepath.Join(root, "reference", "workspace"),
		"-candidate", candidatePath,
	)
	if code == 0 {
		t.Fatalf("intentionally stale candidate unexpectedly passed Preview: %s", previewOut)
	}
	if len(previewOut) == 0 {
		t.Fatalf("stale Preview produced no machine-readable output: stderr=%s", previewErr)
	}
	previewPath := filepath.Join(temp, "preview.json")
	if err := os.WriteFile(previewPath, previewOut, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := buildReviewPacket("ref-spec-01-stale", "", "", previewPath)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "review-needed" ||
		got.Coverage.KnowledgeContext ||
		got.Coverage.CrossView ||
		!got.Coverage.SharedAcceptance {
		t.Fatalf("stale packet coverage/status drifted: %+v", got)
	}
	if !reflect.DeepEqual(got.MissingInputs, []string{"cross-view", "knowledge-context"}) {
		t.Fatalf("stale packet missing inputs=%v", got.MissingInputs)
	}
	var staleFinding *finding
	for i := range got.Findings {
		if got.Findings[i].Code == "STALE_BASE_MODEL_REVISION" {
			staleFinding = &got.Findings[i]
			break
		}
	}
	if staleFinding == nil ||
		staleFinding.Class != "stale-premise" ||
		!strings.Contains(staleFinding.Message, strings.Repeat("0", 64)) {
		t.Fatalf("actual Shared Acceptance stale premise was not preserved: %+v", got.Findings)
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

func runReviewCommandWithExit(t *testing.T, dir, name string, args ...string) ([]byte, string, int) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return stdout.Bytes(), stderr.String(), 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return stdout.Bytes(), stderr.String(), exitErr.ExitCode()
	}
	t.Fatalf("%s %v failed before exit status was available: %v", name, args, err)
	return nil, "", -1
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

func hasReviewEvidence(values []evidenceRef, subject, source string) bool {
	for _, value := range values {
		if value.Subject == subject &&
			value.Source == source &&
			value.Fingerprint != "" {
			return true
		}
	}
	return false
}

func containsReviewString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

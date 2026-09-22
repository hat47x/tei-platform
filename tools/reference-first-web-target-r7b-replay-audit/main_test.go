package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type testFixture struct {
	plan          replayPlan
	receipt       replayPlanReceipt
	metadata      runMetadata
	planBytes     []byte
	receiptBytes  []byte
	metadataBytes []byte
	rawDir        string
}

func TestReplayInputPreflightReusesAuditBindingValidation(t *testing.T) {
	f := newTestFixture(t)
	plan, receipt, planSHA, receiptSHA, err := decodeAndValidateReplayInputs(f.planBytes, f.receiptBytes)
	if err != nil {
		t.Fatal(err)
	}
	if planSHA != sha256Hex(f.planBytes) || receiptSHA != sha256Hex(f.receiptBytes) ||
		plan.ReplayTargetBuildRevision != f.plan.ReplayTargetBuildRevision ||
		receipt.ReplayPlanArtifactSHA256 != planSHA {
		t.Fatalf("unexpected replay-input preflight result")
	}

	t.Run("plan-provenance-drift", func(t *testing.T) {
		f := newTestFixture(t)
		f.plan.HandoverBaselineID = ""
		f.planBytes = encodeJSON(t, f.plan)
		if _, _, _, _, err := decodeAndValidateReplayInputs(f.planBytes, f.receiptBytes); err == nil {
			t.Fatal("accepted replay plan with missing launch provenance")
		}
	})
	t.Run("receipt-plan-digest-drift", func(t *testing.T) {
		f := newTestFixture(t)
		f.receipt.ReplayPlanArtifactSHA256 = strings.Repeat("f", 64)
		f.receiptBytes = encodeJSON(t, f.receipt)
		if _, _, _, _, err := decodeAndValidateReplayInputs(f.planBytes, f.receiptBytes); err == nil {
			t.Fatal("accepted receipt not bound to exact replay-plan bytes")
		}
	})
	t.Run("receipt-already-executed", func(t *testing.T) {
		f := newTestFixture(t)
		f.receipt.ReplayExecuted = true
		f.receiptBytes = encodeJSON(t, f.receipt)
		if _, _, _, _, err := decodeAndValidateReplayInputs(f.planBytes, f.receiptBytes); err == nil {
			t.Fatal("accepted replay-plan receipt that already claims execution")
		}
	})
}

func TestLaunchProvenanceTSVUsesValidatedPlanValues(t *testing.T) {
	f := newTestFixture(t)
	got := launchProvenanceTSV(f.plan)
	want := strings.Join([]string{
		expectedBaselineID,
		expectedBaselineRevision,
		expectedTargetRevision,
		expectedTargetAdapter,
		expectedHost,
	}, "\t")
	if got != want {
		t.Fatalf("launch provenance TSV=%q, want %q", got, want)
	}
	if strings.ContainsAny(got, "\r\n") {
		t.Fatalf("launch provenance TSV contains a line break: %q", got)
	}
}

func TestVerifyOriginArtifactsAcceptsExactPlanArtifacts(t *testing.T) {
	f := newTestFixture(t)
	if err := verifyOriginArtifacts(f.plan, f.rawDir); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyOriginArtifactsRejectsIdentityDrift(t *testing.T) {
	t.Run("digest", func(t *testing.T) {
		f := newTestFixture(t)
		f.plan.Units[0].OriginRawArtifactSHA256 = strings.Repeat("f", 64)
		if err := verifyOriginArtifacts(f.plan, f.rawDir); err == nil {
			t.Fatal("accepted origin raw artifact digest drift")
		}
	})
	t.Run("missing", func(t *testing.T) {
		f := newTestFixture(t)
		if err := os.Remove(filepath.Join(f.rawDir, f.plan.Units[0].OriginRawArtifactFile)); err != nil {
			t.Fatal(err)
		}
		if err := verifyOriginArtifacts(f.plan, f.rawDir); err == nil {
			t.Fatal("accepted missing origin raw artifact")
		}
	})
	t.Run("credential", func(t *testing.T) {
		f := newTestFixture(t)
		path := filepath.Join(f.rawDir, f.plan.Units[0].OriginRawArtifactFile)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, []byte("\neditor-secret")...)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := verifyOriginArtifacts(f.plan, f.rawDir); err == nil {
			t.Fatal("accepted credential material in origin raw artifact")
		}
	})
}

func TestReplayInputPreflightRejectsInvalidOriginPlanIdentity(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*replayUnit)
	}{
		{"filename", func(unit *replayUnit) { unit.OriginRawArtifactFile = "other.json" }},
		{"unsafe-filename", func(unit *replayUnit) { unit.OriginRawArtifactFile = "../read.json" }},
		{"digest", func(unit *replayUnit) { unit.OriginRawArtifactSHA256 = "invalid" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newTestFixture(t)
			tc.mutate(&f.plan.Units[0])
			rebindPlan(t, &f)
			if _, _, _, _, err := decodeAndValidateReplayInputs(f.planBytes, f.receiptBytes); err == nil {
				t.Fatalf("accepted invalid origin plan identity: %s", tc.name)
			}
		})
	}
}

func TestAuditRunProducesSixAuditDerivedObservations(t *testing.T) {
	f := newTestFixture(t)
	got, err := auditRun(f.planBytes, f.receiptBytes, f.metadataBytes, f.rawDir)
	if err != nil {
		t.Fatal(err)
	}
	if got.EvidenceNature != "first-web-target-r7b-run-observation" ||
		got.RunIdentity != f.metadata.RunIdentity ||
		got.TargetProcessInstanceIdentity != f.metadata.TargetProcessInstanceIdentity ||
		got.SourceReplayPlanSHA256 != sha256Hex(f.planBytes) ||
		got.SourceReplayPlanReceiptSHA256 != sha256Hex(f.receiptBytes) ||
		got.ReplayTargetBuildRevision != expectedTargetRevision ||
		got.TargetListenScope != "loopback" {
		t.Fatalf("unexpected run observation header: %+v", got)
	}
	if len(got.Observations) != 6 {
		t.Fatalf("observation count=%d want=6", len(got.Observations))
	}
	for _, observation := range got.Observations {
		if !observation.SemanticValidationPassed || !isSHA256(observation.RawArtifactSHA256) {
			t.Fatalf("observation did not retain audit-derived semantic/digest evidence: %+v", observation)
		}
		data, err := os.ReadFile(filepath.Join(f.rawDir, observation.RawArtifactFile))
		if err != nil {
			t.Fatal(err)
		}
		if observation.RawArtifactSHA256 != sha256Hex(data) {
			t.Fatalf("raw digest mismatch for %q", observation.BehaviorID)
		}
	}
}

func TestAuditRunRejectsExactSourceBindingDrift(t *testing.T) {
	t.Run("plan-digest", func(t *testing.T) {
		f := newTestFixture(t)
		f.metadata.SourceReplayPlanSHA256 = strings.Repeat("f", 64)
		f.metadataBytes = encodeJSON(t, f.metadata)
		assertAuditFails(t, f)
	})
	t.Run("receipt-digest", func(t *testing.T) {
		f := newTestFixture(t)
		f.metadata.SourceReplayPlanReceiptSHA256 = strings.Repeat("e", 64)
		f.metadataBytes = encodeJSON(t, f.metadata)
		assertAuditFails(t, f)
	})
	t.Run("receipt-plan-binding", func(t *testing.T) {
		f := newTestFixture(t)
		f.receipt.ReplayPlanArtifactSHA256 = strings.Repeat("d", 64)
		f.receiptBytes = encodeJSON(t, f.receipt)
		f.metadata.SourceReplayPlanReceiptSHA256 = sha256Hex(f.receiptBytes)
		f.metadataBytes = encodeJSON(t, f.metadata)
		assertAuditFails(t, f)
	})
	t.Run("receipt-source-intake-binding", func(t *testing.T) {
		f := newTestFixture(t)
		f.receipt.SourceExecutionIntakeSHA256 = strings.Repeat("c", 64)
		f.receiptBytes = encodeJSON(t, f.receipt)
		f.metadata.SourceReplayPlanReceiptSHA256 = sha256Hex(f.receiptBytes)
		f.metadataBytes = encodeJSON(t, f.metadata)
		assertAuditFails(t, f)
	})
}

func TestAuditRunRejectsIdentityTargetAndListenDrift(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testFixture)
	}{
		{"empty-run-id", func(f *testFixture) { f.metadata.RunIdentity = "" }},
		{"empty-process-id", func(f *testFixture) { f.metadata.TargetProcessInstanceIdentity = "" }},
		{"target-drift", func(f *testFixture) { f.metadata.ReplayTargetBuildRevision = strings.Repeat("a", 40) }},
		{"non-loopback", func(f *testFixture) { f.metadata.TargetListenScope = "all-interfaces" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newTestFixture(t)
			tc.mutate(&f)
			f.metadataBytes = encodeJSON(t, f.metadata)
			assertAuditFails(t, f)
		})
	}
}

func TestAuditRunRejectsCaptureSetAndHTTPDrift(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		f := newTestFixture(t)
		f.metadata.Captures = f.metadata.Captures[1:]
		f.metadataBytes = encodeJSON(t, f.metadata)
		assertAuditFails(t, f)
	})
	t.Run("duplicate", func(t *testing.T) {
		f := newTestFixture(t)
		f.metadata.Captures[1] = f.metadata.Captures[0]
		f.metadataBytes = encodeJSON(t, f.metadata)
		assertAuditFails(t, f)
	})
	t.Run("unknown", func(t *testing.T) {
		f := newTestFixture(t)
		f.metadata.Captures[0].BehaviorID = "first-web/unknown"
		f.metadataBytes = encodeJSON(t, f.metadata)
		assertAuditFails(t, f)
	})
	t.Run("http", func(t *testing.T) {
		f := newTestFixture(t)
		f.metadata.Captures[0].ObservedHTTPStatus++
		f.metadataBytes = encodeJSON(t, f.metadata)
		assertAuditFails(t, f)
	})
}

func TestAuditRunRejectsUnsafeOrAmbiguousRawArtifactMapping(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testFixture)
	}{
		{"empty", func(f *testFixture) { f.metadata.Captures[0].RawArtifactFile = "" }},
		{"absolute", func(f *testFixture) { f.metadata.Captures[0].RawArtifactFile = filepath.Join(f.rawDir, "read.json") }},
		{"parent-traversal", func(f *testFixture) { f.metadata.Captures[0].RawArtifactFile = "../read.json" }},
		{"duplicate-file", func(f *testFixture) { f.metadata.Captures[1].RawArtifactFile = f.metadata.Captures[0].RawArtifactFile }},
		{"missing-file", func(f *testFixture) { f.metadata.Captures[0].RawArtifactFile = "missing.json" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newTestFixture(t)
			tc.mutate(&f)
			f.metadataBytes = encodeJSON(t, f.metadata)
			assertAuditFails(t, f)
		})
	}
}

func TestAuditRunRejectsPlanAndRawSemanticDrift(t *testing.T) {
	t.Run("plan-http-status", func(t *testing.T) {
		f := newTestFixture(t)
		f.plan.Units[0].ExpectedHTTPStatus++
		rebindPlan(t, &f)
		assertAuditFails(t, f)
	})
	t.Run("plan-observation-class", func(t *testing.T) {
		f := newTestFixture(t)
		f.plan.Units[0].ExpectedObservationClass = "other-class"
		rebindPlan(t, &f)
		assertAuditFails(t, f)
	})
	t.Run("plan-distinct-runs", func(t *testing.T) {
		f := newTestFixture(t)
		f.plan.Units[0].RequiredDistinctRuns = 3
		rebindPlan(t, &f)
		assertAuditFails(t, f)
	})
	t.Run("plan-provenance", func(t *testing.T) {
		f := newTestFixture(t)
		f.plan.HandoverBaselineID = "other-B"
		rebindPlan(t, &f)
		assertAuditFails(t, f)
	})
	t.Run("raw-provenance", func(t *testing.T) {
		f := newTestFixture(t)
		name := captureFile(f.metadata, "first-web/read-only")
		data, err := os.ReadFile(filepath.Join(f.rawDir, name))
		if err != nil {
			t.Fatal(err)
		}
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatal(err)
		}
		raw["targetBuildRevision"] = "other-T"
		if err := os.WriteFile(filepath.Join(f.rawDir, name), encodeJSON(t, raw), 0o600); err != nil {
			t.Fatal(err)
		}
		assertAuditFails(t, f)
	})
	t.Run("credential", func(t *testing.T) {
		f := newTestFixture(t)
		name := captureFile(f.metadata, "first-web/read-only")
		path := filepath.Join(f.rawDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, []byte("\neditor-secret\n")...)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		assertAuditFails(t, f)
	})
}

func TestRunObservationShapeDoesNotCarryExecutionSecretsOrMotorAuthority(t *testing.T) {
	typ := reflect.TypeOf(runObservation{})
	for _, forbidden := range []string{"Credential", "Password", "RequestBody", "Selector", "Coordinate", "Authorized", "CanExecute", "Repeatable"} {
		if _, ok := typ.FieldByName(forbidden); ok {
			t.Fatalf("run observation leaked forbidden field %q", forbidden)
		}
	}
	obsType := reflect.TypeOf(behaviorObservation{})
	for _, forbidden := range []string{"Credential", "Password", "RequestBody", "Selector", "Coordinate", "Authorized", "CanExecute", "Repeatable"} {
		if _, ok := obsType.FieldByName(forbidden); ok {
			t.Fatalf("behavior observation leaked forbidden field %q", forbidden)
		}
	}
}

func newTestFixture(t *testing.T) testFixture {
	t.Helper()
	rules := behaviorRules()
	ids := []string{
		"first-web/read-only",
		"first-web/command-success",
		"first-web/authentication-denied",
		"first-web/authorization-denied",
		"first-web/validation-not-satisfied",
		"first-web/execution-error",
	}
	statuses := map[string]int{
		"first-web/read-only":                200,
		"first-web/command-success":          200,
		"first-web/authentication-denied":    200,
		"first-web/authorization-denied":     200,
		"first-web/validation-not-satisfied": 200,
		"first-web/execution-error":          422,
	}
	files := map[string]string{
		"first-web/read-only":                "read.json",
		"first-web/command-success":          "submit-success.json",
		"first-web/authentication-denied":    "submit-authentication-denied.json",
		"first-web/authorization-denied":     "submit-authorization-denied.json",
		"first-web/validation-not-satisfied": "submit-validation-not-satisfied.json",
		"first-web/execution-error":          "submit-execution-error.json",
	}
	p := map[string]string{
		"handoverBaselineId":         expectedBaselineID,
		"baselineRepositoryRevision": expectedBaselineRevision,
		"targetBuildRevision":        expectedTargetRevision,
		"targetAdapter":              expectedTargetAdapter,
		"host":                       expectedHost,
	}
	units := make([]replayUnit, 0, len(ids))
	captures := make([]runCapture, 0, len(ids))
	for _, id := range ids {
		originBytes := behaviorFixture(t, id, p)
		units = append(units, replayUnit{
			BehaviorID:                id,
			ExpectedHTTPStatus:        statuses[id],
			ExpectedObservationClass:  rules[id].ObservationClass,
			OriginRawArtifactFile:     files[id],
			OriginRawArtifactSHA256:   sha256Hex(originBytes),
			ReplayTargetBuildRevision: expectedTargetRevision,
			RequiredDistinctRuns:      2,
			ReplayExecutionStatus:     "not-executed",
			RepeatabilityStatus:       "not-observed",
			CoverageClass:             "behavior-unit",
		})
		captures = append(captures, runCapture{BehaviorID: id, ObservedHTTPStatus: statuses[id], RawArtifactFile: files[id]})
	}
	plan := replayPlan{
		EvidenceNature:                     "first-web-target-r7b-replay-plan",
		SourceExecutionIntakeSHA256:        strings.Repeat("1", 64),
		SourceExecutionIntakeReceiptSHA256: strings.Repeat("2", 64),
		HandoverBaselineID:                 expectedBaselineID,
		BaselineRepositoryRevision:         expectedBaselineRevision,
		ReplayTargetBuildRevision:          expectedTargetRevision,
		TargetAdapter:                      expectedTargetAdapter,
		Host:                               expectedHost,
		Units:                              units,
	}
	planBytes := encodeJSON(t, plan)
	receipt := replayPlanReceipt{
		EvidenceNature:                     "first-web-target-r7b-replay-plan-acquisition-receipt",
		ReplayPlanToolRepositoryRevision:   strings.Repeat("3", 40),
		RepositoryPostRunRevision:          strings.Repeat("3", 40),
		RepositoryPostRunClean:             true,
		SourceQualificationPassed:          true,
		SourceExecutionIntakeSHA256:        plan.SourceExecutionIntakeSHA256,
		SourceExecutionIntakeReceiptSHA256: plan.SourceExecutionIntakeReceiptSHA256,
		ReplayPlanArtifact:                 "r7b-replay-plan.json",
		ReplayPlanArtifactSHA256:           sha256Hex(planBytes),
		ReplayExecuted:                     false,
		RepeatabilityObserved:              false,
	}
	receiptBytes := encodeJSON(t, receipt)
	metadata := runMetadata{
		EvidenceNature:                "first-web-target-r7b-run-metadata",
		RunIdentity:                   "run-20260920T000000Z-a1",
		SourceReplayPlanSHA256:        sha256Hex(planBytes),
		SourceReplayPlanReceiptSHA256: sha256Hex(receiptBytes),
		ReplayTargetBuildRevision:     expectedTargetRevision,
		TargetProcessInstanceIdentity: "process-20260920T000000Z-a1",
		TargetListenScope:             "loopback",
		Captures:                      captures,
	}
	metadataBytes := encodeJSON(t, metadata)
	rawDir := t.TempDir()
	for _, id := range ids {
		if err := os.WriteFile(filepath.Join(rawDir, files[id]), behaviorFixture(t, id, p), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return testFixture{plan: plan, receipt: receipt, metadata: metadata, planBytes: planBytes, receiptBytes: receiptBytes, metadataBytes: metadataBytes, rawDir: rawDir}
}

func rebindPlan(t *testing.T, f *testFixture) {
	t.Helper()
	f.planBytes = encodeJSON(t, f.plan)
	f.receipt.ReplayPlanArtifactSHA256 = sha256Hex(f.planBytes)
	f.receipt.SourceExecutionIntakeSHA256 = f.plan.SourceExecutionIntakeSHA256
	f.receipt.SourceExecutionIntakeReceiptSHA256 = f.plan.SourceExecutionIntakeReceiptSHA256
	f.receiptBytes = encodeJSON(t, f.receipt)
	f.metadata.SourceReplayPlanSHA256 = sha256Hex(f.planBytes)
	f.metadata.SourceReplayPlanReceiptSHA256 = sha256Hex(f.receiptBytes)
	f.metadata.ReplayTargetBuildRevision = f.plan.ReplayTargetBuildRevision
	f.metadataBytes = encodeJSON(t, f.metadata)
}

func assertAuditFails(t *testing.T, f testFixture) {
	t.Helper()
	if _, err := auditRun(f.planBytes, f.receiptBytes, f.metadataBytes, f.rawDir); err == nil {
		t.Fatal("audit unexpectedly succeeded")
	}
}

func captureFile(metadata runMetadata, behaviorID string) string {
	for _, c := range metadata.Captures {
		if c.BehaviorID == behaviorID {
			return c.RawArtifactFile
		}
	}
	return ""
}

func encodeJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(data, '\n')
}

func behaviorFixture(t *testing.T, id string, p map[string]string) []byte {
	t.Helper()
	base := map[string]any{
		"handoverBaselineId":         p["handoverBaselineId"],
		"baselineRepositoryRevision": p["baselineRepositoryRevision"],
		"targetBuildRevision":        p["targetBuildRevision"],
		"targetAdapter":              p["targetAdapter"],
		"host":                       p["host"],
	}
	switch id {
	case "first-web/read-only":
		base["result"] = map[string]any{
			"scenarioId": "order-entry.read-only-web-slice", "applicationId": "order_entry",
			"runtimeBindings": []any{
				map[string]any{"pattern": "order_entry.recent_orders", "datasource": "sales", "database": "sales_core", "access": "read-write"},
				map[string]any{"pattern": "order_entry.customer_reference", "environment": "development", "instance": "master_data.dev", "provider": "oracle", "executorCapability": "database.query.oracle", "contractQuery": "master_data/master_data.customer/find_by_id"},
				map[string]any{"pattern": "order_entry.order_lines", "environment": "development", "instance": "sales_core.dev", "provider": "postgres", "executorCapability": "database.query.postgres", "contractQuery": "sales_core/sales_core.order_line/find_by_order"},
			},
		}
	case "first-web/command-success":
		base["result"] = map[string]any{
			"scenarioId": "order-entry.submit-order-web-slice", "applicationId": "order_entry", "stage": "completed",
			"authentication": map[string]any{"capability": "security.authenticate", "mode": "outcome", "outcome": "authenticated"},
			"validations":    []any{map[string]any{"validation": "sales.order_line.quantity_positive", "result": "satisfied"}, map[string]any{"validation": "sales.order.lines.line_number_unique", "result": "satisfied"}},
			"authorization":  map[string]any{"capability": "security.authorize", "mode": "outcome", "outcome": "allowed"},
			"command":        map[string]any{"capability": "sales.order.submit", "mode": "output"},
			"observation":    map[string]any{"readIdentity": "synthetic.order_state.read", "nature": "synthetic", "result": "submitted"},
		}
	case "first-web/authentication-denied":
		base["result"] = map[string]any{"scenarioId": "order-entry.submit-order-web-slice", "applicationId": "order_entry", "stage": "authentication-denied", "authentication": map[string]any{"capability": "security.authenticate", "mode": "outcome", "outcome": "denied"}}
	case "first-web/authorization-denied":
		base["result"] = map[string]any{"scenarioId": "order-entry.submit-order-web-slice", "applicationId": "order_entry", "stage": "authorization-denied", "authentication": map[string]any{"capability": "security.authenticate", "mode": "outcome", "outcome": "authenticated"}, "authorization": map[string]any{"capability": "security.authorize", "mode": "outcome", "outcome": "denied"}}
	case "first-web/validation-not-satisfied":
		base["result"] = map[string]any{"scenarioId": "order-entry.submit-order-web-slice", "applicationId": "order_entry", "stage": "validation-not-satisfied", "authentication": map[string]any{"capability": "security.authenticate", "mode": "outcome", "outcome": "authenticated"}, "validations": []any{map[string]any{"validation": "sales.order_line.quantity_positive", "result": "not-satisfied"}}}
	case "first-web/execution-error":
		base["result"] = map[string]any{"scenarioId": "order-entry.submit-order-web-slice", "applicationId": "order_entry", "stage": "command", "authentication": map[string]any{"capability": "security.authenticate", "mode": "outcome", "outcome": "authenticated"}, "validations": []any{map[string]any{"validation": "sales.order_line.quantity_positive", "result": "satisfied"}, map[string]any{"validation": "sales.order.lines.line_number_unique", "result": "satisfied"}}, "authorization": map[string]any{"capability": "security.authorize", "mode": "outcome", "outcome": "allowed"}}
		base["executionError"] = "capability sales.order.submitの実行を完了できませんでした: synthetic submit adapter unavailable for order 1003"
	default:
		t.Fatalf("unknown behavior %q", id)
	}
	return encodeJSON(t, base)
}

func TestSHAHelperMatchesStandardLibrary(t *testing.T) {
	data := []byte("abc")
	sum := sha256.Sum256(data)
	if got, want := sha256Hex(data), hex.EncodeToString(sum[:]); got != want {
		t.Fatalf("%s != %s", got, want)
	}
}

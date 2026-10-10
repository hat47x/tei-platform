package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

const h = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const m = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func fixture() Packet {
	return Packet{
		Schema: schema, Claim: "E2",
		Contract:      Contract{ProducerPath: "tools/reference-shared-acceptance/practical_validation_loop_test.go", ConsumerPath: "tools/reference-first-web-target/practical_business_rule_test.go", PositiveTest: "TestPracticalRuleChangePreviewsAndAcceptsBothMeaningAndApplicationSelection", NegativeTest: "TestReferenceCommandScenarioRejectsMissingValuesAcrossAllLines", EvidenceClass: "actual-observed"},
		Qualification: Qualification{Head: h, Main: m, FullHealth: "PASS", CleanBefore: true, CleanAfter: true, Receipt: "local/health.log"},
		Acceptance:    Acceptance{R0: "sha256:" + strings.Repeat("0", 64), R1: "sha256:" + strings.Repeat("1", 64), CandidateFingerprint: "sha256:" + strings.Repeat("2", 64), ReviewFingerprint: "sha256:" + strings.Repeat("3", 64), DecisionReceipt: "local/accept.json", ReloadReceipt: "local/reload.json", Delta: []string{"add-application-validation-use", "add-validation"}},
		Web:           Web{AcceptedRevision: "sha256:" + strings.Repeat("1", 64), Application: "order_entry", Target: "web", SelectedValidation: "sales.order_line.unit_price_positive", NegativeValidation: "sales.order_line.unit_price_positive", NegativeEvidenceMode: "running-server", ProviderKind: "actual-db", RunMode: "running-server", WorkspaceRevisionReceipt: "local/r1-web-workspace.json", TargetLaunchReceipt: "local/target-launch.json", PositiveReceipt: "local/positive.json", NegativeReceipt: "local/negative.json", UnobservedControlReceipt: "local/n11.json"},
		Physical:      Physical{AcceptedRevision: "sha256:" + strings.Repeat("1", 64), Datasource: "db-orders", SchemaCompatibility: "compatible", SchemaEvidence: "local/schema.json", HostAuthorityEvidence: "local/host.json", CommandReceipt: "local/write.json", IndependentReadReceipt: "local/read.json", RecordID: "o-1", SchemaFailureControl: "local/s02.json", HostFailureControl: "local/s05.json"},
	}
}
func has(r Report, s string) bool {
	for _, d := range r.Diagnostics {
		if d == s {
			return true
		}
	}
	return false
}
func TestCompletePacketNeverGrantsProductGate(t *testing.T) {
	p := fixture()
	result := Audit(p, h, m)
	if result.Status != "REVIEW_REQUIRED" || result.ProductGate != "UNQUALIFIED" || len(result.Diagnostics) > 0 {
		t.Fatalf("%+v", result)
	}
}
func TestSyntheticWebCannotClaimE2(t *testing.T) {
	p := fixture()
	p.Web.ProviderKind = "synthetic"
	result := Audit(p, h, m)
	if !has(result, "SYNTHETIC_WEB_PROVIDER_CANNOT_CLAIM_E2") || result.Status != "INCOMPLETE" {
		t.Fatalf("%+v", result)
	}
}
func TestE1WithSyntheticWebIsOnlyReviewable(t *testing.T) {
	p := fixture()
	p.Claim = "E1"
	p.Web.ProviderKind = "synthetic"
	p.Physical = Physical{}
	result := Audit(p, h, m)
	if result.Status != "REVIEW_REQUIRED" || result.ProductGate != "UNQUALIFIED" {
		t.Fatalf("%+v", result)
	}
}
func TestRuleWithoutExplicitApplicationUseIsRejected(t *testing.T) {
	p := fixture()
	p.Claim = "E1"
	p.Acceptance.Delta = []string{"add-validation"}
	if !has(Audit(p, h, m), "VALIDATION_AND_APPLICATION_USE_NOT_ATOMIC") {
		t.Fatal("missing atomic-use diagnostic")
	}
}
func TestRevisionAndStaleQualificationFail(t *testing.T) {
	p := fixture()
	p.Acceptance.R1 = p.Acceptance.R0
	r := Audit(p, h, "cccccccccccccccccccccccccccccccccccccccc")
	if !has(r, "REVISION_QUALIFICATION_STALE_OR_MISSING") || !has(r, "ACCEPTED_R0_R1_LINEAGE_MISSING") || !has(r, "WEB_NOT_BOUND_TO_ACCEPTED_R1_RULE_SELECTION") {
		t.Fatalf("%+v", r)
	}
}
func TestMissingReadAndNegativeControlsFail(t *testing.T) {
	p := fixture()
	p.Physical.IndependentReadReceipt = p.Physical.CommandReceipt
	p.Physical.SchemaFailureControl = ""
	p.Physical.HostFailureControl = ""
	p.Web.UnobservedControlReceipt = ""
	r := Audit(p, h, m)
	for _, c := range []string{"E2_COMMAND_AND_INDEPENDENT_READ_NOT_CORRELATED", "S02_NEGATIVE_CONTROL_MISSING", "S05_NEGATIVE_CONTROL_MISSING", "N11_UNOBSERVED_CONTROL_MISSING"} {
		if !has(r, c) {
			t.Fatalf("missing %s in %+v", c, r)
		}
	}
}
func TestHealthNotRunCannotQualify(t *testing.T) {
	p := fixture()
	p.Qualification.FullHealth = "NOT_RUN"
	p.Qualification.CleanAfter = false
	if !has(Audit(p, h, m), "TRUSTED_FULL_HEALTH_NOT_RECORDED") {
		t.Fatal("lost trusted gate")
	}
}
func TestE0RejectsEmptyProducerConsumerEvidence(t *testing.T) {
	p := fixture()
	p.Claim = "E0"
	p.Contract = Contract{}
	if !has(Audit(p, h, m), "E0_CONTRACT_OR_POSITIVE_NEGATIVE_TEST_MISSING") {
		t.Fatal("empty E0 evidence can pass preflight")
	}
}
func TestE1HandlerOnlyUnitPriceNegativeIsNotLiveEvidence(t *testing.T) {
	p := fixture()
	p.Claim = "E1"
	// Existing unit-price rejection is a Go HTTP handler test; it does not
	// exercise a separately running Web Target with an invalid unit price.
	p.Web.NegativeEvidenceMode = "handler-only"
	r := Audit(p, h, m)
	if !has(r, "E1_NEGATIVE_SELECTED_RULE_RUNNING_WEB_EVIDENCE_REQUIRED") ||
		r.Status != "INCOMPLETE" || r.ProductGate != "UNQUALIFIED" {
		t.Fatalf("%+v", r)
	}
}

func TestE1OtherValidationCannotStandInForUnitPriceNegative(t *testing.T) {
	p := fixture()
	p.Claim = "E1"
	// The default Web fixture's rejected order has invalid quantity,
	// which does not demonstrate the newly accepted unit-price guard.
	p.Web.NegativeValidation = "sales.order_line.quantity_positive"
	r := Audit(p, h, m)
	if !has(r, "E1_NEGATIVE_SELECTED_RULE_RUNNING_WEB_EVIDENCE_REQUIRED") ||
		r.Status != "INCOMPLETE" {
		t.Fatalf("%+v", r)
	}
}

func TestE1RejectsUnboundWebWorkspaceRevision(t *testing.T) {
	for name, edit := range map[string]func(*Packet){
		"missing workspace revision": func(p *Packet) { p.Web.WorkspaceRevisionReceipt = "" },
		"missing target launch": func(p *Packet) { p.Web.TargetLaunchReceipt = "" },
		"reused positive result": func(p *Packet) { p.Web.WorkspaceRevisionReceipt = p.Web.PositiveReceipt },
		"same evidence for both stages": func(p *Packet) { p.Web.TargetLaunchReceipt = p.Web.WorkspaceRevisionReceipt },
	} {
		t.Run(name, func(t *testing.T) {
			p := fixture()
			p.Claim = "E1"
			edit(&p)
			r := Audit(p, h, m)
			if !has(r, "E1_WEB_WORKSPACE_R1_AND_TARGET_LAUNCH_PROVENANCE_MISSING") ||
				r.Status != "INCOMPLETE" || r.ProductGate != "UNQUALIFIED" {
				t.Fatalf("%+v", r)
			}
		})
	}
}

func TestHandlerOnlyIsNotARealWebRun(t *testing.T) {
	p := fixture()
	p.Claim = "E1"
	p.Web.RunMode = "handler-only"
	if !has(Audit(p, h, m), "E1_RUNNING_WEB_TARGET_RECEIPT_REQUIRED") {
		t.Fatal("handler-only test promoted to E1")
	}
}

func TestStrictInputRejectsUnknownAndTrailing(t *testing.T) {
	p := fixture()
	dir := t.TempDir()
	path := filepath.Join(dir, "packet.json")
	b, _ := json.Marshal(p)
	if err := os.WriteFile(path, append(b, []byte(`{"schema":"duplicate"}`)...), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-packet", path, "-head", h, "-main", m}, new(bytes.Buffer)); err == nil {
		t.Fatal("trailing values accepted")
	}
	var obj map[string]any
	if err := json.Unmarshal(b, &obj); err != nil {
		t.Fatal(err)
	}
	obj["extra"] = "not authorized"
	b, _ = json.Marshal(obj)
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-packet", path, "-head", h, "-main", m}, new(bytes.Buffer)); err == nil {
		t.Fatal("unknown input accepted")
	}
}
func TestDeterministicDiagnosticsOrder(t *testing.T) {
	p := fixture()
	p.Claim = "E2"
	p.Web.ProviderKind = "synthetic"
	p.Physical.SchemaCompatibility = "unresolved"
	p.Qualification.FullHealth = "NOT_RUN"
	a := Audit(p, h, m)
	b := Audit(p, h, m)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("unstable audit")
	}
	if a.Status != "INCOMPLETE" || a.ProductGate != "UNQUALIFIED" {
		t.Fatalf("%+v", a)
	}
}

func TestClaimedE1E2RejectsSourceOnlyEvidenceClass(t *testing.T) {
	for _, tier := range []string{"E1", "E2"} {
		p := fixture()
		p.Claim = tier
		for _, class := range []string{"source-only", "synthetic-tested"} {
			p.Contract.EvidenceClass = class
			r := Audit(p, h, m)
			if r.Status != "INCOMPLETE" || !has(r, "E1_E2_ACTUAL_EXECUTION_CLASS_REQUIRED") {
				t.Fatalf("%s %s was promoted: %+v", tier, class, r)
			}
		}
	}
	p := fixture()
	p.Claim = "E0"
	p.Contract.EvidenceClass = "source-only"
	if r := Audit(p, h, m); r.Status != "REVIEW_REQUIRED" || r.ProductGate != "UNQUALIFIED" {
		t.Fatalf("E0 should permit source-only, but never qualify Product Gate: %+v", r)
	}
}

func TestStrictInputRejectsDuplicateKeysAndInvalidUTF8(t *testing.T) {
	p := fixture()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "packet.json")
	test := func(label string, data []byte) {
		t.Helper()
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if err := run([]string{"-packet", path, "-head", h, "-main", m}, new(bytes.Buffer)); err == nil {
			t.Fatal(label, "accepted")
		}
	}
	for label, data := range map[string][]byte{
		"top-level duplicate": bytes.Replace(b, []byte(`"claim":"E2"`), []byte(`"claim":"E0","claim":"E2"`), 1),
		"case-folded alias":   bytes.Replace(b, []byte(`"claim":"E2"`), []byte(`"claim":"E0","Claim":"E2"`), 1),
		"nested duplicate":    bytes.Replace(b, []byte(`"runMode":"running-server"`), []byte(`"runMode":"handler-only","runMode":"running-server"`), 1),
		"invalid UTF-8":       append(append([]byte{}, b...), 0xff),
	} {
		test(label, data)
	}
}

func TestStrictInputCannotHideTrailingBytesAfterOneMiBLimit(t *testing.T) {
	b, err := json.Marshal(fixture())
	if err != nil {
		t.Fatal(err)
	}
	if len(b) >= (1 << 20) {
		t.Fatal("fixture unexpectedly large")
	}
	data := append(b, bytes.Repeat([]byte(" "), (1<<20)-len(b))...)
	data = append(data, []byte(`{"claim":"E0"}`)...)
	path := filepath.Join(t.TempDir(), "packet.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-packet", path, "-head", h, "-main", m}, new(bytes.Buffer)); err == nil {
		t.Fatal("over-limit packet was silently truncated at artificial EOF")
	}
}

func TestJSONDepthBoundRejectsPathologicalInput(t *testing.T) {
	data := []byte(strings.Repeat("[", 80) + "0" + strings.Repeat("]", 80))
	if err := rejectDuplicateJSONKeys(data); err == nil {
		t.Fatal("pathological nesting was accepted")
	}
}

func TestCLIReviewRequiredRemainsUnqualified(t *testing.T) {
	b, err := json.Marshal(fixture())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "packet.json")
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{"-packet", path, "-head", h, "-main", m}, &out); err != nil {
		t.Fatalf("reviewable packet should emit a report: %v", err)
	}
	var result Report
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "REVIEW_REQUIRED" || result.ProductGate != "UNQUALIFIED" {
		t.Fatalf("CLI accidentally granted Product Gate: %+v", result)
	}
}

func TestCLIIncompleteEmitsDiagnosticAndFails(t *testing.T) {
	p := fixture()
	p.Claim = "E1"
	p.Contract.EvidenceClass = "source-only"
	p.Qualification.FullHealth = "NOT_RUN"
	p.Web.RunMode = "handler-only"
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "packet.json")
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{"-packet", path, "-head", h, "-main", m}, &out); err == nil {
		t.Fatal("incomplete claim did not fail")
	}
	var result Report
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "INCOMPLETE" || result.ProductGate != "UNQUALIFIED" ||
		!has(result, "E1_E2_ACTUAL_EXECUTION_CLASS_REQUIRED") ||
		!has(result, "E1_RUNNING_WEB_TARGET_RECEIPT_REQUIRED") ||
		!has(result, "TRUSTED_FULL_HEALTH_NOT_RECORDED") {
		t.Fatalf("CLI omitted essential incompleteness: %+v", result)
	}
}

func TestCurrentP0FixtureCannotBeSubmittedAsEvidence(t *testing.T) {
	data, err := os.ReadFile("testdata/p0-order-unit-price-incomplete.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		t.Fatal(err)
	}
	var p Packet
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	result := Audit(p, h, m)
	if result.Status != "INCOMPLETE" || result.ProductGate != "UNQUALIFIED" {
		t.Fatalf("existing source-based tests were promoted to actual E1: %+v", result)
	}
	for _, code := range []string{
		"E1_E2_ACTUAL_EXECUTION_CLASS_REQUIRED",
		"ACCEPTED_R0_R1_LINEAGE_MISSING",
		"E1_RUNNING_WEB_TARGET_RECEIPT_REQUIRED",
		"TRUSTED_FULL_HEALTH_NOT_RECORDED",
	} {
		if !has(result, code) {
			t.Fatalf("P0 coverage fixture lost expected gap %q: %+v", code, result)
		}
	}
}

// The preflight receives untrusted packets. Keep the duplicate-key scan linear
// in the number of distinct keys (the input is also bounded to 1 MiB by run).
func TestStrictJSONManyDistinctKeysWithoutQuadraticScan(t *testing.T) {
	var doc strings.Builder
	doc.WriteByte('{')
	for i := 0; i < 10000; i++ {
		if i != 0 {
			doc.WriteByte(',')
		}
		doc.WriteString(strconv.Quote("field" + strconv.Itoa(i)))
		doc.WriteString(":null")
	}
	doc.WriteByte('}')
	if err := rejectDuplicateJSONKeys([]byte(doc.String())); err != nil {
		t.Fatalf("distinct keys rejected: %v", err)
	}
}

func TestStrictJSONUnicodeFoldedAliasRejected(t *testing.T) {
	if !strings.EqualFold("Key", "key") {
		t.Fatal("unexpected Go unicode folding semantics")
	}
	if err := rejectDuplicateJSONKeys([]byte(`{"Key":1,"key":2}`)); err == nil {
		t.Fatal("Unicode folded key shadowing was accepted")
	}
}

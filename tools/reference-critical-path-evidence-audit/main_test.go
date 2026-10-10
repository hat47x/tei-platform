package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const h = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const m = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func fixture() Packet {
	return Packet{
		Schema: schema, Claim: "E2",
		Contract:      Contract{ProducerPath: "tools/reference-shared-acceptance/practical_validation_loop_test.go", ConsumerPath: "tools/reference-first-web-target/practical_business_rule_test.go", PositiveTest: "TestPracticalRuleChangePreviewsAndAcceptsBothMeaningAndApplicationSelection", NegativeTest: "TestReferenceCommandScenarioRejectsMissingValuesAcrossAllLines", EvidenceClass: "synthetic-tested"},
		Qualification: Qualification{Head: h, Main: m, FullHealth: "PASS", CleanBefore: true, CleanAfter: true, Receipt: "local/health.log"},
		Acceptance:    Acceptance{R0: "sha256:" + strings.Repeat("0", 64), R1: "sha256:" + strings.Repeat("1", 64), CandidateFingerprint: "sha256:" + strings.Repeat("2", 64), ReviewFingerprint: "sha256:" + strings.Repeat("3", 64), DecisionReceipt: "local/accept.json", ReloadReceipt: "local/reload.json", Delta: []string{"add-application-validation-use", "add-validation"}},
		Web:           Web{AcceptedRevision: "sha256:" + strings.Repeat("1", 64), Application: "order_entry", Target: "web", SelectedValidation: "sales.order_line.unit_price_positive", ProviderKind: "actual-db", RunMode: "running-server", PositiveReceipt: "local/positive.json", NegativeReceipt: "local/negative.json", UnobservedControlReceipt: "local/n11.json"},
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

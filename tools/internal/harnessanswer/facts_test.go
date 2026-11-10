package harnessanswer

import (
	"strings"
	"testing"
)

const inspectFixture = `{
  "schemaVersion": "tei.cli.inspect/v0",
  "ok": true,
  "result": {
    "subject": {"kind": "semantic-data", "id": "sales.order"},
    "canonical": {
      "id": "sales.order",
      "symbol": "Order",
      "attributes": [
        {"id": "sales.order.total", "symbol": "total", "type": "std.money"},
        {"id": "sales.order.status", "symbol": "status", "type": "sales.order_status"}
      ],
      "relations": [
        {"id": "sales.order.customer", "symbol": "customer", "target": "customer.customer"}
      ]
    },
    "worldFingerprint": "sha256:c92de09f9595b1f46bfc18e6f99260d8391113b6d2639357d372abf634a1d65a",
    "sourceIndexFingerprint": "sha256:1827f4b46169710cbe26179498bf30ef775aecba858c3beff8e2e0d53ca23531",
    "sources": {
      "artifacts": [
        {"path": "semantic/sales/order.tei",
         "fingerprint": "sha256:f2063418b893a60a9f948edbb2b23ec821f7a15ed8e5863c36e1b72abade1e33"}
      ]
    }
  }
}`

const validateFixture = `{
  "schemaVersion": "tei.cli.validate/v0",
  "ok": true,
  "diagnostics": [
    {"diagnostic": {"code": "DATABASE_RELEASE_CONTRACT_NOT_CURRENT", "severity": "warning"}},
    {"diagnostic": {"code": "APPLICATION_RELEASE_CONTRACT_NOT_CURRENT", "severity": "warning"}},
    {"diagnostic": {"code": "DATABASE_RELEASE_CONTRACT_NOT_CURRENT", "severity": "warning"}}
  ],
  "worldFingerprint": "sha256:c92de09f9595b1f46bfc18e6f99260d8391113b6d2639357d372abf634a1d65a",
  "sourceIndexFingerprint": "sha256:1827f4b46169710cbe26179498bf30ef775aecba858c3beff8e2e0d53ca23531"
}`

func buildGroundTruth(t *testing.T) GroundTruth {
	t.Helper()
	gt, err := FromInspect([]byte(inspectFixture))
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	gt, err = gt.WithValidate([]byte(validateFixture))
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	return gt
}

// The facts must come from the captured CLI response, not from a hand-written
// expectation list: every attribute the CLI returned has to be represented.
func TestFactsFollowTheCapturedInspectResponse(t *testing.T) {
	gt := buildGroundTruth(t)

	if gt.SubjectID != "sales.order" || gt.SubjectKind != "semantic-data" {
		t.Fatalf("subject: %+v", gt)
	}
	var hasTotal, hasCustomer bool
	for _, fact := range gt.Facts {
		if fact.Key == "attribute:sales.order.total" {
			hasTotal = true
			// canonical ID, type, and symbol are all accepted forms.
			if len(fact.Tokens) != 3 || fact.Tokens[1] != "std.money" {
				t.Fatalf("attribute tokens: %+v", fact)
			}
		}
		if fact.Key == "relation:sales.order.customer" || fact.Key == "attribute:sales.order.customer" {
			hasCustomer = true
		}
	}
	if !hasTotal {
		t.Fatal("the total attribute must be a fact")
	}
	if !hasCustomer {
		t.Fatal("the customer reference must be a fact")
	}
	if len(gt.SourceArtifacts) != 1 || gt.SourceArtifacts[0] != "semantic/sales/order.tei" {
		t.Fatalf("source artifacts: %v", gt.SourceArtifacts)
	}
}

func TestValidateCountsIgnoringDuplicateCodes(t *testing.T) {
	gt := buildGroundTruth(t)
	if gt.ValidateOK == nil || !*gt.ValidateOK {
		t.Fatalf("validate ok: %v", gt.ValidateOK)
	}
	if gt.ValidateDiagnostics != 3 {
		t.Fatalf("diagnostics: %d", gt.ValidateDiagnostics)
	}
	if gt.ValidateSeverityCounts["warning"] != 3 {
		t.Fatalf("severity counts: %+v", gt.ValidateSeverityCounts)
	}
	// Two distinct codes, even though three diagnostics carried one of them twice.
	if len(gt.ValidateCodes) != 2 {
		t.Fatalf("codes: %v", gt.ValidateCodes)
	}
}

func TestCheckFindsMentionedFactsAndReportsMissing(t *testing.T) {
	gt := buildGroundTruth(t)
	answer := "sales.order の total は std.money です。定義元は semantic/sales/order.tei。"

	report := gt.Check(answer)
	if report.Found == 0 || report.Missing == 0 {
		t.Fatalf("the answer should match some facts and miss others: %+v", report)
	}
	for _, p := range report.Presence {
		if p.Key == "attribute:sales.order.total" && !p.Found {
			t.Fatalf("total must be found: %+v", p)
		}
		if p.Key == "relation:sales.order.customer" && p.Found {
			t.Fatalf("an unmentioned relation must stay missing: %+v", p)
		}
	}
}

// An empty answer must find nothing; the check must not treat absence as a pass.
func TestEmptyAnswerFindsNothing(t *testing.T) {
	gt := buildGroundTruth(t)
	report := gt.Check("")
	if report.Found != 0 {
		t.Fatalf("empty answer matched: %+v", report.Presence)
	}
	if report.Missing != len(gt.Facts) {
		t.Fatalf("missing %d of %d", report.Missing, len(gt.Facts))
	}
}

// The report must carry its own limits so a caller cannot read "found" as "true".
func TestReportStatesWhatItDoesNotEstablish(t *testing.T) {
	gt := buildGroundTruth(t)
	report := gt.Check("sales.order")
	if len(report.Limits) == 0 {
		t.Fatal("the report must state its limits")
	}
	joined := strings.Join(report.Limits, " ")
	for _, want := range []string{"correctly", "complete", "understood"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("limits missing %q: %v", want, report.Limits)
		}
	}
}

func TestFromInspectRejectsFailedEnvelope(t *testing.T) {
	body := []byte(`{"ok": false, "error": {"code": "SUBJECT_NOT_FOUND", "message": "missing"}}`)
	if _, err := FromInspect(body); err == nil {
		t.Fatal("a failed inspect response must not become ground truth")
	}
}

func TestWithValidateRejectsNonJSON(t *testing.T) {
	gt := buildGroundTruth(t)
	if _, err := gt.WithValidate([]byte("nope")); err == nil {
		t.Fatal("non-JSON validate output must be rejected")
	}
}

// Captured CLI output puts the JSON envelope first and PowerShell's own
// diagnostics ("[stderr] ...") after it, so the decoder must read the first
// value and ignore the rest rather than rejecting the whole file.
func TestDecodeIgnoresTrailingHostDiagnostics(t *testing.T) {
	body := append([]byte(inspectFixture), []byte("\n[stderr]\nThis system cannot create types...\n")...)
	gt, err := FromInspect(body)
	if err != nil {
		t.Fatalf("trailing diagnostics must not reject the response: %v", err)
	}
	if gt.SubjectID != "sales.order" {
		t.Fatalf("subject: %q", gt.SubjectID)
	}
}

func TestDecodeRejectsLeadingGarbage(t *testing.T) {
	body := []byte("warning: something first\n" + inspectFixture)
	if _, err := FromInspect(body); err == nil {
		t.Fatal("a body that does not start with JSON must be rejected")
	}
}

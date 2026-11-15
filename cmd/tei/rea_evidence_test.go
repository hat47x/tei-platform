package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tei/internal/externalanalysis"
)

// syntheticEvidence is a minimal record that carries every field the real tool
// requires of an Evidence. It is written by the test, not captured from a run:
// the real bytes are pinned in internal/externalanalysis/testdata, and this
// command's job is the wiring — flags in, envelope and exit code out.
const syntheticEvidence = `{
  "evidence_id": "ev_synthetic_intake_fixture",
  "subject": {"name": "synthetic", "digest": {"sha256": "000d9d6f7f73531725d781c5742c685611c36a74d6f2c29b73c36d5c61c53b54"}, "local_path": "/synthetic/target", "format": "directory"},
  "provider": {"id": "rea-javascript-application", "name": "REA JavaScript application analyzer", "version": "1"},
  "analysis_profile": null,
  "predicate_type": "rea.javascript-application-analysis",
  "operation": "analyze_javascript_application",
  "parameters": {"input_path": "/synthetic/target"},
  "raw_result": null,
  "normalized_result": {"summary": {"modules": 1}},
  "confidence": "derived",
  "authority": "shipped-artifact",
  "environment": null,
  "limitations": ["synthetic fixture: no real target was analyzed"],
  "relationship_coverage": null,
  "locations": [{"kind": "artifact-path", "path": "artifact-root"}],
  "evidence_links": []
}`

// syntheticView is one delivery in the framing the recorded 6.8.0 emits: two
// rows plus the terminal metadata record that names the parent.
const syntheticView = `{"observation_id":"obs_one","kind":"node","node_id":"n1"}
{"observation_id":"obs_two","kind":"node","node_id":"n2"}
{"$rea":"jsonl/v1","type":"metadata","records":{"path":["normalized_result","rows"],"emitted":2,"source":2},"context":{"evidence_id":"ev_synthetic_view","normalized_result":{"parent_evidence_id":"ev_synthetic_intake_fixture","coverage":{"status":"page","examined":2,"total":2,"exhausted":true},"limitations":["synthetic fixture"],"unknowns":[]}}}
`

func writeSynthetic(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func runReaEvidence(t *testing.T, args reaEvidenceOuterArgs) (int, cliResultEnvelope) {
	t.Helper()
	var out bytes.Buffer
	code, err := reaEvidenceJSON(&out, args, 0)
	if err != nil {
		t.Fatalf("the command must answer with one envelope, not an error: %v", err)
	}
	var report cliResultEnvelope
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("stdout is not one envelope: %v (%s)", err, out.String())
	}
	// ok reports that the command produced a receipt, not that the delivery
	// was complete: a partial receipt is still a receipt, and a caller that
	// reads ok as "complete" would launder the finding.
	if code == 0 && !report.OK {
		t.Fatalf("a command that produced a receipt must say ok: %s", out.String())
	}
	if code != 0 && report.OK {
		t.Fatalf("a failing command must not say ok: %s", out.String())
	}
	return code, report
}

func receiptOf(t *testing.T, report cliResultEnvelope) externalanalysis.Receipt {
	t.Helper()
	raw, err := json.Marshal(report.Result)
	if err != nil {
		t.Fatal(err)
	}
	var result reaEvidenceResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("the result is not a receipt: %v", err)
	}
	return result.Receipt
}

// TestReaEvidenceReceivesOneSavedEvidence is the normal case: the saved output
// of an analysis, received and measured, with the receipt carrying every field
// the coordination needs and promoting nothing.
func TestReaEvidenceReceivesOneSavedEvidence(t *testing.T) {
	dir := t.TempDir()
	evidence := writeSynthetic(t, dir, "analysis.json", syntheticEvidence)
	tool := filepath.Join(dir, "tool")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	code, report := runReaEvidence(t, reaEvidenceOuterArgs{evidencePath: evidence, toolPath: tool})
	if code != 0 {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	if report.SchemaVersion != "tei.cli.rea-evidence/v0" {
		t.Fatalf("schemaVersion=%s", report.SchemaVersion)
	}
	if report.Mode != "evidence" {
		t.Fatalf("mode=%s", report.Mode)
	}
	receipt := receiptOf(t, report)
	if receipt.State != externalanalysis.StateComplete {
		t.Fatalf("state=%s reason=%s", receipt.State, receipt.Reason)
	}
	if receipt.Evidence == nil || receipt.Evidence.ID != "ev_synthetic_intake_fixture" {
		t.Fatalf("evidence=%+v", receipt.Evidence)
	}
	if receipt.Source == nil || receipt.Source.Revision != "sha256:000d9d6f7f73531725d781c5742c685611c36a74d6f2c29b73c36d5c61c53b54" {
		t.Fatalf("source=%+v", receipt.Source)
	}
	if len(receipt.Source.Positions) != 1 || receipt.Source.Positions[0].Kind != "artifact-path" {
		t.Fatalf("positions=%+v", receipt.Source.Positions)
	}
	if receipt.Raw.Fingerprint == "" || receipt.Raw.Bytes != len(syntheticEvidence) {
		t.Fatalf("raw=%+v", receipt.Raw)
	}
	if !receipt.Tool.Present || receipt.Tool.ResolvedPath != tool {
		t.Fatalf("tool=%+v", receipt.Tool)
	}
	if receipt.Promotion != (externalanalysis.Promotion{}) {
		t.Fatalf("promotion=%+v: receiving an analysis promotes nothing", receipt.Promotion)
	}
	if receipt.AcquiredAt == "" {
		t.Fatal("the receipt must say when the bytes were received")
	}
}

// TestReaEvidenceInspectsTheSavedView is the second half of the loop: the saved
// projection, checked against the framing of the declared version.
func TestReaEvidenceInspectsTheSavedView(t *testing.T) {
	dir := t.TempDir()
	evidence := writeSynthetic(t, dir, "analysis.json", syntheticEvidence)
	view := writeSynthetic(t, dir, "view.jsonl", syntheticView)

	code, report := runReaEvidence(t, reaEvidenceOuterArgs{
		evidencePath: evidence, viewPath: view, providerVersion: "6.8.0",
	})
	if code != 0 {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	if report.Mode != "view" {
		t.Fatalf("mode=%s", report.Mode)
	}
	receipt := receiptOf(t, report)
	if receipt.View == nil || receipt.View.Framing != "jsonl/v1-rows-and-terminal" {
		t.Fatalf("view=%+v", receipt.View)
	}
	if receipt.View.Records != 2 || receipt.View.Emitted != 2 {
		t.Fatalf("view=%+v", receipt.View)
	}
	if !receipt.View.ParentLinked || receipt.View.ParentEvidenceID != receipt.Evidence.ID {
		t.Fatalf("the delivery must be linked to the received Evidence: %+v", receipt.View)
	}
}

// TestReaEvidenceReportsAnInterruptedView: a delivery whose terminal record is
// missing is exit 1 with EVIDENCE_PARTIAL, and the rows that arrived are still
// counted rather than dropped.
func TestReaEvidenceReportsAnInterruptedView(t *testing.T) {
	dir := t.TempDir()
	evidence := writeSynthetic(t, dir, "analysis.json", syntheticEvidence)
	records := strings.Split(strings.TrimRight(syntheticView, "\n"), "\n")
	view := writeSynthetic(t, dir, "view.jsonl", strings.Join(records[:len(records)-1], "\n")+"\n")

	code, report := runReaEvidence(t, reaEvidenceOuterArgs{
		evidencePath: evidence, viewPath: view, providerVersion: "6.8.0",
	})
	if code != 1 {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	if report.Error == nil || report.Error.Code != "EVIDENCE_PARTIAL" {
		t.Fatalf("error=%+v", report.Error)
	}
	if !strings.Contains(report.Error.Message, "終端metadata recordがありません") {
		t.Fatalf("message=%s", report.Error.Message)
	}
	if receipt := receiptOf(t, report); receipt.View.Records != 2 {
		t.Fatalf("the rows that arrived are still counted: %+v", receipt.View)
	}
}

// TestReaEvidenceReportsAVersionMismatch pins the version differential in both
// directions: a delivery from the other framing is not silently read as this
// one's.
func TestReaEvidenceReportsAVersionMismatch(t *testing.T) {
	dir := t.TempDir()
	evidence := writeSynthetic(t, dir, "analysis.json", syntheticEvidence)
	view := writeSynthetic(t, dir, "view.jsonl", syntheticView)

	// A single document read as the terminal framing: the terminal is missing,
	// so the delivery is incomplete rather than complete.
	single := `{"evidence_id":"ev_synthetic_view","operation":"inspect_analysis_view","normalized_result":{"parent_evidence_id":"ev_synthetic_intake_fixture","coverage":{"status":"page","exhausted":true},"limitations":[],"unknowns":[],"rows":[]}}`
	document := writeSynthetic(t, dir, "document.jsonl", single+"\n")
	code, report := runReaEvidence(t, reaEvidenceOuterArgs{
		evidencePath: evidence, viewPath: document, providerVersion: "6.8.0",
	})
	if code != 1 || report.Error == nil || report.Error.Code != "EVIDENCE_PARTIAL" {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}

	// The terminal framing read as the single-document version: the delivery
	// carries a control record that version does not produce, which is a
	// contradiction, not a partial one.
	code, report = runReaEvidence(t, reaEvidenceOuterArgs{
		evidencePath: evidence, viewPath: view, providerVersion: "6.7.0",
	})
	if code != 2 || report.Error == nil || report.Error.Code != "EVIDENCE_MISMATCH" {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
}

// TestReaEvidenceRefusesTheEnvelopeAsTheEvidenceBody: the CLI response wrapper
// standing in for the Evidence body is the confusion observed against the real
// tool, and it is a mismatch here rather than a request with a quirk.
func TestReaEvidenceRefusesTheEnvelopeAsTheEvidenceBody(t *testing.T) {
	dir := t.TempDir()
	evidence := writeSynthetic(t, dir, "analysis.json", syntheticEvidence)
	envelope := writeSynthetic(t, dir, "envelope.json", `{"ok":true,"data":`+syntheticEvidence+`,"meta":{"command":"analyze-javascript-application","duration":"1ms"}}`)
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(syntheticEvidence), &wrapper); err != nil {
		t.Fatal(err)
	}
	envelopeDocument := map[string]any{}
	if err := json.Unmarshal([]byte(mustRead(t, envelope)), &envelopeDocument); err != nil {
		t.Fatal(err)
	}
	request := map[string]any{
		"source": map[string]any{"kind": "inline", "evidence": envelopeDocument},
		"view":   map[string]any{"kind": "javascript-facts"},
	}
	requestPath := writeSynthetic(t, dir, "request.json", mustJSONText(t, request))

	code, report := runReaEvidence(t, reaEvidenceOuterArgs{
		evidencePath: evidence, requestPath: requestPath, providerVersion: "6.8.0",
	})
	if code != 2 || report.Error == nil || report.Error.Code != "EVIDENCE_MISMATCH" {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	if !strings.Contains(report.Error.Message, "標識") {
		t.Fatalf("message=%s", report.Error.Message)
	}
}

// TestReaEvidenceRefusesASourceKindTheVersionDoesNotAccept: 6.7.0 refused
// evidence-file with invalid_input in a real run, so a request naming it for
// that version is refused before it is sent.
func TestReaEvidenceRefusesASourceKindTheVersionDoesNotAccept(t *testing.T) {
	dir := t.TempDir()
	evidence := writeSynthetic(t, dir, "analysis.json", syntheticEvidence)
	requestPath := writeSynthetic(t, dir, "request.json", mustJSONText(t, map[string]any{
		"source": map[string]any{"kind": "evidence-file", "path": evidence},
		"view":   map[string]any{"kind": "javascript-facts"},
	}))

	code, report := runReaEvidence(t, reaEvidenceOuterArgs{
		evidencePath: evidence, requestPath: requestPath, providerVersion: "6.7.0",
	})
	if code != 2 || report.Error == nil || report.Error.Code != "EVIDENCE_MISMATCH" {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	if !strings.Contains(report.Error.Message, "evidence-file") {
		t.Fatalf("message=%s", report.Error.Message)
	}
}

// TestReaEvidenceDerivesARequestWithItsOwnDigest: the derived request is a new
// document with its own digest, written once and never overwritten, because it
// is the record of what was sent.
func TestReaEvidenceDerivesARequestWithItsOwnDigest(t *testing.T) {
	dir := t.TempDir()
	evidence := writeSynthetic(t, dir, "analysis.json", syntheticEvidence)
	requestOut := filepath.Join(dir, "derived", "request.json")

	code, report := runReaEvidence(t, reaEvidenceOuterArgs{
		evidencePath: evidence, requestOut: requestOut, providerVersion: "6.8.0",
	})
	if code != 0 {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	if report.Mode != "request" {
		t.Fatalf("mode=%s", report.Mode)
	}
	receipt := receiptOf(t, report)
	if receipt.Derived == nil || receipt.Derived.SourceKind != "inline" {
		t.Fatalf("derived=%+v", receipt.Derived)
	}
	if receipt.Derived.Fingerprint == receipt.Raw.Fingerprint {
		t.Fatal("the derived request must not share the Evidence digest")
	}
	written := mustRead(t, requestOut)
	var parsed struct {
		Source struct {
			Kind     string          `json:"kind"`
			Evidence json.RawMessage `json:"evidence"`
		} `json:"source"`
	}
	if err := json.Unmarshal([]byte(written), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Source.Kind != "inline" {
		t.Fatalf("source kind=%s", parsed.Source.Kind)
	}
	if string(parsed.Source.Evidence) != syntheticEvidence {
		t.Fatal("the request must embed the Evidence bytes as received")
	}

	// The derived request is a record of what was sent, so a second run
	// refuses to overwrite it rather than destroying the earlier record.
	code, report = runReaEvidence(t, reaEvidenceOuterArgs{
		evidencePath: evidence, requestOut: requestOut, providerVersion: "6.8.0",
	})
	if code != 2 || report.Error == nil || report.Error.Code != "EVIDENCE_REQUEST_WRITE_FAILED" {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}

	// The request that was written validates against the Evidence it came from.
	code, report = runReaEvidence(t, reaEvidenceOuterArgs{
		evidencePath: evidence, requestPath: requestOut, providerVersion: "6.8.0",
	})
	if code != 0 {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
}

// TestReaEvidenceReportsMissingAndUnusableInput: a missing flag, an unreadable
// file, bytes that are not the declared shape, and bytes that carry no Evidence
// are four different answers, because a caller fixes them differently.
func TestReaEvidenceReportsMissingAndUnusableInput(t *testing.T) {
	dir := t.TempDir()
	evidence := writeSynthetic(t, dir, "analysis.json", syntheticEvidence)

	code, report := runReaEvidence(t, reaEvidenceOuterArgs{})
	if code != 2 || report.Error == nil || report.Error.Code != "INVALID_ARGUMENT" {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}

	code, report = runReaEvidence(t, reaEvidenceOuterArgs{evidencePath: filepath.Join(dir, "absent.json")})
	if code != 2 || report.Error == nil || report.Error.Code != "EVIDENCE_READ_FAILED" {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}

	notJSON := writeSynthetic(t, dir, "not.json", "this is not json")
	code, report = runReaEvidence(t, reaEvidenceOuterArgs{evidencePath: notJSON})
	if code != 2 || report.Error == nil || report.Error.Code != "EVIDENCE_MISMATCH" {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}

	empty := writeSynthetic(t, dir, "empty.json", "")
	code, report = runReaEvidence(t, reaEvidenceOuterArgs{evidencePath: empty})
	if code != 2 || report.Error == nil || report.Error.Code != "EVIDENCE_MISSING" {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}

	// The evidence itself is fine; only the pins are wrong.
	code, report = runReaEvidence(t, reaEvidenceOuterArgs{evidencePath: evidence})
	if code != 0 {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
}

// TestReaEvidenceChecksTheRecordedPins: the receipt is compared against the
// values recorded when the Evidence was first admitted, and a moved source is
// named as such.
func TestReaEvidenceChecksTheRecordedPins(t *testing.T) {
	dir := t.TempDir()
	evidence := writeSynthetic(t, dir, "analysis.json", syntheticEvidence)

	code, report := runReaEvidence(t, reaEvidenceOuterArgs{
		evidencePath:        evidence,
		expectedEvidenceID:  "ev_synthetic_intake_fixture",
		expectedFingerprint: receiptFingerprint(t, evidence),
		expectedSourceRev:   "sha256:000d9d6f7f73531725d781c5742c685611c36a74d6f2c29b73c36d5c61c53b54",
	})
	if code != 0 {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}

	revised := strings.Replace(syntheticEvidence,
		"000d9d6f7f73531725d781c5742c685611c36a74d6f2c29b73c36d5c61c53b54",
		strings.Repeat("f", 64), 1)
	revisedPath := writeSynthetic(t, dir, "revised.json", revised)
	code, report = runReaEvidence(t, reaEvidenceOuterArgs{
		evidencePath:       revisedPath,
		expectedEvidenceID: "ev_synthetic_intake_fixture",
		expectedSourceRev:  "sha256:000d9d6f7f73531725d781c5742c685611c36a74d6f2c29b73c36d5c61c53b54",
	})
	if code != 1 || report.Error == nil || report.Error.Code != "EVIDENCE_PIN_MISMATCH" {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	if !strings.Contains(report.Error.Message, "revision") {
		t.Fatalf("message=%s", report.Error.Message)
	}
}

// TestReaEvidenceRecordsTheToolAbsence: an environment without the external
// tool still receives and inspects the saved bytes, and the receipt says the
// bytes could not have been produced here.
func TestReaEvidenceRecordsTheToolAbsence(t *testing.T) {
	dir := t.TempDir()
	evidence := writeSynthetic(t, dir, "analysis.json", syntheticEvidence)
	absent := filepath.Join(dir, "not-installed")

	code, report := runReaEvidence(t, reaEvidenceOuterArgs{evidencePath: evidence, toolPath: absent})
	if code != 0 {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	receipt := receiptOf(t, report)
	if receipt.State != externalanalysis.StateToolAbsent {
		t.Fatalf("state=%s reason=%s", receipt.State, receipt.Reason)
	}
	if receipt.Tool.Present || receipt.Tool.AbsentReason == "" {
		t.Fatalf("tool=%+v", receipt.Tool)
	}
	if receipt.Evidence == nil || receipt.Raw.Fingerprint == "" {
		t.Fatalf("the bytes are still the bytes: %+v", receipt.Evidence)
	}
}

// TestReaEvidenceRequiresADeclaredVersion: the framing and the accepted
// source.kind both belong to a version, so the inputs that need one refuse to
// run without it instead of defaulting.
func TestReaEvidenceRequiresADeclaredVersion(t *testing.T) {
	dir := t.TempDir()
	evidence := writeSynthetic(t, dir, "analysis.json", syntheticEvidence)
	view := writeSynthetic(t, dir, "view.jsonl", syntheticView)

	for name, args := range map[string]reaEvidenceOuterArgs{
		"view without version":        {evidencePath: evidence, viewPath: view},
		"request-out without version": {evidencePath: evidence, requestOut: filepath.Join(dir, "r.json")},
	} {
		t.Run(name, func(t *testing.T) {
			code, report := runReaEvidence(t, args)
			if code != 2 || report.Error == nil || report.Error.Code != "INVALID_ARGUMENT" {
				t.Fatalf("code=%d error=%+v", code, report.Error)
			}
		})
	}

	// An unobserved version is refused for the same reason: the expectation
	// was never recorded, so there is nothing to check against.
	code, report := runReaEvidence(t, reaEvidenceOuterArgs{
		evidencePath: evidence, viewPath: view, providerVersion: "9.9.9",
	})
	if code != 2 || report.Error == nil || report.Error.Code != "INVALID_ARGUMENT" {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	if !strings.Contains(report.Error.Message, "9.9.9") {
		t.Fatalf("message=%s", report.Error.Message)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// TestReaEvidenceProjectsTheSelectedCandidates: the CLI entry is the read-only
// consumer the T1-A exit asks for — it receives one observation and one directed
// relation from the same attempt, losslessly, and says what it does not grant.
func TestReaEvidenceProjectsTheSelectedCandidates(t *testing.T) {
	evidence := filepath.Join("..", "..", "internal", "externalanalysis", "testdata", "rea-6.8.0", "analysis.json")

	code, report := runReaEvidence(t, reaEvidenceOuterArgs{
		evidencePath:    evidence,
		providerVersion: "6.8.0",
		observationID:   "jsrg_node_237478f2a27c9614a522cbfc7ab60fa96943a458d164945d25af634f1240b652",
		relationID:      "jsrg_relation_0f6757a241c56fd84ecaec91f538d8297d222eb5625e4cdbcb2de7603f330718",
	})
	if code != 0 {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	if report.Mode != "candidate" {
		t.Fatalf("mode=%s", report.Mode)
	}
	raw, err := json.Marshal(report.Result)
	if err != nil {
		t.Fatal(err)
	}
	var result reaEvidenceResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result.Candidates == nil {
		t.Fatal("the candidate projection is missing")
	}
	view := result.Candidates
	if view.Attempt.EvidenceID != "ev_a88c70bcd066b2f04fdf97cb349369f0779b9c68cc4ac6c0e5f38f3539477a93" {
		t.Fatalf("attempt evidence id=%s", view.Attempt.EvidenceID)
	}
	if view.Attempt.ReceivedDigest == "" {
		t.Fatal("the attempt must name the digest of the bytes that were read")
	}
	if view.Observation == nil || view.Relation == nil {
		t.Fatal("both candidates must be projected")
	}
	if view.Relation.Derived {
		t.Fatal("the producer states its own relation identity; a derived handle must not replace it")
	}
	if view.Coverage == nil || len(view.Unknowns) == 0 {
		t.Fatalf("coverage=%+v unknowns=%d: the producer's unknowns must survive the projection", view.Coverage, len(view.Unknowns))
	}
	for _, want := range []string{
		"external-candidate-is-not-accepted-meaning",
		"external-identity-is-not-a-tei-semantic-subject",
	} {
		if !containsLimit(view.AuthorityLimits, want) {
			t.Fatalf("authorityLimits=%v want %s", view.AuthorityLimits, want)
		}
	}

	// A selection the Evidence does not carry is exit 1 with a named failure,
	// not a silent empty projection.
	code, report = runReaEvidence(t, reaEvidenceOuterArgs{
		evidencePath:    evidence,
		providerVersion: "6.8.0",
		observationID:   "jsrg_node_absent",
		relationID:      "jsrg_relation_absent",
	})
	if code != 1 || report.Error == nil {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	if !strings.Contains(report.Error.Message, "jsrg_node_absent") {
		t.Fatalf("message=%s: the failure must name what was not found", report.Error.Message)
	}

	// The projection needs a declared version, and it needs both selections.
	code, report = runReaEvidence(t, reaEvidenceOuterArgs{
		evidencePath:  evidence,
		observationID: "jsrg_node_absent",
		relationID:    "jsrg_relation_absent",
	})
	if code != 2 || report.Error == nil || report.Error.Code != "INVALID_ARGUMENT" {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	code, report = runReaEvidence(t, reaEvidenceOuterArgs{
		evidencePath:    evidence,
		providerVersion: "6.8.0",
		observationID:   "jsrg_node_absent",
	})
	if code != 2 || report.Error == nil || report.Error.Code != "INVALID_ARGUMENT" {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
}

func containsLimit(limits []string, want string) bool {
	for _, limit := range limits {
		if limit == want {
			return true
		}
	}
	return false
}

func mustJSONText(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func receiptFingerprint(t *testing.T, path string) string {
	t.Helper()
	receipt := externalanalysis.Receive([]byte(mustRead(t, path)), externalanalysis.Tool{Name: "rea", Present: true}, time.Now())
	return receipt.Raw.Fingerprint
}

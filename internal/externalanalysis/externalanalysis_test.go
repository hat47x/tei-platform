package externalanalysis_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"tei/internal/externalanalysis"
)

// The fixtures are real captured output of the REA CLI, saved under testdata
// with the run record in testdata/manifest.json. The expectations below are
// pinned by hand, so a fixture that changes fails here instead of quietly
// redefining the contract.

const (
	evidenceID67  = "ev_c9d5026bb51c7e100892eb076ebb4b812ea0f4a7874dcf4c835d33098ba9fb85"
	evidenceID68  = "ev_a88c70bcd066b2f04fdf97cb349369f0779b9c68cc4ac6c0e5f38f3539477a93"
	viewID67      = "ev_7d13b168b2f95c3f2725c13590e0e1d293a38eef43c79756ecfced6cff6150dc"
	viewID68      = "ev_16fb8cd4b854f983b4e5aea1461d5a3b1fa4a19ad7d46e618fee2a548eca3e8d"
	sourceRev     = "sha256:000d9d6f7f73531725d781c5742c685611c36a74d6f2c29b73c36d5c61c53b54"
	fingerprint67 = "sha256:3a88503c353ecca1edad5715921ede7398f0eca86457316871dde85cd73bdd5e"
	fingerprint68 = "sha256:7ea3780d8dc23d00c380145af62d63bf194e39decbaefab3d82c5a3b4134bba9"
)

var acquisition = time.Date(2026, 11, 11, 5, 5, 0, 0, time.UTC)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", filepath.FromSlash(name)))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return raw
}

// presentTool declares a tool this environment could have run. The tests pass
// it explicitly rather than probing the ambient PATH, so a machine without the
// tool installed still exercises the byte-level contract.
func presentTool(version string) externalanalysis.Tool {
	return externalanalysis.Tool{Name: "rea", Version: version, Present: true, ResolvedPath: "test"}
}

func absentTool() externalanalysis.Tool {
	return externalanalysis.Tool{Name: "rea", Present: false, AbsentReason: `exec: "rea": executable file not found in $PATH`}
}

func decode(t *testing.T, raw []byte) string {
	t.Helper()
	if len(raw) >= 2 && raw[0] == 0xFF && raw[1] == 0xFE {
		units := make([]uint16, 0, (len(raw)-2)/2)
		for i := 2; i+1 < len(raw); i += 2 {
			units = append(units, uint16(raw[i])|uint16(raw[i+1])<<8)
		}
		return string(utf16.Decode(units))
	}
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		return string(raw[3:])
	}
	return string(raw)
}

// lines splits a delivery into its records the way the intake does: on LF, with
// a trailing empty line dropped.
func lines(t *testing.T, raw []byte) []string {
	t.Helper()
	split := strings.Split(decode(t, raw), "\n")
	for len(split) > 0 && split[len(split)-1] == "" {
		split = split[:len(split)-1]
	}
	return split
}

// reencode builds a derived delivery from records. It is deliberately UTF-8
// with LF: the derived copies below are the test's own documents, and the
// encoding rules are covered by the real fixtures.
func reencode(t *testing.T, records []string) []byte {
	t.Helper()
	return []byte(strings.Join(records, "\n") + "\n")
}

func mustJSON(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("fixture is not a JSON object: %v", err)
	}
	return value
}

// TestReceiptSchemaVersionIsExplicitAndStable pins the external receipt shape
// independently of the producer's Evidence format. A received envelope is not
// an Owner-accepted semantic model or an authorization to execute the result.
func TestReceiptSchemaVersionIsExplicitAndStable(t *testing.T) {
	raw := fixture(t, "rea-6.8.0/analysis.json")
	receipt := externalanalysis.Receive(raw, presentTool("6.8.0"), acquisition)
	if receipt.State != externalanalysis.StateComplete {
		t.Fatalf("receipt state=%s reason=%s", receipt.State, receipt.Reason)
	}
	const expected = "tei.externalanalysis.receipt/v0"
	if receipt.SchemaVersion != expected {
		t.Fatalf("schema version=%q want %q", receipt.SchemaVersion, expected)
	}
	if receipt.Promotion.AcceptedMeaning || receipt.Promotion.CanonicalSubject || receipt.Promotion.DatabaseObserved {
		t.Fatalf("receipt must not silently promote external evidence: %+v", receipt.Promotion)
	}
}

// TestReceiveRealSavedEvidence is the normal case: the saved output of a real
// analysis run, received and measured. Every field the coordination needs is
// asserted against the pinned value, including the digest of the bytes as they
// arrived and the revision of the analyzed source.
func TestReceiveRealSavedEvidence(t *testing.T) {
	for _, testCase := range []struct {
		version     string
		evidenceID  string
		fingerprint string
	}{
		{"6.7.0", evidenceID67, fingerprint67},
		{"6.8.0", evidenceID68, fingerprint68},
	} {
		t.Run(testCase.version, func(t *testing.T) {
			raw := fixture(t, "rea-"+testCase.version+"/analysis.json")
			receipt := externalanalysis.Receive(raw, presentTool(testCase.version), acquisition)

			if receipt.State != externalanalysis.StateComplete {
				t.Fatalf("state=%s reason=%s", receipt.State, receipt.Reason)
			}
			if receipt.Evidence == nil {
				t.Fatal("the receipt carries no Evidence")
			}
			if receipt.Evidence.ID != testCase.evidenceID {
				t.Fatalf("evidence id=%s want %s", receipt.Evidence.ID, testCase.evidenceID)
			}
			if receipt.Evidence.Operation != "analyze_javascript_application" {
				t.Fatalf("operation=%s", receipt.Evidence.Operation)
			}
			if receipt.Evidence.Provider.ID != "rea-javascript-application" {
				t.Fatalf("provider=%s", receipt.Evidence.Provider.ID)
			}
			if !receipt.Evidence.Envelope {
				t.Fatal("the saved bytes carried the CLI response wrapper; the receipt must say so")
			}
			if receipt.Raw.Fingerprint != testCase.fingerprint {
				t.Fatalf("raw fingerprint=%s want %s", receipt.Raw.Fingerprint, testCase.fingerprint)
			}
			if receipt.Raw.Encoding != "utf-16le" {
				t.Fatalf("encoding=%s: the real run on this host wrote UTF-16LE with a BOM", receipt.Raw.Encoding)
			}
			if receipt.Raw.Bytes != len(raw) {
				t.Fatalf("bytes=%d want %d", receipt.Raw.Bytes, len(raw))
			}
			if receipt.Source == nil || receipt.Source.Revision != sourceRev {
				t.Fatalf("source revision=%v want %s", receipt.Source, sourceRev)
			}
			if receipt.Source.Subject != "rea-t005" {
				t.Fatalf("subject=%s", receipt.Source.Subject)
			}
			if len(receipt.Source.Positions) == 0 || receipt.Source.Positions[0].Kind != "artifact-path" {
				t.Fatalf("positions=%+v", receipt.Source.Positions)
			}
			if receipt.Envelope == nil || receipt.Envelope.Command != "analyze-javascript-application" {
				t.Fatalf("envelope=%+v: the producer's own execution record is missing", receipt.Envelope)
			}
			if receipt.AcquiredAt != "2026-11-11T05:05:00Z" {
				t.Fatalf("acquiredAt=%s", receipt.AcquiredAt)
			}
			if receipt.EvidenceBody() == nil {
				t.Fatal("the Evidence body must stay available for deriving a request")
			}
			if receipt.Promotion != (externalanalysis.Promotion{}) {
				t.Fatalf("promotion=%+v: receiving an analysis promotes nothing", receipt.Promotion)
			}
		})
	}
}

// TestInspectRealSavedViewIsComplete is the normal case for the second half of
// the loop: the saved projection of that Evidence, checked against the framing
// the recorded version emits. The two versions frame the same projection
// differently, and both are complete for their own version.
func TestInspectRealSavedViewIsComplete(t *testing.T) {
	for _, testCase := range []struct {
		version    string
		records    int
		control    string
		viewID     string
		emitted    int
		collection string
	}{
		{version: "6.7.0", records: 1, control: "none", viewID: viewID67},
		{version: "6.8.0", records: 7, control: "metadata", viewID: viewID68, emitted: 7, collection: "normalized_result,rows"},
	} {
		t.Run(testCase.version, func(t *testing.T) {
			raw := fixture(t, "rea-"+testCase.version+"/analysis.json")
			view := fixture(t, "rea-"+testCase.version+"/view.jsonl")
			framing, ok := externalanalysis.FramingFor(testCase.version)
			if !ok {
				t.Fatalf("%s must be an observed version", testCase.version)
			}
			receipt := externalanalysis.Receive(raw, presentTool(testCase.version), acquisition).WithView(view, framing)

			if receipt.State != externalanalysis.StateComplete {
				t.Fatalf("state=%s reason=%s", receipt.State, receipt.Reason)
			}
			if receipt.View == nil {
				t.Fatal("the receipt carries no view")
			}
			if receipt.View.Records != testCase.records {
				t.Fatalf("records=%d want %d", receipt.View.Records, testCase.records)
			}
			if receipt.View.Control != testCase.control {
				t.Fatalf("control=%s want %s", receipt.View.Control, testCase.control)
			}
			if receipt.View.EvidenceID != testCase.viewID {
				t.Fatalf("view evidence id=%s want %s: the view is its own record, not the parent", receipt.View.EvidenceID, testCase.viewID)
			}
			if receipt.View.EvidenceID == receipt.Evidence.ID {
				t.Fatal("the view's Evidence ID must not be the parent's")
			}
			if receipt.View.Emitted != testCase.emitted {
				t.Fatalf("emitted=%d want %d", receipt.View.Emitted, testCase.emitted)
			}
			if testCase.collection != "" && strings.Join(receipt.View.Collection, ",") != testCase.collection {
				t.Fatalf("collection=%v want %s", receipt.View.Collection, testCase.collection)
			}
			if !receipt.View.ParentLinked {
				t.Fatal("the delivery names the received Evidence as its parent; the link must be checked")
			}
			if receipt.View.ParentEvidenceID != receipt.Evidence.ID {
				t.Fatalf("parent=%s want %s", receipt.View.ParentEvidenceID, receipt.Evidence.ID)
			}
			if receipt.View.Coverage == nil || receipt.View.Coverage.Status != "page" {
				t.Fatalf("coverage=%+v", receipt.View.Coverage)
			}
			if receipt.View.Coverage.Exhausted == nil || !*receipt.View.Coverage.Exhausted {
				t.Fatalf("the real delivery exhausted its page: %+v", receipt.View.Coverage)
			}
			if len(receipt.View.Limitations) == 0 {
				t.Fatal("the producer declared limitations; a receipt that drops them overstates the delivery")
			}
			if receipt.View.Encoding != "utf-16le" {
				t.Fatalf("view encoding=%s", receipt.View.Encoding)
			}
		})
	}
}

// TestInterruptedDeliveryIsPartial covers the two ways a delivery ends early:
// the terminal record is missing, and a line is cut mid-record. Neither is a
// complete receipt, and the rows that did arrive are not the whole thing.
func TestInterruptedDeliveryIsPartial(t *testing.T) {
	framing, ok := externalanalysis.FramingFor("6.8.0")
	if !ok {
		t.Fatal("6.8.0 must be an observed version")
	}
	raw := fixture(t, "rea-6.8.0/view.jsonl")
	evidence := externalanalysis.Receive(fixture(t, "rea-6.8.0/analysis.json"), presentTool("6.8.0"), acquisition)

	t.Run("terminal record missing", func(t *testing.T) {
		records := lines(t, raw)
		interrupted := reencode(t, records[:len(records)-1])
		view := evidence.WithView(interrupted, framing)
		if view.State != externalanalysis.StatePartial {
			t.Fatalf("state=%s reason=%s", view.State, view.Reason)
		}
		if !strings.Contains(view.Reason, "終端metadata recordがありません") {
			t.Fatalf("reason=%s", view.Reason)
		}
		if view.View.Records != 7 {
			t.Fatalf("records=%d: the rows that arrived are still counted", view.View.Records)
		}
	})

	t.Run("line cut mid record", func(t *testing.T) {
		cut := raw[:len(raw)-100]
		view := evidence.WithView(cut, framing)
		if view.State != externalanalysis.StatePartial {
			t.Fatalf("state=%s reason=%s", view.State, view.Reason)
		}
		if !strings.Contains(view.Reason, "途中切断") {
			t.Fatalf("reason=%s", view.Reason)
		}
	})
}

// TestVersionFramingIsNotInterchangeable pins the differential that was
// observed against the real tool: 6.7.0 writes one document with no terminal,
// 6.8.0 writes rows plus a terminal. Feeding each version's delivery to the
// other version's expectation fails in both directions, and differently: a
// missing terminal is an incomplete delivery, an undeclared terminal is a
// contradiction.
func TestVersionFramingIsNotInterchangeable(t *testing.T) {
	legacy, _ := externalanalysis.FramingFor("6.7.0")
	control, _ := externalanalysis.FramingFor("6.8.0")
	view67 := fixture(t, "rea-6.7.0/view.jsonl")
	view68 := fixture(t, "rea-6.8.0/view.jsonl")
	evidence := externalanalysis.Receive(fixture(t, "rea-6.8.0/analysis.json"), presentTool("6.8.0"), acquisition)

	old := evidence.WithView(view67, control)
	if old.State != externalanalysis.StatePartial {
		t.Fatalf("a 6.7.0 delivery read as 6.8.0 is not complete: state=%s reason=%s", old.State, old.Reason)
	}
	newer := evidence.WithView(view68, legacy)
	if newer.State != externalanalysis.StateMismatch {
		t.Fatalf("a 6.8.0 delivery read as 6.7.0 contradicts the declaration: state=%s reason=%s", newer.State, newer.Reason)
	}
}

// TestUndeclaredFramingIsNotGuessed: an unobserved tool version produces a
// framing nobody declared, and the inspection says so instead of picking the
// likelier interpretation.
func TestUndeclaredFramingIsNotGuessed(t *testing.T) {
	framing, ok := externalanalysis.FramingFor("9.9.9")
	if ok {
		t.Fatal("an unobserved version must not return an expectation")
	}
	receipt := externalanalysis.Receive(fixture(t, "rea-6.8.0/analysis.json"), presentTool("6.8.0"), acquisition).
		WithView(fixture(t, "rea-6.8.0/view.jsonl"), framing)
	if receipt.State != externalanalysis.StatePartial {
		t.Fatalf("state=%s reason=%s", receipt.State, receipt.Reason)
	}
	if !strings.Contains(receipt.Reason, "宣言されていない") {
		t.Fatalf("reason=%s", receipt.Reason)
	}
}

// TestDuplicateIdentityIsMismatch takes the real delivery and makes two rows
// name the same observation. The counts still add up, so only the identity
// check catches it.
func TestDuplicateIdentityIsMismatch(t *testing.T) {
	framing, _ := externalanalysis.FramingFor("6.8.0")
	records := lines(t, fixture(t, "rea-6.8.0/view.jsonl"))
	if len(records) < 3 {
		t.Fatalf("fixture has %d records", len(records))
	}
	first := mustJSON(t, []byte(records[0]))
	second := mustJSON(t, []byte(records[1]))
	identity, ok := first["observation_id"].(string)
	if !ok || identity == "" {
		t.Fatalf("the first record has no observation_id: %v", first)
	}
	if second["observation_id"] == identity {
		t.Fatal("the fixture already carries a duplicate; the test would prove nothing")
	}
	second["observation_id"] = identity
	mutated, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	records[1] = string(mutated)
	receipt := externalanalysis.Receive(fixture(t, "rea-6.8.0/analysis.json"), presentTool("6.8.0"), acquisition).
		WithView(reencode(t, records), framing)
	if receipt.State != externalanalysis.StateMismatch {
		t.Fatalf("state=%s reason=%s", receipt.State, receipt.Reason)
	}
	if len(receipt.View.Duplicates) != 1 || receipt.View.Duplicates[0] != identity {
		t.Fatalf("duplicates=%v want [%s]", receipt.View.Duplicates, identity)
	}
}

// TestEnvelopeStandingInForTheEvidenceIsRefused covers the confusion that was
// observed against the real tool: the CLI response wrapper handed over as the
// Evidence body. It is refused when deriving a request, and reported as a
// mismatch when validating one.
func TestEnvelopeStandingInForTheEvidenceIsRefused(t *testing.T) {
	raw := fixture(t, "rea-6.8.0/analysis.json")
	receipt := externalanalysis.Receive(raw, presentTool("6.8.0"), acquisition)
	envelope := mustJSON(t, []byte(decode(t, raw)))

	if _, err := externalanalysis.DeriveViewRequest(raw, viewRequest(), acceptedKinds68(t)); err == nil {
		t.Fatal("deriving a request from the response envelope must be refused")
	} else if !strings.Contains(err.Error(), "包装") {
		t.Fatalf("the refusal must name the confusion: %v", err)
	}

	request := map[string]any{
		"source": map[string]any{"kind": "inline", "evidence": envelope},
		"view":   map[string]any{"kind": "javascript-facts"},
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	checked := receipt.WithRequest(encoded, acceptedKinds68(t))
	if checked.State != externalanalysis.StateMismatch {
		t.Fatalf("state=%s reason=%s", checked.State, checked.Reason)
	}
	if !strings.Contains(checked.Reason, "標識") {
		t.Fatalf("reason=%s", checked.Reason)
	}
}

// TestSourceKindTheVersionDoesNotAcceptIsMismatch: the accepted set travels with
// the receipt, and a request naming a kind outside it is not sent. 6.7.0
// refused evidence-file with invalid_input; that refusal is a real run, saved
// under testdata.
func TestSourceKindTheVersionDoesNotAcceptIsMismatch(t *testing.T) {
	kinds67, ok := externalanalysis.AcceptedSourceKinds("6.7.0")
	if !ok {
		t.Fatal("6.7.0 must be an observed version")
	}
	if len(kinds67) != 2 || kinds67[0] != "retained-evidence" || kinds67[1] != "inline" {
		t.Fatalf("6.7.0 accepted kinds=%v", kinds67)
	}
	kinds68, ok := externalanalysis.AcceptedSourceKinds("6.8.0")
	if !ok {
		t.Fatal("6.8.0 must be an observed version")
	}
	if len(kinds68) != 3 || kinds68[2] != "evidence-file" {
		t.Fatalf("6.8.0 accepted kinds=%v: the version difference is the point", kinds68)
	}
	if _, ok := externalanalysis.AcceptedSourceKinds("9.9.9"); ok {
		t.Fatal("an unobserved version must not return an accepted set")
	}

	request := map[string]any{
		"source": map[string]any{"kind": "evidence-file", "path": "analysis.json"},
		"view":   map[string]any{"kind": "javascript-facts"},
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	receipt := externalanalysis.Receive(fixture(t, "rea-6.7.0/analysis.json"), presentTool("6.7.0"), acquisition).
		WithRequest(encoded, kinds67)
	if receipt.State != externalanalysis.StateMismatch {
		t.Fatalf("state=%s reason=%s", receipt.State, receipt.Reason)
	}
	if !strings.Contains(receipt.Reason, "evidence-file") {
		t.Fatalf("reason=%s", receipt.Reason)
	}
}

// TestDerivedRequestIsSeparateFromTheEvidence: the request is a new document
// with its own digest, embedding the Evidence bytes exactly as they were
// received. A request that shared the Evidence's digest would mean the Evidence
// was passed on unchanged.
func TestDerivedRequestIsSeparateFromTheEvidence(t *testing.T) {
	raw := fixture(t, "rea-6.8.0/analysis.json")
	receipt := externalanalysis.Receive(raw, presentTool("6.8.0"), acquisition)
	request, err := externalanalysis.DeriveViewRequest(receipt.EvidenceBody(), viewRequest(), acceptedKinds68(t))
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	derived := receipt.WithRequest(request, acceptedKinds68(t))
	if derived.State != externalanalysis.StateComplete {
		t.Fatalf("state=%s reason=%s", derived.State, derived.Reason)
	}
	if derived.Derived == nil {
		t.Fatal("the derived request is not recorded")
	}
	if derived.Derived.Fingerprint == derived.Raw.Fingerprint {
		t.Fatal("the derived request must not share the Evidence digest")
	}
	if derived.Derived.SourceKind != "inline" {
		t.Fatalf("source kind=%s", derived.Derived.SourceKind)
	}
	var parsed struct {
		Source struct {
			Kind     string          `json:"kind"`
			Evidence json.RawMessage `json:"evidence"`
		} `json:"source"`
		View map[string]any `json:"view"`
	}
	if err := json.Unmarshal(request, &parsed); err != nil {
		t.Fatal(err)
	}
	if string(parsed.Source.Evidence) != string(receipt.EvidenceBody()) {
		t.Fatal("the request must embed the Evidence bytes as received, not a re-serialized copy")
	}
	if parsed.View["kind"] != "javascript-facts" {
		t.Fatalf("view=%v", parsed.View)
	}
}

// TestRequestForADifferentEvidenceIsMismatch: a request that embeds another
// Evidence than the one received is not a request about this receipt.
func TestRequestForADifferentEvidenceIsMismatch(t *testing.T) {
	raw := fixture(t, "rea-6.8.0/analysis.json")
	receipt := externalanalysis.Receive(raw, presentTool("6.8.0"), acquisition)
	other := fixture(t, "rea-6.7.0/analysis.json")
	otherEnvelope := mustJSON(t, []byte(decode(t, other)))
	request := map[string]any{
		"source": map[string]any{"kind": "inline", "evidence": otherEnvelope["data"]},
		"view":   map[string]any{"kind": "javascript-facts"},
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	checked := receipt.WithRequest(encoded, acceptedKinds68(t))
	if checked.State != externalanalysis.StateMismatch {
		t.Fatalf("state=%s reason=%s", checked.State, checked.Reason)
	}
	if !strings.Contains(checked.Reason, evidenceID67) {
		t.Fatalf("the reason must name the Evidence the request actually embeds: %s", checked.Reason)
	}
}

// TestMissingEvidenceIsNotComplete: empty bytes, and an envelope whose record
// is missing its required content, are both 未取得 — not a receipt with holes
// papered over.
func TestMissingEvidenceIsNotComplete(t *testing.T) {
	empty := externalanalysis.Receive(nil, presentTool("6.8.0"), acquisition)
	if empty.State != externalanalysis.StateMissing {
		t.Fatalf("state=%s", empty.State)
	}

	partial := map[string]any{"ok": true, "data": map[string]any{"evidence_id": "ev_partial", "operation": "analyze_javascript_application"}}
	encoded, err := json.Marshal(partial)
	if err != nil {
		t.Fatal(err)
	}
	receipt := externalanalysis.Receive(encoded, presentTool("6.8.0"), acquisition)
	if receipt.State != externalanalysis.StateMissing {
		t.Fatalf("state=%s reason=%s", receipt.State, receipt.Reason)
	}
	for _, want := range []string{"subject", "provider", "normalized_result", "confidence", "authority"} {
		if !strings.Contains(receipt.Reason, want) {
			t.Fatalf("reason must name the absent field %s: %s", want, receipt.Reason)
		}
	}
}

// TestBytesThatAreNotTheDeclaredShapeAreMismatch: not JSON, and JSON that is
// not an analysis output at all, are 形式不一致 — a different shape, not a
// missing one.
func TestBytesThatAreNotTheDeclaredShapeAreMismatch(t *testing.T) {
	for name, raw := range map[string][]byte{
		"not json":        []byte("this is not json"),
		"a json array":    []byte(`[1,2,3]`),
		"another object":  []byte(`{"hello":"world"}`),
		"ok without data": []byte(`{"ok":true}`),
	} {
		t.Run(name, func(t *testing.T) {
			receipt := externalanalysis.Receive(raw, presentTool("6.8.0"), acquisition)
			if receipt.State != externalanalysis.StateMismatch {
				t.Fatalf("state=%s reason=%s", receipt.State, receipt.Reason)
			}
			if receipt.Reason == "" {
				t.Fatal("a mismatch must say what was observed")
			}
		})
	}
}

// TestSourceRevisionChangeIsNotTheSameObservation: the same Evidence ID
// received after the analyzed source was revised is a different observation.
// The pin check names the revision, so a caller does not quietly re-admit it.
func TestSourceRevisionChangeIsNotTheSameObservation(t *testing.T) {
	raw := fixture(t, "rea-6.8.0/analysis.json")
	first := externalanalysis.Receive(raw, presentTool("6.8.0"), acquisition)
	pins := externalanalysis.Pins{
		EvidenceID:     first.Evidence.ID,
		Fingerprint:    first.Raw.Fingerprint,
		SourceRevision: first.Source.Revision,
	}
	if err := first.VerifyPins(pins); err != nil {
		t.Fatalf("the receipt it was built from must match its own pins: %v", err)
	}

	// The digest is replaced in the decoded text: the fixture is UTF-16LE on
	// this host, so an ASCII replace over the raw bytes would find nothing.
	text := decode(t, raw)
	revised := strings.Replace(text, "000d9d6f7f73531725d781c5742c685611c36a74d6f2c29b73c36d5c61c53b54", strings.Repeat("f", 64), 1)
	if revised == text {
		t.Fatal("the fixture does not carry the subject digest the test expects")
	}
	second := externalanalysis.Receive([]byte(revised), presentTool("6.8.0"), acquisition)
	if second.Evidence == nil {
		t.Fatalf("the revised bytes are still an Evidence record: state=%s reason=%s", second.State, second.Reason)
	}
	if second.Evidence.ID != first.Evidence.ID {
		t.Fatalf("the Evidence ID changed too: %s", second.Evidence.ID)
	}
	err := second.VerifyPins(pins)
	if err == nil {
		t.Fatal("a revised source must not pass the recorded pins")
	}
	if !strings.Contains(err.Error(), "revision") {
		t.Fatalf("the failure must name the revision change: %v", err)
	}

	// One extra UTF-16LE line ending keeps the byte count even, so the bytes
	// still decode; only the digest moved.
	replaced := append(append([]byte(nil), raw...), '\n', 0)
	third := externalanalysis.Receive(replaced, presentTool("6.8.0"), acquisition)
	if third.Evidence == nil {
		t.Fatalf("the replaced bytes are still an Evidence record: state=%s reason=%s", third.State, third.Reason)
	}
	err = third.VerifyPins(pins)
	if err == nil {
		t.Fatal("replaced bytes must not pass the recorded fingerprint")
	}
	if !strings.Contains(err.Error(), "digest") {
		t.Fatalf("the failure must name the digest: %v", err)
	}
}

// TestToolAbsenceIsRecordedRatherThanPassed: in an environment without the
// external tool, the receipt says the bytes are complete and that they could
// not have been produced here. Both halves are needed; either alone misleads.
func TestToolAbsenceIsRecordedRatherThanPassed(t *testing.T) {
	raw := fixture(t, "rea-6.8.0/analysis.json")
	receipt := externalanalysis.Receive(raw, absentTool(), acquisition)
	if receipt.State != externalanalysis.StateToolAbsent {
		t.Fatalf("state=%s reason=%s", receipt.State, receipt.Reason)
	}
	if receipt.Tool.Present {
		t.Fatal("the tool section must record the absence")
	}
	if receipt.Tool.AbsentReason == "" {
		t.Fatal("the absence needs a reason a reader can act on")
	}
	if receipt.Evidence == nil || receipt.Evidence.ID != evidenceID68 {
		t.Fatalf("the bytes are still the bytes: %+v", receipt.Evidence)
	}
	if receipt.Raw.Fingerprint != fingerprint68 {
		t.Fatalf("fingerprint=%s", receipt.Raw.Fingerprint)
	}

	// A delivery with a problem of its own keeps that problem as its state:
	// the missing tool is a caveat on a whole receipt, not a way to launder an
	// interrupted one.
	framing, _ := externalanalysis.FramingFor("6.8.0")
	records := lines(t, fixture(t, "rea-6.8.0/view.jsonl"))
	interrupted := externalanalysis.Receive(fixture(t, "rea-6.8.0/analysis.json"), absentTool(), acquisition).
		WithView(reencode(t, records[:len(records)-1]), framing)
	if interrupted.State != externalanalysis.StatePartial {
		t.Fatalf("state=%s reason=%s", interrupted.State, interrupted.Reason)
	}
	if interrupted.Tool.Present {
		t.Fatal("the tool section must still record the absence")
	}
}

// TestRealRefusalDeliveriesAreFailed: the tool's own refusals, captured from
// real runs, are receipts that say the tool failed — not empty successes, and
// not evidence at all.
func TestRealRefusalDeliveriesAreFailed(t *testing.T) {
	for _, testCase := range []struct {
		file    string
		version string
		code    string
	}{
		{"rea-6.7.0/error-source-kind.jsonl", "6.7.0", "invalid_request"},
		{"rea-6.7.0/error-envelope-as-evidence.jsonl", "6.7.0", "invalid_request"},
		{"rea-6.8.0/error-source-kind.jsonl", "6.8.0", "invalid_request"},
		{"rea-6.8.0/error-envelope-as-evidence.jsonl", "6.8.0", "invalid_request"},
		{"rea-6.8.0/error-evidence-file-utf16.jsonl", "6.8.0", "evidence_integrity_mismatch"},
	} {
		t.Run(testCase.file, func(t *testing.T) {
			raw := fixture(t, testCase.file)
			framing, _ := externalanalysis.FramingFor(testCase.version)
			receipt := externalanalysis.Receive(raw, presentTool(testCase.version), acquisition).WithView(raw, framing)
			if receipt.State != externalanalysis.StateFailed {
				t.Fatalf("state=%s reason=%s", receipt.State, receipt.Reason)
			}
			if receipt.Failure == nil || receipt.Failure.Code != testCase.code {
				t.Fatalf("failure=%+v want code %s", receipt.Failure, testCase.code)
			}
			if receipt.Evidence != nil {
				t.Fatal("a refusal is not Evidence")
			}
		})
	}
}

// TestDerivingWithoutADeclaredVersionIsRefused: the accepted source kinds come
// from the recorded tool version. Deriving a request without one is refused
// rather than defaulted, because the accepted set changed between versions.
func TestDerivingWithoutADeclaredVersionIsRefused(t *testing.T) {
	raw := fixture(t, "rea-6.8.0/analysis.json")
	receipt := externalanalysis.Receive(raw, presentTool("6.8.0"), acquisition)
	if _, err := externalanalysis.DeriveViewRequest(receipt.EvidenceBody(), viewRequest(), nil); err == nil {
		t.Fatal("deriving without a declared version must be refused")
	}
	unobserved, ok := externalanalysis.AcceptedSourceKinds("7.0.0")
	if ok {
		t.Fatal("an unobserved version must not report an accepted set")
	}
	if _, err := externalanalysis.DeriveViewRequest(receipt.EvidenceBody(), viewRequest(), unobserved); err == nil {
		t.Fatal("deriving against an unobserved version must be refused")
	}
}

// TestEmptyPinsAskNoQuestion: a caller that recorded no pins is not asking
// whether anything matches, so an empty Pins is not a failure — including for a
// receipt that carries no Evidence at all, whose state already says so.
func TestEmptyPinsAskNoQuestion(t *testing.T) {
	missing := externalanalysis.Receive(nil, presentTool("6.8.0"), acquisition)
	if err := missing.VerifyPins(externalanalysis.Pins{}); err != nil {
		t.Fatalf("empty pins must not manufacture a mismatch: %v", err)
	}
}

func viewRequest() externalanalysis.ViewRequest {
	return externalanalysis.ViewRequest{
		Kind:     "javascript-facts",
		Selector: json.RawMessage(`{"kind":"javascript-facts","property_keys":[],"offset":0,"limit":32}`),
	}
}

func acceptedKinds68(t *testing.T) []string {
	t.Helper()
	kinds, ok := externalanalysis.AcceptedSourceKinds("6.8.0")
	if !ok {
		t.Fatal("6.8.0 must be an observed version")
	}
	return kinds
}

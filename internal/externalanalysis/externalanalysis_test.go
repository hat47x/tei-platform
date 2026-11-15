package externalanalysis

import (
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", filepath.FromSlash(name)))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return raw
}

func TestReceiveRealSavedEvidence(t *testing.T) {
	for _, tc := range []struct {
		version    string
		evidenceID string
	}{
		{"6.7.0", "ev_c9d5026bb51c7e100892eb076ebb4b812ea0f4a7874dcf4c835d33098ba9fb85"},
		{"6.8.0", "ev_a88c70bcd066b2f04fdf97cb349369f0779b9c68cc4ac6c0e5f38f3539477a93"},
	} {
		t.Run(tc.version, func(t *testing.T) {
			raw := fixture(t, "rea-"+tc.version+"/analysis.json")
			receipt := Receive(raw, Tool{Name: "rea", Version: tc.version, Present: true, ResolvedPath: "test"}, testTime)

			if receipt.State != StateComplete {
				t.Fatalf("state=%s reason=%s", receipt.State, receipt.Reason)
			}
			if receipt.Evidence == nil || receipt.Evidence.ID != tc.evidenceID {
				t.Fatalf("evidence id=%v want %s", receipt.Evidence, tc.evidenceID)
			}
			if receipt.Evidence.Provider.ID != "rea-javascript-application" {
				t.Fatalf("provider id=%s", receipt.Evidence.Provider.ID)
			}
			if receipt.Source == nil || receipt.Source.Revision != "sha256:000d9d6f7f73531725d781c5742c685611c36a74d6f2c29b73c36d5c61c53b54" {
				t.Fatalf("source revision=%v", receipt.Source)
			}
			if receipt.Envelope == nil || receipt.Envelope.Command != "analyze-javascript-application" {
				t.Fatalf("envelope=%v", receipt.Envelope)
			}
			if receipt.Raw.Bytes != len(raw) {
				t.Fatalf("raw bytes=%d want %d", receipt.Raw.Bytes, len(raw))
			}
			if receipt.EvidenceBody() == nil {
				t.Fatal("evidence body is missing")
			}
		})
	}
}

// Actual upstream JSONL failures use a reserved control record. This
// synthetic transport witness proves classification, not a live tool run.
func TestReceiveTypedProducerFailureDoesNotPromoteEvidence(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		want    State
	}{
		{"jsonl-error", `{"$rea":"jsonl/v1","type":"error","ok":false,"result":{"ok":false,"error":{"code":"INVALID_SOURCE_KIND","category":"validation","message":"synthetic"}}}`, StateFailed},
		{"json-error", `{"ok":false,"error":{"code":"BAD_REQUEST","category":"validation","message":"synthetic"}}`, StateFailed},
		{"missing-code", `{"$rea":"jsonl/v1","type":"error","ok":false,"result":{"error":{"message":"no code"}}}`, StateMismatch},
		{"missing-ok", `{"error":{"code":"BAD_REQUEST"}}`, StateMismatch},
		{"not-a-failure", `{"$rea":"jsonl/v1","type":"metadata","records":{"emitted":0}}`, StateMismatch},
		{"false-with-no-error", `{"ok":false}`, StateMismatch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(tc.content)
			receipt := Receive(raw, Tool{Name: "rea", Version: "6.8.0", Present: true}, testTime)
			if receipt.State != tc.want {
				t.Fatalf("state=%s want=%s reason=%s", receipt.State, tc.want, receipt.Reason)
			}
			if receipt.Evidence != nil || receipt.EvidenceBody() != nil ||
				receipt.Promotion.AcceptedMeaning || receipt.Promotion.CanonicalWorld ||
				receipt.Promotion.DatabaseObservation {
				t.Fatalf("negative transport became Evidence or accepted meaning: %+v", receipt)
			}
			if tc.want == StateFailed {
				if receipt.Failure == nil || receipt.Failure.Code == "" {
					t.Fatalf("typed upstream failure was lost: %+v", receipt)
				}
				if receipt.Raw.Bytes != len(raw) || receipt.Raw.Fingerprint == "" {
					t.Fatalf("failed receipt lost received byte provenance: %+v", receipt.Raw)
				}
			}
		})
	}
}

func TestReceiveEmptyBytes(t *testing.T) {
	receipt := Receive([]byte{}, Tool{}, testTime)
	if receipt.State != StateMissing {
		t.Fatalf("state=%s", receipt.State)
	}
}

func TestReceiveInvalidJSON(t *testing.T) {
	receipt := Receive([]byte("not json"), Tool{}, testTime)
	if receipt.State != StateMismatch {
		t.Fatalf("state=%s", receipt.State)
	}
}

func TestInspectRealSavedViewIsComplete(t *testing.T) {
	for _, tc := range []struct {
		version string
		records int
		viewID  string
	}{
		{"6.7.0", 1, "ev_7d13b168b2f95c3f2725c13590e0e1d293a38eef43c79756ecfced6cff6150dc"},
		{"6.8.0", 7, "ev_16fb8cd4b854f983b4e5aea1461d5a3b1fa4a19ad7d46e618fee2a548eca3e8d"},
	} {
		t.Run(tc.version, func(t *testing.T) {
			raw := fixture(t, "rea-"+tc.version+"/analysis.json")
			view := fixture(t, "rea-"+tc.version+"/view.jsonl")
			framing, ok := FramingFor(tc.version)
			if !ok {
				t.Fatalf("missing framing for %s", tc.version)
			}
			receipt := Receive(raw, Tool{Name: "rea", Version: tc.version, Present: true, ResolvedPath: "test"}, testTime).WithView(view, framing)

			if receipt.State != StateComplete {
				t.Fatalf("state=%s reason=%s", receipt.State, receipt.Reason)
			}
			if receipt.View == nil {
				t.Fatal("view is missing")
			}
			if receipt.View.Records != tc.records {
				t.Fatalf("view records=%d want %d", receipt.View.Records, tc.records)
			}
			if receipt.View.EvidenceID != tc.viewID {
				t.Fatalf("view evidence id=%s want %s", receipt.View.EvidenceID, tc.viewID)
			}
		})
	}
}

// New REA 6.8 output reports a JSON metadata record and an actual
// emitted count. Legacy captured fixtures used "$rea=..." with seven rows.
func TestInspectREA68StructuredJSONLMetadata(t *testing.T) {
	raw := fixture(t, "rea-6.8.0/analysis.json")
	framing, ok := FramingFor("6.8.0")
	if !ok {
		t.Fatal("missing REA 6.8 framing")
	}
	const rows = "{\"name\":\"one\"}\n{\"name\":\"two\"}\n"
	const good = `{"$rea":"jsonl/v1","type":"metadata","records":{"path":["normalized_result","rows"],"emitted":2,"source":10},"context":{"evidence_id":"ev-fresh-view"}}`
	for _, tc := range []struct {
		name    string
		content string
		state   State
		control string
	}{
		{"current-json-metadata", rows + good + "\n", StateComplete, "metadata"},
		{"zero-records-with-explicit-metadata", `{"$rea":"jsonl/v1","type":"metadata","records":{"path":["normalized_result","rows"],"emitted":0,"source":0},"context":null}` + "\n", StateComplete, "metadata"},
		{"missing-terminal", rows, StatePartial, "none"},
		{"truncated-delivery", rows + `{"$rea":"jsonl/v1","type":"metadata","records":{"path":["normalized_result","rows"],"emitted":3,"source":10}}` + "\n", StatePartial, "metadata"},
		{"wrong-type", rows + `{"$rea":"jsonl/v1","type":"error","records":{"path":["normalized_result","rows"],"emitted":2,"source":10}}` + "\n", StateMismatch, "metadata"},
		{"unsupported-protocol", rows + `{"$rea":"jsonl/v9","type":"metadata","records":{"path":["normalized_result","rows"],"emitted":2,"source":10}}` + "\n", StateMismatch, "metadata"},
		{"missing-emitted", rows + `{"$rea":"jsonl/v1","type":"metadata","records":{"path":["normalized_result","rows"],"source":10}}` + "\n", StateMismatch, "metadata"},
		{"source-less-than-emitted", rows + `{"$rea":"jsonl/v1","type":"metadata","records":{"path":["normalized_result","rows"],"emitted":2,"source":1}}` + "\n", StateMismatch, "metadata"},
		{"malformed-data-line", "NOT JSON\n" + good + "\n", StateMismatch, "metadata"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			receipt := Receive(raw, Tool{Name: "rea", Version: "6.8.0", Present: true}, testTime).
				WithView([]byte(tc.content), framing)
			if receipt.State != tc.state {
				t.Fatalf("state=%s want=%s reason=%s", receipt.State, tc.state, receipt.Reason)
			}
			if receipt.View == nil || receipt.View.Control != tc.control {
				t.Fatalf("view/control=%+v want=%s", receipt.View, tc.control)
			}
			if tc.name == "current-json-metadata" {
				if receipt.View.Records != 2 || receipt.View.Emitted != 2 || receipt.View.SourceCount != 10 ||
					receipt.View.EvidenceID != "ev-fresh-view" ||
					len(receipt.View.Collection) != 2 || receipt.View.Collection[0] != "normalized_result" {
					t.Fatalf("structured metadata not preserved: %+v", receipt.View)
				}
				if receipt.View.ParentLinked {
					t.Fatalf("view identity was promoted to an unproven parent link: %+v", receipt.View)
				}
			}
		})
	}
}

func TestViewMissingTerminalIsPartialFor68(t *testing.T) {
	raw := fixture(t, "rea-6.8.0/analysis.json")
	view := fixture(t, "rea-6.8.0/view.jsonl")
	trimmed := string(view)
	if idx := lastIndexString(trimmed, "\n$rea="); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	framing, ok := FramingFor("6.8.0")
	if !ok {
		t.Fatal("missing framing")
	}
	receipt := Receive(raw, Tool{Name: "rea", Version: "6.8.0", Present: true, ResolvedPath: "test"}, testTime).WithView([]byte(trimmed), framing)
	if receipt.State != StatePartial {
		t.Fatalf("state=%s reason=%s", receipt.State, receipt.Reason)
	}
	if receipt.View == nil || receipt.View.Control != "none" {
		t.Fatalf("control=%v", receipt.View)
	}
}

func TestDeriveViewRequestEmbedsEvidence(t *testing.T) {
	raw := fixture(t, "rea-6.8.0/analysis.json")
	receipt := Receive(raw, Tool{Name: "rea", Version: "6.8.0", Present: true, ResolvedPath: "test"}, testTime)
	kinds, ok := AcceptedSourceKinds("6.8.0")
	if !ok {
		t.Fatal("missing accepted kinds")
	}
	derived, err := DeriveViewRequest(receipt.EvidenceBody(), ViewRequest{Kind: "javascript-facts", Selector: []byte(`{"kind":"javascript-facts","offset":0,"limit":10}`)}, kinds)
	if err != nil {
		t.Fatalf("derive request: %v", err)
	}
	updated := receipt.WithRequest(derived, kinds)
	if updated.State != StateComplete {
		t.Fatalf("state=%s reason=%s", updated.State, updated.Reason)
	}
	if updated.Derived == nil || updated.Derived.SourceKind != "inline" {
		t.Fatalf("derived source kind=%v", updated.Derived)
	}
	if !containsString(updated.Derived.AcceptedKinds, "evidence-file") {
		t.Fatalf("accepted kinds=%v", updated.Derived.AcceptedKinds)
	}
}

func TestDeriveViewRequestRejectsEnvelope(t *testing.T) {
	raw := fixture(t, "rea-6.8.0/analysis.json")
	receipt := Receive(raw, Tool{Name: "rea", Version: "6.8.0", Present: true, ResolvedPath: "test"}, testTime)
	kinds, _ := AcceptedSourceKinds("6.8.0")
	envelope := []byte(`{"ok":true,"data":` + string(receipt.EvidenceBody()) + `,"meta":{}}`)
	if _, err := DeriveViewRequest(envelope, ViewRequest{Kind: "javascript-facts", Selector: []byte(`{"kind":"javascript-facts"}`)}, kinds); err == nil {
		t.Fatal("envelope must be refused")
	}
}

func TestVerifyPinsDetectsMismatch(t *testing.T) {
	raw := fixture(t, "rea-6.8.0/analysis.json")
	receipt := Receive(raw, Tool{Name: "rea", Version: "6.8.0", Present: true, ResolvedPath: "test"}, testTime)
	if err := receipt.VerifyPins(Pins{EvidenceID: "wrong", Fingerprint: "wrong", SourceRevision: "wrong"}); err == nil {
		t.Fatal("pin mismatch must fail")
	}
}

func lastIndexString(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

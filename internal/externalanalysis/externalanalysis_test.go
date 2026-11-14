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

package main

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

// The Wails bridge is a pass-through by design: it returns the host's response
// type unchanged rather than re-mapping it. That is what keeps the unified
// failure fields (kind, retryable, effectUnknown) from being dropped on the way
// to the frontend, and this test pins it, because the failure of that property
// is silent - the code and message would still arrive and only the axes would
// go missing.
func TestStudioBridgeKeepsTheUnifiedFailureAxes(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	host, err := newStudioHost([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	bridge := &StudioBridge{host: host}

	// A workspace the host does not allow is a business rejection: the caller
	// must not resend it unchanged, and no effect was applied because nothing
	// ran.
	denied := bridge.InspectSubject("workspace-2", "semantic-data", "sales.order")
	if denied.OK || denied.Failure == nil {
		t.Fatalf("an unallowed workspace was not refused: %+v", denied)
	}
	if denied.Failure.Code != "WORKSPACE_NOT_ALLOWED" {
		t.Fatalf("code=%q, want WORKSPACE_NOT_ALLOWED", denied.Failure.Code)
	}
	if denied.Failure.Kind != "business_rejection" {
		t.Fatalf("kind=%q, want business_rejection: the bridge dropped the classification", denied.Failure.Kind)
	}
	if denied.Failure.Retryable != "not_retryable" {
		t.Fatalf("retryable=%q, want not_retryable", denied.Failure.Retryable)
	}
	if denied.Failure.EffectUnknown {
		t.Fatal("a refusal before anything ran was reported as having an unknown effect")
	}

	// A source the host cannot read is its own failure. Whether anything was
	// applied is not decidable from here, so the caller must look the result up
	// before resending. This is the case that would silently read as "the
	// request was refused" if the classification were dropped, which is the
	// confusion the whole contract exists to prevent.
	unreadable := bridge.ObserveXead("workspace-1", "../secret.xead")
	if unreadable.OK || unreadable.Failure == nil {
		t.Fatalf("an unreadable source was not reported: %+v", unreadable)
	}
	if unreadable.Failure.Code != "XEAD_SOURCE_READ_FAILED" {
		t.Fatalf("code=%q, want XEAD_SOURCE_READ_FAILED", unreadable.Failure.Code)
	}
	if unreadable.Failure.Kind != "technical_failure" {
		t.Fatalf("kind=%q, want technical_failure", unreadable.Failure.Kind)
	}
	if unreadable.Failure.Retryable != "lookup_first" {
		t.Fatalf("retryable=%q, want lookup_first", unreadable.Failure.Retryable)
	}
	if !unreadable.Failure.EffectUnknown {
		t.Fatal("an unreadable source was reported as having a known effect")
	}

	// The same three axes must reach the wire, because the frontend reads JSON
	// rather than the Go struct.
	encoded, err := json.Marshal(unreadable)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		OK      bool `json:"ok"`
		Failure struct {
			Code          string `json:"code"`
			Kind          string `json:"kind"`
			Retryable     string `json:"retryable"`
			EffectUnknown bool   `json:"effectUnknown"`
		} `json:"failure"`
	}
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatalf("the bridge response is not one JSON document: %v (%s)", err, encoded)
	}
	if wire.OK || wire.Failure.Kind != "technical_failure" ||
		wire.Failure.Retryable != "lookup_first" || !wire.Failure.EffectUnknown {
		t.Fatalf("the wire form dropped the classification: %s", encoded)
	}
}

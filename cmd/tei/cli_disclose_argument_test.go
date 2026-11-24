package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/knowledgecontext"
)

// TestDiscloseUnknownSectionIsInvalidArgument pins that an unknown -disclose
// section is an argument mistake, not a Workspace/build failure: it answers
// INVALID_ARGUMENT with exit 2 before any Workspace is read, and the message
// names the offending value and lists every published section. A published
// section is never refused as an argument error; it reaches the disclosure
// builder (#3242).
func TestDiscloseUnknownSectionIsInvalidArgument(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")

	// (a) An unknown section is INVALID_ARGUMENT (exit 2), and the message
	// names both the bad value and every published section.
	var out bytes.Buffer
	code, err := knowledgeContextJSON(&out, reference, "semantic-data", "sales.order", "not-a-section", "", "", 0)
	if err != nil {
		t.Fatalf("unknown section returned an emitter error: %v", err)
	}
	if code != 2 {
		t.Fatalf("unknown section: exit=%d, want 2 (body=%s)", code, out.String())
	}
	var body struct {
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(out.Bytes(), &body); err != nil {
		t.Fatalf("unknown section: not one envelope: %v (%s)", err, out.String())
	}
	if body.Error == nil || body.Error.Code != "INVALID_ARGUMENT" {
		t.Fatalf("unknown section: error.code=%q, want INVALID_ARGUMENT (body=%s)", body.Error.Code, out.String())
	}
	if !strings.Contains(body.Error.Message, "not-a-section") {
		t.Fatalf("unknown section: message does not name the bad value: %q", body.Error.Message)
	}
	for _, section := range knowledgecontext.SelectableDisclosureSections {
		if !strings.Contains(body.Error.Message, section) {
			t.Fatalf("unknown section: message omits published section %q: %q", section, body.Error.Message)
		}
	}

	// (b) Each published section is not refused as an argument error; it
	// reaches the disclosure builder instead.
	for _, section := range knowledgecontext.SelectableDisclosureSections {
		var sectionOut bytes.Buffer
		if _, err := knowledgeContextJSON(&sectionOut, reference, "semantic-data", "sales.order", section, "", "", 0); err != nil {
			t.Fatalf("published section %q returned an emitter error: %v", section, err)
		}
		if failure := errorCodeOf(t, sectionOut.Bytes()); failure == "INVALID_ARGUMENT" {
			t.Fatalf("published section %q was refused as an argument error: %s", section, sectionOut.String())
		}
	}
}

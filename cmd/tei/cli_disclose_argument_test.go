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

	// (c) A value that selects no section is an argument mistake too. It used
	// to pass this check and fail deeper with KNOWLEDGE_BUILD_FAILED, which
	// names a build problem for a flag value that selected nothing.
	for _, selection := range []string{" , ", ",", " "} {
		t.Run("empty selection "+selection, func(t *testing.T) {
			var emptyOut bytes.Buffer
			code, err := knowledgeContextJSON(&emptyOut, reference, "semantic-data", "sales.order", selection, "", "", 0)
			if err != nil {
				t.Fatalf("empty selection %q returned an emitter error: %v", selection, err)
			}
			if code != 2 {
				t.Fatalf("empty selection %q: exit=%d, want 2 (body=%s)", selection, code, emptyOut.String())
			}
			var body struct {
				Error *struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(emptyOut.Bytes(), &body); err != nil {
				t.Fatalf("empty selection %q: not one envelope: %v (%s)", selection, err, emptyOut.String())
			}
			if body.Error == nil || body.Error.Code != "INVALID_ARGUMENT" {
				t.Fatalf("empty selection %q: error.code=%v, want INVALID_ARGUMENT (body=%s)",
					selection, body.Error, emptyOut.String())
			}
			if !strings.Contains(body.Error.Message, "at least one") {
				t.Fatalf("empty selection %q: message does not say a section is required: %q",
					selection, body.Error.Message)
			}
			// The message still names every section, so a caller learns what to
			// select without reading the source.
			for _, section := range knowledgecontext.SelectableDisclosureSections {
				if !strings.Contains(body.Error.Message, section) {
					t.Fatalf("empty selection %q: message omits published section %q: %q",
						selection, section, body.Error.Message)
				}
			}
		})
	}
}

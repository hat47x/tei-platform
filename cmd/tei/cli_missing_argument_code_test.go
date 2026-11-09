package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// TestMissingRequiredArgumentIsOneCode pins #3733: a flag that was never
// supplied is the same failure whatever command it belongs to, so a reader can
// decide "the caller forgot an argument" with one condition. Before this,
// inspect answered INVALID_SUBJECT and the knowledge commands answered
// KNOWLEDGE_BUILD_FAILED, which named a subject problem or a build failure
// instead of a missing flag.
func TestMissingRequiredArgumentIsOneCode(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	cases := []struct {
		name string
		run  func(*bytes.Buffer) (int, error)
	}{
		{"inspect without -subject-id", func(out *bytes.Buffer) (int, error) {
			return inspectWorkspaceJSON(out, reference, "", "semantic-data", "", 0)
		}},
		{"inspect without -workspace", func(out *bytes.Buffer) (int, error) {
			return inspectWorkspaceJSON(out, "", "", "semantic-data", "sales.order", 0)
		}},
		{"trace without -scope", func(out *bytes.Buffer) (int, error) {
			return traceWorkspaceJSON(out, reference, "", "", "semantic-data", "sales.order",
				"both", "summary", "", 0, 0, 0)
		}},
		{"knowledge-context without -subject-id", func(out *bytes.Buffer) (int, error) {
			return knowledgeContextJSON(out, reference, "semantic-data", "", "", "", "", 0)
		}},
		{"knowledge-qualify without -subject-id", func(out *bytes.Buffer) (int, error) {
			return knowledgeQualifyJSON(out, reference, "semantic-data", "", "", 0)
		}},
		{"knowledge-review-plan without -subject-id", func(out *bytes.Buffer) (int, error) {
			return knowledgeReviewPlanJSON(out, reference, "semantic-data", "", "", 0)
		}},
		{"knowledge-review-handoff without -subject-id", func(out *bytes.Buffer) (int, error) {
			return knowledgeReviewHandoffJSON(out, reference, "semantic-data", "", "", "", 0)
		}},
		{"knowledge-review-disclose without -subject-id", func(out *bytes.Buffer) (int, error) {
			return knowledgeReviewDiscloseJSON(out, reference, "semantic-data", "", "", "", "", 0)
		}},
		{"model catalog without -workspace", func(out *bytes.Buffer) (int, error) {
			return runModelCommand([]string{"catalog"}, out)
		}},
		{"requirements without a Workspace", func(out *bytes.Buffer) (int, error) {
			return resultCommandJSON(out, "requirements", "", "", "", "", false, false, 0)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			code, err := tc.run(&out)
			if err != nil {
				t.Fatalf("returned an emitter error: %v", err)
			}
			if code != 2 {
				t.Fatalf("exit=%d, want 2", code)
			}
			failure := errorCodeOf(t, out.Bytes())
			if failure != "INVALID_ARGUMENT" {
				t.Fatalf("error.code=%q, want INVALID_ARGUMENT (body=%s)", failure, out.String())
			}
		})
	}
}

// A present but malformed subject keeps its own code: INVALID_SUBJECT still
// means "this subject value is wrong", so the two cases stay distinguishable.
func TestMalformedSubjectKeepsItsOwnCode(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	var out bytes.Buffer
	if _, err := inspectWorkspaceJSON(&out, reference, "", "semantic-data", "sales.order", 0); err != nil {
		t.Fatalf("baseline inspect: %v", err)
	}
	// A blank ID passes the flag guard (the flag was supplied) but is not a
	// subject, which is what INVALID_SUBJECT is for.
	var blank bytes.Buffer
	code, err := inspectWorkspaceJSON(&blank, reference, "", "semantic-data", " ", 0)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if code == 0 {
		t.Fatalf("a blank subject was accepted: %s", blank.String())
	}
	// Whatever code it reports, it must not be the missing-argument code path
	// for a flag that was supplied: the flag guard is what changed here.
	failure := errorCodeOf(t, blank.Bytes())
	if failure == "" {
		t.Fatalf("no error document: %s", blank.String())
	}
	if failure == "INVALID_ARGUMENT" && strings.Contains(blank.String(), "inspect requires") {
		t.Fatalf("a supplied flag was reported as missing: %s", blank.String())
	}
}

// errorCodeOf reads error.code from one envelope document.
func errorCodeOf(t *testing.T, raw []byte) string {
	t.Helper()
	var body struct {
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("not one JSON document: %v (%q)", err, string(raw))
	}
	if body.Error == nil {
		return ""
	}
	return body.Error.Code
}

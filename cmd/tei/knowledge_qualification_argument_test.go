package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runKnowledgeCLI builds the CLI once and returns a runner. Each call reports
// the exit code and the single stdout envelope.
func runKnowledgeCLI(t *testing.T) func(args ...string) (int, []byte) {
	t.Helper()
	binary := buildCLIForLifecycle(t)
	return func(args ...string) (int, []byte) {
		t.Helper()
		command := exec.Command(binary, args...)
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		err := command.Run()
		code := 0
		if err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("%q: CLI execution: %v", args, err)
			}
			code = exit.ExitCode()
		}
		if stderr.Len() != 0 {
			t.Fatalf("%q wrote to stderr: %s", args, stderr.String())
		}
		return code, stdout.Bytes()
	}
}

func knowledgeEnvelopeFailure(t *testing.T, body []byte) (string, string) {
	t.Helper()
	var response struct {
		OK    bool `json:"ok"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("not one envelope: %v (%s)", err, body)
	}
	if response.OK || response.Error == nil {
		t.Fatalf("expected a failure envelope: %s", body)
	}
	return response.Error.Code, response.Error.Message
}

// TestKnowledgeQualificationRejectsABadRequirementsFileAsAnArgument pins #3943:
// a requirements file that cannot be read, is malformed, or carries an unknown
// field is the caller's input problem. It used to answer KNOWLEDGE_BUILD_FAILED,
// which reads as a defect in TEI, so a caller could not tell "fix your file"
// from "the knowledge model is broken". Exit 2 is the CLI convention for an
// argument or load failure; 1 is a semantic finding.
func TestKnowledgeQualificationRejectsABadRequirementsFileAsAnArgument(t *testing.T) {
	run := runKnowledgeCLI(t)
	reference := filepath.Join("..", "..", "reference", "workspace")
	base := []string{"knowledge-qualify", "-workspace", reference,
		"-subject-kind", "semantic-data", "-subject-id", "security.authenticated_subject",
		"-qualification-requirements"}

	dir := t.TempDir()
	cases := []struct {
		name    string
		path    func() string
		wantIn  string
		wantNot string
	}{
		{
			name: "absent",
			path: func() string { return filepath.Join(dir, "absent.json") },
			// The message must name the flag, because that is the thing to fix.
			wantIn: "-qualification-requirements",
		},
		{
			name: "malformed",
			path: func() string {
				path := filepath.Join(dir, "malformed.json")
				if err := os.WriteFile(path, []byte(`{"expectedKind":`), 0o644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			wantIn: "not valid JSON",
		},
		{
			name: "unknown-field",
			path: func() string {
				path := filepath.Join(dir, "unknown.json")
				if err := os.WriteFile(path, []byte(`{"expectedKind":"semantic-data","accepted":true}`), 0o644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			wantIn: "unknown field",
		},
		{
			name: "trailing-value",
			path: func() string {
				path := filepath.Join(dir, "trailing.json")
				if err := os.WriteFile(path, []byte(`{"expectedKind":"semantic-data"} {"a":1}`), 0o644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			wantIn: "more than one JSON value",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, body := run(append(append([]string{}, base...), tc.path())...)
			failure, message := knowledgeEnvelopeFailure(t, body)
			if failure != "INVALID_ARGUMENT" {
				t.Fatalf("a bad requirements file answered %s, want INVALID_ARGUMENT: %s", failure, body)
			}
			if code != 2 {
				t.Fatalf("exit=%d, want 2 for an argument failure", code)
			}
			if !strings.Contains(message, tc.wantIn) {
				t.Fatalf("message does not say what to fix (%q missing): %q", tc.wantIn, message)
			}
			// The old code must not come back: it is what made the failure look
			// like a broken model.
			if strings.Contains(message, "KNOWLEDGE_BUILD_FAILED") || failure == "KNOWLEDGE_BUILD_FAILED" {
				t.Fatalf("a bad requirements file still reports a build failure: %s", body)
			}
		})
	}
}

// A JSON *string* passed where a path belongs is the mistake a caller makes
// after reading the help text; the message must say the value is treated as a
// path so the cause is obvious without reading the source.
func TestKnowledgeQualificationTreatsInlineJSONAsAPath(t *testing.T) {
	run := runKnowledgeCLI(t)
	reference := filepath.Join("..", "..", "reference", "workspace")
	code, body := run("knowledge-qualify", "-workspace", reference,
		"-subject-kind", "semantic-data", "-subject-id", "security.authenticated_subject",
		"-qualification-requirements", `{"qualifications":[{"id":"owner-approved"}]}`)
	failure, _ := knowledgeEnvelopeFailure(t, body)
	if failure != "INVALID_ARGUMENT" || code != 2 {
		t.Fatalf("inline JSON answered %s exit=%d, want INVALID_ARGUMENT exit=2: %s", failure, code, body)
	}
}

// knowledge-review-handoff shares the loader, so it must report the same way.
// A fix applied to one entrance only is how the codes drifted before.
func TestKnowledgeReviewHandoffRejectsABadRequirementsFileAsAnArgument(t *testing.T) {
	run := runKnowledgeCLI(t)
	reference := filepath.Join("..", "..", "reference", "workspace")
	code, body := run("knowledge-review-handoff", "-workspace", reference,
		"-subject-kind", "semantic-data", "-subject-id", "security.authenticated_subject",
		"-qualification-requirements", filepath.Join(t.TempDir(), "absent.json"))
	failure, message := knowledgeEnvelopeFailure(t, body)
	if failure != "INVALID_ARGUMENT" || code != 2 {
		t.Fatalf("handoff answered %s exit=%d, want INVALID_ARGUMENT exit=2: %s", failure, code, body)
	}
	if !strings.Contains(message, "-qualification-requirements") {
		t.Fatalf("the message does not name the flag: %q", message)
	}
}

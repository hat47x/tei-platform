package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runCLIForClassification runs one invocation and returns exit code, ok and
// failure code. It builds once per call so the test sees the real process.
func runCLIForClassification(t *testing.T, args ...string) (int, bool, string) {
	t.Helper()
	binary := sharedCLIBinary(t)
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(binary, args...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		exit, isExit := err.(*exec.ExitError)
		if !isExit {
			t.Fatalf("argv=%q: %v (%s)", args, err, stderr.String())
		}
		code = exit.ExitCode()
	}
	var doc struct {
		OK    bool `json:"ok"`
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("argv=%q did not return one JSON document: %v (stdout=%q stderr=%q)",
			args, err, stdout.String(), stderr.String())
	}
	failure := ""
	if doc.Error != nil {
		failure = doc.Error.Code
	}
	return code, doc.OK, failure
}

// Leaving out a required flag is one condition, so every command answers it
// with the same stable code and exit code. Before this, inspect answered
// INVALID_SUBJECT and knowledge-context answered KNOWLEDGE_BUILD_FAILED for the
// same caller mistake, so a reader could not detect "a flag is missing" with one
// rule (#3733).
func TestMissingRequiredFlagIsAlwaysInvalidArgument(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	// A valid requirements document, so the qualify cases reach the argument
	// guard instead of failing in the strict decoder.
	requirements := filepath.Join(t.TempDir(), "requirements.json")
	if err := os.WriteFile(requirements, []byte(`{"expectedKind":"semantic-data"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		args []string
	}{
		{"inspect without -subject-id", []string{"inspect", "-workspace", reference, "-subject-kind", "semantic-data"}},
		{"inspect without -subject-kind", []string{"inspect", "-workspace", reference, "-subject-id", "sales.order"}},
		{"inspect without -workspace", []string{"inspect", "-subject-kind", "semantic-data", "-subject-id", "sales.order"}},
		{"trace without -scope", []string{"trace", "-workspace", reference, "-subject-kind", "semantic-data", "-subject-id", "sales.order"}},
		{"trace without -subject-id", []string{"trace", "-workspace", reference, "-scope", "s", "-subject-kind", "semantic-data"}},
		{"model catalog without -workspace", []string{"model", "catalog"}},
		{"source-index without -workspace", []string{"source-index"}},
		{"requirements without input", []string{"requirements"}},
		{"knowledge-context without -subject-kind", []string{"knowledge-context", "-workspace", reference}},
		// The requirement file itself is supplied, so the only missing input
		// is the subject id.
		{"knowledge-qualify without -subject-id", []string{"knowledge-qualify", "-workspace", reference,
			"-subject-kind", "semantic-data", "-qualification-requirements", requirements}},
		{"knowledge-review-plan without -subject-kind", []string{"knowledge-review-plan", "-workspace", reference}},
		{"knowledge-qualify without -qualification-requirements", []string{"knowledge-qualify",
			"-workspace", reference, "-subject-kind", "semantic-data", "-subject-id", "sales.order"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, ok, failure := runCLIForClassification(t, tc.args...)
			if code != 2 {
				t.Fatalf("exit=%d, want 2 (input error)", code)
			}
			if ok {
				t.Fatalf("a missing flag answered ok")
			}
			if failure != "INVALID_ARGUMENT" {
				t.Fatalf("failure=%q, want INVALID_ARGUMENT", failure)
			}
		})
	}
}

// The code must keep meaning what it says: a supplied but unusable subject
// value is INVALID_SUBJECT, not INVALID_ARGUMENT. Collapsing both would lose the
// distinction the two names carry.
func TestUnusableSubjectValueStaysInvalidSubject(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")

	for _, tc := range []struct {
		name string
		args []string
	}{
		{"whitespace-only id", []string{"inspect", "-workspace", reference,
			"-subject-kind", "semantic-data", "-subject-id", "  "}},
		{"non-exact id", []string{"inspect", "-workspace", reference,
			"-subject-kind", "semantic-data", "-subject-id", " sales.order"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, ok, failure := runCLIForClassification(t, tc.args...)
			if code != 2 || ok {
				t.Fatalf("exit=%d ok=%v", code, ok)
			}
			if failure != "INVALID_SUBJECT" {
				t.Fatalf("failure=%q, want INVALID_SUBJECT", failure)
			}
		})
	}
}

// A subject that is well formed but absent is a finding, not an input error:
// exit 1 and SUBJECT_NOT_FOUND. This separates "you asked wrong" from "it is not
// there", which the reference-allow-list style numbers must keep apart.
func TestAbsentSubjectIsAFindingNotAnInputError(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	code, ok, failure := runCLIForClassification(t, "inspect", "-workspace", reference,
		"-subject-kind", "semantic-data", "-subject-id", "sales.not_found")
	if code != 1 || ok {
		t.Fatalf("exit=%d ok=%v", code, ok)
	}
	if failure != "SUBJECT_NOT_FOUND" {
		t.Fatalf("failure=%q, want SUBJECT_NOT_FOUND", failure)
	}
}

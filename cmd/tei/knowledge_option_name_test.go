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

// knowledgeCLI builds the CLI once and returns a runner that reports the exit
// code and the single stdout envelope.
func knowledgeCLI(t *testing.T) func(args ...string) (int, []byte) {
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

func knowledgeFailureOf(t *testing.T, body []byte) (string, string) {
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

// A file holding two JSON documents is a file the caller got wrong. It used to
// answer KNOWLEDGE_BUILD_FAILED, which reads as a defect in TEI, and it was the
// last path the loader still returned without marking as an argument error
// (#3943).
func TestKnowledgeQualificationRefusesATrailingDocument(t *testing.T) {
	run := knowledgeCLI(t)
	reference := filepath.Join("..", "..", "reference", "workspace")

	path := filepath.Join(t.TempDir(), "trailing.json")
	if err := os.WriteFile(path, []byte(`{"expectedKind":"semantic-data"} {"expectedKind":"other"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	code, body := run("knowledge-qualify", "-workspace", reference,
		"-subject-kind", "semantic-data", "-subject-id", "sales.order",
		"-qualification-requirements", path)
	failure, message := knowledgeFailureOf(t, body)
	if failure != "INVALID_ARGUMENT" {
		t.Fatalf("a trailing document answered %s, want INVALID_ARGUMENT: %s", failure, body)
	}
	if code != 2 {
		t.Fatalf("exit=%d, want 2 for an argument failure", code)
	}
	if !strings.Contains(message, "trailing JSON value") {
		t.Fatalf("the message does not explain the trailing value: %q", message)
	}
	if !strings.Contains(message, "-qualification-requirements") {
		t.Fatalf("the message does not name the option: %q", message)
	}
}

// One loader serves both -qualification-requirements and -review-handoff. It
// used to name -qualification-requirements in every message, so a caller who
// passed -review-handoff to knowledge-review-disclose was told to fix a flag
// they had not used (#3943).
func TestKnowledgeReviewDiscloseNamesTheOptionItRead(t *testing.T) {
	run := knowledgeCLI(t)
	reference := filepath.Join("..", "..", "reference", "workspace")

	code, body := run("knowledge-review-disclose", "-workspace", reference,
		"-subject-kind", "semantic-data", "-subject-id", "sales.order",
		"-review-handoff", filepath.Join(t.TempDir(), "absent.json"))
	failure, message := knowledgeFailureOf(t, body)
	if failure != "INVALID_ARGUMENT" || code != 2 {
		t.Fatalf("disclose answered %s exit=%d, want INVALID_ARGUMENT exit=2: %s", failure, code, body)
	}
	if !strings.Contains(message, "-review-handoff") {
		t.Fatalf("the message does not name -review-handoff: %q", message)
	}
	// Naming an option the caller never passed sends them to the wrong place.
	if strings.Contains(message, "-qualification-requirements") {
		t.Fatalf("the message names an option this command did not read: %q", message)
	}
}

// The same mistake on -qualification-requirements must still name that option,
// so fixing the shared loader did not simply swap one wrong name for another.
func TestKnowledgeQualifyStillNamesItsOwnOption(t *testing.T) {
	run := knowledgeCLI(t)
	reference := filepath.Join("..", "..", "reference", "workspace")

	code, body := run("knowledge-qualify", "-workspace", reference,
		"-subject-kind", "semantic-data", "-subject-id", "sales.order",
		"-qualification-requirements", filepath.Join(t.TempDir(), "absent.json"))
	failure, message := knowledgeFailureOf(t, body)
	if failure != "INVALID_ARGUMENT" || code != 2 {
		t.Fatalf("qualify answered %s exit=%d, want INVALID_ARGUMENT exit=2: %s", failure, code, body)
	}
	if !strings.Contains(message, "-qualification-requirements") {
		t.Fatalf("the message does not name -qualification-requirements: %q", message)
	}
	if strings.Contains(message, "-review-handoff") {
		t.Fatalf("the message names an option this command did not read: %q", message)
	}
}

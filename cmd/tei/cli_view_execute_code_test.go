package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runExecuteForTest runs one `view -execute` invocation against the reference
// workspace and reports the exit code, ok and error.code.
func runExecuteForTest(t *testing.T, binary, workspace string, extra ...string) (int, bool, string, string) {
	t.Helper()
	args := append([]string{"view", "-workspace", workspace,
		"-application", "order_entry",
		"-pattern", "order_entry.submit_order",
		"-command", "sales.order.submit",
		"-execute", "-format", "json"}, extra...)
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(binary, args...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		exit, isExit := err.(*exec.ExitError)
		if !isExit {
			t.Fatalf("%v (%s)", err, stderr.String())
		}
		code = exit.ExitCode()
	}
	var doc struct {
		OK    bool `json:"ok"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("not one JSON document: %v (%s)", err, stdout.String())
	}
	failure, message := "", ""
	if doc.Error != nil {
		failure, message = doc.Error.Code, doc.Error.Message
	}
	return code, doc.OK, failure, message
}

func buildExecuteCLI(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "tei")
	if os.PathSeparator == '\\' {
		binary += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
	return binary
}

// A reference execution can fail for reasons the caller fixes differently:
// a missing input, or a capability the authorizer does not allow. Both must
// answer with their own code, not one generic COMMAND_FAILED whose real reason
// is only readable by parsing the message text (#3797).
func TestExecuteFailureKeepsItsOwnCode(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	binary := buildExecuteCLI(t)

	t.Run("missing input", func(t *testing.T) {
		code, ok, failure, _ := runExecuteForTest(t, binary, reference)
		if ok || code != 1 {
			t.Fatalf("exit=%d ok=%v", code, ok)
		}
		if failure != "INVALID_COMMAND_INPUT" {
			t.Fatalf("code=%q, want INVALID_COMMAND_INPUT", failure)
		}
	})

	t.Run("unauthorized capability", func(t *testing.T) {
		code, ok, failure, _ := runExecuteForTest(t, binary, reference,
			"-input", "order=ORD-1", "-allow-capability", "sales.other.capability")
		if ok || code != 1 {
			t.Fatalf("exit=%d ok=%v", code, ok)
		}
		if failure != "COMMAND_UNAUTHORIZED" {
			t.Fatalf("code=%q, want COMMAND_UNAUTHORIZED", failure)
		}
	})

	// The message must not repeat the code as a prefix. A caller that reads
	// only error.code should get the whole reason from error.message.
	t.Run("message does not repeat the code", func(t *testing.T) {
		_, _, failure, message := runExecuteForTest(t, binary, reference)
		if failure == "" {
			t.Fatal("expected a failure")
		}
		if len(message) > len(failure)+2 && message[:len(failure)+2] == failure+": " {
			t.Fatalf("message still carries the code prefix: %q", message)
		}
	})
}

// The lifecycle refusal was already distinct; this pins that the wider code set
// did not disturb it.
func TestExecuteLifecycleRefusalStaysDistinct(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	binary := buildExecuteCLI(t)
	db := filepath.Join(t.TempDir(), "life.db")

	if code, ok, failure, _ := runExecuteForTest(t, binary, reference,
		"-input", "order=ORD-1", "-lifecycle-state", db, "-lifecycle-command", "c1"); !ok || code != 0 {
		t.Fatalf("first run: exit=%d ok=%v code=%q", code, ok, failure)
	}
	code, ok, failure, _ := runExecuteForTest(t, binary, reference,
		"-input", "order=ORD-1", "-lifecycle-state", db, "-lifecycle-command", "c1")
	if ok || code != 1 || failure != "COMMAND_LIFECYCLE_REFUSED" {
		t.Fatalf("resend: exit=%d ok=%v code=%q", code, ok, failure)
	}
}

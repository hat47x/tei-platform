package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runViewForTest builds the CLI once and runs one view invocation.
func runViewForTest(t *testing.T, args ...string) (int, bool, string, []byte) {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "tei")
	if os.PathSeparator == '\\' {
		binary += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
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
		t.Fatalf("argv=%q did not return one JSON document: %v (%s)", args, err, stdout.String())
	}
	failure := ""
	if doc.Error != nil {
		failure = doc.Error.Code
	}
	return code, doc.OK, failure, stdout.Bytes()
}

// -input only feeds the command plan, and the plan exists only when -command is
// given. Before this, `view -application X -input order=ORD-1` succeeded and
// dropped the value: the response was byte-identical to the same call without
// -input, and a malformed pair was never parsed. A caller could therefore
// believe it had supplied input to a command that was never planned (#3778).
func TestViewRefusesInputWithoutCommand(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")

	cases := map[string][]string{
		"application only": {"-application", "order_entry", "-input", "order=ORD-1"},
		"malformed pair":   {"-application", "order_entry", "-input", "no-equals-sign"},
		"with pattern":     {"-application", "order_entry", "-pattern", "order_entry.order_record", "-input", "order=ORD-1"},
	}
	for name, extra := range cases {
		t.Run(name, func(t *testing.T) {
			args := append([]string{"view", "-workspace", reference, "-format", "json"}, extra...)
			code, ok, failure, raw := runViewForTest(t, args...)
			if ok {
				t.Fatalf("view accepted -input without -command: %s", raw)
			}
			if code != 2 {
				t.Fatalf("exit=%d, want 2 (input error)", code)
			}
			if failure != "INVALID_ARGUMENT" {
				t.Fatalf("code=%q, want INVALID_ARGUMENT", failure)
			}
		})
	}
}

// The fix must not break the legitimate pairing. -input with -command plans the
// command with the supplied values and needs no -execute.
func TestViewKeepsInputWithCommand(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	args := []string{"view", "-workspace", reference,
		"-application", "order_entry",
		"-pattern", "order_entry.order_record",
		"-command", "sales.submit_order",
		"-input", "order=ORD-1",
		"-format", "json"}
	code, ok, failure, raw := runViewForTest(t, args...)
	if code == 2 {
		t.Fatalf("-input with -command was refused as an argument error: %s", raw)
	}
	if failure == "INVALID_ARGUMENT" {
		t.Fatalf("-input with -command was refused: %s", raw)
	}
	_ = ok // the plan may succeed or report a semantic finding; either is fine
}

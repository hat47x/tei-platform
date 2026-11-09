package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func buildCLIForLifecycle(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	return binary
}

func runCLIProcess(t *testing.T, binary string, args ...string) ([]byte, int) {
	t.Helper()
	command := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("CLI execution: %v", err)
		}
		code = exit.ExitCode()
	}
	if stderr.Len() > 0 {
		t.Fatalf("argv=%q emitted stderr: %q", args, stderr.String())
	}
	return stdout.Bytes(), code
}

// #2564 product path: a second CLI process must refuse a command the first
// process already observed. This uses the durable SQLite stage store, not the
// in-memory lifecycle.
func TestCLIProcessRestartRefusesObservedCommandReplay(t *testing.T) {
	binary := buildCLIForLifecycle(t)
	root := productSliceWorkspace(t)
	state := filepath.Join(t.TempDir(), "lifecycle.db")
	args := []string{
		"view", "-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_record",
		"-command", "sales.submit_order",
		"-input", "order=ORD-1",
		"-execute", "-format", "json",
		"-lifecycle-state", state,
		"-lifecycle-command", "cmd-cli-restart",
	}

	first, code := runCLIProcess(t, binary, args...)
	if code != 0 {
		t.Fatalf("first exit=%d body=%s", code, first)
	}
	var firstReport struct {
		OK     bool `json:"ok"`
		Result struct {
			Outcome *struct {
				Name string `json:"name"`
			} `json:"outcome"`
		} `json:"result"`
	}
	if err := json.Unmarshal(first, &firstReport); err != nil {
		t.Fatal(err)
	}
	if !firstReport.OK || firstReport.Result.Outcome == nil || firstReport.Result.Outcome.Name != "submitted" {
		t.Fatalf("first report=%s", first)
	}

	second, code := runCLIProcess(t, binary, args...)
	if code != 1 {
		t.Fatalf("second exit=%d body=%s", code, second)
	}
	var secondReport struct {
		OK    bool `json:"ok"`
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(second, &secondReport); err != nil {
		t.Fatal(err)
	}
	if secondReport.OK || secondReport.Error == nil || secondReport.Error.Code != "COMMAND_LIFECYCLE_REFUSED" {
		t.Fatalf("second report=%s", second)
	}
}

func TestCLILifecycleStateRequiresACommandID(t *testing.T) {
	binary := buildCLIForLifecycle(t)
	root := productSliceWorkspace(t)
	state := filepath.Join(t.TempDir(), "lifecycle.db")
	body, code := runCLIProcess(t, binary,
		"view", "-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_record",
		"-command", "sales.submit_order",
		"-execute",
		"-lifecycle-state", state,
	)
	if code != 2 {
		t.Fatalf("exit=%d body=%s", code, body)
	}
	var report struct {
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &report); err != nil {
		t.Fatal(err)
	}
	if report.Error == nil || report.Error.Code != "INVALID_ARGUMENT" {
		t.Fatalf("report=%s", body)
	}
}

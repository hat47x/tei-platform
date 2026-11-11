package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestPluginInventoryEnvelopeAndModes covers the CLI boundary of
// `plugin-inventory` (#4376): the envelope, the two alternate shapes, the
// failure codes, and the guarantee that the report never carries an "enabled"
// claim or a settings version it cannot derive.
func TestPluginInventoryEnvelopeAndModes(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}

	reference := filepath.Join("..", "..", "reference", "workspace")

	runRaw := func(expectedCode int, args ...string) []byte {
		t.Helper()
		cmd := exec.Command(binary, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		got := 0
		if err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("CLI execution: %v", err)
			}
			got = exit.ExitCode()
		}
		if got != expectedCode {
			t.Fatalf("argv=%q exit=%d want=%d stdout=%q stderr=%q", args, got, expectedCode, stdout.String(), stderr.String())
		}
		if stderr.Len() > 0 {
			t.Fatalf("argv=%q emitted stderr in the envelope path: %q", args, stderr.String())
		}
		return stdout.Bytes()
	}

	run := func(expectedCode int, args ...string) cliResultEnvelope {
		raw := runRaw(expectedCode, args...)
		var envelope cliResultEnvelope
		if err := json.Unmarshal(raw, &envelope); err != nil {
			t.Fatalf("argv=%q did not return envelope JSON: %v", args, err)
		}
		if envelope.SchemaVersion != "tei.cli.plugin-inventory/v0" {
			t.Fatalf("argv=%q schemaVersion=%q", args, envelope.SchemaVersion)
		}
		return envelope
	}

	lockPath := filepath.Join(t.TempDir(), "plugin-lock.json")
	lockRaw := runRaw(0, "plugin-lock", "-workspace", reference)
	if err := os.WriteFile(lockPath, lockJSONBytes(t, lockRaw), 0o644); err != nil {
		t.Fatal(err)
	}

	list := run(0, "plugin-inventory", "-workspace", reference, "-environment", "development")
	if !list.OK || list.Result == nil {
		t.Fatalf("unexpected list envelope: %+v", list)
	}
	if list.Mode != pluginInventoryModeList {
		t.Fatalf("list mode=%q want %q", list.Mode, pluginInventoryModeList)
	}
	var listShape map[string]json.RawMessage
	if err := reMarshal(list.Result, &listShape); err != nil {
		t.Fatalf("list result shape: %v", err)
	}
	for _, key := range []string{"schemaVersion", "environment", "scanState", "settingsVersion", "confirmed", "resolver", "rows", "totals", "notModelled", "messages", "authorityLimits"} {
		if _, ok := listShape[key]; !ok {
			t.Errorf("list result is missing %q", key)
		}
	}
	// TEI has no settings store yet: the field must be present and null.
	if string(listShape["settingsVersion"]) != "null" {
		t.Errorf("settingsVersion must be null, got %s", listShape["settingsVersion"])
	}
	if strings.Contains(string(mustJson(t, list.Result)), `"enabled"`) {
		t.Errorf("the inventory must not claim an enabled setting")
	}

	detail := run(0, "plugin-inventory", "-workspace", reference, "-environment", "development",
		"-plugin", "tei.database.postgres", "-plugin-lock", lockPath)
	if !detail.OK || detail.Mode != pluginInventoryModeDetail {
		t.Fatalf("unexpected detail envelope: %+v", detail)
	}
	var detailShape map[string]json.RawMessage
	if err := reMarshal(detail.Result, &detailShape); err != nil {
		t.Fatalf("detail result shape: %v", err)
	}
	for _, key := range []string{"row", "message", "resolver", "authorityLimits"} {
		if _, ok := detailShape[key]; !ok {
			t.Errorf("detail result is missing %q", key)
		}
	}
	var row struct {
		PluginID        string   `json:"pluginId"`
		Declared        bool     `json:"declared"`
		EnvSelected     bool     `json:"envSelected"`
		Resolved        bool     `json:"resolved"`
		LockPinned      *bool    `json:"lockPinned"`
		RuntimeBoundary string   `json:"runtimeBoundary"`
		ReasonCodes     []string `json:"reasonCodes"`
		NotModelled     []string `json:"notModelled"`
	}
	if err := reMarshal(detailShape["row"], &row); err != nil {
		t.Fatalf("detail row: %v", err)
	}
	if row.PluginID != "tei.database.postgres" {
		t.Fatalf("detail row is for %q", row.PluginID)
	}
	if !row.Declared || !row.EnvSelected || !row.Resolved {
		t.Fatalf("the reference plugin must be declared, selected and resolved: %+v", row)
	}
	if row.LockPinned == nil || !*row.LockPinned {
		t.Fatalf("the supplied lock pins this plugin: %+v", row)
	}
	if row.RuntimeBoundary != "in-process" {
		t.Fatalf("a builtin plugin runs in-process, got %q", row.RuntimeBoundary)
	}
	if !hasReason(row.NotModelled, "install") || !hasReason(row.NotModelled, "enable") {
		t.Fatalf("the row must name the install and enable gaps: %+v", row.NotModelled)
	}

	diagnose := run(0, "plugin-inventory", "-workspace", reference, "-environment", "production", "-diagnose")
	if !diagnose.OK || diagnose.Mode != pluginInventoryModeDiagnose {
		t.Fatalf("unexpected diagnose envelope: %+v", diagnose)
	}
	var diagnoseShape struct {
		Rows []struct {
			PluginID    string   `json:"pluginId"`
			ReasonCodes []string `json:"reasonCodes"`
		} `json:"rows"`
		Messages map[string]string `json:"messages"`
	}
	if err := reMarshal(diagnose.Result, &diagnoseShape); err != nil {
		t.Fatalf("diagnose result: %v", err)
	}
	for _, r := range diagnoseShape.Rows {
		if len(r.ReasonCodes) == 0 {
			t.Errorf("diagnose mode must only list rows with a reason code: %+v", r)
		}
	}
	for _, r := range diagnoseShape.Rows {
		for _, code := range r.ReasonCodes {
			if _, ok := diagnoseShape.Messages[code]; !ok {
				t.Errorf("reason code %q has no message", code)
			}
		}
	}

	// Argument failures keep the existing codes: no second failure vocabulary.
	if got := run(2, "plugin-inventory", "-workspace", reference); !got.OK && got.Error != nil && got.Error.Code == resultCodeInvalidArgument {
	} else {
		t.Fatalf("missing -environment must be INVALID_ARGUMENT: %+v", got)
	}
	if got := run(2, "plugin-inventory", "-workspace", reference, "-environment", "no_such_environment"); got.Error == nil || got.Error.Code != resultCodeUnknownEnvironment {
		t.Fatalf("unknown environment code: %+v", got)
	}
	if got := run(2, "plugin-inventory", "-workspace", reference, "-environment", "development", "-plugin-lock", filepath.Join(t.TempDir(), "absent.json")); got.Error == nil || got.Error.Code != resultCodeInvalidPluginLock {
		t.Fatalf("unreadable lock code: %+v", got)
	}
	if got := run(2, "plugin-inventory", "-workspace", reference, "-environment", "development", "-plugin", "tei.no.such.plugin"); got.Error == nil || got.Error.Code != resultCodeInvalidArgument {
		t.Fatalf("unknown plugin detail code: %+v", got)
	}
	if got := run(2, "plugin-inventory", "-workspace", reference, "-environment", "development", "-no-such-option", "x"); got.Error == nil || got.Error.Code != resultCodeInvalidArgument {
		t.Fatalf("unrelated flag code: %+v", got)
	}
}

func mustJson(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func hasReason(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// #2354: the result envelope commands must expose one machine-readable JSON
// shape for success and failure, at the real argv/exit/stdout boundary.
func TestResultEnvelopeCommandsJSONContract(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}

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
		wantVersion := "tei.cli." + args[0] + "/v0"
		if envelope.SchemaVersion != wantVersion {
			t.Fatalf("argv=%q schemaVersion=%q want %q", args, envelope.SchemaVersion, wantVersion)
		}
		return envelope
	}

	reference := filepath.Join("..", "..", "reference", "workspace")

	// world stays outside the envelope (#2354): its stdout is one bare
	// ir.World JSON, digested byte-for-byte by the critical-path evidence audit.
	worldRaw := runRaw(0, "world", "-workspace", reference)
	var worldBare map[string]any
	if err := json.Unmarshal(worldRaw, &worldBare); err != nil {
		t.Fatalf("world did not return JSON: %v", err)
	}
	if _, enveloped := worldBare["schemaVersion"]; enveloped || worldBare["applications"] == nil {
		t.Fatalf("world must stay a bare ir.World write, got: %.80s", worldRaw)
	}

	filesEnvelope := run(0, "workspace-files", "-workspace", reference)
	if !filesEnvelope.OK || filesEnvelope.Result == nil {
		t.Fatalf("unexpected workspace-files envelope: %+v", filesEnvelope)
	}

	wholeIndex := run(0, "source-index", "-workspace", reference)
	if !wholeIndex.OK || wholeIndex.Mode != resultModeIndex || wholeIndex.Result == nil {
		t.Fatalf("unexpected whole-index envelope: %+v", wholeIndex)
	}
	subjectIndex := run(0, "source-index", "-workspace", reference, "-subject-kind", "query", "-subject-id", "order_entry.recent_orders")
	if !subjectIndex.OK || subjectIndex.Mode != resultModeSubject || subjectIndex.Result == nil {
		t.Fatalf("unexpected subject envelope: %+v", subjectIndex)
	}

	requirementsEnvelope := run(0, "requirements", "-workspace", reference)
	if !requirementsEnvelope.OK || requirementsEnvelope.Result == nil {
		t.Fatalf("unexpected requirements envelope: %+v", requirementsEnvelope)
	}

	lockEnvelope := run(0, "plugin-lock", "-workspace", reference)
	if !lockEnvelope.OK || lockEnvelope.Result == nil {
		t.Fatalf("unexpected plugin-lock envelope: %+v", lockEnvelope)
	}

	resolveEnvelope := run(0, "plugin-resolve", "-workspace", reference, "-environment", "development")
	if !resolveEnvelope.OK || resolveEnvelope.Result == nil {
		t.Fatalf("unexpected plugin-resolve envelope: %+v", resolveEnvelope)
	}

	lockPath := filepath.Join(t.TempDir(), "plugin-lock.json")
	lockRaw := runRaw(0, "plugin-lock", "-workspace", reference)
	if err := os.WriteFile(lockPath, lockJSONBytes(t, lockRaw), 0o644); err != nil {
		t.Fatal(err)
	}
	planEnvelope := run(0, "plugin-runtime-plan", "-workspace", reference, "-environment", "development", "-plugin-lock", lockPath)
	if !planEnvelope.OK || planEnvelope.Result == nil {
		t.Fatalf("unexpected plugin-runtime-plan envelope: %+v", planEnvelope)
	}

	loadFailure := run(2, "workspace-files", "-workspace", filepath.Join(t.TempDir(), "absent"))
	if loadFailure.OK || loadFailure.Error == nil || loadFailure.Error.Code != resultCodeWorkspaceLoadFailed {
		t.Fatalf("wrong load failure envelope: %+v", loadFailure)
	}

	unknownEnv := run(2, "plugin-resolve", "-workspace", reference, "-environment", "no_such_environment")
	if unknownEnv.OK || unknownEnv.Error == nil || unknownEnv.Error.Code != resultCodeUnknownEnvironment {
		t.Fatalf("wrong unknown environment envelope: %+v", unknownEnv)
	}

	badLock := run(2, "plugin-resolve", "-workspace", reference, "-environment", "development", "-plugin-lock", "no-such-lock.json")
	if badLock.OK || badLock.Error == nil || badLock.Error.Code != resultCodeInvalidPluginLock {
		t.Fatalf("wrong plugin lock failure envelope: %+v", badLock)
	}

	worldFlagCmd := exec.Command(binary, "world", "-no-such-option", "x")
	var worldFlagStdout, worldFlagStderr bytes.Buffer
	worldFlagCmd.Stdout, worldFlagCmd.Stderr = &worldFlagStdout, &worldFlagStderr
	if err := worldFlagCmd.Run(); err != nil {
		var worldFlagExit *exec.ExitError
		if !errors.As(err, &worldFlagExit) || worldFlagExit.ExitCode() != 2 {
			t.Fatalf("world flag failure exit: %v", err)
		}
	} else {
		t.Fatal("world flag failure must exit non-zero")
	}
	if worldFlagStdout.Len() != 0 || worldFlagStderr.Len() == 0 {
		t.Fatalf("world flag failure must stay the stderr usage exit (stdout %.80q stderr %.80q)",
			worldFlagStdout.String(), worldFlagStderr.String())
	}

	requirementsFlag := run(2, "requirements", "-no-such-option", "x")
	if requirementsFlag.OK || requirementsFlag.Error == nil || requirementsFlag.Error.Code != resultCodeInvalidArgument {
		t.Fatalf("wrong flag failure envelope: %+v", requirementsFlag)
	}

	unrelatedFlag := run(2, "workspace-files", "-workspace", reference, "-environment", "dev")
	if unrelatedFlag.OK || unrelatedFlag.Error == nil || unrelatedFlag.Error.Code != resultCodeInvalidArgument {
		t.Fatalf("wrong unrelated flag envelope: %+v", unrelatedFlag)
	}

	bothSources := run(2, "workspace-files", "-workspace", reference, "-world", "other.json")
	if bothSources.OK || bothSources.Error == nil || bothSources.Error.Code != resultCodeInvalidArgument {
		t.Fatalf("wrong both-sources envelope: %+v", bothSources)
	}
}

func lockJSONBytes(t *testing.T, raw []byte) []byte {
	t.Helper()
	var envelope cliResultEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("plugin-lock envelope: %v", err)
	}
	lock, err := json.Marshal(envelope.Result)
	if err != nil {
		t.Fatal(err)
	}
	return lock
}

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

// TestPluginPlanEnvelopeAndRefusals covers the CLI boundary of `plugin-plan`
// (#4377): the plan is a proposal, the destructive actions are refused until
// their approval and recovery path exist, and no argument failure invents a
// new code.
func TestPluginPlanEnvelopeAndRefusals(t *testing.T) {
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
		t.Helper()
		raw := runRaw(expectedCode, args...)
		var envelope cliResultEnvelope
		if err := json.Unmarshal(raw, &envelope); err != nil {
			t.Fatalf("argv=%q did not return envelope JSON: %v", args, err)
		}
		if envelope.SchemaVersion != "tei.cli.plugin-plan/v0" {
			t.Fatalf("argv=%q schemaVersion=%q", args, envelope.SchemaVersion)
		}
		return envelope
	}

	lockPath := filepath.Join(t.TempDir(), "plugin-lock.json")
	lockRaw := runRaw(0, "plugin-lock", "-workspace", reference)
	if err := os.WriteFile(lockPath, lockJSONBytes(t, lockRaw), 0o644); err != nil {
		t.Fatal(err)
	}

	disable := run(0, "plugin-plan", "-workspace", reference, "-environment", "development",
		"-plan-action", "disable-plugin", "-plan-plugin", "tei.editor.intelligence.sql")
	if !disable.OK || disable.Result == nil || disable.Mode != pluginPlanMode {
		t.Fatalf("unexpected plan envelope: %+v", disable)
	}
	var plan struct {
		SchemaVersion       string   `json:"schemaVersion"`
		Action              string   `json:"action"`
		Requested           string   `json:"requested"`
		Candidates          []string `json:"candidates"`
		Unchanged           []string `json:"unchanged"`
		Refused             bool     `json:"refused"`
		Confirmation        bool     `json:"confirmationRequired"`
		SourcesPreserved    bool     `json:"sourcesPreserved"`
		DataPreserved       bool     `json:"dataPreserved"`
		SelectionChanged    bool     `json:"selectionChanged"`
		Digest              string   `json:"digest"`
		Restart             []string `json:"restartRequired"`
		NotModelled         []string `json:"notModelled"`
		OptionalNotModelled bool     `json:"optionalDependencyNotModelled"`
		UnattendedUnsafe    []string `json:"unattendedUnsafe"`
		AuthorityLimits     []string `json:"authorityLimits"`
	}
	if err := reMarshal(disable.Result, &plan); err != nil {
		t.Fatalf("plan result: %v", err)
	}
	if plan.SchemaVersion != "tei.plugin-management.plan/v0" || plan.Action != "disable-plugin" {
		t.Fatalf("plan identity: %+v", plan)
	}
	if plan.Requested != "tei.editor.intelligence.sql" {
		t.Fatalf("plan requested = %q", plan.Requested)
	}
	if plan.Refused {
		t.Fatalf("a disable of an unrequired plugin must be proposed")
	}
	if !plan.SourcesPreserved || !plan.DataPreserved {
		t.Fatalf("disable must preserve Source and data: %+v", plan)
	}
	if plan.SelectionChanged {
		t.Fatalf("the plan must not change the selection")
	}
	if plan.Digest == "" {
		t.Fatalf("the plan must carry a digest")
	}
	if len(plan.Restart) == 0 {
		t.Fatalf("a static target needs a restart and the plan must name it")
	}
	if !plan.OptionalNotModelled {
		t.Fatalf("optional dependencies are not modelled and the plan must say so")
	}
	if len(plan.AuthorityLimits) == 0 {
		t.Fatalf("the plan must name what it does not guarantee")
	}
	if !containsPlanValue(plan.Candidates, "tei.editor.intelligence.sql") {
		t.Fatalf("candidates = %+v", plan.Candidates)
	}
	if len(plan.Unchanged) == 0 {
		t.Fatalf("unrelated plugins must be listed as unchanged: %+v", plan)
	}

	// Data deletion and conversion are refused: no approval or recovery path
	// exists yet (#4380).
	for _, action := range []string{"delete-data", "convert-data"} {
		result := run(0, "plugin-plan", "-workspace", reference, "-environment", "development",
			"-plan-action", action, "-plan-plugin", "tei.database.postgres")
		if !result.OK {
			t.Fatalf("%s must answer a refusal inside a successful envelope: %+v", action, result)
		}
		var refused struct {
			Refused          bool     `json:"refused"`
			Reasons          []string `json:"reasons"`
			UnattendedUnsafe []string `json:"unattendedUnsafe"`
			DataPreserved    bool     `json:"dataPreserved"`
		}
		if err := reMarshal(result.Result, &refused); err != nil {
			t.Fatalf("%s result: %v", action, err)
		}
		if !refused.Refused {
			t.Fatalf("%s must be refused", action)
		}
		if !containsPlanValue(refused.Reasons, "requires-owner-approval") ||
			!containsPlanValue(refused.Reasons, "recovery-path-not-established") {
			t.Fatalf("%s reasons = %+v", action, refused.Reasons)
		}
		if !containsPlanValue(refused.UnattendedUnsafe, action) {
			t.Fatalf("%s must be listed as unsafe unattended: %+v", action, refused.UnattendedUnsafe)
		}
		if refused.DataPreserved {
			t.Fatalf("%s must not claim data is preserved", action)
		}
	}

	// An update without the target lock is refused, not guessed.
	update := run(0, "plugin-plan", "-workspace", reference, "-environment", "development",
		"-plan-action", "update-plugin", "-plan-plugin", "tei.database.postgres", "-plugin-lock", lockPath)
	var updatePlan struct {
		Refused bool     `json:"refused"`
		Reasons []string `json:"reasons"`
	}
	if err := reMarshal(update.Result, &updatePlan); err != nil {
		t.Fatalf("update result: %v", err)
	}
	if !updatePlan.Refused || !containsPlanValue(updatePlan.Reasons, "target-lock-required") {
		t.Fatalf("update must refuse without the target lock: %+v", updatePlan)
	}

	// Argument failures keep the existing codes.
	if got := run(2, "plugin-plan", "-workspace", reference, "-environment", "development", "-plan-plugin", "tei.database.postgres"); got.Error == nil || got.Error.Code != resultCodeInvalidArgument {
		t.Fatalf("missing -plan-action must be INVALID_ARGUMENT: %+v", got)
	}
	if got := run(2, "plugin-plan", "-workspace", reference, "-environment", "development", "-plan-action", "nonsense", "-plan-plugin", "tei.database.postgres"); got.Error == nil || got.Error.Code != resultCodeInvalidArgument {
		t.Fatalf("an unknown action must be INVALID_ARGUMENT: %+v", got)
	}
	if got := run(2, "plugin-plan", "-workspace", reference, "-environment", "no_such_environment", "-plan-action", "disable-plugin", "-plan-plugin", "tei.database.postgres"); got.Error == nil || got.Error.Code != resultCodeUnknownEnvironment {
		t.Fatalf("unknown environment code: %+v", got)
	}
	if got := run(2, "plugin-plan", "-workspace", reference, "-environment", "development", "-plan-action", "disable-plugin", "-plan-plugin", "tei.database.postgres", "-plugin-lock", filepath.Join(t.TempDir(), "absent.json")); got.Error == nil || got.Error.Code != resultCodeInvalidPluginLock {
		t.Fatalf("unreadable lock code: %+v", got)
	}
	if got := run(2, "plugin-plan", "-workspace", reference, "-environment", "development", "-plan-action", "disable-plugin", "-plan-plugin", "tei.database.postgres", "-no-such-option", "x"); got.Error == nil || got.Error.Code != resultCodeInvalidArgument {
		t.Fatalf("unrelated flag code: %+v", got)
	}
}

// TestPluginPlanCannotBeMistakenForAnApply keeps the word "apply" out of the
// command surface: the command has no flag that applies a plan.
func TestPluginPlanCannotBeMistakenForAnApply(t *testing.T) {
	record := func() (cliCommandRecord, bool) {
		for _, cmd := range commandCatalog().Commands {
			if cmd.Name == "plugin-plan" {
				return cmd, true
			}
		}
		return cliCommandRecord{}, false
	}
	cmd, ok := record()
	if !ok {
		t.Fatalf("plugin-plan must be advertised")
	}
	for _, text := range append(append([]string{}, cmd.Required...), cmd.Optional...) {
		if strings.Contains(text, "apply") || strings.Contains(text, "confirm") {
			t.Fatalf("plugin-plan must not advertise an applying flag: %q", text)
		}
	}
	if cmd.Effect != "read-only" {
		t.Fatalf("plugin-plan effect = %q, want read-only", cmd.Effect)
	}
}

func containsPlanValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

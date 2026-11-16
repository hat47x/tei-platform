package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/plugin"
	"tei/internal/plugin/management"
)

// TestPluginPlanCommandsAdvertisement checks that `plugin-plan` appears in
// `tei commands` with exactly the flags it accepts and no write flags, since it
// is a read-only command that creates and modifies nothing.
func TestPluginPlanCommandsAdvertisement(t *testing.T) {
	binary := buildCLIForLifecycle(t)
	commandsRaw, code := runCLIProcess(t, binary, "commands")
	if code != 0 {
		t.Fatalf("tei commands exit=%d body=%s", code, commandsRaw)
	}
	var catalog cliCommandCatalog
	if err := json.Unmarshal(commandsRaw, &catalog); err != nil {
		t.Fatalf("tei commands is not the advertised catalog: %v", err)
	}
	var record *cliCommandRecord
	for i := range catalog.Commands {
		if catalog.Commands[i].Name == "plugin-plan" {
			record = &catalog.Commands[i]
		}
	}
	if record == nil {
		t.Fatal("plugin-plan is not advertised by tei commands")
	}
	if record.Effect != "read-only" || record.SchemaVersion != "tei.cli.plugin-plan/v0" || record.JSONContract != "enveloped" {
		t.Fatalf("plugin-plan record: %+v", record)
	}
	if len(record.WriteFlags) != 0 {
		t.Fatalf("a read-only plan must advertise no writeFlags, got %v", record.WriteFlags)
	}
	got := append(append([]string{}, record.RequiredFlags...), record.OptionalFlags...)
	sort.Strings(got)
	want := []string{"action", "environment", "plugin", "plugin-lock", "source-index", "target-lock", "workspace", "world"}
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("plugin-plan advertises flags %v, want %v", got, want)
	}
	// The closed action set is advertised, and there is no apply.
	sets := record.ArgumentValues["action"]
	if len(sets) != 5 {
		t.Fatalf("plugin-plan must advertise the five actions, got %v", sets)
	}
	for _, a := range sets {
		if a == "apply" {
			t.Fatal("plugin-plan must not advertise an apply action")
		}
	}
}

// TestPluginPlanReasonCodesMatchManagement pins the plan vocabulary to the
// management vocabulary (#4376): internal/operations cannot import
// internal/plugin/management (it transitively imports net), so the shared codes
// are referenced by value and pinned equal here, in a package that sees both.
func TestPluginPlanReasonCodesMatchManagement(t *testing.T) {
	overlap := map[string]string{
		operations.PlanReasonPluginNotDeclared:         management.ReasonPluginNotDeclared,
		operations.PlanReasonEnvNotSelected:            management.ReasonEnvNotSelected,
		operations.PlanReasonVersionUnsupported:        management.ReasonVersionUnsupported,
		operations.PlanReasonDependencyCycle:           management.ReasonDependencyCycle,
		operations.PlanReasonDependencyUnresolved:      management.ReasonDependencyUnresolved,
		operations.PlanReasonContractIncompatible:      management.ReasonContractIncompatible,
		operations.PlanReasonLockNotProvided:           management.ReasonLockNotProvided,
		operations.PlanReasonSourceFingerprintMismatch: management.ReasonSourceFingerprintMismatch,
	}
	for planCode, mgmtCode := range overlap {
		if planCode != mgmtCode {
			t.Errorf("plan reason %q drifted from management reason %q", planCode, mgmtCode)
		}
	}
}

// digestTree is a stable digest of every file under root, used to prove the
// read-only command wrote nothing.
func digestTree(t *testing.T, root string) string {
	t.Helper()
	var lines []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		sum := sha256.Sum256(raw)
		lines = append(lines, rel+" "+hex.EncodeToString(sum[:]))
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

// TestPluginPlanCLIEnvelopesAndNonMutation runs the real binary against the
// reference Workspace: a disable plan and two update plans, and proves the
// on-disk Workspace tree is byte-identical before and after every run.
func TestPluginPlanCLIEnvelopesAndNonMutation(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	reference := filepath.Join("..", "..", "reference", "workspace")

	run := func(expectedCode int, args ...string) cliResultEnvelope {
		t.Helper()
		cmd := exec.Command(binary, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		got := 0
		if err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("argv=%q: %v", args, err)
			}
			got = exit.ExitCode()
		}
		if got != expectedCode {
			t.Fatalf("argv=%q exit=%d want=%d stdout=%q stderr=%q", args, got, expectedCode, stdout.String(), stderr.String())
		}
		if stderr.Len() > 0 {
			t.Fatalf("argv=%q wrote stderr on the envelope path: %q", args, stderr.String())
		}
		var envelope cliResultEnvelope
		if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
			t.Fatalf("argv=%q is not one envelope: %v (%q)", args, err, stdout.String())
		}
		if envelope.SchemaVersion != "tei.cli.plugin-plan/v0" {
			t.Fatalf("argv=%q schemaVersion=%q", args, envelope.SchemaVersion)
		}
		return envelope
	}

	before := digestTree(t, reference)

	// A disable plan for a provider nothing requires: only the target changes.
	disable := run(0, "plugin-plan", "-workspace", reference, "-environment", "development",
		"-action", "disable", "-plugin", "tei.database.oracle")
	if !disable.OK || disable.Result == nil {
		t.Fatalf("disable plan envelope: %+v", disable)
	}
	var disablePlan operations.Plan
	if err := reMarshal(disable.Result, &disablePlan); err != nil {
		t.Fatalf("disable plan shape: %v", err)
	}
	if disablePlan.SchemaVersion != operations.PluginPlanSchemaVersion {
		t.Fatalf("plan schemaVersion=%q", disablePlan.SchemaVersion)
	}
	if disablePlan.Action != "disable" || disablePlan.Requested.Plugin != "tei.database.oracle" {
		t.Fatalf("disable request: %+v", disablePlan.Requested)
	}
	if len(disablePlan.Candidates) != 1 || disablePlan.Candidates[0] != "tei.database.oracle" {
		t.Fatalf("disable candidates=%v, want [tei.database.oracle]", disablePlan.Candidates)
	}
	if len(disablePlan.Additional) != 0 || disablePlan.ConfirmationRequired {
		t.Fatalf("nothing requires the oracle provider: additional=%v confirm=%t", disablePlan.Additional, disablePlan.ConfirmationRequired)
	}
	if !disablePlan.SourcesPreserved || !disablePlan.DataPreserved || disablePlan.SelectionChanged {
		t.Fatalf("disable must preserve sources and data: %+v", disablePlan)
	}
	if len(disablePlan.Digest) == 0 {
		t.Fatal("a plan must carry a digest")
	}

	// An update plan without a target lock names the missing input rather than
	// inventing a diff (the issue's verification invocation).
	updateNoTarget := run(0, "plugin-plan", "-workspace", reference, "-environment", "development",
		"-action", "update", "-plugin", "tei.database.postgres")
	var updateNoTargetPlan operations.Plan
	if err := reMarshal(updateNoTarget.Result, &updateNoTargetPlan); err != nil {
		t.Fatalf("update plan shape: %v", err)
	}
	if updateNoTargetPlan.Diff != nil {
		t.Fatalf("no target lock must yield no diff, got %+v", updateNoTargetPlan.Diff)
	}
	if !hasPlanCode(updateNoTargetPlan, operations.PlanReasonLockNotProvided) {
		t.Fatalf("want lock-not-provided, got %+v", updateNoTargetPlan.Reasons)
	}

	// A real update diff: the current lock is the lock the workspace resolves
	// to; the target lock is the same lock with the postgres artifact digest
	// moved, which is a build-input change requiring regeneration and a
	// restart for the in-process binary.
	dir := t.TempDir()
	lockRaw, code := runCLIProcess(t, binary, "plugin-lock", "-workspace", reference)
	if code != 0 {
		t.Fatalf("plugin-lock exit=%d body=%s", code, lockRaw)
	}
	currentLockPath := filepath.Join(dir, "current-lock.json")
	if err := os.WriteFile(currentLockPath, lockJSONBytes(t, lockRaw), 0o600); err != nil {
		t.Fatal(err)
	}
	var realLock plugin.Lock
	if err := json.Unmarshal(lockJSONBytes(t, lockRaw), &realLock); err != nil {
		t.Fatalf("decode real lock: %v", err)
	}
	targetLock := realLock
	for ei := range targetLock.Environments {
		if targetLock.Environments[ei].Environment != "development" {
			continue
		}
		for pi := range targetLock.Environments[ei].Plugins {
			if targetLock.Environments[ei].Plugins[pi].ID == "tei.database.postgres" {
				targetLock.Environments[ei].Plugins[pi].ArtifactDigest = "sha256:" + strings.Repeat("c", 64)
			}
		}
	}
	targetRaw, err := json.Marshal(targetLock)
	if err != nil {
		t.Fatal(err)
	}
	targetLockPath := filepath.Join(dir, "target-lock.json")
	if err := os.WriteFile(targetLockPath, targetRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	updateDiff := run(0, "plugin-plan", "-workspace", reference, "-environment", "development",
		"-action", "update", "-plugin", "tei.database.postgres",
		"-plugin-lock", currentLockPath, "-target-lock", targetLockPath)
	var updateDiffPlan operations.Plan
	if err := reMarshal(updateDiff.Result, &updateDiffPlan); err != nil {
		t.Fatalf("update diff shape: %v", err)
	}
	if updateDiffPlan.Diff == nil || !updateDiffPlan.Diff.ArtifactChanged || updateDiffPlan.Diff.VersionChanged {
		t.Fatalf("update diff must be an artifact change only: %+v", updateDiffPlan.Diff)
	}
	if len(updateDiffPlan.RegenerationRequired) != 1 || updateDiffPlan.RegenerationRequired[0] != "tei.database.postgres" {
		t.Fatalf("a build-input change must require regeneration, got %v", updateDiffPlan.RegenerationRequired)
	}
	if len(updateDiffPlan.RestartRequired) != 1 {
		t.Fatalf("an in-process plugin update must require a restart, got %v", updateDiffPlan.RestartRequired)
	}

	// Argument failures keep the machine-readable codes.
	if got := run(2, "plugin-plan", "-workspace", reference, "-environment", "development", "-action", "apply", "-plugin", "tei.database.oracle"); !got.OK && got.Error != nil && got.Error.Code == resultCodeInvalidArgument {
	} else {
		t.Fatalf("apply must be refused as INVALID_ARGUMENT: %+v", got)
	}
	if got := run(2, "plugin-plan", "-workspace", reference, "-environment", "development", "-action", "disable"); got.Error == nil || got.Error.Code != resultCodeInvalidArgument {
		t.Fatalf("missing -plugin must be INVALID_ARGUMENT: %+v", got)
	}
	if got := run(2, "plugin-plan", "-environment", "development", "-action", "disable", "-plugin", "tei.database.oracle"); got.Error == nil || got.Error.Code != resultCodeInvalidArgument {
		t.Fatalf("missing -workspace/-world must be INVALID_ARGUMENT: %+v", got)
	}
	if got := run(2, "plugin-plan", "-workspace", reference, "-environment", "no_such_environment", "-action", "disable", "-plugin", "tei.database.oracle"); got.Error == nil || got.Error.Code != resultCodeUnknownEnvironment {
		t.Fatalf("unknown environment code: %+v", got)
	}

	// Nothing was written: the Workspace tree is byte-identical.
	if after := digestTree(t, reference); after != before {
		t.Fatalf("plugin-plan modified the Workspace tree: before=%s after=%s", before, after)
	}
}

func hasPlanCode(plan operations.Plan, code string) bool {
	for _, r := range plan.Reasons {
		if r.Code == code {
			return true
		}
	}
	return false
}
